package jetbrains_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yasuomaidana/dendrite-mcp-hub/pkg/jetbrains"
)

func TestHTTPTransport_SendAndDecode(t *testing.T) {
	ctx := context.Background()

	// Spin up test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req jetbrains.JSONRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		res := jetbrains.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  json.RawMessage(`{"status":"ok"}`),
		}
		_ = json.NewEncoder(w).Encode(res)
	}))
	defer server.Close()

	// Test default parameters constructor
	defaultTransport := jetbrains.NewHTTPTransport("", nil)
	_ = defaultTransport.Close()

	httpTransport := jetbrains.NewHTTPTransport(server.URL, server.Client())
	defer func() { _ = httpTransport.Close() }()

	req := &jetbrains.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "test_method",
	}

	resp, err := httpTransport.Send(ctx, req)
	if err != nil {
		t.Fatalf("http transport Send failed: %v", err)
	}
	if string(resp.Result) != `{"status":"ok"}` {
		t.Errorf("unexpected result: %s", string(resp.Result))
	}
}

func TestHTTPTransport_ErrorStatus(t *testing.T) {
	ctx := context.Background()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer server.Close()

	httpTransport := jetbrains.NewHTTPTransport(server.URL, server.Client())
	req := &jetbrains.JSONRPCRequest{JSONRPC: "2.0", ID: 1, Method: "fail"}

	_, err := httpTransport.Send(ctx, req)
	if err == nil {
		t.Fatal("expected error on HTTP 500")
	}
}

func TestStdioTransport_Echo(t *testing.T) {
	ctx := context.Background()

	// Use 'cat' on mac/unix as a loopback stdio JSON-RPC process
	stdio, err := jetbrains.NewStdioTransport("cat")
	if err != nil {
		t.Fatalf("failed to start stdio cat: %v", err)
	}
	defer func() { _ = stdio.Close() }()

	req := &jetbrains.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      42,
		Method:  "echo",
	}

	resp, err := stdio.Send(ctx, req)
	if err != nil {
		t.Fatalf("stdio Send failed: %v", err)
	}

	if resp.JSONRPC != "2.0" {
		t.Errorf("expected jsonrpc 2.0, got %s", resp.JSONRPC)
	}
}
