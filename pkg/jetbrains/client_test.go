package jetbrains_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/yasuomaidana/dendrite-mcp-hub/pkg/jetbrains"
)

func TestClient_Lifecycle_And_Tools(t *testing.T) {
	ctx := context.Background()
	_, transport := newScriptedServer(t)

	client, err := jetbrains.NewClient(transport,
		jetbrains.WithClientInfo("test-agent", "1.0.0"),
		jetbrains.WithProtocolVersion(jetbrains.DefaultProtocolVersion),
	)
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	if client.IsConnected() {
		t.Fatal("expected client to not be connected initially")
	}

	// 1. Connect Handshake
	initRes, err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("connect failed: %v", err)
	}

	if !client.IsConnected() {
		t.Fatal("expected client to be connected after Connect()")
	}

	if initRes.ProtocolVersion != jetbrains.DefaultProtocolVersion {
		t.Errorf("expected protocol version %s, got %s", jetbrains.DefaultProtocolVersion, initRes.ProtocolVersion)
	}

	if client.GetServerInfo() == nil || client.GetServerInfo().Name != "JetBrains GoLand MCP Server" {
		t.Errorf("unexpected server info: %+v", client.GetServerInfo())
	}

	if client.GetServerCapabilities() == nil || client.GetServerCapabilities().Tools == nil {
		t.Error("expected server capabilities to include tools")
	}

	// Connect again should return ErrAlreadyConnected
	if _, err := client.Connect(ctx); !errors.Is(err, jetbrains.ErrAlreadyConnected) {
		t.Errorf("expected ErrAlreadyConnected, got: %v", err)
	}

	// 2. Ping
	latency, err := client.Ping(ctx)
	if err != nil {
		t.Fatalf("ping failed: %v", err)
	}
	if latency < 0 {
		t.Errorf("invalid ping latency: %v", latency)
	}

	// 3. List Tools
	tools, err := client.ListTools(ctx)
	if err != nil {
		t.Fatalf("list tools failed: %v", err)
	}
	if len(tools) == 0 {
		t.Fatal("expected non-empty tool list")
	}

	foundOpen := false
	for _, tool := range tools {
		if tool.Name == "open_file_in_editor" {
			foundOpen = true
			break
		}
	}
	if !foundOpen {
		t.Error("expected 'open_file_in_editor' tool in list")
	}

	// 4. Call Tool - Happy Path
	callRes, err := client.CallTool(ctx, "open_file_in_editor", map[string]interface{}{
		"file_path": "main.go",
		"line":      10,
	})
	if err != nil {
		t.Fatalf("call tool failed: %v", err)
	}
	if len(callRes.Content) == 0 || callRes.Content[0].Text != "File opened successfully" {
		t.Errorf("unexpected call tool content: %+v", callRes.Content)
	}

	// 5. Call Tool - Unknown Tool
	_, err = client.CallTool(ctx, "non_existent_tool", nil)
	if !errors.Is(err, jetbrains.ErrToolNotFound) {
		t.Errorf("expected ErrToolNotFound, got: %v", err)
	}

	// 6. Call Tool - Empty Name
	_, err = client.CallTool(ctx, "", nil)
	if !errors.Is(err, jetbrains.ErrEmptyToolName) {
		t.Errorf("expected ErrEmptyToolName, got: %v", err)
	}

	// 7. Disconnect
	if err := client.Disconnect(); err != nil {
		t.Fatalf("disconnect failed: %v", err)
	}
	if client.IsConnected() {
		t.Fatal("expected client to be disconnected")
	}

	// Double disconnect should be idempotent
	if err := client.Disconnect(); err != nil {
		t.Errorf("expected no error on duplicate disconnect, got: %v", err)
	}

	// Operations after disconnect should fail
	if _, err := client.ListTools(ctx); !errors.Is(err, jetbrains.ErrNotConnected) {
		t.Errorf("expected ErrNotConnected, got: %v", err)
	}
	if _, err := client.CallTool(ctx, "open_file_in_editor", nil); !errors.Is(err, jetbrains.ErrNotConnected) {
		t.Errorf("expected ErrNotConnected, got: %v", err)
	}
	if _, err := client.Ping(ctx); !errors.Is(err, jetbrains.ErrNotConnected) {
		t.Errorf("expected ErrNotConnected, got: %v", err)
	}
}

func TestClient_NilTransport(t *testing.T) {
	_, err := jetbrains.NewClient(nil)
	if !errors.Is(err, jetbrains.ErrNilTransport) {
		t.Errorf("expected ErrNilTransport, got: %v", err)
	}
}

func TestClient_ErrorResponses(t *testing.T) {
	ctx := context.Background()
	srv, transport := newScriptedServer(t)

	// Handler returning JSON-RPC error
	srv.RegisterHandler("initialize", func(req *jetbrains.JSONRPCRequest) (*jetbrains.JSONRPCResponse, error) {
		return &jetbrains.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &jetbrains.JSONRPCError{
				Code:    -32000,
				Message: "handshake rejected",
			},
		}, nil
	})

	client, err := jetbrains.NewClient(transport)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = client.Connect(ctx)
	if !errors.Is(err, jetbrains.ErrInvalidResponse) {
		t.Errorf("expected ErrInvalidResponse on handshake rejection, got: %v", err)
	}
}

func TestClient_ListTools_And_Ping_Errors(t *testing.T) {
	ctx := context.Background()
	srv, transport := newScriptedServer(t)

	client, err := jetbrains.NewClient(transport)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, _ = client.Connect(ctx)

	// ListTools returns JSON-RPC error
	srv.RegisterHandler("tools/list", func(req *jetbrains.JSONRPCRequest) (*jetbrains.JSONRPCResponse, error) {
		return &jetbrains.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &jetbrains.JSONRPCError{
				Code:    -32603,
				Message: "failed listing tools",
			},
		}, nil
	})

	if _, err := client.ListTools(ctx); !errors.Is(err, jetbrains.ErrInvalidResponse) {
		t.Errorf("expected ErrInvalidResponse from ListTools error, got: %v", err)
	}

	// Ping returns JSON-RPC error
	srv.RegisterHandler("ping", func(req *jetbrains.JSONRPCRequest) (*jetbrains.JSONRPCResponse, error) {
		return &jetbrains.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &jetbrains.JSONRPCError{
				Code:    -32000,
				Message: "ping failed",
			},
		}, nil
	})

	if _, err := client.Ping(ctx); !errors.Is(err, jetbrains.ErrInvalidResponse) {
		t.Errorf("expected ErrInvalidResponse from Ping error, got: %v", err)
	}
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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, _ = client.Connect(ctx)

	_, err = client.CallTool(ctx, "build_project", nil)
	if !errors.Is(err, jetbrains.ErrExecutionFailed) {
		t.Errorf("expected ErrExecutionFailed, got: %v", err)
	}
}
