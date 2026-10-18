<!-- Imported from Dendrite docs/mcp/dendrite_ref_list.md @ 69ea048 (heal/qa-m0) on 2026-10-18. Generated reference: regenerate in Dendrite until pkg/mcpserver is extracted (DEN-971). -->

---
id: dendrite_ref_list
title: "dendrite_ref_list"
sidebar_label: "dendrite_ref_list"
---

# dendrite_ref_list

Lists registered Tier 1 source references for library reuse

## Input Schema

```json
{
  "properties": {},
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
    "arguments": {},
    "name": "dendrite_ref_list"
  }
}
```

### CLI

```bash
dendrite mcp call dendrite_ref_list
```
