package mcphttp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/yasuomaidana/dendrite-mcp-hub/pkg/mcpserver"
)

func TestCallRejectsArgumentsThatAreNotAnObject(t *testing.T) {
	handler := call(func(context.Context, map[string]interface{}) (*mcpserver.CallToolResult, error) {
		return &mcpserver.CallToolResult{}, nil
	})
	req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Name: "x", Arguments: json.RawMessage(`[1,2]`)}}
	if _, err := handler(context.Background(), req); err == nil || !strings.Contains(err.Error(), "invalid tool arguments") {
		t.Fatalf("err = %v", err)
	}
	req.Params.Arguments = nil // no arguments at all is fine
	if _, err := handler(context.Background(), req); err != nil {
		t.Fatalf("no arguments: %v", err)
	}
}

func TestObjectSchemaKeepsPropertiesAndForcesTheObjectRoot(t *testing.T) {
	in := map[string]interface{}{"type": "string", "properties": map[string]interface{}{}}
	out := objectSchema(in)
	if out["type"] != "object" || out["properties"] == nil || in["type"] != "string" {
		t.Fatalf("out = %v, in = %v (the input must not change)", out, in)
	}
	if got := objectSchema(nil); got["type"] != "object" {
		t.Fatalf("nil schema = %v", got)
	}
}
