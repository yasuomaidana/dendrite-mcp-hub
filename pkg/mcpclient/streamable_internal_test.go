package mcpclient

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestToResponseMapsWireErrorsAndPlainErrors(t *testing.T) {
	wire := toResponse(&jsonrpc.Response{ID: mustID(t, 1), Error: &jsonrpc.Error{Code: -32601, Message: "no such method"}})
	if wire.Error == nil || wire.Error.Code != -32601 || wire.Error.Message != "no such method" {
		t.Fatalf("wire error = %+v", wire.Error)
	}
	plain := toResponse(&jsonrpc.Response{ID: mustID(t, 1), Error: errors.New("boom")})
	if plain.Error == nil || plain.Error.Code != -32603 || plain.Error.Message != "boom" {
		t.Fatalf("plain error = %+v", plain.Error)
	}
	if ok := toResponse(&jsonrpc.Response{ID: mustID(t, 1)}); ok.Error != nil {
		t.Fatalf("a result must carry no error: %+v", ok)
	}
}

func TestAResponseNobodyAwaitsIsDropped(t *testing.T) {
	tr := NewStreamableHTTPTransport("http://127.0.0.1:1", nil)
	tr.deliver(&jsonrpc.Response{ID: mustID(t, 999)}) // must not panic or block
	if len(tr.pending) != 0 {
		t.Fatalf("pending = %v", tr.pending)
	}
}

func mustID(t *testing.T, v float64) jsonrpc.ID {
	t.Helper()
	id, err := jsonrpc.MakeID(v)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// parkedConn is a connection whose server never answers: Read blocks until Close, and Close itself can be
// made to block until released, like a DELETE the server is slow to finish.
type parkedConn struct {
	closed   chan struct{}
	holdOpen chan struct{} // when non-nil, Close waits for it to be closed
}

func newParkedConn() *parkedConn { return &parkedConn{closed: make(chan struct{})} }

func (p *parkedConn) Read(context.Context) (jsonrpc.Message, error) {
	<-p.closed
	return nil, errors.New("parked connection closed")
}
func (p *parkedConn) Write(context.Context, jsonrpc.Message) error { return nil }
func (p *parkedConn) SessionID() string                            { return "" }
func (p *parkedConn) Close() error {
	if p.holdOpen != nil {
		<-p.holdOpen
	}
	select {
	case <-p.closed:
	default:
		close(p.closed)
	}
	return nil
}

func parked(conn *parkedConn) *StreamableHTTPTransport {
	tr := NewStreamableHTTPTransport("http://parked.invalid", nil)
	tr.dial = func(context.Context) (mcp.Connection, error) { return conn, nil }
	return tr
}

func ping(id int) *JSONRPCRequest { return &JSONRPCRequest{JSONRPC: "2.0", ID: id, Method: "ping"} }

func TestAwaitEndsWithTheContextOrTheTransportTimeout(t *testing.T) {
	tr := parked(newParkedConn())
	t.Cleanup(func() { _ = tr.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := tr.Send(ctx, ping(1)); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("context deadline: %v", err)
	}
	tr.SetTimeout(50 * time.Millisecond)
	if _, err := tr.Send(context.Background(), ping(2)); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("transport timeout: %v", err)
	}
	if len(tr.pending) != 0 {
		t.Fatalf("timed-out calls must be forgotten: %v", tr.pending)
	}
}

func TestACallWaitingWhenTheConnectionEndsGetsThatError(t *testing.T) {
	conn := newParkedConn()
	tr := parked(conn)
	t.Cleanup(func() { _ = tr.Close() })
	done := make(chan error, 1)
	go func() { _, err := tr.Send(context.Background(), ping(1)); done <- err }()
	time.Sleep(100 * time.Millisecond)
	_ = conn.Close() // the connection dies under the call: Read fails, the reader fails every waiter
	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "connection ended") {
			t.Fatalf("err = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the waiting call was not woken")
	}
}

func TestCloseReturnsAfterTheGraceWhenTheServerIsStillBusy(t *testing.T) {
	conn := newParkedConn()
	conn.holdOpen = make(chan struct{})
	tr := parked(conn)
	tr.SetTimeout(50 * time.Millisecond)
	if _, err := tr.Send(context.Background(), ping(1)); err == nil { // opens the connection; it never answers
		t.Fatal("expected a timeout")
	}
	start := time.Now()
	if err := tr.Close(); err != nil {
		t.Fatal(err)
	}
	close(conn.holdOpen)
	if waited := time.Since(start); waited < closeGrace-100*time.Millisecond || waited > closeGrace+3*time.Second {
		t.Fatalf("Close waited %v, want about %v", waited, closeGrace)
	}
}

func TestADialFailureIsReported(t *testing.T) {
	tr := NewStreamableHTTPTransport("http://x.invalid", nil)
	tr.dial = func(context.Context) (mcp.Connection, error) { return nil, errors.New("no route") }
	if _, err := tr.Send(context.Background(), ping(1)); err == nil || !strings.Contains(err.Error(), "failed to connect") {
		t.Fatalf("err = %v", err)
	}
}
