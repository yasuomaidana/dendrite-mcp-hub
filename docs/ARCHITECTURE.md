# dendrite-mcp-hub Architecture

`dendrite-mcp-hub` (`github.com/yasuomaidana/dendrite-mcp-hub`) is a Go library with three jobs:

- an MCP server with a tool registry,
- a generic outbound MCP client (`pkg/mcpclient`, stdio and streamable HTTP), an HTTP server handler for the registry (`pkg/mcphttp`) and a bridge into JetBrains IDEs built on the client,
- a cross-language contract drift monitor.

It has no `main` package. Hosts (the `dendrite` CLI's `inter serve` command, IDE plugins) build a server, register tools and serve it.

## 1. MCP server: `pkg/mcpserver`

- **Protocol:** MCP `2024-11-05` over JSON-RPC 2.0. Methods: `initialize`, `notifications/initialized`, `ping`, `tools/list`, `tools/call`. Unknown methods return `-32601`, malformed JSON `-32700`, bad params `-32602`.
- **Registry:** `Server` keeps `map[name]Tool` and `map[name]ToolHandler` behind an `RWMutex`.
  - `RegisterTool(tool, handler)` replaces any existing entry, so hosts can override a default.
  - `ListTools` returns every registered tool (in map order).
  - `Handler(name)` lets hosts compose tools.
- **Calls:** `tools/call` decodes `CallToolParams{name, arguments}` and invokes the handler with the request context. A handler error becomes a `CallToolResult{isError: true}` with the message as text content. It is not a JSON-RPC error, so the client model sees why the tool failed.
- **Transport:** `ServeStdio(ctx, in, out)` reads one JSON message per line (scanner buffer up to 10 MB for large tool arguments) and writes one response per line. Notifications get no response. It stops on EOF or context cancellation.
- **Identity:** `ServerConfig{Name, Version}`, defaulting to `Dendrite MCP Hub` / `0.1.0`.

## 2. Generic MCP client: `pkg/mcpclient`

It speaks to any MCP server and knows nothing about JetBrains or any other one.

- **Transports** implement `Transport{Send, Notify, Close}`. `Send` waits for the response with the same id; `Notify` writes a notification and returns, because a server never answers one.
  - `HTTPTransport` posts JSON-RPC to a server's HTTP endpoint (default: the IDE's built-in `http://localhost:63342/api/mcp`).
  - `StdioTransport` runs an MCP server as a subprocess:
    - One reader goroutine routes each response to the call waiting for its id, so calls may overlap and answers may arrive in any order.
    - Lines that are not responses (server notifications, server requests, log output on stdout) are skipped.
    - A call ends with its context, or after `DefaultCallTimeout` (30 s) when the context has no deadline, or when the child exits. The error quotes the last 4 KB of the child's stderr.
    - `Close` closes stdin, gives the child 2 s to leave, then kills it and always reaps it.
- **Client:**
  - `NewClient(transport, WithClientInfo, WithProtocolVersion)`.
  - `Connect` performs `initialize`, then sends `notifications/initialized` with `Notify` and records server info and capabilities. (Before DMCP-34 the notification went through `Send`, which waited for a reply a real server never sends, so the handshake hung.)
  - Also: `ListTools`, `CallTool(name, args)`, `Ping` (returns the round trip), `Disconnect`.
- **Verified** against a real stdio MCP server built on `pkg/mcpserver`, and live against `engram mcp --tools=agent` (23 `mem_*` tools listed, `mem_context` answered).

## 2b. Streamable HTTP: `pkg/mcphttp` and `StreamableHTTPTransport` (DMCP-37)

The official [go-sdk](https://github.com/modelcontextprotocol/go-sdk) carries the streamable-HTTP transport of the MCP specification (sessions through `Mcp-Session-Id`, JSON or SSE answers, stream resumption, DNS-rebinding protection). The hand-rolled stdio server and client stay as they are; the go-sdk is used for HTTP only (decision of DEN-433).

- **Server:** `mcphttp.Handler(srv, Options{JSONResponse})` returns an `http.Handler`. It builds a go-sdk server whose tools call the hub server's handlers, so every tool registered with `RegisterTool` is served over HTTP with the same behaviour as over stdio: a handler error becomes an `isError` result (`Tool error: ...`), image items travel as base64, and a schema root is always `{"type":"object"}`. Register every tool before calling `Handler`; tools added later are not served.
- **Client:** `mcpclient.NewStreamableHTTPTransport(endpoint, httpClient)` implements `Transport` on the go-sdk client connection. As with `StdioTransport`, one reader goroutine matches each response to its call by id, so calls may overlap; a call ends with its context or after `DefaultCallTimeout`. `Close` ends the session (the go-sdk sends the DELETE) and wakes waiting calls with `ErrTransportClosed`; it returns after 2 s at the latest, even while the server is busy with a call.
- **Tests:** the client and the server are tested against each other (both response modes), plus a lost session, timeouts, `Close` under a waiting call and a connection that dies.

## 2a. JetBrains bridge: `pkg/jetbrains`

`pkg/jetbrains` re-exports the client types from `pkg/mcpclient` (`NewClient` keeps the identity `dendrite-jetbrains-bridge`) and adds what is specific to the IDE:

- **Bridge:** typed wrappers over the IDE's tools:
  - `OpenFile(path, line, col)`
  - `GetFileProblems(path) → []Problem`
  - `SearchSymbol(query) → []Symbol`
  - `BuildProject → BuildResult`
  - `ExecuteRunConfiguration(name) → output`
  - `LintFiles(paths) → []Problem`

  Text content is parsed into the typed results. Empty or unstructured content falls back to the raw text rather than being invented. When the IDE is not running, calls fail with a connection error instead of returning placeholder diagnostics.
- **Provisioning:** `SetupJetBrainsToolset(SetupOptions)` writes:
  - the project and user MCP configs (`.ai/mcp/mcp.json`, `~/.ai/mcp/mcp.json`) registering the host binary as an MCP server,
  - `.idea/runConfigurations/*.xml`, named after the host's commands,
  - `.idea/tools/External_Tools.xml`.

  It reports what was created versus left untouched. `UpsertMCPServer` / `RemoveMCPServer` edit one server entry in an `mcp.json` without disturbing the others.

## 3. Contract drift monitor: `pkg/polyglot/contract`

Detects interface drift between a canonical schema and the data models that implement it in each language.

1. **Canonical schemas:** `ScanSchemas(dirs)` parses Protobuf messages, JSON Schema objects and OpenAPI component schemas into `ContractSchema{name, fields: [{name, type, required}]}`.
2. **Code models:** `ScanCodeModels({lang: dir})` parses Go structs (JSON tags), Rust serde structs (`rename`), TypeScript interfaces and Python Pydantic models into the same shape.
3. **Normalization:** `NormalizeType` maps language types onto a shared vocabulary (`string`, `int32`, `int64`, `float32`, `float64`, `bool`, `bytes`, `array`, `map`), unwrapping `Option<…>`/optional markers into an `optional` flag. Field names are compared after applying serialization renames.
4. **Comparison:** `CompareSchemas` / `CompareAll` produce `DriftViolation`s: `FIELD_MISSING`, `TYPE_MISMATCH`, `SERIALIZATION_TAG_MISMATCH` and `OPTIONALITY_MISMATCH`, each with the file and language. They are collected in a `DriftReport`.
5. **IDE notification:** `NotifyIDE(report)` opens each violating file at the offending declaration through the JetBrains bridge.

## 4. Design constraints

- Standard library only for the protocol core. No global state, so several servers can coexist in one process.
- Tool handlers are supplied by the host. The hub never imports the host's packages, which keeps the module dependency one-way (host → hub).
- Upstream MCP services (Penpot, Plane, disassembler bridges) can be reached with `pkg/mcpclient`. Re-exporting an upstream's tools through the registry is not built. Engram is not an upstream: `dendrite soma install engram` registers `engram mcp --tools=agent` directly with Claude Code and Antigravity (DEN-427), which avoids one Engram subprocess per `inter serve` process.

## 5. Consumers and siblings

- **dendrite-control** registers its tool catalog on `mcpserver.Server`, including tools backed by **dendrite-core-rs**, **dendrite-rag-rs** and **dendrite-git-rs** through its sidecar client. It also uses `pkg/jetbrains` for the `bridge` CLI commands and IDE setup.
- IDE plugins and agent CLIs reach the hub through the MCP configs written by `SetupJetBrainsToolset`.

See [RESPONSIBILITIES.md](RESPONSIBILITIES.md) for the ownership boundary, the tool-family map and the ticket map.
