<!-- Imported from Dendrite docs/mcp/dendrite_store_query_partial.md @ 69ea048 (heal/qa-m0) on 2026-10-18. Generated reference: regenerate in Dendrite until pkg/mcpserver is extracted (DEN-971). -->

---
id: dendrite_store_query_partial
title: "dendrite_store_query_partial"
sidebar_label: "dendrite_store_query_partial"
---

# dendrite_store_query_partial

Query committed segments for a project, including projects currently in ANALYZING_PARTIAL state

## Input Schema

```json
{
  "properties": {
    "project": {
      "description": "Project identifier",
      "type": "string"
    }
  },
  "required": [
    "project"
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
      "project": "sample"
    },
    "name": "dendrite_store_query_partial"
  }
}
```

### CLI

```bash
dendrite mcp call dendrite_store_query_partial '{"project":"sample"}'
```
