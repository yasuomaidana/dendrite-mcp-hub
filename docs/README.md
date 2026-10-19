# `dendrite-mcp-hub` Documentation Index

This directory contains architectural boundary documentation and imported specifications for **`dendrite-mcp-hub`** (`github.com/yasuomaidana/dendrite-mcp-hub`), the unified Model Context Protocol (MCP) host, IDE communication bridge, and upstream MCP connector hub in the Dendrite 5-repository decomposition (Pillar 5 / Pillar 9).

## Repository Documentation

- [Repository Responsibilities & Boundary Definition](RESPONSIBILITIES.md) (`docs/RESPONSIBILITIES.md`) — Authoritative scope, ownership boundaries, exposed interfaces, MCP tool family ownership vs. `dendrite-control`, extracted Go packages, sibling repository dependencies, Plane ticket map (`DMCP`), and open architectural questions.
- [Imported Documentation Manifest](imports.json) (`docs/imports.json`) — Machine-readable list of feature specifications, CLI references, and setup guides imported from the control-plane repository (`Dendrite`).

## Imported from Dendrite

<!-- Generated from imports.json by scripts/sync_subproject_docs.py (control-plane repo). The list is regenerated; descriptions are kept and must describe this repository. -->

Specifications that define `dendrite-mcp-hub`'s responsibilities, listed in [`imports.json`](imports.json) and copied into `docs/imported/`. The first line of each copy records where it was copied from. The copies are regenerated, so do not edit them here. Each entry below says what the spec defines for this repository.

### Architecture

- [`2026-10-quality-audit-execution-plan`](imported/architecture/2026-10-quality-audit-execution-plan.md) — Boundary of dendrite-mcp-hub in the five-repository split: the MCP server, tool registry, IDE bridges and upstream connectors it owns.

### Feature specifications

- [`FEAT-003-reverse-engineering-rea`](imported/features/FEAT-003-reverse-engineering-rea.md) — REA bridge tools the hub exposes, including `dendrite_rea_showcase`.
- [`FEAT-014-jetbrains-ide-mcp-integration`](imported/features/FEAT-014-jetbrains-ide-mcp-integration.md) — JetBrains reverse MCP client and bridge implemented in the hub's `pkg/jetbrains`.
- [`FEAT-028-local-agent-triage-gateway-token-efficiency`](imported/features/FEAT-028-local-agent-triage-gateway-token-efficiency.md) — Local agent triage tool the hub registers, and the multi-MCP orchestration it fronts for IDE chats.
- [`FEAT-037-penpot-ui-documentation-mcp-integration`](imported/features/FEAT-037-penpot-ui-documentation-mcp-integration.md) — Penpot design-token sync the hub implements (`pkg/ui/penpot`) and exposes as `dendrite_ui_penpot_sync`.
- [`FEAT-042-artifact-package-publisher-mcp`](imported/features/FEAT-042-artifact-package-publisher-mcp.md) — Artifact and package publisher service the hub hosts as MCP tools.
- [`FEAT-045-code-knowledge-mcp-graphify`](imported/features/FEAT-045-code-knowledge-mcp-graphify.md) — Code-knowledge MCP server the hub hosts in front of the retrieval backend.
- [`FEAT-046-multi-ide-plugins`](imported/features/FEAT-046-multi-ide-plugins.md) — IDE plugins (JetBrains, VS Code, Neovim, Zed) that connect to the hub, and the hub's contract drift monitor.
- [`FEAT-100-multi-ide-ecosystem-support`](imported/features/FEAT-100-multi-ide-ecosystem-support.md) — Shared JSON-RPC 2.0 / stdio core and thin IDE adapters that make up the hub.
- [`FEAT-101-personal-context-topic-memory-installer`](imported/features/FEAT-101-personal-context-topic-memory-installer.md) — Canvas bridge and personal-context MCP connectors the hub exposes.

### CLI reference

- [`mcp`](imported/cli/mcp.md) — `mcp serve`, `tools` and `call`: how the hub's server is started and queried.
- [`jetbrains`](imported/cli/jetbrains.md) — IDE probe, open, problems and setup commands served by the hub's JetBrains bridge.
- [`canvas`](imported/cli/canvas.md) — Canvas actions and the Penpot bridge served by the hub.

### MCP tool reference

- [`dendrite_canvas_action_dispatch`](imported/mcp/dendrite_canvas_action_dispatch.md) — Tool the hub registers to dispatch canvas actions.
- [`dendrite_canvas_penpot_bridge`](imported/mcp/dendrite_canvas_penpot_bridge.md) — Tool the hub registers to bridge canvas state to Penpot.
- [`dendrite_contract_drift_check`](imported/mcp/dendrite_contract_drift_check.md) — Tool backed by the hub's cross-language contract drift monitor (`pkg/polyglot/contract`).
- [`dendrite_jetbrains_setup`](imported/mcp/dendrite_jetbrains_setup.md) — Tool that provisions IDE MCP configuration through the hub's JetBrains bridge.
- [`dendrite_local_agent_triage`](imported/mcp/dendrite_local_agent_triage.md) — Hub-hosted triage tool that routes prompts to local or cloud agents.
- [`dendrite_pcontext_greeting_generate`](imported/mcp/dendrite_pcontext_greeting_generate.md) — Hub-hosted personal-context greeting tool.
- [`dendrite_rea_showcase`](imported/mcp/dendrite_rea_showcase.md) — Hub-hosted REA bridge tool for cleanroom parity showcases.
- [`dendrite_ui_penpot_sync`](imported/mcp/dendrite_ui_penpot_sync.md) — Tool backed by the hub's Penpot design-token synchroniser.

### Guides

- [`jetbrains-mcp-and-agents-setup`](imported/guides/jetbrains-mcp-and-agents-setup.md) — Connecting IDEs and coding agents to the hub, with its tool catalogue.
- [`using-dendrite-tools-with-jetbrains`](imported/guides/using-dendrite-tools-with-jetbrains.md) — Day-to-day use of the hub's tools from a JetBrains IDE.

