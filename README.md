# dendrite-mcp-hub

Go unified MCP host bridging JetBrains IDEs, Penpot MCP, Plane AI, Engram and Dendrite tools (Pillar 5/9)

`dendrite-mcp-hub` (`github.com/yasuomaidana/dendrite-mcp-hub`) provides the Model Context Protocol (MCP) JSON-RPC 2.0 server and tool registry (`pkg/mcpserver`), the reverse IDE client and configuration bridge (`pkg/jetbrains`), and cross-language IPC contract drift monitoring (`pkg/polyglot/contract`).
It aggregates and proxies upstream MCP servers and external tool connectors—including Penpot UI design-token synchronization, Plane v2 AI/issue workflows, Engram topic memory, REA disassemblers, package registry publishers, and local Code Knowledge (`graphify`) queries.
It also coordinates the multi-IDE plugin adapters (JetBrains, VS Code, Neovim, Zed) while delegating control-plane orchestration to `dendrite-control` and AST, Git, and RAG compute to the Rust sidecar crates (`dendrite-core-rs`, `dendrite-git-rs`, `dendrite-rag-rs`).
See [docs/RESPONSIBILITIES.md](docs/RESPONSIBILITIES.md) and the [Documentation Index](docs/README.md) for the authoritative boundary definition, MCP tool family ownership map, and Plane (`DMCP`) ticket tracking.
