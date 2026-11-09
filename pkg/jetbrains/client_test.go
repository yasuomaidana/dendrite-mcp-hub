package jetbrains_test

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"

	"github.com/yasuomaidana/dendrite-mcp-hub/pkg/jetbrains"
)

func connectedClient(t *testing.T) *jetbrains.Client {
	t.Helper()
	_, transport := newScriptedServer(t)
	client, err := jetbrains.NewClient(transport,
		jetbrains.WithClientInfo("test-agent", "1.0.0"),
		jetbrains.WithProtocolVersion(jetbrains.DefaultProtocolVersion),
	)
	requireOK(t, err)
	expect(t, !client.IsConnected(), "expected client to not be connected initially")
	return client
}

func TestClient_Connect(t *testing.T) {
	ctx := context.Background()
	client := connectedClient(t)

	initRes, err := client.Connect(ctx)
	requireOK(t, err)
	expect(t, client.IsConnected(), "expected client to be connected after Connect()")
	expect(t, initRes.ProtocolVersion == jetbrains.DefaultProtocolVersion, "unexpected protocol version %s", initRes.ProtocolVersion)
	info := client.GetServerInfo()
	expect(t, info != nil && info.Name == "JetBrains GoLand MCP Server", "unexpected server info: %+v", info)
	caps := client.GetServerCapabilities()
	expect(t, caps != nil && caps.Tools != nil, "expected server capabilities to include tools")

	_, err = client.Connect(ctx)
	requireIs(t, err, jetbrains.ErrAlreadyConnected)
}

func TestClient_PingAndListTools(t *testing.T) {
	ctx := context.Background()
	client := connectedClient(t)
	_, err := client.Connect(ctx)
	requireOK(t, err)

	latency, err := client.Ping(ctx)
	requireOK(t, err)
	expect(t, latency >= 0, "invalid ping latency: %v", latency)

	tools, err := client.ListTools(ctx)
	requireOK(t, err)
	names := make([]string, 0, len(tools))
	for _, tool := range tools {
		names = append(names, tool.Name)
	}
	expect(t, slices.Contains(names, "open_file_in_editor"), "expected 'open_file_in_editor' tool in list, got %v", names)
}

func TestClient_CallTool(t *testing.T) {
	ctx := context.Background()
	client := connectedClient(t)
	_, err := client.Connect(ctx)
	requireOK(t, err)

	callRes, err := client.CallTool(ctx, "open_file_in_editor", map[string]interface{}{
		"file_path": "main.go",
		"line":      10,
	})
	requireOK(t, err)
	expect(t, len(callRes.Content) > 0 && callRes.Content[0].Text == "File opened successfully", "unexpected content: %+v", callRes.Content)

	_, err = client.CallTool(ctx, "non_existent_tool", nil)
	requireIs(t, err, jetbrains.ErrToolNotFound)
	_, err = client.CallTool(ctx, "", nil)
	requireIs(t, err, jetbrains.ErrEmptyToolName)
}

func TestClient_Disconnect(t *testing.T) {
	ctx := context.Background()
	client := connectedClient(t)
	_, err := client.Connect(ctx)
	requireOK(t, err)

	requireOK(t, client.Disconnect())
	expect(t, !client.IsConnected(), "expected client to be disconnected")
	requireOK(t, client.Disconnect()) // idempotent

	_, err = client.ListTools(ctx)
	requireIs(t, err, jetbrains.ErrNotConnected)
	_, err = client.CallTool(ctx, "open_file_in_editor", nil)
	requireIs(t, err, jetbrains.ErrNotConnected)
	_, err = client.Ping(ctx)
	requireIs(t, err, jetbrains.ErrNotConnected)
}

// rpcErrorHandler answers every request with a JSON-RPC error.
func rpcErrorHandler(code int, msg string) func(*jetbrains.JSONRPCRequest) (*jetbrains.JSONRPCResponse, error) {
	return func(req *jetbrains.JSONRPCRequest) (*jetbrains.JSONRPCResponse, error) {
		return &jetbrains.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &jetbrains.JSONRPCError{Code: code, Message: msg},
		}, nil
	}
}

func TestClient_NilTransport(t *testing.T) {
	_, err := jetbrains.NewClient(nil)
	if !errors.Is(err, jetbrains.ErrNilTransport) {
		t.Errorf("expected ErrNilTransport, got: %v", err)
	}
}

func TestClient_ErrorResponses(t *testing.T) {
	srv, transport := newScriptedServer(t)
	srv.RegisterHandler("initialize", rpcErrorHandler(-32000, "handshake rejected"))

	client, err := jetbrains.NewClient(transport)
	requireOK(t, err)

	_, err = client.Connect(context.Background())
	requireIs(t, err, jetbrains.ErrInvalidResponse)
}

func TestClient_ListTools_And_Ping_Errors(t *testing.T) {
	ctx := context.Background()
	srv, transport := newScriptedServer(t)

	client, err := jetbrains.NewClient(transport)
	requireOK(t, err)
	_, _ = client.Connect(ctx)

	srv.RegisterHandler("tools/list", rpcErrorHandler(-32603, "failed listing tools"))
	_, err = client.ListTools(ctx)
	requireIs(t, err, jetbrains.ErrInvalidResponse)

	srv.RegisterHandler("ping", rpcErrorHandler(-32000, "ping failed"))
	_, err = client.Ping(ctx)
	requireIs(t, err, jetbrains.ErrInvalidResponse)
}

func TestClient_ToolExecutionFailure(t *testing.T) {
	ctx := context.Background()
	srv, transport := newScriptedServer(t)

	srv.RegisterHandler("tools/call", func(req *jetbrains.JSONRPCRequest) (*jetbrains.JSONRPCResponse, error) {
		res := jetbrains.CallToolResult{
			IsError: true,
			Content: []jetbrains.ContentItem{{Type: "text", Text: "Internal compilation failure"}},
		}
		data, _ := json.Marshal(res)
		return &jetbrains.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  data,
		}, nil
	})

	client, err := jetbrains.NewClient(transport)
	requireOK(t, err)
	_, _ = client.Connect(ctx)

	_, err = client.CallTool(ctx, "build_project", nil)
	requireIs(t, err, jetbrains.ErrExecutionFailed)
}

type recordedTransport struct {
	sendFn  func(ctx context.Context, req *jetbrains.JSONRPCRequest) (*jetbrains.JSONRPCResponse, error)
	closeFn func() error
}

func (r *recordedTransport) Send(ctx context.Context, req *jetbrains.JSONRPCRequest) (*jetbrains.JSONRPCResponse, error) {
	return r.sendFn(ctx, req)
}

func (r *recordedTransport) Close() error {
	if r.closeFn != nil {
		return r.closeFn()
	}
	return nil
}

type sendFn = func(context.Context, *jetbrains.JSONRPCRequest) (*jetbrains.JSONRPCResponse, error)

func failWith(msg string) sendFn {
	return func(context.Context, *jetbrains.JSONRPCRequest) (*jetbrains.JSONRPCResponse, error) {
		return nil, errors.New(msg)
	}
}

func replyWith(result string) sendFn {
	return func(_ context.Context, req *jetbrains.JSONRPCRequest) (*jetbrains.JSONRPCResponse, error) {
		return &jetbrains.JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: json.RawMessage(result)}, nil
	}
}

func newRecordedClient(t *testing.T, fn sendFn) (*jetbrains.Client, *recordedTransport) {
	t.Helper()
	rt := &recordedTransport{sendFn: fn}
	c, err := jetbrains.NewClient(rt)
	requireOK(t, err)
	return c, rt
}

func TestClient_ConnectErrorPaths(t *testing.T) {
	ctx := context.Background()
	c, rt := newRecordedClient(t, failWith("network failure"))
	expect(t, c.GetServerInfo() == nil, "expected nil ServerInfo before Connect")
	expect(t, c.GetServerCapabilities() == nil, "expected nil ServerCapabilities before Connect")

	_, err := c.Connect(ctx)
	requireErr(t, err)

	rt.sendFn = replyWith(`{bad`)
	_, err = c.Connect(ctx)
	requireIs(t, err, jetbrains.ErrInvalidResponse)
}

func TestClient_ListToolsErrorPaths(t *testing.T) {
	ctx := context.Background()
	c, rt := newRecordedClient(t, replyWith(`{"protocolVersion":"2024-11-05"}`))
	_, err := c.Connect(ctx)
	requireOK(t, err)

	rt.sendFn = failWith("send failed")
	_, err = c.ListTools(ctx)
	requireErr(t, err)

	rt.sendFn = replyWith(`{bad`)
	_, err = c.ListTools(ctx)
	requireIs(t, err, jetbrains.ErrInvalidResponse)
}

func TestClient_CallToolErrorPaths(t *testing.T) {
	ctx := context.Background()
	c, rt := newRecordedClient(t, replyWith(`{"protocolVersion":"2024-11-05"}`))
	_, err := c.Connect(ctx)
	requireOK(t, err)

	cases := []struct {
		name string
		fn   sendFn
		want error
	}{
		{"transport error", failWith("send failed"), nil},
		{"non-32601 rpc error", func(_ context.Context, req *jetbrains.JSONRPCRequest) (*jetbrains.JSONRPCResponse, error) {
			return &jetbrains.JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Error: &jetbrains.JSONRPCError{Code: -32000, Message: "server failure"}}, nil
		}, jetbrains.ErrExecutionFailed},
		{"invalid result json", replyWith(`{bad`), jetbrains.ErrInvalidResponse},
		{"isError with empty content", replyWith(`{"isError":true}`), jetbrains.ErrExecutionFailed},
	}
	for _, tc := range cases {
		rt.sendFn = tc.fn
		_, err := c.CallTool(ctx, "x", nil)
		requireErr(t, err)
		expect(t, tc.want == nil || errors.Is(err, tc.want), "%s: expected %v, got %v", tc.name, tc.want, err)
	}
}

func TestClient_PingTransportError(t *testing.T) {
	ctx := context.Background()
	c, rt := newRecordedClient(t, replyWith(`{"protocolVersion":"2024-11-05"}`))
	_, err := c.Connect(ctx)
	requireOK(t, err)

	rt.sendFn = failWith("ping network down")
	_, err = c.Ping(ctx)
	requireErr(t, err)
}
