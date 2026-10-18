<!-- Imported from Dendrite docs/mcp/dendrite_story_resolve.md @ 69ea048 (heal/qa-m0) on 2026-10-18. Generated reference: regenerate in Dendrite until pkg/mcpserver is extracted (DEN-971). -->

---
id: dendrite_story_resolve
title: "dendrite_story_resolve"
sidebar_label: "dendrite_story_resolve"
---

# dendrite_story_resolve

Merge winning variant worktree, persist benchmark metrics into store, and purge alternate worktrees

## Input Schema

```json
{
  "properties": {
    "story_id": {
      "description": "Target story identifier",
      "type": "string"
    },
    "winner": {
      "description": "Name of the winning variant to merge (e.g. 'v2' or 'astar-bitvec')",
      "type": "string"
    }
  },
  "required": [
    "story_id",
    "winner"
  ],
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
      "story_id": "sample",
      "winner": "sample"
    },
    "name": "dendrite_story_resolve"
  }
}
```

### CLI

```bash
dendrite mcp call dendrite_story_resolve '{"story_id":"sample","winner":"sample"}'
```
