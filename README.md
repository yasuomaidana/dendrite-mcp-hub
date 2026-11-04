# dendrite-mcp-hub

Go unified MCP host bridging JetBrains IDEs, Penpot MCP, Plane AI, Engram and Dendrite tools (Pillar 5/9)

`dendrite-mcp-hub` (`github.com/yasuomaidana/dendrite-mcp-hub`) provides the Model Context Protocol (MCP) JSON-RPC 2.0 server and tool registry (`pkg/mcpserver`), the reverse IDE client and configuration bridge (`pkg/jetbrains`), and cross-language IPC contract drift monitoring (`pkg/polyglot/contract`).
It aggregates and proxies upstream MCP servers and external tool connectors—including Penpot UI design-token synchronization, Plane v2 AI/issue workflows, Engram topic memory, REA disassemblers, package registry publishers, and local Code Knowledge (`graphify`) queries.
It also coordinates the multi-IDE plugin adapters (JetBrains, VS Code, Neovim, Zed) while delegating control-plane orchestration to `dendrite-control` and AST, Git, and RAG compute to the Rust sidecar crates (`dendrite-core-rs`, `dendrite-git-rs`, `dendrite-rag-rs`).
See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for the technical specification and [docs/RESPONSIBILITIES.md](docs/RESPONSIBILITIES.md) for the ownership boundary, MCP tool-family map and Plane (`DMCP`) ticket tracking.

## Status of 0.1.0

| Package | State |
|---|---|
| `pkg/jetbrains` | complete: reverse IDE client, merge-safe MCP registration with per-synapse scopes, run configurations; 100% test coverage |
| `pkg/polyglot/contract` | complete: cross-language IPC contract drift monitor; 100% test coverage |
| `pkg/mcpserver` | the server core and types only. The tool implementations (about 11,000 lines) still live in the Dendrite control repository and move here under DMCP-1 once the hub defines ports for what they need; the hub never imports the control repository |

## Build and test

```bash
go build ./... && go vet ./... && go test ./... -cover
```

See [CHANGELOG.md](CHANGELOG.md) for what each release contains.
