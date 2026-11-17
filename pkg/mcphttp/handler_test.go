package mcphttp_test

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yasuomaidana/dendrite-mcp-hub/pkg/mcpclient"
	"github.com/yasuomaidana/dendrite-mcp-hub/pkg/mcphttp"
	"github.com/yasuomaidana/dendrite-mcp-hub/pkg/mcpserver"
)

// hubServer is a hub server with one tool per behaviour the bridge must carry over.
func hubServer() *mcpserver.Server {
	srv := mcpserver.NewServer(mcpserver.ServerConfig{Name: "bridge", Version: "3.1"})
	text := func(s string) *mcpserver.CallToolResult {
		return &mcpserver.CallToolResult{Content: []mcpserver.ContentItem{{Type: "text", Text: s}}}
	}
	srv.RegisterTool(mcpserver.Tool{Name: "echo", Description: "echoes", InputSchema: map[string]interface{}{
		"properties": map[string]interface{}{"text": map[string]interface{}{"type": "string"}},
	}}, func(_ context.Context, args map[string]interface{}) (*mcpserver.CallToolResult, error) {
		s, _ := args["text"].(string)
		return text("echo:" + s), nil
	})
	srv.RegisterTool(mcpserver.Tool{Name: "boom"}, func(context.Context, map[string]interface{}) (*mcpserver.CallToolResult, error) {
		return nil, errors.New("exploded")
	})
	srv.RegisterTool(mcpserver.Tool{Name: "flagged"}, func(context.Context, map[string]interface{}) (*mcpserver.CallToolResult, error) {
		res := text("soft failure")
		res.IsError = true
		return res, nil
	})
	srv.RegisterTool(mcpserver.Tool{Name: "image"}, func(context.Context, map[string]interface{}) (*mcpserver.CallToolResult, error) {
		return &mcpserver.CallToolResult{Content: []mcpserver.ContentItem{
			{Type: "image", Data: base64.StdEncoding.EncodeToString([]byte("PNG")), MimeType: "image/png"},
			{Type: "image", Data: "!!not base64!!", Text: "fallback text"},
		}}, nil
	})
	return srv
}

func connect(t *testing.T, opts mcphttp.Options) *mcpclient.Client {
	t.Helper()
	ts := httptest.NewServer(mcphttp.Handler(hubServer(), opts))
	t.Cleanup(ts.Close)
	c, err := mcpclient.NewClient(mcpclient.NewStreamableHTTPTransport(ts.URL, ts.Client()))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Connect(context.Background()); err != nil {
		t.Fatalf("handshake: %v", err)
	}
	t.Cleanup(func() { _ = c.Disconnect() })
	return c
}

// Both response modes of the specification: an SSE stream per POST, and a plain JSON body.
func TestEveryToolOfTheHubServerIsReachableOverHTTP(t *testing.T) {
	for name, opts := range map[string]mcphttp.Options{"sse": {}, "json": {JSONResponse: true}} {
		t.Run(name, func(t *testing.T) {
			c := connect(t, opts)
			checkIdentityAndSchemas(t, c)
			ctx := context.Background()
			res, err := c.CallTool(ctx, "echo", map[string]interface{}{"text": "hi"})
			if err != nil || res.Content[0].Text != "echo:hi" {
				t.Fatalf("echo = %+v, err = %v", res, err)
			}
			if _, err := c.Ping(ctx); err != nil {
				t.Errorf("ping: %v", err)
			}
		})
	}
}

func checkIdentityAndSchemas(t *testing.T, c *mcpclient.Client) {
	t.Helper()
	if info := c.GetServerInfo(); info.Name != "bridge" || info.Version != "3.1" {
		t.Errorf("server info = %+v", info)
	}
	tools, err := c.ListTools(context.Background())
	if err != nil || len(tools) != 4 {
		t.Fatalf("tools = %+v, err = %v", tools, err)
	}
	for _, tool := range tools {
		if tool.InputSchema["type"] != "object" {
			t.Errorf("%s: schema root is %v, want object", tool.Name, tool.InputSchema["type"])
		}
	}
}

func TestHandlerErrorsAndFlaggedResultsBecomeIsErrorResults(t *testing.T) {
	c := connect(t, mcphttp.Options{JSONResponse: true})
	for tool, want := range map[string]string{"boom": "Tool error: exploded", "flagged": "soft failure"} {
		res, err := c.CallTool(context.Background(), tool, nil)
		if !errors.Is(err, mcpclient.ErrExecutionFailed) || res == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: res = %+v, err = %v", tool, res, err)
		}
	}
}

func TestImagesTravelAsBase64AndBadDataFallsBackToText(t *testing.T) {
	c := connect(t, mcphttp.Options{JSONResponse: true})
	res, err := c.CallTool(context.Background(), "image", nil)
	if err != nil || len(res.Content) != 2 {
		t.Fatalf("image = %+v, err = %v", res, err)
	}
	if got := res.Content[0]; got.Type != "image" || got.MimeType != "image/png" || got.Data != base64.StdEncoding.EncodeToString([]byte("PNG")) {
		t.Errorf("image item = %+v", got)
	}
	if got := res.Content[1]; got.Type != "text" || got.Text != "fallback text" {
		t.Errorf("fallback item = %+v", got)
	}
}

func TestAnUnknownToolIsAProtocolError(t *testing.T) {
	c := connect(t, mcphttp.Options{JSONResponse: true})
	if _, err := c.CallTool(context.Background(), "nope", nil); err == nil {
		t.Fatal("an unknown tool must fail")
	}
}
