# `dendrite-mcp-hub` — Repository Responsibilities & Boundary Definition

> **Plane Project**: `DMCP` (workspace `dendrite-ws`, `http://localhost:8090`)  
> **Extraction Umbrella**: `DEN-971` (first child: `DMCP-1`)  
> **Target Go Module**: `dendrite.dev/mcphub` (currently initialized in `go.mod` as `module dendrite-mcp-hub`, `go 1.22`)

---

## 1. Purpose

`dendrite-mcp-hub` (`dendrite.dev/mcphub`) is the unified Model Context Protocol (MCP) JSON-RPC 2.0 server, tool registry, and bidirectional IDE/upstream connector bridge for the Dendrite ecosystem (Pillar 5 and Pillar 9 of the 2026-10 Architecture Execution Plan). It exposes Dendrite's tool catalog to MCP-capable AI clients and IDE plugins (JetBrains IDEs, VS Code, Neovim, Zed, Claude Code, Antigravity, Cursor) while acting as an outbound MCP client and proxy to IDE servers (JetBrains built-in MCP at `:63342/api/mcp`) and upstream MCP services (Penpot MCP, Plane v2 AI/issue workflows, Engram topic memory, REA disassemblers, and local Graphify Code Knowledge). Extracting this boundary isolates protocol transport, tool schema registration, IDE plugins, and external SaaS/self-hosted MCP connectors from the core control-plane daemon (`dendrite-control`) and the Rust compute sidecars (`dendrite-core-rs`, `dendrite-rag-rs`, `dendrite-git-rs`).

---

## 2. Owns vs. Does NOT Own

### Owns (`dendrite-mcp-hub` / `dendrite.dev/mcphub`)

- **MCP JSON-RPC 2.0 Server & Tool Registry (`pkg/mcpserver`)**:
  - Protocol version `2024-11-05` lifecycle (`initialize`, `notifications/initialized`, `ping`, `tools/list`, `tools/call`), line-delimited stdio server (`ServeStdio` with 10 MB scanner buffer), and thread-safe tool registration (`RegisterTool`, `ListTools`, `HandleRequest`).
- **JetBrains Reverse MCP Client, Bridge & IDE Provisioning (`pkg/jetbrains`)**:
  - Outbound MCP client (`Client`) over HTTP (`http://localhost:63342/api/mcp`) and stdio (`StdioTransport`).
  - High-level IDE `Bridge` (`OpenFile`, `GetFileProblems`, `SearchSymbol`, `BuildProject`, `ExecuteRunConfiguration`, `LintFiles`).
  - Zero-touch IDE toolset provisioning (`SetupJetBrainsToolset` generating `.ai/mcp/mcp.json`, `~/.ai/mcp/mcp.json`, `.idea/runConfigurations/*.xml`, and `.idea/tools/External_Tools.xml`).
- **Cross-Language Schema & IPC Contract Drift Monitor (`pkg/polyglot/contract`)**:
  - Contract drift detection across Go, Rust, Python, and TypeScript schemas (`ContractMonitor.VerifyContracts`, `dendrite_contract_drift_check`) and automatic JetBrains IDE diagnostic navigation (`NotifyIDE`).
- **Penpot MCP Upstream Integration & Design-Token Synchronization (`pkg/ui/penpot`, `pkg/personalcontext/canvas` bridge)**:
  - Penpot UI component specification parsing, bidirectional token diff synthesis (`dendrite_ui_penpot_sync`), Canvas Penpot action dispatch (`dendrite_canvas_action_dispatch`, `dendrite_canvas_penpot_bridge`), and planned official Penpot MCP server proxy (`DEN-930`, `DEN-960`).
- **Upstream & Domain MCP Servers (Planned / In Extraction)**:
  - **Artifact & Package Publisher MCP (`pkg/mcp/publisher`, `internal/mcp/publisher`)**: Topological workspace publishing (`Cargo.toml`, `package.json`, `pyproject.toml`, Go tags), dry-run SemVer guard, secret-isolated execution gate, and `dendrite_publisher_*` tools (`DEN-091`, `DEN-092`, `DEN-093` — *planned / `To Do`*).
  - **Code Knowledge & Graphify MCP (`DEND-201`)**: Local AST and `.dendrite/knowledge.db` query server exposing `graphify.*` and `knowledge.*` MCP tools (*`To Verify` in Plane, not yet present in `pkg/mcpserver`*).
  - **REA Disassembler MCP Bridge (`DEN-009`, `pkg/rea/client`, `pkg/mcpserver/tools_rea.go`)**: Existing `dendrite_rea_showcase` tool and planned JSON-RPC client adapter for GhidraMCP, r2mcp, and RzMCP (*`DEN-009` `To Do`*).
  - **Personal Context & Enterprise MCP Connectors (`DEN-808`)** and **Bidirectional Multi-Vendor Sync Engine (`DEN-812`, Plane ↔ Jira, Penpot ↔ Figma)** (*planned / `Backlog`*).
  - **Multi-IDE Client Adapters (`DEND-302`..`DEND-305`)**: Thin IDE plugins for JetBrains (`dendrite-jetbrains`), VS Code (`dendrite-vscode`), Neovim (`dendrite.nvim`), and Zed (`dendrite-zed`) communicating with the MCP hub (*planned / `To Do`*).

### Does NOT Own (Explicit Exclusions & Owning Repositories)

| Capability / Domain | Owning Repository | Module / Crate | Rationale |
| :--- | :--- | :--- | :--- |
| CLI binary (`cmd/dendrite`), environment replication (`pkg/env`), LiteLLM token governor & gateway (`pkg/gateway`, `pkg/scheduler`), Plane v2 REST client & DAG engine (`pkg/tracker/plane`, `pkg/tracker/dag`), multi-repo pool orchestration (`pkg/pool`), web UI & REST server (`pkg/ui`, `pkg/server`), autonomous agent runner (`pkg/agent`) | **`dendrite-control`** (currently `/Users/yasuomaidana/GoLandProjects/Dendrite`) | `dendrite.dev/control` (currently `module Dendrite`) | Core control plane and orchestration state machine; `dendrite-mcp-hub` only exposes or proxies MCP tool surfaces for these subsystems. |
| Tree-sitter AST parsing, code folding, last-node & cyclomatic/cognitive complexity analysis, BLAKE3 hashing, token counting, and CFG slicing (replacing Go heuristics in `pkg/ast` and `pkg/ai/tokenopt`) | **`dendrite-core-rs`** | `dendrite-core` (binary `dendrite-core`) | CPU-intensive deterministic AST/CFG and token analysis executed via Rust JSON-RPC 2.0 stdio sidecar. |
| Hybrid BM25 + vector retrieval, content-addressable storage (CAS), and Engram/topic memory sync engine (replacing `pkg/rag` and `pkg/store`) | **`dendrite-rag-rs`** | `dendrite-rag` (binary `dendrite-rag`) | High-throughput indexing, vector search, and segmented storage executed via Rust JSON-RPC 2.0 stdio sidecar. |
| Pure-Rust `gix` hunk dependency graph, semantic hunk clusterer, and review-branch engine (replacing `pkg/vcs` / `pkg/story/fork` git shelling) | **`dendrite-git-rs`** | `dendrite-git` (binary `dendrite-git`) | Git object graph manipulation and atomic hunk staging executed via Rust JSON-RPC 2.0 stdio sidecar. |
| Foundational DAG data structures, topological sorting, and pattern/graph matchers (`features = ["core", "matcher"]`) | **`embedag`** (`../../RustRoverProjects/embedag/crates/embedag`) | `embedag` | Shared Rust library crate consumed by `dendrite-core-rs` and `dendrite-rag-rs`. |

---

## 3. Interfaces

### What `dendrite-mcp-hub` Exposes
1. **Go Module API (`dendrite.dev/mcphub`)**:
   - `pkg/mcpserver`: `Server`, `ServerConfig`, `NewServer`, `RegisterTool`, `RegisterDefaultTools`, `ListTools`, `HandleRequest`, `ServeStdio`, `Tool`, `ToolHandler`, `CallToolParams`, `CallToolResult`, `ContentItem`, `Dependencies`.
   - `pkg/jetbrains`: `Client`, `NewClient`, `Transport`, `HTTPTransport`, `StdioTransport`, `Bridge`, `NewBridge`, `SetupJetBrainsToolset`, `SetupOptions`, `SetupResult`, `MCPConfig`.
   - `pkg/polyglot/contract`: `ContractMonitor`, `NewContractMonitor`, `SchemaContract`, `DriftViolation`.
2. **MCP JSON-RPC 2.0 Stdio & HTTP Endpoints**:
   - Standard MCP `2024-11-05` methods (`initialize`, `notifications/initialized`, `ping`, `tools/list`, `tools/call`) over `os.Stdin`/`os.Stdout` (invoked via `dendrite mcp serve`) and planned SSE/HTTP proxy endpoints for upstream MCP aggregation.

### Who Consumes `dendrite-mcp-hub`
- **`dendrite-control` (`cmd/dendrite/mcp.go`, `cmd/dendrite/main.go`)**: Wires `mcpserver.NewServer` and `jetbrains.NewBridge` into the `dendrite mcp` (`serve`, `tools`, `call`) and `dendrite jetbrains` (`probe`, `open`, `problems`, `symbols`, `build`, `run`, `setup`) CLI commands.
- **External AI Coding Agents & IDEs**: JetBrains AI Assistant, Junie, Claude Code, Antigravity (AGY), Cursor, Windsurf, Aider, OpenHands, and the planned quad-IDE plugins (`dendrite-jetbrains`, `dendrite-vscode`, `dendrite.nvim`, `dendrite-zed`) via `.ai/mcp/mcp.json`.

### Sibling Ecosystem Interface Contracts (Pillar 9)
- **Rust Sidecar Crates (`dendrite-core-rs`, `dendrite-rag-rs`, `dendrite-git-rs`)**: Each ships a `dendrite-<x>` binary speaking JSON-RPC 2.0 over stdio with SemVer-versioned schemas in `proto/*.json`. The Go side uses a `sidecar.Client` with a health handshake and falls back to the in-process Go implementation when the binary is absent.
- **`embedag`**: Exposes a Rust crate API (`default-features = false, features = ["core", "matcher"]`) consumed by `dendrite-core-rs` and `dendrite-rag-rs`.

---

## 4. Code Being Extracted / Replaced & MCP Tool Family Ownership

### Verified Existing Go Packages and Files in `Dendrite`

Every file listed below was verified by reading `/Users/yasuomaidana/GoLandProjects/Dendrite`:

1. **`pkg/mcpserver/` (26 files verified — 69 tools registered in `RegisterDefaultTools`)**:
   - **Core Protocol & Registry (self-contained stdlib Go)**:
     - `pkg/mcpserver/types.go` — MCP `2024-11-05` & JSON-RPC 2.0 request/response envelopes, error codes, `Tool`, `ToolHandler`, `CallToolParams`, `CallToolResult`, `ContentItem`.
     - `pkg/mcpserver/server.go` — `Server`, `ServerConfig`, `NewServer`, `RegisterTool`, `ListTools`, `HandleRequest`, `ServeStdio`.
     - `pkg/mcpserver/server_test.go` — Unit tests for initialization, ping, tool execution, stdio framing, and error codes.
   - **Tool Registration Files (`pkg/mcpserver/tools*.go`)**:
     - `pkg/mcpserver/tools.go` — Defines `Dependencies` and `RegisterDefaultTools(deps)`; registers 17 inline tools (`dendrite_env_status`, `dendrite_ticket_list`, `dendrite_ticket_update`, `dendrite_plane_issues`, `dendrite_ollama_probe`, `dendrite_topology_advisor`, `dendrite_gateway_render`, `dendrite_gateway_update`, `dendrite_agent_run`, `dendrite_telemetry`, `dendrite_inference_list`, `dendrite_inference_probe`, `dendrite_plane_replicate`, `dendrite_plane_push_status`, `dendrite_plane_doc_sync`, `dendrite_jetbrains_setup`, `dendrite_commit`) and invokes 22 domain registration helpers:
     - `pkg/mcpserver/tools_agentsearch.go` (`dendrite_agent_search`)
     - `pkg/mcpserver/tools_audit.go` (`dendrite_audit_changes`, `dendrite_github_sync`, `dendrite_vcs_guard`, `dendrite_remote_sync`, `dendrite_quality_audit`, `dendrite_hitl_sync`)
     - `pkg/mcpserver/tools_canvas.go` (`dendrite_canvas_action_dispatch`, `dendrite_canvas_penpot_bridge`)
     - `pkg/mcpserver/tools_contract.go` (`dendrite_contract_drift_check`)
     - `pkg/mcpserver/tools_distcodebase.go` (`dendrite_dist_sources`, `dendrite_dist_decompose`, `dendrite_dist_interop`, `dendrite_dist_spawn`)
     - `pkg/mcpserver/tools_docusaurus.go` (`dendrite_docusaurus_generate`, `dendrite_docusaurus_share`, `dendrite_docusaurus_audit`)
     - `pkg/mcpserver/tools_env.go` (`dendrite_env_prepare`)
     - `pkg/mcpserver/tools_greeting.go` (`dendrite_pcontext_greeting_generate`)
     - `pkg/mcpserver/tools_inference.go` (`dendrite_inference_migrate_model`)
     - `pkg/mcpserver/tools_migrate.go` (`dendrite_service_move`, `dendrite_service_plan`)
     - `pkg/mcpserver/tools_penpot.go` (`dendrite_ui_penpot_sync`)
     - `pkg/mcpserver/tools_perf.go` (`dendrite_bench_run`)
     - `pkg/mcpserver/tools_plane_dag.go` (`dendrite_ticket_smart_select`, `dendrite_plane_ticket_smart_select`, `dendrite_plane_relation_link`, `dendrite_plane_issue_create`)
     - `pkg/mcpserver/tools_pool.go` (`dendrite_pool_resolve`, `dendrite_pool_list`, `dendrite_pool_add`, `dendrite_pool_switch`, `dendrite_pool_split`)
     - `pkg/mcpserver/tools_query.go` (`dendrite_query_slice`, `dendrite_query_swap`, `dendrite_query_transpile`, `dendrite_query_fuse`, `dendrite_query_reuse_audit`, `dendrite_ref_list`, `dendrite_ref_add`)
     - `pkg/mcpserver/tools_rag.go` (`dendrite_rag_ask`, `dendrite_rag_retrieve`)
     - `pkg/mcpserver/tools_rea.go` (`dendrite_rea_showcase`)
     - `pkg/mcpserver/tools_scheduler.go` (`dendrite_analyze_policy`, `dendrite_analyze_status`, `dendrite_analyze_trigger`, `dendrite_analyze_switch`, `dendrite_tokens_status`)
     - `pkg/mcpserver/tools_store.go` (`dendrite_store_status`, `dendrite_store_commit_chunk`, `dendrite_store_query_partial`, `dendrite_store_inspect`, `dendrite_store_rm`)
     - `pkg/mcpserver/tools_story.go` (`dendrite_story_fork`, `dendrite_story_benchmark`, `dendrite_story_resolve`, `dendrite_story_abort`)
     - `pkg/mcpserver/tools_testgen.go` (`dendrite_test_auto`)
     - `pkg/mcpserver/tools_triage.go` (`dendrite_local_agent_triage`)
2. **`pkg/jetbrains/` (6 files verified — moved wholesale to `dendrite-mcp-hub`)**:
   - `pkg/jetbrains/types.go` — Client-side JSON-RPC 2.0 & MCP types, `Problem`, `Symbol`, `BuildResult`, domain errors.
   - `pkg/jetbrains/transport.go` — `Transport` interface, `HTTPTransport` (`http://localhost:63342/api/mcp`), `StdioTransport`.
   - `pkg/jetbrains/client.go` — `Client` (`NewClient`, `Connect`, `Disconnect`, `ListTools`, `CallTool`, `Ping`).
   - `pkg/jetbrains/bridge.go` — `Bridge` (`OpenFile`, `GetFileProblems`, `SearchSymbol`, `BuildProject`, `ExecuteRunConfiguration`, `LintFiles`).
   - `pkg/jetbrains/setup.go` — `SetupJetBrainsToolset` writing `.ai/mcp/mcp.json`, `~/.ai/mcp/mcp.json`, `.idea/runConfigurations/*.xml`, and `.idea/tools/External_Tools.xml`.
   - `pkg/jetbrains/jetbrains_test.go` and `pkg/jetbrains/setup_test.go`.
3. **`pkg/polyglot/contract/` (verified — moved to `dendrite-mcp-hub` per `DEN-063`)**:
   - `pkg/polyglot/contract/types.go` and `pkg/polyglot/contract/monitor.go` (`ContractMonitor`, imports `Dendrite/pkg/jetbrains`).
4. **Shared / Split Packages Across `dendrite-mcp-hub` and `dendrite-control`**:
   - `pkg/ui/penpot/penpot.go` — Penpot component spec parser and token diff synchronizer (`DEN-076` split between `pkg/ui/penpot` and `pkg/mcpserver/tools_penpot.go`).
   - `pkg/personalcontext/canvas/types.go` (and `PenpotUIBridge`) — Canvas writing and Penpot UI bridge actions exposed by `pkg/mcpserver/tools_canvas.go`.
   - `pkg/agent/triage/triage.go` — Local Agent Triage engine (`DEN-050` split across `pkg/agent/triage`, `pkg/jetbrains`, `pkg/mcpserver/tools_triage.go`, and `pkg/gateway`).
   - `cmd/dendrite/mcp.go` and `cmd/dendrite/main.go` (lines 1127–1300) — Control-plane CLI entrypoints (`dendrite mcp` and `dendrite jetbrains`) that remain in `dendrite-control` as callers of `dendrite.dev/mcphub`.
5. **Planned Packages Referenced in Tickets (Verified Not Yet Present in `Dendrite/pkg/`)**:
   - `pkg/mcp/publisher` and `internal/mcp/publisher` (`DEN-091`, `DEN-092`, `DEN-093` — `To Do`).
   - `pkg/rea/client` (`DEN-009` — `To Do`).
   - `pkg/routines/cicd` (`DEN-096` — `To Do`).
   - `pkg/stories/next` (`DEN-075` — `In Progress`).

### Fallback & Degradation Behavior

- **Rust Sidecar Fallback**: When MCP tools backed by Rust compute crates (`dendrite_query_*` via `dendrite-core-rs`, `dendrite_rag_*` / `dendrite_store_*` via `dendrite-rag-rs`, `dendrite_story_*` / `dendrite_commit` via `dendrite-git-rs`) are invoked, the Go `sidecar.Client` performs a health handshake over JSON-RPC 2.0 stdio and falls back to the in-process Go implementation when the `dendrite-<x>` binary is missing.
- **Plane v2 Offline Fallback**: Tracker and DAG MCP tools (`dendrite_plane_*`, `dendrite_ticket_*`) attempt live Plane v2 REST calls (`http://localhost:8090`) via `deps.PlaneClient` or `plane.NewClientFromEnv()`, and fall back to local markdown/cache state (`DEV_STATUS.md` / Turso outbox) when Plane is unreachable.
- **JetBrains Live Connection Requirement**: Per the Zero-Mock Policy enforced in `cmd/dendrite/main.go` (`getJetBrainsClient`), outbound calls to the JetBrains IDE MCP server (`http://localhost:63342/api/mcp`) use a 1500ms timeout and return an explicit connection error rather than fabricating mock IDE diagnostics when the IDE is closed.

### MCP Tool Families: Hub Ownership vs. Control-Plane Boundary

All 87 generated MCP tool reference docs under `Dendrite/docs/mcp/` are copied mechanically into `docs/imported/mcp/`. Below is the grouping of all registered tool families by prefix, showing which families belong natively in `dendrite-mcp-hub` vs. which families are **control-plane or sidecar domain tools** that should either remain registered in `dendrite-control` (using `dendrite.dev/mcphub` as a library) or be proxied over RPC to prevent circular Go module dependencies:

| Tool Name Prefix / Family | Registered Tools in `pkg/mcpserver` | Target Owner | Boundary Recommendation |
| :--- | :--- | :--- | :--- |
| **`dendrite_jetbrains_*`** | `dendrite_jetbrains_setup` (+ outbound `pkg/jetbrains` bridge tools: `open_file_in_editor`, `get_file_problems`, `search_symbol`, `build_project`, `execute_run_configuration`) | **`dendrite-mcp-hub`** | **Hub Native**: Core IDE MCP bridge and `.ai/mcp/mcp.json` provisioning. |
| **`dendrite_ui_penpot_*` & `dendrite_canvas_*`** | `dendrite_ui_penpot_sync`, `dendrite_canvas_action_dispatch`, `dendrite_canvas_penpot_bridge` | **`dendrite-mcp-hub`** | **Hub Native**: Penpot MCP design-token diffing and Canvas-to-Penpot/Plane bridge (`DEN-076`, `DEN-930`, `DEN-960`). |
| **`dendrite_contract_*`** | `dendrite_contract_drift_check` | **`dendrite-mcp-hub`** | **Hub Native**: Cross-language schema & IPC contract drift monitor wired to `pkg/jetbrains` (`DEN-063`). |
| **`dendrite_rea_*`** | `dendrite_rea_showcase` (+ planned `DEN-009` disassembler protocol tools) | **`dendrite-mcp-hub`** | **Hub Native**: Upstream REA disassembler MCP bridge (GhidraMCP / r2mcp / RzMCP). |
| **`dendrite_publisher_*` & `graphify.*` / `knowledge.*`** | *Planned* (`DEN-091`..`093`, `DEND-201`) | **`dendrite-mcp-hub`** | **Hub Native**: Package registry publisher MCP tools and local Code Knowledge MCP server. |
| **`dendrite_plane_*` & `dendrite_ticket_*`** | `dendrite_plane_issues`, `dendrite_plane_replicate`, `dendrite_plane_push_status`, `dendrite_plane_doc_sync`, `dendrite_plane_ticket_smart_select`, `dendrite_plane_relation_link`, `dendrite_plane_issue_create`, `dendrite_ticket_list`, `dendrite_ticket_update`, `dendrite_ticket_smart_select` | **Shared / Proxy to `dendrite-control`** | **Flag — Control-Plane Tracker**: Handlers depend on `pkg/tracker/plane` and `pkg/tracker/dag`. Keep tracker implementation in `dendrite-control`; expose schemas in the hub or register from `dendrite-control`. |
| **`dendrite_local_agent_triage` & `dendrite_pcontext_*`** | `dendrite_local_agent_triage`, `dendrite_pcontext_greeting_generate` | **Shared (`dendrite-mcp-hub` / `dendrite-control`)** | **Flag — Split (`DEN-050`)**: MCP routing surface and JetBrains AI Chat orchestrator belong in the hub, but `pkg/agent/triage` depends on `pkg/inference`, `pkg/ollama`, and `pkg/rag`. |
| **`dendrite_pool_*`** | `dendrite_pool_resolve`, `dendrite_pool_list`, `dendrite_pool_add`, `dendrite_pool_switch`, `dendrite_pool_split` | **`dendrite-control`** | **Stay in Control Plane**: Directly mutates multi-repo `.dendrite/pool.json`, `go.work`, and `[patch]` tables via `pkg/pool/{dag,switcher,split}`. |
| **`dendrite_env_*` & `dendrite_service_*`** | `dendrite_env_status`, `dendrite_env_prepare`, `dendrite_service_move`, `dendrite_service_plan` | **`dendrite-control`** | **Stay in Control Plane**: Container/host replication (`pkg/env`), CI/CD bootstrap (`pkg/env/bootstrap`), and SSH node migration (`pkg/node/migrate`). |
| **`dendrite_gateway_*`, `dendrite_analyze_*`, `dendrite_tokens_*`, `dendrite_telemetry`** | `dendrite_gateway_render`, `dendrite_gateway_update`, `dendrite_analyze_policy`, `dendrite_analyze_status`, `dendrite_analyze_trigger`, `dendrite_analyze_switch`, `dendrite_tokens_status`, `dendrite_telemetry` | **`dendrite-control`** | **Stay in Control Plane**: LiteLLM config rendering (`pkg/gateway`), token quota governor (`pkg/scheduler`), and spend telemetry (`pkg/telemetry`). |
| **`dendrite_inference_*`, `dendrite_ollama_*`, `dendrite_topology_*`** | `dendrite_inference_list`, `dendrite_inference_probe`, `dendrite_inference_migrate_model`, `dendrite_ollama_probe`, `dendrite_topology_advisor` | **`dendrite-control`** | **Stay in Control Plane**: Local inference probing (`pkg/inference`, `pkg/ollama`) and hardware topology sizing (`pkg/topology/advisor`). |
| **`dendrite_agent_*`, `dendrite_test_*`, `dendrite_bench_*`, `dendrite_dist_*`, `dendrite_docusaurus_*`** | `dendrite_agent_run`, `dendrite_agent_search`, `dendrite_test_auto`, `dendrite_bench_run`, `dendrite_dist_sources`, `dendrite_dist_decompose`, `dendrite_dist_interop`, `dendrite_dist_spawn`, `dendrite_docusaurus_generate`, `dendrite_docusaurus_share`, `dendrite_docusaurus_audit` | **`dendrite-control`** | **Stay in Control Plane**: Autonomous agent loops (`pkg/agent`), test synthesis (`pkg/testgen`), benchmarking (`pkg/perf`), distributed spawning (`pkg/distcodebase`), and docs generation (`pkg/docusaurus`). |
| **`dendrite_query_*` & `dendrite_ref_*`** | `dendrite_query_slice`, `dendrite_query_swap`, `dendrite_query_transpile`, `dendrite_query_fuse`, `dendrite_query_reuse_audit`, `dendrite_ref_list`, `dendrite_ref_add` | **`dendrite-core-rs` / `dendrite-control`** | **Delegate to `dendrite-core-rs` Sidecar**: AST slicing, transpilation, and library reuse auditing should route to `dendrite-core` JSON-RPC sidecar with Go fallback in `dendrite-control`. |
| **`dendrite_rag_*` & `dendrite_store_*`** | `dendrite_rag_ask`, `dendrite_rag_retrieve`, `dendrite_store_status`, `dendrite_store_commit_chunk`, `dendrite_store_query_partial`, `dendrite_store_inspect`, `dendrite_store_rm` | **`dendrite-rag-rs` / `dendrite-control`** | **Delegate to `dendrite-rag-rs` Sidecar**: Hybrid BM25+vector retrieval and segmented CAS storage belong to `dendrite-rag`. |
| **`dendrite_story_*`, `dendrite_audit_*`, `dendrite_vcs_*`, `dendrite_remote_*`, `dendrite_github_*`, `dendrite_quality_*`, `dendrite_hitl_*`, `dendrite_commit`** | `dendrite_story_fork`, `dendrite_story_benchmark`, `dendrite_story_resolve`, `dendrite_story_abort`, `dendrite_audit_changes`, `dendrite_vcs_guard`, `dendrite_remote_sync`, `dendrite_github_sync`, `dendrite_quality_audit`, `dendrite_hitl_sync`, `dendrite_commit` | **`dendrite-git-rs` / `dendrite-control`** | **Delegate to `dendrite-git-rs` & `dendrite-control`**: Git worktree branching and hunk clustering belong to `dendrite-git`; quality/HITL governance stays in `dendrite-control`. |

---

## 5. Dependencies on Sibling Repositories

The five-repository workspace is synchronized by `dendrite pool` under `~/GoLandProjects/`:

- **Go Workspace (`~/GoLandProjects/go.work`)**:
  - Links `./dendrite-control` (`dendrite.dev/control`) and `./dendrite-mcp-hub` (`dendrite.dev/mcphub`) for atomic multi-module Go builds (`pool resolve` generates `go.work` and JetBrains `.idea/modules.xml` scopes).
- **`dendrite-mcp-hub` → `dendrite-control` Directionality**:
  - To prevent circular Go module imports (`dendrite.dev/control` ↔ `dendrite.dev/mcphub`), `dendrite-mcp-hub` must **not** directly import internal `dendrite.dev/control/pkg/...` domain packages. Instead, `dendrite-mcp-hub` defines the MCP server, protocol types, `ToolHandler` registration interfaces, `pkg/jetbrains` bridge, `pkg/polyglot/contract`, and upstream MCP connectors; `dendrite-control` imports `dendrite.dev/mcphub` (or communicates over JSON-RPC/HTTP) to register control-plane tool handlers.
- **Rust Sidecar Crates (`dendrite-core-rs`, `dendrite-rag-rs`, `dendrite-git-rs`) & `embedag`**:
  - `dendrite-core-rs` (`dendrite-core`) and `dendrite-rag-rs` (`dendrite-rag`) consume the sibling **`embedag`** crate via path dependency `../../RustRoverProjects/embedag/crates/embedag` with `default-features = false, features = ["core", "matcher"]` (and `[patch."https://github.com/yasuomaidana/dendrite-core-rs"]` managed by `dendrite pool resolve`).
  - When `dendrite-mcp-hub` serves Code Knowledge (`DEND-201`) or RAG/AST MCP requests directly, it invokes the `dendrite-core` and `dendrite-rag` binaries over JSON-RPC 2.0 stdio (`proto/*.json`).

---

## 6. Ticket Map (Plane Project `DMCP`)

Tracked in Plane v2 project **`DMCP`** (workspace `dendrite-ws`, `http://localhost:8090`). Extraction umbrella in Dendrite: **`DEN-971`**; first child ticket: **`DMCP-1`**. Tickets relocated/split on 2026-10-18 grouped by capability:

### A. Core MCP Server, JetBrains Bridge & Routine Control
| Ticket | Title & Target Packages | Action | Status |
| :--- | :--- | :--- | :--- |
| **DEN-971 / DMCP-1** | Extract `dendrite-mcp-hub` (`dendrite.dev/mcphub`) repository boundary & module | umbrella / child | Backlog |
| **DEN-040** | Dendrite Native MCP Server & Tool Registry Bridge (`pkg/mcpserver`, `cmd/dendrite`) | move | To Verify |
| **DEN-028** | JetBrains IDE MCP Integration & Communication Bridge (`pkg/jetbrains`) | move | To Verify |
| **DEN-046** | JetBrains IDE Toolset Integration, 19 MCP Tools & Agentic Developer Workflow Guide (`pkg/jetbrains`, `pkg/mcpserver`, `cmd/dendrite`) | split | To Verify |
| **DEN-050** | Local Agent Triage Gateway, Token-Efficient Prompt Router & JetBrains AI Chat Multi-MCP Orchestrator (`pkg/agent/triage`, `pkg/jetbrains`, `pkg/mcpserver`, `pkg/gateway`) | split | In Progress |
| **DEN-075** | Context-Aware `dendrite stories next` Scheduler & Active Story Context Injector (`pkg/stories/next`, `pkg/jetbrains`) | split | In Progress |
| **DEN-096** | Token-Efficient Structured Log Tailing & MCP Routine Control Tools (`pkg/routines/cicd`, `pkg/mcpserver`, `cmd/dendrite`) | split | To Do |

### B. Penpot UI MCP, Multi-Vendor Sync & Personal Context Connectors
| Ticket | Title & Target Packages | Action | Status |
| :--- | :--- | :--- | :--- |
| **DEN-960** | Penpot MCP service + hub registration | move | Backlog |
| **DEN-930** | Spike: Penpot official MCP server deployment shape | move | Backlog |
| **DEN-076** | UI File Mutation Watcher & Penpot Documentation Token Diff Synthesizer (`pkg/ui/penpot`, `pkg/mcpserver`) | split | To Verify |
| **DEN-812** | Bidirectional Multi-Vendor Migration & Real-Time Sync Engine (Plane ↔ Jira, Penpot ↔ Figma) | move | Backlog |
| **DEN-808** | Personal Context MCP Connectors: Workspace & Enterprise Tool Adapters | move | Backlog |

### C. Multi-IDE Plugins & Cross-Language Contract Monitoring
| Ticket | Title & Target Packages | Action | Status |
| :--- | :--- | :--- | :--- |
| **DEN-063** | Quad-IDE Cross-Language Schema & IPC Contract Drift Monitor (`pkg/polyglot/contract`, `pkg/jetbrains`) | move | To Verify |
| **DEND-302** | [Plugin - JetBrains]: JetBrains Plugin (`dendrite-jetbrains`) with local AI chat & Graphify markers | split | To Do |
| **DEND-303** | [Plugin - VS Code]: VS Code Extension (`dendrite-vscode`) built with bun & MCP context | split | To Do |
| **DEND-304** | [Plugin - Neovim]: Headless Lua RPC client (`dendrite.nvim`) | split | To Do |
| **DEND-305** | [Plugin - Zed]: Rust WASM extension (`dendrite-zed`) | split | To Do |

### D. Artifact Publisher, Code Knowledge (`graphify`) & REA Disassembler MCP
| Ticket | Title & Target Packages | Action | Status |
| :--- | :--- | :--- | :--- |
| **DEN-091** | Pluggable Registry Adapter Interface & Cargo Workspace Topological Publisher (`pkg/mcp/publisher`, `internal/mcp/publisher`) | move | To Do |
| **DEN-092** | Pre-Flight Validation, Dry-Run SemVer Guard & Secret-Isolated Execution Gate (`pkg/mcp/publisher`, `cmd/dendrite`) | split | To Do |
| **DEN-093** | Post-Publish Git Tag Lifecycle & Stdio JSON-RPC MCP Tool Registry (`pkg/mcp/publisher`, `pkg/mcpserver`) | move | To Do |
| **DEND-201** | [Code Knowledge MCP]: Local AST query MCP server | move | To Verify |
| **DEN-009** | REA MCP Bridge & Disassembler Protocol Client | move | To Do |

### Plane keys (synced with Plane on 2026-10-18)

Each relocated ticket has its own key in `DMCP`. The DEN original either is archived with the `Relocated` state (**move**) or stays open in DEN as the Go-side umbrella with the `relocated-split` label (**split**).

| DMCP key | Origin in DEN | Action | DMCP state |
| :--- | :--- | :--- | :--- |
| **DMCP-1** | umbrella DEN-971 | first child | Backlog |
| **DMCP-2** | DEN-960 (DEN #185) | move | Backlog |
| **DMCP-3** | DEN-930 (DEN #183) | move | Backlog |
| **DMCP-4** | DEN-812 (DEN #141) | move | Backlog |
| **DMCP-5** | DEN-808 (DEN #136) | move | Backlog |
| **DMCP-6** | DEND-305 (DEN #113) | split | To Do |
| **DMCP-7** | DEND-304 (DEN #112) | split | To Do |
| **DMCP-8** | DEND-303 (DEN #111) | split | To Do |
| **DMCP-9** | DEND-302 (DEN #110) | split | To Do |
| **DMCP-10** | DEND-201 (DEN #106) | move | To Verify |
| **DMCP-11** | DEN-096 (DEN #96) | split | To Do |
| **DMCP-12** | DEN-093 (DEN #93) | move | To Do |
| **DMCP-13** | DEN-092 (DEN #92) | split | To Do |
| **DMCP-14** | DEN-091 (DEN #91) | move | To Do |
| **DMCP-15** | DEN-076 (DEN #76) | split | To Verify |
| **DMCP-16** | DEN-075 (DEN #75) | split | In Progress |
| **DMCP-17** | DEN-063 (DEN #63) | move | To Verify |
| **DMCP-18** | DEN-050 (DEN #50) | split | In Progress |
| **DMCP-19** | DEN-046 (DEN #46) | split | To Verify |
| **DMCP-20** | DEN-040 (DEN #40) | move | To Verify |
| **DMCP-21** | DEN-028 (DEN #28) | move | To Verify |
| **DMCP-22** | DEN-009 (DEN #9) | move | To Do |

---

## 7. Imported Docs

The following 15 documents in [`docs/imports.json`](imports.json) were verified in `/Users/yasuomaidana/GoLandProjects/Dendrite` and selected for import into `docs/imported/`:

1. `docs/architecture/2026-10-quality-audit-execution-plan.md` — Defines the authoritative Pillar 5 (Penpot MCP + Plane AI) and Pillar 9 five-repo decomposition boundary for `dendrite-mcp-hub`.
2. `docs/features/FEAT-003-reverse-engineering-rea.md` — Specifies the REA Disassembler MCP Bridge (`DEN-009`, `pkg/rea/client`) and `dendrite_rea_showcase` MCP tool (`pkg/mcpserver/tools_rea.go`).
3. `docs/features/FEAT-014-jetbrains-ide-mcp-integration.md` — Specifies the `pkg/jetbrains` MCP client, HTTP/Stdio transports, IDE `Bridge`, and `.ai/mcp/mcp.json` configuration generator (`DEN-028`, `DEN-046`).
4. `docs/features/FEAT-028-local-agent-triage-gateway-token-efficiency.md` — Specifies the `dendrite_local_agent_triage` MCP tool (`pkg/mcpserver/tools_triage.go`) and JetBrains AI Chat multi-MCP routing (`DEN-050`).
5. `docs/features/FEAT-037-penpot-ui-documentation-mcp-integration.md` — Specifies Penpot UI design-token diff synthesis (`pkg/ui/penpot`), `dendrite_ui_penpot_sync` (`pkg/mcpserver/tools_penpot.go`), and Plane issue deep-linking (`DEN-076`).
6. `docs/features/FEAT-042-artifact-package-publisher-mcp.md` — Specifies the Artifact & Package Publisher MCP service (`pkg/mcp/publisher`) and `dendrite_publisher_*` MCP tools (`DEN-091`..`093`).
7. `docs/features/FEAT-045-code-knowledge-mcp-graphify.md` — Specifies the Code Knowledge MCP server (`DEND-201`) exposing `graphify.*` and `knowledge.*` tools over `.dendrite/knowledge.db`.
8. `docs/features/FEAT-046-multi-ide-plugins.md` — Specifies the quad-IDE plugin suite (`DEND-302`..`305`) and cross-language IPC contract drift monitor (`DEN-063`, `pkg/polyglot/contract`).
9. `docs/features/FEAT-100-multi-ide-ecosystem-support.md` — Defines the shared JSON-RPC 2.0 / MCP stdio core engine and thin IDE adapter architecture.
10. `docs/features/FEAT-101-personal-context-topic-memory-installer.md` — Defines the Canvas Penpot UI bridge (`pkg/personalcontext/canvas`, `pkg/mcpserver/tools_canvas.go`) and Personal Context MCP Connectors (`DEN-808`).
11. `docs/cli/mcp.md` — Documents the `dendrite mcp serve`, `dendrite mcp tools`, and `dendrite mcp call` CLI commands backed by `pkg/mcpserver`.
12. `docs/cli/jetbrains.md` — Documents the `dendrite jetbrains` CLI subcommands (`probe`, `open`, `problems`, `symbols`, `build`, `run`, `setup`) backed by `pkg/jetbrains`.
13. `docs/cli/canvas.md` — Documents the `dendrite canvas` CLI and the `dendrite_canvas_action_dispatch` / `dendrite_canvas_penpot_bridge` MCP tools.
14. `docs/guides/jetbrains-mcp-and-agents-setup.md` — Provides the full setup guide and reference catalog of Dendrite MCP tools and JetBrains IDE configurations.
15. `docs/guides/using-dendrite-tools-with-jetbrains.md` — Documents developer workflows combining the Dendrite MCP server (`pkg/mcpserver`) and reverse IDE bridge (`pkg/jetbrains`).

*(Note: All 87 generated MCP tool reference files under `Dendrite/docs/mcp/*.md` are copied mechanically into `docs/imported/mcp/` by the orchestrator and intentionally omitted from `docs/imports.json`.)*

---

## 8. Open Questions & Boundary Conflicts Noticed

1. **`pkg/mcpserver` Is Not Currently Self-Contained (28+ Control-Plane Package Imports)**:
   - Section 5 of `2026-10-quality-audit-execution-plan.md` states that `pkg/mcpserver` is *"already self-contained, 7.5k LOC"*. While `pkg/mcpserver/server.go` and `pkg/mcpserver/types.go` use only Go stdlib, `pkg/mcpserver/tools*.go` directly imports **28+ internal `Dendrite/pkg/...` control-plane packages** (`pkg/agent`, `pkg/audit`, `pkg/devtool`, `pkg/distcodebase`, `pkg/docusaurus`, `pkg/env`, `pkg/gateway`, `pkg/inference`, `pkg/node/migrate`, `pkg/ollama`, `pkg/perf`, `pkg/pool/...`, `pkg/quality`, `pkg/query/...`, `pkg/rag`, `pkg/scheduler/...`, `pkg/store/...`, `pkg/story/fork`, `pkg/telemetry`, `pkg/testgen`, `pkg/tokens`, `pkg/topology/...`, `pkg/tracker/...`, `pkg/vcs`, `pkg/workspace`).
   - **Resolution Needed in `DMCP-1`**: Extract `pkg/mcpserver/{server,types}.go`, `pkg/jetbrains`, `pkg/polyglot/contract`, and hub-native tool handlers into `dendrite-mcp-hub` first, and either keep control-plane `tools_*.go` registration in `dendrite-control` (injecting them via `Server.RegisterTool`) or decouple them behind a provider interface / RPC proxy so `dendrite-mcp-hub` never imports `dendrite-control`.
2. **Go Module Path & Version Mismatch (`dendrite-mcp-hub` vs. `dendrite.dev/mcphub`)**:
   - `/Users/yasuomaidana/GoLandProjects/dendrite-mcp-hub/go.mod` currently declares `module dendrite-mcp-hub` (`go 1.22`), and `/Users/yasuomaidana/GoLandProjects/Dendrite/go.mod` declares `module Dendrite` (`go 1.25`), whereas Pillar 9 specifies `module dendrite.dev/mcphub` and `module dendrite.dev/control` linked via `~/GoLandProjects/go.work`.
3. **Duplicated JSON-RPC 2.0 / MCP Protocol Types Across `pkg/mcpserver` and `pkg/jetbrains`**:
   - Both `pkg/mcpserver/types.go` (server-side) and `pkg/jetbrains/types.go` (client-side) independently define `JSONRPCRequest`, `JSONRPCResponse`, `JSONRPCError`, `Tool`, `CallToolParams`, `CallToolResult`, and `ContentItem` (with slight differences: `mcpserver.CallToolParams.Arguments` is `map[string]interface{}` vs. `map[string]any` in `jetbrains`, and `mcpserver.ServerInfo` includes `Capabilities`). Extracting both into `dendrite-mcp-hub` should unify shared MCP wire types.
4. **`DEND-201` (Code Knowledge MCP) Status vs. Codebase Reality**:
   - `DEND-201` is marked `To Verify` in Plane and documented in `FEAT-045-code-knowledge-mcp-graphify.md` (`graphify.*` / `knowledge.*` tools), but no `graphify` or `knowledge` MCP tools exist in `pkg/mcpserver/` today.
5. **`MockTransport` Documentation Drift in `FEAT-014`**:
   - `docs/features/FEAT-014-jetbrains-ide-mcp-integration.md` documents a `MockTransport` in `pkg/jetbrains/transport.go`, whereas actual code in `pkg/jetbrains/transport.go` implements only `HTTPTransport` and `StdioTransport` (using `scriptedServer` in `jetbrains_test.go` and enforcing the Zero-Mock live connection policy in `cmd/dendrite/main.go`).
