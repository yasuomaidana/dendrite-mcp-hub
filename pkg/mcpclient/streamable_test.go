package mcpclient_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/yasuomaidana/dendrite-mcp-hub/pkg/mcpclient"
	"github.com/yasuomaidana/dendrite-mcp-hub/pkg/mcphttp"
	"github.com/yasuomaidana/dendrite-mcp-hub/pkg/mcpserver"
)

// httpHub serves a hub server with an instant tool and one that blocks until the test releases it.
// httpHub serves in the given response mode: JSON answers arrive with the HTTP headers, SSE streams open at once.
func httpHub(t *testing.T, opts mcphttp.Options) (*mcpclient.StreamableHTTPTransport, func()) {
	t.Helper()
	release := make(chan struct{})
	var once sync.Once
	releaseAll := func() { once.Do(func() { close(release) }) }
	srv := mcpserver.NewServer(mcpserver.ServerConfig{Name: "h", Version: "1"})
	srv.RegisterTool(mcpserver.Tool{Name: "fast"}, func(context.Context, map[string]interface{}) (*mcpserver.CallToolResult, error) {
		return &mcpserver.CallToolResult{Content: []mcpserver.ContentItem{{Type: "text", Text: "ok"}}}, nil
	})
	srv.RegisterTool(mcpserver.Tool{Name: "slow"}, func(ctx context.Context, _ map[string]interface{}) (*mcpserver.CallToolResult, error) {
		select {
		case <-release:
		case <-ctx.Done():
		}
		return &mcpserver.CallToolResult{}, nil
	})
	ts := httptest.NewServer(mcphttp.Handler(srv, opts))
	t.Cleanup(ts.Close)
	t.Cleanup(releaseAll) // runs before ts.Close, which waits for a blocked tool
	return mcpclient.NewStreamableHTTPTransport(ts.URL, ts.Client()), releaseAll
}

var jsonMode = mcphttp.Options{JSONResponse: true}

// eachMode runs a test in both response modes of the specification.
func eachMode(t *testing.T, run func(t *testing.T, opts mcphttp.Options)) {
	t.Helper()
	for name, opts := range map[string]mcphttp.Options{"sse": {}, "json": jsonMode} {
		t.Run(name, func(t *testing.T) { run(t, opts) })
	}
}

func handshake(t *testing.T, tr *mcpclient.StreamableHTTPTransport) {
	t.Helper()
	c, err := mcpclient.NewClient(tr)
	requireOK(t, err)
	_, err = c.Connect(context.Background())
	requireOK(t, err)
}

func callTool(id interface{}, name string) *mcpclient.JSONRPCRequest {
	r := rpc(id, "tools/call")
	r.Params = map[string]interface{}{"name": name, "arguments": map[string]interface{}{}}
	return r
}

func TestStreamableMatchesNumericAndStringIDs(t *testing.T) {
	tr, _ := httpHub(t, jsonMode)
	c, err := mcpclient.NewClient(tr)
	requireOK(t, err)
	_, err = c.Connect(context.Background())
	requireOK(t, err)
	for _, id := range []interface{}{uint64(41), "req-42"} {
		resp, err := tr.Send(context.Background(), callTool(id, "fast"))
		requireOK(t, err)
		expect(t, resp.Error == nil && len(resp.Result) > 0, "id %v: %+v", id, resp)
	}
}

func TestStreamableCallsMayOverlap(t *testing.T) {
	tr, release := httpHub(t, jsonMode)
	c, _ := mcpclient.NewClient(tr)
	_, err := c.Connect(context.Background())
	requireOK(t, err)
	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i, name := range []string{"slow", "fast"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, errs[i] = tr.Send(context.Background(), callTool(100+i, name))
		}()
	}
	time.Sleep(100 * time.Millisecond)
	release()
	within(t, 10*time.Second, "both calls", wg.Wait)
	for i, err := range errs {
		expect(t, err == nil, "call %d: %v", i, err)
	}
}

func TestStreamableHonoursContextAndTimeout(t *testing.T) {
	eachMode(t, func(t *testing.T, opts mcphttp.Options) { honoursContextAndTimeout(t, opts) })
}

func honoursContextAndTimeout(t *testing.T, opts mcphttp.Options) {
	tr, _ := httpHub(t, opts)
	handshake(t, tr)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := tr.Send(ctx, callTool(2, "slow"))
	expect(t, errors.Is(err, context.DeadlineExceeded), "context deadline: %v", err)
	tr.SetTimeout(100 * time.Millisecond)
	_, err = tr.Send(context.Background(), callTool(3, "slow"))
	expect(t, errors.Is(err, context.DeadlineExceeded), "transport timeout: %v", err)
}

func TestStreamableCloseWakesAWaitingCallAndIsFinal(t *testing.T) {
	eachMode(t, func(t *testing.T, opts mcphttp.Options) { closeWakesAWaitingCall(t, opts) })
}

func closeWakesAWaitingCall(t *testing.T, opts mcphttp.Options) {
	tr, _ := httpHub(t, opts)
	handshake(t, tr)
	done := make(chan error, 1)
	go func() {
		_, err := tr.Send(context.Background(), callTool(2, "slow"))
		done <- err
	}()
	time.Sleep(100 * time.Millisecond)
	requireOK(t, tr.Close())
	select {
	case err := <-done:
		expect(t, errors.Is(err, mcpclient.ErrTransportClosed), "waiting call: %v", err)
	case <-time.After(10 * time.Second):
		t.Fatal("Close did not wake the waiting call")
	}
	requireOK(t, tr.Close())
	_, err := tr.Send(context.Background(), callTool(3, "fast"))
	expect(t, errors.Is(err, mcpclient.ErrTransportClosed), "send after Close: %v", err)
	expect(t, errors.Is(tr.Notify(context.Background(), rpc(nil, "x")), mcpclient.ErrTransportClosed), "notify after Close")
}

func TestStreamableCloseBeforeAnyCallIsFine(t *testing.T) {
	requireOK(t, mcpclient.NewStreamableHTTPTransport("http://127.0.0.1:1", nil).Close())
}

func TestStreamableFailuresAreReported(t *testing.T) {
	dead := mcpclient.NewStreamableHTTPTransport("http://127.0.0.1:1", nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := dead.Send(ctx, rpc(1, "initialize"))
	requireErr(t, err)
	requireErr(t, mcpclient.NewStreamableHTTPTransport("http://127.0.0.1:1", nil).Notify(ctx, rpc(nil, "x")))

	tr, _ := httpHub(t, jsonMode)
	unmarshalable := rpc(1, "x")
	unmarshalable.Params = make(chan int)
	_, err = tr.Send(ctx, unmarshalable)
	requireErr(t, err)
	requireErr(t, tr.Notify(ctx, unmarshalable))
	badID := rpc(make(chan int), "x")
	_, err = tr.Send(ctx, badID)
	requireErr(t, err)
	_, err = tr.Send(ctx, rpc(struct{ A int }{1}, "x")) // an id that is not a number or a string
	requireErr(t, err)
}

// A server that forgets the session (a restart) answers 404: the connection fails for good and says so.
func TestStreamableReportsALostSession(t *testing.T) {
	eachMode(t, func(t *testing.T, opts mcphttp.Options) {
		var mu sync.Mutex
		current := mcphttp.Handler(mcpserver.NewServer(mcpserver.ServerConfig{}), opts)
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			h := current
			mu.Unlock()
			h.ServeHTTP(w, r)
		}))
		t.Cleanup(ts.Close)
		tr := mcpclient.NewStreamableHTTPTransport(ts.URL, ts.Client())
		handshake(t, tr)
		mu.Lock()
		current = mcphttp.Handler(mcpserver.NewServer(mcpserver.ServerConfig{}), opts) // new server: no sessions
		mu.Unlock()
		_, err := tr.Send(context.Background(), rpc(7, "ping"))
		requireErr(t, err)
		time.Sleep(200 * time.Millisecond)
		_, err = tr.Send(context.Background(), rpc(8, "ping"))
		requireErr(t, err)
	})
}

func TestStreamableRejectsAnEndpointThatIsNotAURL(t *testing.T) {
	_, err := mcpclient.NewStreamableHTTPTransport("http://\x7f", nil).Send(context.Background(), rpc(1, "ping"))
	requireErr(t, err)
}
