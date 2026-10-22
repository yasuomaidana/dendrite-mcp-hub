# dendrite-mcp-hub Architecture

`dendrite-mcp-hub` (`github.com/yasuomaidana/dendrite-mcp-hub`) is a Go library with three jobs:

- an MCP server with a tool registry,
- an MCP client and bridge into JetBrains IDEs,
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

## 2. JetBrains bridge: `pkg/jetbrains`

- **Transports:**
  - `HTTPTransport` posts JSON-RPC to the IDE's built-in MCP server (`http://localhost:63342/api/mcp`, configurable).
  - `StdioTransport` talks to a stdio MCP process.
  - Both implement `Transport.Send(ctx, req) (resp, error)`.
- **Client:**
  - `NewClient(transport, WithClientInfo, WithProtocolVersion)`.
  - `Connect` performs `initialize` + `notifications/initialized` and records server info and capabilities.
  - Also: `ListTools`, `CallTool(name, args)`, `Ping` (returns the round trip), `Disconnect`.
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
- Upstream MCP services (Penpot, Plane, Engram, disassembler bridges) are planned as additional clients alongside the JetBrains one, each behind the same `Transport` interface. Their tools are re-exported through the registry.

## 5. Consumers and siblings

- **dendrite-control** registers its tool catalog on `mcpserver.Server`, including tools backed by **dendrite-core-rs**, **dendrite-rag-rs** and **dendrite-git-rs** through its sidecar client. It also uses `pkg/jetbrains` for the `bridge` CLI commands and IDE setup.
- IDE plugins and agent CLIs reach the hub through the MCP configs written by `SetupJetBrainsToolset`.

See [RESPONSIBILITIES.md](RESPONSIBILITIES.md) for the ownership boundary, the tool-family map and the ticket map.
