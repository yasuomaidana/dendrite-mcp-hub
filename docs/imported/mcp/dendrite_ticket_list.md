<!-- Imported from Dendrite docs/mcp/dendrite_ticket_list.md @ 69ea048 (heal/qa-m0) on 2026-10-18. Generated reference: regenerate in Dendrite until pkg/mcpserver is extracted (DEN-971). -->

---
id: dendrite_ticket_list
title: "dendrite_ticket_list"
sidebar_label: "dendrite_ticket_list"
---

# dendrite_ticket_list

List all project tickets, their statuses, complexity scores, and milestones from DEV_STATUS.md

## Input Schema

```json
{
  "properties": {
    "file": {
      "description": "Path to DEV_STATUS.md (default: DEV_STATUS.md)",
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
      "file": "DEV_STATUS.md"
    },
    "name": "dendrite_ticket_list"
  }
}
```

### CLI

```bash
dendrite mcp call dendrite_ticket_list '{"file":"DEV_STATUS.md"}'
```
