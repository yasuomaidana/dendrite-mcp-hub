package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func call(t *testing.T, s *Server, req string) *JSONRPCResponse {
	t.Helper()
	return s.HandleRequest(context.Background(), []byte(req))
}

func echoServer() *Server {
	s := NewServer(ServerConfig{Name: "test", Version: "9.9.9"})
	s.RegisterTool(Tool{Name: "echo", Description: "echoes msg"}, func(ctx context.Context, args map[string]interface{}) (*CallToolResult, error) {
		msg, _ := args["msg"].(string)
		return &CallToolResult{Content: []ContentItem{{Type: "text", Text: msg}}}, nil
	})
	s.RegisterTool(Tool{Name: "boom"}, func(ctx context.Context, args map[string]interface{}) (*CallToolResult, error) {
		return nil, errors.New("exploded")
	})
	return s
}

func TestNewServerDefaultsAndAccessors(t *testing.T) {
	s := NewServer(ServerConfig{})
	if info := s.Info(); info.Name != DefaultName || info.Version != DefaultVersion {
		t.Fatalf("defaults = %+v", info)
	}
	s = echoServer()
	if s.Info().Name != "test" || s.Info().Version != "9.9.9" {
		t.Fatalf("info = %+v", s.Info())
	}
	if _, ok := s.Handler("echo"); !ok {
		t.Fatal("echo handler missing")
	}
	if _, ok := s.Handler("nope"); ok {
		t.Fatal("unexpected handler")
	}
	if n := len(s.ListTools()); n != 2 {
		t.Fatalf("ListTools = %d", n)
	}
}

func TestHandleRequestProtocol(t *testing.T) {
	s := echoServer()

	resp := call(t, s, `{"jsonrpc":"2.0","id":1,"method":"initialize"}`)
	init, ok := resp.Result.(InitializeResult)
	if !ok || init.ProtocolVersion != ProtocolVersion || init.ServerInfo.Name != "test" || init.Capabilities.Tools == nil {
		t.Fatalf("initialize = %+v", resp)
	}
	if call(t, s, `{"jsonrpc":"2.0","method":"notifications/initialized"}`) != nil {
		t.Fatal("notification must not be answered")
	}
	if resp := call(t, s, `{"jsonrpc":"2.0","id":2,"method":"ping"}`); resp.Error != nil {
		t.Fatalf("ping = %+v", resp)
	}
	if resp := call(t, s, `{"jsonrpc":"2.0","id":3,"method":"tools/list"}`); len(resp.Result.(ListToolsResult).Tools) != 2 {
		t.Fatalf("tools/list = %+v", resp)
	}

	resp = call(t, s, `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"echo","arguments":{"msg":"hi"}}}`)
	if res, ok := resp.Result.(*CallToolResult); !ok || res.Content[0].Text != "hi" {
		t.Fatalf("echo = %+v", resp)
	}
	resp = call(t, s, `{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"boom"}}`)
	if res, ok := resp.Result.(CallToolResult); !ok || !res.IsError || !strings.Contains(res.Content[0].Text, "exploded") {
		t.Fatalf("boom = %+v", resp)
	}

	for req, code := range map[string]int{
		`{not json`: ErrCodeParseError,
		`{"jsonrpc":"2.0","id":6,"method":"tools/call","params":"x"}`:              ErrCodeInvalidParams,
		`{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"ghost"}}`: ErrCodeMethodNotFound,
		`{"jsonrpc":"2.0","id":8,"method":"resources/list"}`:                       ErrCodeMethodNotFound,
	} {
		if resp := call(t, s, req); resp.Error == nil || resp.Error.Code != code {
			t.Errorf("%s: resp = %+v, want code %d", req, resp, code)
		}
	}
}

func TestServeStdio(t *testing.T) {
	s := echoServer()
	in := strings.NewReader(strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"ping"}`,
		``,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"echo","arguments":{"msg":"x"}}}`,
	}, "\n"))
	var out bytes.Buffer
	if err := s.ServeStdio(context.Background(), in, &out); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("responses = %q", out.String())
	}
	var resp JSONRPCResponse
	if err := json.Unmarshal([]byte(lines[1]), &resp); err != nil || resp.Error != nil {
		t.Fatalf("second response = %s, %v", lines[1], err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := s.ServeStdio(ctx, strings.NewReader(`{"id":1,"method":"ping"}`), &out); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled ServeStdio err = %v", err)
	}
	if err := s.ServeStdio(context.Background(), strings.NewReader(`{"id":1,"method":"ping"}`), failingWriter{}); err == nil {
		t.Fatal("expected write error")
	}
}

// failingWriter is an io.Writer whose writes always fail (a closed pipe).
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("broken pipe") }
