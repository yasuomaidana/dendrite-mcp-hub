<!-- Imported from Dendrite docs/mcp/dendrite_remote_sync.md @ 69ea048 (heal/qa-m0) on 2026-10-18. Generated reference: regenerate in Dendrite until pkg/mcpserver is extracted (DEN-971). -->

---
id: dendrite_remote_sync
title: "dendrite_remote_sync"
sidebar_label: "dendrite_remote_sync"
---

# dendrite_remote_sync

Fetch and detect remote commits/hotfixes, pull with rebase, and reconcile tickets.

## Input Schema

```json
{
  "properties": {
    "branch": {
      "description": "Branch to sync (default: active branch)",
      "type": "string"
    }
  },
  "type": "object"
}
```

## Example Call

### JSON-RPC

```json
{
  "id": 1,
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "arguments": {
      "branch": "active branch"
    },
    "name": "dendrite_remote_sync"
  }
}
```

### CLI

```bash
dendrite mcp call dendrite_remote_sync '{"branch":"active branch"}'
```
