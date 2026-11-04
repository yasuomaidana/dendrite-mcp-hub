# DMCP Project Development Status & Roadmap

This document maintains the real-time development status, milestones, and active ticket execution pipeline for
**DMCP**, managed following the autonomous ticket lifecycle protocol.

---

## Current Status Overview

- **Current Focus**: `DMCP-031: Track DEV_STATUS.md generated from its own Plane project`
- **Last Updated**: 2026-11-03
- **Summary**: Backlog: 7 | To Do: 10 | In Progress: 5 | To Verify: 7 | Done: 2 (Total: 31)
- **Overall Progress**: 2 / 31 Tickets Completed (6.5%)

---

## Milestone Execution Pipeline

### M7: Go/Rust Decomposition

- [ ] **DMCP-1**: dendrite-mcp-hub: extract pkg/mcpserver into a standalone Go module [State: Backlog, Priority: Medium]

### Milestone 27: Advanced Polyglot Contract Sync, Speculative Branching &amp; Upstream Drift

- [ ] **DMCP-17**: [DEN-063] Quad-IDE Cross-Language Schema & IPC Contract Drift Monitor (`pkg/polyglot/contract`, `pkg/jetbrains`) [State: To Verify, Priority: Medium]

### Milestone 31: Ticket-Milestone DAG Linker &amp; Story Prioritization Engine

- [ ] **DMCP-16**: [DEN-075] Context-Aware `dendrite stories next` Scheduler & Active Story Context Injector (`pkg/stories/next`, `pkg/jetbrains`) [State: In Progress, Priority: High]

### 10: IDE Intelligence &amp; JetBrains MCP Bridge.

- [ ] **DMCP-21**: [DEN-028] JetBrains IDE MCP Integration & Communication Bridge [State: To Verify, Priority: Medium]

### 21: Model Context Protocol (MCP) Server &amp; Tool Registry Bridge.

- [ ] **DMCP-20**: [DEN-040] Dendrite Native MCP Server & Tool Registry Bridge (`pkg/mcpserver`, `cmd/dendrite`) [State: To Verify, Priority: Medium]

### 22: Distributed Codebase Ecosystem, Modular REA &amp; Cross-Project Interoperability Coordinator.

- [ ] **DMCP-19**: [DEN-046] JetBrains IDE Toolset Integration, 19 MCP Tools & Agentic Developer Workflow Guide (pkg/jetbrains, pkg/mcpserver, cmd/dendrite) [State: To Verify, Priority: Medium]

### 23: Hardening, Benchmark Scenarios, Cleanroom Invariants &amp; Local Agent Triage.

- [ ] **DMCP-18**: [DEN-050] Local Agent Triage Gateway, Token-Efficient Prompt Router & JetBrains AI Chat Multi-MCP Orchestrator (`pkg/agent/triage`, `pkg/jetbrains`, `pkg/mcpserver`, `pkg/gateway`) [State: In Progress, Priority: Medium]

### 5: Reverse Engineering &amp; Sandboxed Intelligence.

- [ ] **DMCP-22**: [DEN-009] REA MCP Bridge & Disassembler Protocol Client [State: To Do, Priority: Medium]

### General

- [ ] **DMCP-2**: [DEN-960] Penpot MCP service + hub registration (was DEN #185) [State: Backlog, Priority: Medium]
- [ ] **DMCP-3**: [DEN-930] Spike: Penpot official MCP server deployment shape (was DEN #183) [State: Backlog, Priority: Medium]
- [ ] **DMCP-4**: [DEN-812] Bidirectional Multi-Vendor Migration & Real-Time Sync Engine (Plane <-> Jira, Penpot <-> Figma) (was DEN #141) [State: Backlog, Priority: High]
- [ ] **DMCP-5**: [DEN-808] Personal Context MCP Connectors: Workspace & Enterprise Tool Adapters (was DEN #136) [State: Backlog, Priority: Medium]
- [ ] **DMCP-6**: [DEND-305] [Plugin - Zed]: Rust WASM extension (dendrite-zed) (was DEN #113) [State: To Do, Priority: Medium]
- [ ] **DMCP-7**: [DEND-304] [Plugin - Neovim]: Headless Lua RPC client (dendrite.nvim) (was DEN #112) [State: To Do, Priority: Medium]
- [ ] **DMCP-8**: [DEND-303] [Plugin - VS Code]: VS Code Extension (dendrite-vscode) built with bun & MCP context (was DEN #111) [State: To Do, Priority: High]
- [ ] **DMCP-9**: [DEND-302] [Plugin - JetBrains]: JetBrains Plugin (dendrite-jetbrains) with local AI chat & Graphify markers (was DEN #110) [State: To Do, Priority: Urgent]
- [ ] **DMCP-10**: [DEND-201] [Code Knowledge MCP]: Local AST query MCP server (was DEN #106) [State: To Do, Priority: High]
- [ ] **DMCP-11**: [DEN-096] Token-Efficient Structured Log Tailing & MCP Routine Control Tools (`pkg/routines/cicd`, `pkg/mcpserver`, `cmd/dendrite`) [State: To Do, Priority: Medium]
- [ ] **DMCP-12**: [DEN-093] Post-Publish Git Tag Lifecycle & Stdio JSON-RPC MCP Tool Registry (`pkg/mcp/publisher`, `pkg/mcpserver`) [State: To Do, Priority: Medium]
- [ ] **DMCP-13**: [DEN-092] Pre-Flight Validation, Dry-Run SemVer Guard & Secret-Isolated Execution Gate (`pkg/mcp/publisher`, `cmd/dendrite`) [State: To Do, Priority: Medium]
- [ ] **DMCP-14**: [DEN-091] Pluggable Registry Adapter Interface & Cargo Workspace Topological Publisher (`pkg/mcp/publisher`, `internal/mcp/publisher`) [State: To Do, Priority: Medium]
- [ ] **DMCP-15**: [DEN-076] UI File Mutation Watcher & Penpot Documentation Token Diff Synthesizer (`pkg/ui/penpot`, `pkg/mcpserver`) [State: To Verify, Priority: Medium]
- [x] **DMCP-23**: docs: own technical specification (ARCHITECTURE.md), drop imported copies [State: Done, Priority: High]
- [ ] **DMCP-025**: Bring mcp-hub develop to a passing quality gate before the develop to main merge [State: To Verify, Priority: Medium]
- [ ] **DMCP-26**: CHANGELOG for 0.1.0 and README purpose [State: In Progress, Priority: Medium]
- [ ] **DMCP-27**: Second complexity check in CI with gocyclo and gocognit [State: In Progress, Priority: Medium]
- [ ] **DMCP-28**: Clear the Go complexity debt: 29 functions over a limit [State: Backlog, Priority: Medium]
- [ ] **DMCP-29**: Adopt generated CI and clear the quality gate (29 complexity, 1 clone) [State: Backlog, Priority: Medium]
- [ ] **DMCP-030**: Untrack .dendrite local state [State: To Verify, Priority: Medium]
- [ ] **DMCP-031**: Track DEV_STATUS.md generated from its own Plane project [State: In Progress, Priority: Medium]

### Rust RAG only + quality gate

- [x] **DMCP-11**: ci(quality): max 500 lines per code file and jscpd duplicate gate; split the 4 long Go files [State: Done, Priority: Medium]

