package mcpclient_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yasuomaidana/dendrite-mcp-hub/pkg/mcpclient"
)

func expect(t *testing.T, ok bool, format string, args ...interface{}) {
	t.Helper()
	if !ok {
		t.Fatalf(format, args...)
	}
}

func requireOK(t *testing.T, err error) {
	t.Helper()
	expect(t, err == nil, "unexpected error: %v", err)
}

func requireErr(t *testing.T, err error) {
	t.Helper()
	expect(t, err != nil, "expected an error")
}

func jsonServer(t *testing.T, handler http.HandlerFunc) *mcpclient.HTTPTransport {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return mcpclient.NewHTTPTransport(srv.URL, srv.Client())
}

func TestHTTPTransportSendAndNotify(t *testing.T) {
	ctx := context.Background()
	var methods []string
	tr := jsonServer(t, func(w http.ResponseWriter, r *http.Request) {
		var req mcpclient.JSONRPCRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		methods = append(methods, req.Method)
		_ = json.NewEncoder(w).Encode(mcpclient.JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: json.RawMessage(`{"status":"ok"}`)})
	})
	resp, err := tr.Send(ctx, rpc(1, "test_method"))
	requireOK(t, err)
	expect(t, string(resp.Result) == `{"status":"ok"}`, "unexpected result: %s", resp.Result)
	requireOK(t, tr.Notify(ctx, rpc(nil, "notifications/initialized")))
	expect(t, strings.Join(methods, ",") == "test_method,notifications/initialized", "methods = %v", methods)
	requireOK(t, tr.Close())
	requireOK(t, mcpclient.NewHTTPTransport("", nil).Close())
}

func TestHTTPTransportErrors(t *testing.T) {
	ctx := context.Background()
	failing := jsonServer(t, func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	})
	_, err := failing.Send(ctx, rpc(1, "fail"))
	requireErr(t, err)
	requireErr(t, failing.Notify(ctx, rpc(nil, "fail")))

	dead := mcpclient.NewHTTPTransport("http://127.0.0.1:1", nil)
	_, err = dead.Send(ctx, &mcpclient.JSONRPCRequest{Params: make(chan int)})
	requireErr(t, err) // marshal
	_, err = mcpclient.NewHTTPTransport("http://\x7f", nil).Send(ctx, rpc(1, "x"))
	requireErr(t, err) // bad URL
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	_, err = dead.Send(cancelled, rpc(1, "x"))
	requireErr(t, err) // network

	garbled := jsonServer(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("{not-json")) })
	_, err = garbled.Send(ctx, rpc(1, "x"))
	requireErr(t, err) // decode
}

// The regression of DMCP-34: the handshake against a real server, which never answers a notification.
func TestStdioHandshakeAgainstARealServer(t *testing.T) {
	client, err := mcpclient.NewClient(startHelper(t, "server"))
	requireOK(t, err)
	within(t, 10*time.Second, "handshake", func() {
		res, err := client.Connect(context.Background())
		requireOK(t, err)
		expect(t, res.ServerInfo.Name == "helper" && res.ServerInfo.Version == "9.9", "server info = %+v", res.ServerInfo)
	})
	ctx := context.Background()
	tools, err := client.ListTools(ctx)
	requireOK(t, err)
	expect(t, len(tools) == 1 && tools[0].Name == "echo", "tools = %+v", tools)
	out, err := client.CallTool(ctx, "echo", map[string]interface{}{"text": "hello"})
	requireOK(t, err)
	expect(t, out.Content[0].Text == "echo:hello", "call result = %+v", out)
	_, err = client.Ping(ctx)
	requireOK(t, err)
	requireOK(t, client.Disconnect())
}

func TestStdioSkipsNoiseAndMatchesIDs(t *testing.T) {
	tr := startHelper(t, "noisy")
	for _, id := range []interface{}{1, "req-2"} {
		resp, err := tr.Send(context.Background(), rpc(id, "tools/list"))
		requireOK(t, err)
		expect(t, string(resp.Result) == `{"ok":true}`, "id %v: response = %s", id, resp.Result)
	}
	requireOK(t, tr.Notify(context.Background(), rpc(nil, "notifications/initialized")))
}

func TestStdioAnswersMayArriveOutOfOrder(t *testing.T) {
	tr := startHelper(t, "reorder")
	results := make([]string, 2)
	var wg sync.WaitGroup
	for i := range results {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := tr.Send(context.Background(), rpc(i+1, "x"))
			if err == nil {
				results[i] = string(resp.Result)
			}
		}()
	}
	within(t, 10*time.Second, "both calls", wg.Wait)
	for i, got := range results {
		want := `{"answered":` + string(rune('1'+i)) + `}`
		expect(t, got == want, "call %d got %q, want %s", i+1, got, want)
	}
}

func TestStdioHonoursContextAndTimeout(t *testing.T) {
	tr := startHelper(t, "silent")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	within(t, 5*time.Second, "send with a deadline", func() {
		_, err := tr.Send(ctx, rpc(1, "x"))
		expect(t, errors.Is(err, context.DeadlineExceeded), "err = %v", err)
	})
	tr.SetTimeout(100 * time.Millisecond)
	within(t, 5*time.Second, "send with the default timeout", func() {
		_, err := tr.Send(context.Background(), rpc(2, "x"))
		expect(t, errors.Is(err, context.DeadlineExceeded), "err = %v", err)
	})
}

func TestStdioReportsACrashWithItsStderr(t *testing.T) {
	tr := startHelper(t, "crash")
	within(t, 10*time.Second, "send to a crashing server", func() {
		_, err := tr.Send(context.Background(), rpc(1, "x"))
		requireErr(t, err)
		expect(t, strings.Contains(err.Error(), "fatal: boom"), "stderr missing: %v", err)
		expect(t, !strings.Contains(err.Error(), strings.Repeat("x", 4500)), "stderr tail not bounded: %d bytes", len(err.Error()))
	})
	// The child is gone: the next write fails and quotes the same stderr.
	_, err := tr.Send(context.Background(), rpc(2, "x"))
	requireErr(t, err)
}

func TestStdioCloseReapsTheChild(t *testing.T) {
	polite := startHelper(t, "silent")
	within(t, 5*time.Second, "closing a server that leaves with its stdin", func() {
		requireOK(t, polite.Close())
		requireOK(t, polite.Close())
	})
	_, err := polite.Send(context.Background(), rpc(1, "x"))
	requireErr(t, err)

	stuck := startHelper(t, "stuck")
	within(t, 10*time.Second, "closing a server that ignores its stdin", func() { requireOK(t, stuck.Close()) })
}

func TestStdioStartAndMarshalErrors(t *testing.T) {
	_, err := mcpclient.NewStdioTransport("/nonexistent-dendrite-binary-xyz")
	requireErr(t, err)
	tr := startHelper(t, "silent")
	requireErr(t, tr.Notify(context.Background(), &mcpclient.JSONRPCRequest{Params: make(chan int)}))
	_, err = tr.Send(context.Background(), &mcpclient.JSONRPCRequest{Params: make(chan int)})
	requireErr(t, err)
}
