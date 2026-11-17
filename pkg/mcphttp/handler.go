// Package mcphttp serves a hub MCP server over the streamable-HTTP transport of the MCP specification
// (sessions, JSON or SSE responses, resumption), using the official go-sdk for the protocol. The hand-rolled
// stdio server stays as it is; this package only bridges its tools onto an HTTP handler.
package mcphttp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/yasuomaidana/dendrite-mcp-hub/pkg/mcpserver"
)

// Options tunes the HTTP handler; the zero value is valid.
type Options struct {
	// JSONResponse answers each POST with application/json instead of a text/event-stream.
	JSONResponse bool
}

// Handler serves the tools registered on srv at the moment of the call. Register every tool first: the
// handler does not see tools added later.
func Handler(srv *mcpserver.Server, opts Options) http.Handler {
	sdk := bridge(srv)
	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return sdk }, &mcp.StreamableHTTPOptions{
		JSONResponse: opts.JSONResponse,
	})
}

// bridge builds a go-sdk server whose tools call the hub server's handlers.
func bridge(srv *mcpserver.Server) *mcp.Server {
	info := srv.Info()
	sdk := mcp.NewServer(&mcp.Implementation{Name: info.Name, Version: info.Version}, nil)
	for _, tool := range srv.ListTools() {
		if handler, ok := srv.Handler(tool.Name); ok {
			sdk.AddTool(&mcp.Tool{Name: tool.Name, Description: tool.Description, InputSchema: objectSchema(tool.InputSchema)}, call(handler))
		}
	}
	return sdk
}

// objectSchema guarantees the {"type":"object"} root the go-sdk insists on.
func objectSchema(schema map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(schema)+1)
	for k, v := range schema {
		out[k] = v
	}
	out["type"] = "object"
	return out
}

// call adapts a hub tool handler. Like the stdio server, a handler error becomes an isError result.
func call(handler mcpserver.ToolHandler) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := map[string]interface{}{}
		if len(req.Params.Arguments) > 0 {
			if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
				return nil, fmt.Errorf("invalid tool arguments: %w", err)
			}
		}
		res, err := handler(ctx, args)
		if err != nil {
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Tool error: %v", err)}}}, nil
		}
		return convert(res), nil
	}
}

// convert maps hub content items to go-sdk content: images carry base64 data, anything else is text.
func convert(res *mcpserver.CallToolResult) *mcp.CallToolResult {
	out := &mcp.CallToolResult{IsError: res.IsError, Content: make([]mcp.Content, 0, len(res.Content))}
	for _, item := range res.Content {
		out.Content = append(out.Content, content(item))
	}
	return out
}

func content(item mcpserver.ContentItem) mcp.Content {
	if item.Type == "image" {
		if data, err := base64.StdEncoding.DecodeString(item.Data); err == nil {
			return &mcp.ImageContent{Data: data, MIMEType: item.MimeType}
		}
	}
	return &mcp.TextContent{Text: item.Text}
}
