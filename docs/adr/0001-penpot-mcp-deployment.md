# ADR 0001: How Dendrite reaches Penpot through the hub (DMCP #3, spike)

Status: accepted, 2026-11-17. Ticket: DMCP #3 (was DEN-930). Consumers: DMCP #2 (Penpot upstream) and DEN-836 (Penpot / code / Plane glue).

## Question

Penpot ships an official MCP server. Can the hub adopt it as the Penpot upstream, or must the hub build its own tools? The adopt-or-build rule of the triad spec (`docs/architecture/2026-11-penpot-triad.md` in the control repository) says to adopt when the official one fits.

## What the official server is (verified 2026-11-17)

Sources: the [`penpot/penpot` repository, `mcp/`](https://github.com/penpot/penpot/tree/develop/mcp) and the [Penpot MCP help page](https://help.penpot.app/mcp/). The npm registry lists `@penpot/mcp` 2.15.4 as `latest` and `stable`.

| Aspect | Fact |
|---|---|
| Package | npm `@penpot/mcp` (`npx -y @penpot/mcp@stable`), Node.js 20 or later. No official Docker image: the repository documents none, and the Docker builds that exist are third-party. |
| Transport | Streamable HTTP at `http://localhost:4401/mcp` (`PENPOT_MCP_SERVER_PORT`). The legacy `/sse` and `/messages` endpoints are gone. Stdio-only clients use `mcp-remote`. |
| How it reaches the design | Through a **Penpot plugin running in a browser tab**. The server opens a WebSocket (`PENPOT_MCP_WEBSOCKET_PORT`, 4402); the plugin, loaded from `http://localhost:4400/manifest.json`, runs the Plugin API calls and answers. Only one browser tab works at a time, it acts on the focused page, and the tab must stay open and active. |
| Backend API | Only `submit_feedback` uses it (`PENPOT_MCP_BACKEND_URI`). |
| Auth | Local single-user: none. Multi-user: OAuth (`PENPOT_MCP_AUTH_URI`, needs `--multi-user`). The hosted server takes a personal MCP key as `userToken` in the URL. |
| Tools | Retrieve, modify and create design elements, run Plugin API code, `import_image`, `export_shape`; the repository lists no fixed tool inventory. |

## Decision

1. **Do not make the official MCP server the headless source of design context.** It needs a human's browser with a live plugin session, so a git hook, a `chronos context` call or a post-merge job (the three edges of DEN-836) cannot depend on it. No process of ours can be sure the tab is open.
2. **The hub's Penpot tools read the Penpot RPC API with an access token** (`/api/rpc/command/*`, DEN-835). This is headless, scriptable and testable against an `httptest` server. The client lives in the control repository (`pkg/ui/penpot/rpc`); the hub gets the tool set `penpot_*` (tokens, components, frames, SVG export) through a host-neutral port, because the hub never imports the control repository.
3. **The official server stays available as an interactive upstream.** The generic upstream proxy (DMCP #2) can register it (`transport = "http"`, `url = http://localhost:4401/mcp`) for sessions where a person has Penpot open and wants an agent to edit the canvas. It is not required by any automated path and is off by default.
4. **Revisit** when the official server gains a headless mode (a backend-driven tool set) or a stable Docker image.

## Consequences

- DMCP #2 builds the generic `pkg/upstream` proxy first (stdio and streamable HTTP, through `pkg/mcpclient`) and the host-neutral `penpot_*` tool set second.
- Self-hosted Penpot needs the `enable-access-tokens` flag (DEN-835, done) for the RPC path. The official server's own flags (`enable-mcp`, `PENPOT_MCP_URI` on the frontend) are only needed if the interactive upstream is used.
- Not verified live: running `@penpot/mcp` against the local Penpot needs a browser tab, which no automated session can provide. What was verified is the published documentation and the package versions above.

## Alternatives considered

- **Adopt the official server only.** Rejected: not headless.
- **A third-party Penpot MCP server** (for example the community Python ones advertising dozens of tools). Rejected: they add a second implementation to maintain and trust, and the RPC client already covers the reads DEN-836 needs.
