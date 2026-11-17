package mcpclient_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/yasuomaidana/dendrite-mcp-hub/pkg/mcpclient"
)

// scripted is a Transport whose answers per method are fixed by the test.
type scripted struct {
	results   map[string]string                  // method -> result JSON
	rpcErrors map[string]*mcpclient.JSONRPCError // method -> error object
	failSend  map[string]error                   // method -> transport failure
	notifyErr error
	closed    int
}

func (s *scripted) Send(_ context.Context, req *mcpclient.JSONRPCRequest) (*mcpclient.JSONRPCResponse, error) {
	if err := s.failSend[req.Method]; err != nil {
		return nil, err
	}
	return &mcpclient.JSONRPCResponse{
		JSONRPC: "2.0", ID: req.ID, Result: json.RawMessage(s.results[req.Method]), Error: s.rpcErrors[req.Method],
	}, nil
}

func (s *scripted) Notify(context.Context, *mcpclient.JSONRPCRequest) error { return s.notifyErr }

func (s *scripted) Close() error { s.closed++; return nil }

const initOK = `{"protocolVersion":"2024-11-05","serverInfo":{"name":"s","version":"1"},"capabilities":{"tools":{}}}`

func newScripted() *scripted {
	return &scripted{
		results: map[string]string{
			"initialize": initOK,
			"tools/list": `{"tools":[{"name":"a"}]}`,
			"tools/call": `{"content":[{"type":"text","text":"done"}]}`,
			"ping":       `{}`,
		},
		rpcErrors: map[string]*mcpclient.JSONRPCError{},
		failSend:  map[string]error{},
	}
}

func connected(t *testing.T, s *scripted) *mcpclient.Client {
	t.Helper()
	c, err := mcpclient.NewClient(s, mcpclient.WithClientInfo("test", "1"), mcpclient.WithProtocolVersion("2024-11-05"))
	requireOK(t, err)
	_, err = c.Connect(context.Background())
	requireOK(t, err)
	return c
}

// operations runs each client call that needs a connection.
var operations = map[string]func(*mcpclient.Client) error{
	"tools/list": func(c *mcpclient.Client) error { _, err := c.ListTools(context.Background()); return err },
	"tools/call": func(c *mcpclient.Client) error { _, err := c.CallTool(context.Background(), "a", nil); return err },
	"ping":       func(c *mcpclient.Client) error { _, err := c.Ping(context.Background()); return err },
}

func TestClientLifecycle(t *testing.T) {
	_, err := mcpclient.NewClient(nil)
	expect(t, errors.Is(err, mcpclient.ErrNilTransport), "nil transport = %v", err)

	s := newScripted()
	c := connected(t, s)
	expect(t, c.IsConnected(), "not connected after Connect")
	expect(t, c.GetServerInfo().Name == "s" && c.GetServerCapabilities().Tools != nil, "server details missing")
	_, err = c.Connect(context.Background())
	expect(t, errors.Is(err, mcpclient.ErrAlreadyConnected), "second Connect = %v", err)
	for _, op := range operations {
		requireOK(t, op(c))
	}
	requireOK(t, c.Disconnect())
	requireOK(t, c.Disconnect())
	expect(t, s.closed == 1 && !c.IsConnected(), "closed %d times, connected %v", s.closed, c.IsConnected())
	expect(t, c.GetServerInfo() == nil && c.GetServerCapabilities() == nil, "details kept after Disconnect")
}

func TestClientRequiresAConnection(t *testing.T) {
	c, _ := mcpclient.NewClient(newScripted())
	for name, op := range operations {
		expect(t, errors.Is(op(c), mcpclient.ErrNotConnected), "%s before Connect", name)
	}
}

func TestClientReportsFailuresPerMethod(t *testing.T) {
	for method, op := range operations {
		s := newScripted()
		c := connected(t, s)
		s.failSend[method] = errors.New("network down")
		requireErr(t, op(c))
		delete(s.failSend, method)
		s.rpcErrors[method] = &mcpclient.JSONRPCError{Code: -32000, Message: "server says no"}
		requireErr(t, op(c))
		delete(s.rpcErrors, method)
		if method != "ping" {
			s.results[method] = `not json`
			expect(t, errors.Is(op(c), mcpclient.ErrInvalidResponse), "%s with bad JSON", method)
		}
	}
}

func TestClientConnectFailures(t *testing.T) {
	cases := map[string]func(*scripted){
		"transport": func(s *scripted) { s.failSend["initialize"] = errors.New("down") },
		"rpc error": func(s *scripted) { s.rpcErrors["initialize"] = &mcpclient.JSONRPCError{Code: -1, Message: "no"} },
		"bad json":  func(s *scripted) { s.results["initialize"] = `[` },
		"notify":    func(s *scripted) { s.notifyErr = errors.New("pipe closed") },
	}
	for name, breakIt := range cases {
		s := newScripted()
		breakIt(s)
		c, _ := mcpclient.NewClient(s)
		_, err := c.Connect(context.Background())
		expect(t, err != nil && !c.IsConnected(), "%s: err=%v connected=%v", name, err, c.IsConnected())
	}
}

func TestClientCallToolErrors(t *testing.T) {
	s := newScripted()
	c := connected(t, s)
	_, err := c.CallTool(context.Background(), "", nil)
	expect(t, errors.Is(err, mcpclient.ErrEmptyToolName), "empty name = %v", err)

	s.rpcErrors["tools/call"] = &mcpclient.JSONRPCError{Code: -32601, Message: "unknown"}
	_, err = c.CallTool(context.Background(), "a", nil)
	expect(t, errors.Is(err, mcpclient.ErrToolNotFound), "unknown tool = %v", err)
	delete(s.rpcErrors, "tools/call")

	for result, want := range map[string]string{
		`{"content":[{"type":"text","text":"bad input"}],"isError":true}`: "bad input",
		`{"content":[],"isError":true}`:                                   "unknown tool execution error",
	} {
		s.results["tools/call"] = result
		res, err := c.CallTool(context.Background(), "a", map[string]interface{}{"k": 1})
		expect(t, errors.Is(err, mcpclient.ErrExecutionFailed) && res != nil, "isError result: res=%v err=%v", res, err)
		expect(t, strings.HasSuffix(err.Error(), want), "error %q lacks %q", err, want)
	}
}
