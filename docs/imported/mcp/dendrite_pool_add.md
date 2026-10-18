<!-- Imported from Dendrite docs/mcp/dendrite_pool_add.md @ 69ea048 (heal/qa-m0) on 2026-10-18. Generated reference: regenerate in Dendrite until pkg/mcpserver is extracted (DEN-971). -->

---
id: dendrite_pool_add
title: "dendrite_pool_add"
sidebar_label: "dendrite_pool_add"
---

# dendrite_pool_add

Register one or more project directories into .dendrite/pool.json

## Input Schema

```json
{
  "properties": {
    "directories": {
      "description": "List of project directory paths to register",
      "items": {
        "type": "string"
      },
      "type": "array"
    }
  },
  "required": [
    "directories"
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
      "directories": [
        "item1"
      ]
    },
    "name": "dendrite_pool_add"
  }
}
```

### CLI

```bash
dendrite mcp call dendrite_pool_add '{"directories":["item1"]}'
```
