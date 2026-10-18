<!-- Imported from Dendrite docs/mcp/dendrite_inference_list.md @ 69ea048 (heal/qa-m0) on 2026-10-18. Generated reference: regenerate in Dendrite until pkg/mcpserver is extracted (DEN-971). -->

---
id: dendrite_inference_list
title: "dendrite_inference_list"
sidebar_label: "dendrite_inference_list"
---

# dendrite_inference_list

List registered multi-provider inference engines (Ollama, OxiLLaMa, vLLM, TGI, TEI, llama.cpp), endpoints, and failover cascade order

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
    "name": "dendrite_inference_list"
  }
}
```

### CLI

```bash
dendrite mcp call dendrite_inference_list
```
