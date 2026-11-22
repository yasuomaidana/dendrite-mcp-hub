package mcpclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ErrTransportClosed means a call was made on, or was waiting on, a closed streamable-HTTP transport.
var ErrTransportClosed = errors.New("streamable http transport is closed")

// StreamableHTTPTransport implements Transport on the go-sdk's streamable-HTTP client transport: it keeps
// one session with the server (Mcp-Session-Id), accepts JSON or SSE answers and reconnects streams.
//
// One reader goroutine routes each response to the call waiting for its id, like StdioTransport, so calls
// may overlap. Messages that are not responses (server notifications and requests) are skipped.
type StreamableHTTPTransport struct {
	endpoint string
	client   *http.Client
	timeout  time.Duration

	dial func(context.Context) (mcp.Connection, error) // the go-sdk transport; tests replace it

	life context.Context // cancelled by Close: it ends calls blocked in an HTTP write
	stop context.CancelFunc

	mu      sync.Mutex // guards conn, pending, err
	conn    mcp.Connection
	pending map[string]chan *JSONRPCResponse
	err     error
}

// NewStreamableHTTPTransport points a transport at an MCP endpoint; a nil client means http.DefaultClient.
func NewStreamableHTTPTransport(endpoint string, client *http.Client) *StreamableHTTPTransport {
	life, stop := context.WithCancel(context.Background())
	t := &StreamableHTTPTransport{endpoint: endpoint, client: client, timeout: DefaultCallTimeout, life: life, stop: stop, pending: map[string]chan *JSONRPCResponse{}}
	t.dial = t.sdkDial
	return t
}

// sdkDial opens the go-sdk streamable-HTTP connection; the standalone SSE stream stays off, since no
// caller listens for server-initiated messages.
func (t *StreamableHTTPTransport) sdkDial(ctx context.Context) (mcp.Connection, error) {
	return (&mcp.StreamableClientTransport{Endpoint: t.endpoint, HTTPClient: t.client, DisableStandaloneSSE: true}).Connect(ctx)
}

// SetTimeout bounds a Send whose context has no deadline of its own.
func (t *StreamableHTTPTransport) SetTimeout(d time.Duration) { t.timeout = d }

// connect opens the session on first use. The connection outlives ctx: the go-sdk detaches it.
func (t *StreamableHTTPTransport) connect(ctx context.Context) (mcp.Connection, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.err != nil {
		return nil, t.err
	}
	if t.conn != nil {
		return t.conn, nil
	}
	conn, err := t.dial(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", t.endpoint, err)
	}
	t.conn = conn
	go t.readLoop(conn)
	return conn, nil
}

// readLoop delivers responses until the connection fails or closes, then fails every waiting call.
func (t *StreamableHTTPTransport) readLoop(conn mcp.Connection) {
	for {
		msg, err := conn.Read(context.Background())
		if err != nil {
			t.fail(err)
			return
		}
		if resp, ok := msg.(*jsonrpc.Response); ok {
			t.deliver(resp)
		}
	}
}

func (t *StreamableHTTPTransport) deliver(resp *jsonrpc.Response) {
	key := idKey(resp.ID.Raw())
	t.mu.Lock()
	ch := t.pending[key]
	delete(t.pending, key)
	t.mu.Unlock()
	if ch != nil {
		ch <- toResponse(resp)
	}
}

// fail records why the connection ended and wakes every waiting call by closing its channel.
func (t *StreamableHTTPTransport) fail(cause error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.err == nil {
		t.err = fmt.Errorf("streamable http connection ended: %w", cause)
	}
	for key, ch := range t.pending {
		close(ch)
		delete(t.pending, key)
	}
}

// toResponse converts a go-sdk response to the hub's envelope.
func toResponse(resp *jsonrpc.Response) *JSONRPCResponse {
	out := &JSONRPCResponse{JSONRPC: "2.0", ID: resp.ID.Raw(), Result: resp.Result}
	var wire *jsonrpc.Error
	if errors.As(resp.Error, &wire) {
		out.Error = &JSONRPCError{Code: int(wire.Code), Message: wire.Message}
	} else if resp.Error != nil {
		out.Error = &JSONRPCError{Code: -32603, Message: resp.Error.Error()}
	}
	return out
}

// message converts the hub's envelope to a go-sdk request; a nil ID makes a notification.
func message(req *JSONRPCRequest) (*jsonrpc.Request, error) {
	out := &jsonrpc.Request{Method: req.Method}
	if req.Params != nil {
		params, err := json.Marshal(req.Params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		out.Params = params
	}
	if req.ID != nil {
		id, err := wireID(req.ID)
		if err != nil {
			return nil, err
		}
		out.ID = id
	}
	return out, nil
}

// wireID turns any JSON-encodable id (the client counts with uint64) into a go-sdk id.
func wireID(id interface{}) (jsonrpc.ID, error) {
	raw, err := json.Marshal(id)
	if err != nil {
		return jsonrpc.ID{}, fmt.Errorf("invalid request id: %w", err)
	}
	var decoded interface{}
	_ = json.Unmarshal(raw, &decoded) // raw came from json.Marshal, so it is valid JSON
	return jsonrpc.MakeID(decoded)
}

// Notify writes a notification; a server never answers one.
func (t *StreamableHTTPTransport) Notify(ctx context.Context, req *JSONRPCRequest) error {
	msg, err := message(req)
	if err != nil {
		return err
	}
	ctx, cancel := t.bound(ctx)
	defer cancel()
	conn, err := t.connect(ctx)
	if err != nil {
		return err
	}
	if err := conn.Write(ctx, msg); err != nil {
		return t.cause(err)
	}
	return nil
}

// Send writes a request and waits for its response, the context's deadline or the transport timeout.
func (t *StreamableHTTPTransport) Send(ctx context.Context, req *JSONRPCRequest) (*JSONRPCResponse, error) {
	msg, err := message(req)
	if err != nil {
		return nil, err
	}
	ctx, cancel := t.bound(ctx)
	defer cancel()
	conn, err := t.connect(ctx)
	if err != nil {
		return nil, err
	}
	key := idKey(req.ID)
	ch := make(chan *JSONRPCResponse, 1)
	t.mu.Lock()
	t.pending[key] = ch
	t.mu.Unlock()
	if err := conn.Write(ctx, msg); err != nil {
		t.forget(key)
		return nil, fmt.Errorf("failed to send request: %w", t.cause(err))
	}
	return t.await(ctx, key, ch)
}

// bound gives a context without a deadline the transport timeout (the write blocks on the HTTP answer too)
// and ties every context to the transport's life, so Close ends the call.
func (t *StreamableHTTPTransport) bound(ctx context.Context) (context.Context, context.CancelFunc) {
	var cancel context.CancelFunc
	if _, hasDeadline := ctx.Deadline(); hasDeadline {
		ctx, cancel = context.WithCancel(ctx)
	} else {
		ctx, cancel = context.WithTimeout(ctx, t.timeout)
	}
	unlink := context.AfterFunc(t.life, cancel)
	return ctx, func() { unlink(); cancel() }
}

// cause reports ErrTransportClosed instead of the context error a Close produced.
func (t *StreamableHTTPTransport) cause(err error) error {
	if closed := t.failure(); errors.Is(closed, ErrTransportClosed) {
		return closed
	}
	return err
}

func (t *StreamableHTTPTransport) await(ctx context.Context, key string, ch chan *JSONRPCResponse) (*JSONRPCResponse, error) {
	select {
	case resp, ok := <-ch:
		if !ok {
			return nil, t.failure()
		}
		return resp, nil
	case <-ctx.Done():
		t.forget(key)
		return nil, t.cause(ctx.Err())
	}
}

func (t *StreamableHTTPTransport) forget(key string) {
	t.mu.Lock()
	delete(t.pending, key)
	t.mu.Unlock()
}

func (t *StreamableHTTPTransport) failure() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.err
}

// Close ends the session (the go-sdk sends the DELETE) and wakes any waiting call. It is safe to call twice,
// and it returns after closeGrace at the latest, even when the server is still busy with a call.
func (t *StreamableHTTPTransport) Close() error {
	t.mu.Lock()
	conn := t.conn
	t.conn = nil
	if t.err == nil {
		t.err = ErrTransportClosed
	}
	t.mu.Unlock()
	t.stop()
	t.fail(ErrTransportClosed)
	if conn == nil {
		return nil
	}
	// The DELETE that ends the session waits for calls still running on the server: give it closeGrace.
	done := make(chan error, 1)
	go func() { done <- conn.Close() }()
	select {
	case err := <-done:
		return err
	case <-time.After(closeGrace):
		return nil
	}
}
