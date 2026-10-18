<!-- Imported from Dendrite docs/cli/mcp.md @ 69ea048 on 2026-10-18. Source of truth for this repo's scope; keep in sync with the Dendrite copy until the extraction umbrella lands. -->

---
id: mcp
title: "dendrite mcp"
sidebar_label: "mcp"
---

# dendrite mcp

Model Context Protocol (MCP) server over stdio

## Category

Synaptic Cleft & Gating

## Aliases

- `interneuron-mcp`
- `inter`

## Subcommands

| Subcommand | Description |
| :--- | :--- |
| `serve` | Launch Dendrite MCP server listening on stdio |
| `tools` | List all registered Dendrite MCP tools |
| `call` | Invoke an MCP tool directly with JSON arguments |

## Flags

None

## Examples

```bash
dendrite mcp serve
dendrite mcp tools
dendrite mcp call dendrite_env_status
```
