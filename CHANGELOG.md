# Changelog

All notable changes are documented here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project follows [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added
- CI: `.github/workflows/release-refs.yml` runs `scripts/check-release-refs.sh` on every pull request, the same guard as the other Dendrite repositories (DMCP-33).
- CI: `.github/workflows/complexity.yml` cross-checks function complexity with `gocyclo` and `gocognit`; report only while the debt (DMCP-28) is open (DMCP-27).

## [0.1.0]

First release as a standalone module, `github.com/yasuomaidana/dendrite-mcp-hub` (DEN-971, DMCP-1).

### Added
- `pkg/mcpserver`: the MCP JSON-RPC 2.0 server core and its types, merged from the Dendrite control repository.
- `pkg/jetbrains`: the reverse IDE client and configuration bridge, with merge-safe MCP registration and per-synapse IDE scopes (DEN-1004); `inter serve` registration and run configurations that use the cobra command names (DEN-1032).
- `pkg/polyglot/contract`: monitor for cross-language IPC contract drift.
- Synapse identity and the `dendrite-ecosystem` MCP entry for this repository; own technical specification (`docs/ARCHITECTURE.md`) and responsibilities (`docs/RESPONSIBILITIES.md`).
- CI quality workflow: no code file over 500 lines and no duplicated blocks (DMCP-11).

### Changed
- The default MCP binary written into IDE configuration is the running `dendrite` executable, never `<project>/bin/dendrite`.
- The four Go files over the line limit were split (DMCP-11).
- `pkg/jetbrains` and `pkg/polyglot/contract` reach 100% test coverage: repeated blocks were removed, error paths that cannot happen were dropped, and process-level calls (working directory, executable lookup, stdin and stdout) sit behind small seams so their failures are tested (DMCP-25).
- Imported Dendrite documents were reduced to the ones native to the hub.

### Not in this release
- The MCP tool implementations (`pkg/mcpserver` tools) still live in the control repository; their move is tracked as DMCP-1.
