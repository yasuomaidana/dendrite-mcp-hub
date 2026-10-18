# `dendrite-mcp-hub` Documentation Index

This directory contains architectural boundary documentation and imported specifications for **`dendrite-mcp-hub`** (`github.com/yasuomaidana/dendrite-mcp-hub`), the unified Model Context Protocol (MCP) host, IDE communication bridge, and upstream MCP connector hub in the Dendrite 5-repository decomposition (Pillar 5 / Pillar 9).

## Repository Documentation

- [Repository Responsibilities & Boundary Definition](RESPONSIBILITIES.md) (`docs/RESPONSIBILITIES.md`) — Authoritative scope, ownership boundaries, exposed interfaces, MCP tool family ownership vs. `dendrite-control`, extracted Go packages, sibling repository dependencies, Plane ticket map (`DMCP`), and open architectural questions.
- [Imported Documentation Manifest](imports.json) (`docs/imports.json`) — Machine-readable list of feature specifications, CLI references, and setup guides imported from the control-plane repository (`Dendrite`).

## Imported from Dendrite

The following specifications and guides are imported from `/Users/yasuomaidana/GoLandProjects/Dendrite` as listed in [`docs/imports.json`](imports.json):

### Architecture
- [docs/imported/architecture/2026-10-quality-audit-execution-plan.md](docs/imported/architecture/2026-10-quality-audit-execution-plan.md) — Pillar 5 (Penpot MCP + Plane AI Integration) & Pillar 9 (Five Sub-Project Decomposition & Multi-Directory Sync Proof).

### Feature Specifications
- [docs/imported/features/FEAT-003-reverse-engineering-rea.md](docs/imported/features/FEAT-003-reverse-engineering-rea.md) — Reverse Engineering Architecture (REA) MCP Bridge (`DEN-009`) and cleanroom parity verification tools (`dendrite_rea_showcase`).
- [docs/imported/features/FEAT-014-jetbrains-ide-mcp-integration.md](docs/imported/features/FEAT-014-jetbrains-ide-mcp-integration.md) — JetBrains IDE MCP Client, Transport Layer, High-Level Bridge (`pkg/jetbrains`), and IDE toolset configuration (`DEN-028`, `DEN-046`).
- [docs/imported/features/FEAT-028-local-agent-triage-gateway-token-efficiency.md](docs/imported/features/FEAT-028-local-agent-triage-gateway-token-efficiency.md) — Local Agent Triage MCP tool (`dendrite_local_agent_triage`) and JetBrains AI Chat multi-MCP orchestration (`DEN-050`).
- [docs/imported/features/FEAT-037-penpot-ui-documentation-mcp-integration.md](docs/imported/features/FEAT-037-penpot-ui-documentation-mcp-integration.md) — Penpot UI file mutation watcher, design-token diff synthesizer (`pkg/ui/penpot`), and `dendrite_ui_penpot_sync` MCP tool (`DEN-076`, `DEN-930`, `DEN-960`).
- [docs/imported/features/FEAT-042-artifact-package-publisher-mcp.md](docs/imported/features/FEAT-042-artifact-package-publisher-mcp.md) — Pluggable Artifact & Package Publisher MCP service (`pkg/mcp/publisher`, `DEN-091`, `DEN-092`, `DEN-093`).
- [docs/imported/features/FEAT-045-code-knowledge-mcp-graphify.md](docs/imported/features/FEAT-045-code-knowledge-mcp-graphify.md) — Local AST & Graphify Code Knowledge MCP server (`DEND-201`).
- [docs/imported/features/FEAT-046-multi-ide-plugins.md](docs/imported/features/FEAT-046-multi-ide-plugins.md) — Quad-IDE plugin suite (JetBrains `DEND-302`, VS Code `DEND-303`, Neovim `DEND-304`, Zed `DEND-305`) and cross-language IPC contract drift monitor (`DEN-063`).
- [docs/imported/features/FEAT-100-multi-ide-ecosystem-support.md](docs/imported/features/FEAT-100-multi-ide-ecosystem-support.md) — Shared JSON-RPC 2.0 / MCP stdio core engine and thin IDE adapter architecture.
- [docs/imported/features/FEAT-101-personal-context-topic-memory-installer.md](docs/imported/features/FEAT-101-personal-context-topic-memory-installer.md) — Canvas Penpot UI bridge (`pkg/personalcontext/canvas`) and Personal Context MCP Connectors (`DEN-808`).

### CLI References
- [docs/imported/cli/mcp.md](docs/imported/cli/mcp.md) — CLI reference for `dendrite mcp serve`, `dendrite mcp tools`, and `dendrite mcp call`.
- [docs/imported/cli/jetbrains.md](docs/imported/cli/jetbrains.md) — CLI reference for `dendrite jetbrains` (`probe`, `open`, `problems`, `symbols`, `build`, `run`, `setup`).
- [docs/imported/cli/canvas.md](docs/imported/cli/canvas.md) — CLI reference for `dendrite canvas` (`actions`, `dispatch`, `bridge`) and Penpot MCP bridge tools.

### Developer Guides
- [docs/imported/guides/jetbrains-mcp-and-agents-setup.md](docs/imported/guides/jetbrains-mcp-and-agents-setup.md) — Comprehensive setup guide and catalog of Dendrite MCP tools and JetBrains IDE integration.
- [docs/imported/guides/using-dendrite-tools-with-jetbrains.md](docs/imported/guides/using-dendrite-tools-with-jetbrains.md) — End-to-end developer workflow guide for bidirectional MCP usage in JetBrains IDEs.

### Mechanically Copied MCP Tool Reference Docs
- `docs/imported/mcp/` — All 87 generated MCP tool reference documents (`dendrite_*.md`) copied mechanically by the orchestrator from `Dendrite/docs/mcp/` (omitted from `docs/imports.json` per repository conventions; see [RESPONSIBILITIES.md](RESPONSIBILITIES.md) for tool family ownership analysis).
