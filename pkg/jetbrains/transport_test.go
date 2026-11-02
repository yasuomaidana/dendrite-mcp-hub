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

func TestHTTPTransport_AdditionalErrors(t *testing.T) {
	ctx := context.Background()

	// 1. json.Marshal error (channel in Params)
	tr := jetbrains.NewHTTPTransport("http://127.0.0.1:1", nil)
	if _, err := tr.Send(ctx, &jetbrains.JSONRPCRequest{Params: make(chan int)}); err == nil {
		t.Errorf("expected marshal error")
	}

	// 2. http.NewRequestWithContext error (invalid control char in URL)
	badURLTr := jetbrains.NewHTTPTransport("http://\x7f", nil)
	if _, err := badURLTr.Send(ctx, &jetbrains.JSONRPCRequest{JSONRPC: "2.0", Method: "x"}); err == nil {
		t.Errorf("expected NewRequestWithContext error")
	}

	// 3. client.Do network error (canceled context)
	cancelCtx, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := tr.Send(cancelCtx, &jetbrains.JSONRPCRequest{JSONRPC: "2.0", Method: "x"}); err == nil {
		t.Errorf("expected client.Do error on canceled context")
	}

	// 4. Invalid JSON response from HTTP server
	badServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("{not-json"))
	}))
	defer badServer.Close()
	badJSONTr := jetbrains.NewHTTPTransport(badServer.URL, badServer.Client())
	if _, err := badJSONTr.Send(ctx, &jetbrains.JSONRPCRequest{JSONRPC: "2.0", Method: "x"}); err == nil {
		t.Errorf("expected decode error on invalid JSON response")
	}
}

func TestStdioTransport_AdditionalErrors(t *testing.T) {
	ctx := context.Background()

	// 1. Non-existent binary fails cmd.Start()
	if _, err := jetbrains.NewStdioTransport("/nonexistent-dendrite-binary-xyz"); err == nil {
		t.Errorf("expected error starting nonexistent command")
	}

	// 2. Marshal error on Send
	stdio, err := jetbrains.NewStdioTransport("cat")
	if err != nil {
		t.Fatalf("failed to start cat: %v", err)
	}
	if _, err := stdio.Send(ctx, &jetbrains.JSONRPCRequest{Params: make(chan int)}); err == nil {
		t.Errorf("expected marshal error on StdioTransport.Send")
	}

	// 3. After Close(), stdin is closed -> Write error
	_ = stdio.Close()
	if _, err := stdio.Send(ctx, &jetbrains.JSONRPCRequest{JSONRPC: "2.0", Method: "x"}); err == nil {
		t.Errorf("expected write error after Close")
	}

	// 4. Subprocess outputs non-JSON line -> Unmarshal error
	badOut, err := jetbrains.NewStdioTransport("sh", "-c", "read line; echo not-json")
	if err != nil {
		t.Fatalf("failed to start sh: %v", err)
	}
	defer func() { _ = badOut.Close() }()
	if _, err := badOut.Send(ctx, &jetbrains.JSONRPCRequest{JSONRPC: "2.0", Method: "x"}); err == nil {
		t.Errorf("expected unmarshal error from non-JSON stdout")
	}

	// 5. Subprocess exits immediately without writing -> ReadBytes EOF error
	eofProc, err := jetbrains.NewStdioTransport("true")
	if err != nil {
		t.Fatalf("failed to start true: %v", err)
	}
	defer func() { _ = eofProc.Close() }()
	if _, err := eofProc.Send(ctx, &jetbrains.JSONRPCRequest{JSONRPC: "2.0", Method: "x"}); err == nil {
		t.Errorf("expected EOF read error from true")
	}
}
