package mcpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Client coordinates JSON-RPC 2.0 communication with one MCP server.
type Client struct {
	transport          Transport
	clientInfo         ClientInfo
	protocolVersion    string
	connected          bool
	serverInfo         *ServerInfo
	serverCapabilities *ServerCapabilities
	reqCounter         uint64
	mu                 sync.RWMutex
}

// Option configures Client parameters.
type Option func(*Client)

// WithClientInfo configures custom client identification.
func WithClientInfo(name, version string) Option {
	return func(c *Client) {
		c.clientInfo = ClientInfo{Name: name, Version: version}
	}
}

// WithProtocolVersion overrides the MCP protocol version string.
func WithProtocolVersion(ver string) Option {
	return func(c *Client) {
		c.protocolVersion = ver
	}
}

// NewClient initializes a client over the given transport.
func NewClient(transport Transport, opts ...Option) (*Client, error) {
	if transport == nil {
		return nil, ErrNilTransport
	}
	c := &Client{
		transport:       transport,
		protocolVersion: DefaultProtocolVersion,
		clientInfo:      ClientInfo{Name: "dendrite-mcp-hub", Version: "0.1.0"},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

// nextID generates an incremental JSON-RPC request ID.
func (c *Client) nextID() uint64 {
	return atomic.AddUint64(&c.reqCounter, 1)
}

// call sends one request and returns its result, mapping a JSON-RPC error object to ErrInvalidResponse.
func (c *Client) call(ctx context.Context, method string, params interface{}) (json.RawMessage, *JSONRPCError, error) {
	resp, err := c.transport.Send(ctx, &JSONRPCRequest{JSONRPC: "2.0", ID: c.nextID(), Method: method, Params: params})
	if err != nil {
		return nil, nil, err
	}
	return resp.Result, resp.Error, nil
}

// Connect initiates the MCP handshake: initialize, then the initialized notification.
func (c *Client) Connect(ctx context.Context) (*InitializeResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.connected {
		return nil, ErrAlreadyConnected
	}
	raw, rpcErr, err := c.call(ctx, "initialize", InitializeParams{
		ProtocolVersion: c.protocolVersion,
		ClientInfo:      c.clientInfo,
		Capabilities:    ClientCapabilities{Roots: &RootsCapability{ListChanged: true}},
	})
	if err != nil {
		return nil, fmt.Errorf("handshake failed: %w", err)
	}
	if rpcErr != nil {
		return nil, fmt.Errorf("%w: %s (code %d)", ErrInvalidResponse, rpcErr.Message, rpcErr.Code)
	}
	var initResult InitializeResult
	if err := json.Unmarshal(raw, &initResult); err != nil {
		return nil, fmt.Errorf("%w: failed to decode initialize result: %s", ErrInvalidResponse, err)
	}
	// A notification gets no reply: it is written, never awaited.
	if err := c.transport.Notify(ctx, &JSONRPCRequest{JSONRPC: "2.0", Method: "notifications/initialized"}); err != nil {
		return nil, fmt.Errorf("handshake failed: %w", err)
	}
	c.connected = true
	c.serverInfo = &initResult.ServerInfo
	c.serverCapabilities = &initResult.Capabilities
	return &initResult, nil
}

// Disconnect shuts down the connection and closes the transport.
func (c *Client) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.connected {
		return nil
	}
	c.connected = false
	c.serverInfo = nil
	c.serverCapabilities = nil
	return c.transport.Close()
}

// IsConnected returns whether the client is currently connected.
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// GetServerInfo returns details about the connected server.
func (c *Client) GetServerInfo() *ServerInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.serverInfo == nil {
		return nil
	}
	cpy := *c.serverInfo
	return &cpy
}

// GetServerCapabilities returns capabilities advertised by the server.
func (c *Client) GetServerCapabilities() *ServerCapabilities {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.serverCapabilities == nil {
		return nil
	}
	cpy := *c.serverCapabilities
	return &cpy
}

// ListTools discovers all tools exposed by the server.
func (c *Client) ListTools(ctx context.Context) ([]Tool, error) {
	if !c.IsConnected() {
		return nil, ErrNotConnected
	}
	raw, rpcErr, err := c.call(ctx, "tools/list", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}
	if rpcErr != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidResponse, rpcErr.Message)
	}
	var res ListToolsResult
	if err := json.Unmarshal(raw, &res); err != nil {
		return nil, fmt.Errorf("%w: failed to decode tools list: %s", ErrInvalidResponse, err)
	}
	return res.Tools, nil
}

// CallTool invokes a tool on the server with the given arguments.
func (c *Client) CallTool(ctx context.Context, name string, args map[string]interface{}) (*CallToolResult, error) {
	if !c.IsConnected() {
		return nil, ErrNotConnected
	}
	if name == "" {
		return nil, ErrEmptyToolName
	}
	if args == nil {
		args = make(map[string]interface{})
	}
	raw, rpcErr, err := c.call(ctx, "tools/call", CallToolParams{Name: name, Arguments: args})
	if err != nil {
		return nil, fmt.Errorf("tool call network failure: %w", err)
	}
	if rpcErr != nil {
		return nil, rpcToolError(rpcErr)
	}
	var callRes CallToolResult
	if err := json.Unmarshal(raw, &callRes); err != nil {
		return nil, fmt.Errorf("%w: failed to decode tool result: %s", ErrInvalidResponse, err)
	}
	if callRes.IsError {
		return &callRes, fmt.Errorf("%w: %s", ErrExecutionFailed, toolErrorText(callRes))
	}
	return &callRes, nil
}

// rpcToolError maps a JSON-RPC error object to the matching sentinel error.
func rpcToolError(e *JSONRPCError) error {
	if e.Code == -32601 {
		return fmt.Errorf("%w: %s", ErrToolNotFound, e.Message)
	}
	return fmt.Errorf("%w: %s (code %d)", ErrExecutionFailed, e.Message, e.Code)
}

// toolErrorText extracts the failure message of a tool result flagged as an error.
func toolErrorText(res CallToolResult) string {
	if len(res.Content) > 0 && res.Content[0].Text != "" {
		return res.Content[0].Text
	}
	return "unknown tool execution error"
}

// Ping measures round-trip latency to the server.
func (c *Client) Ping(ctx context.Context) (time.Duration, error) {
	if !c.IsConnected() {
		return 0, ErrNotConnected
	}
	start := time.Now()
	_, rpcErr, err := c.call(ctx, "ping", nil)
	if err != nil {
		return 0, fmt.Errorf("ping failed: %w", err)
	}
	if rpcErr != nil {
		return 0, fmt.Errorf("%w: %s", ErrInvalidResponse, rpcErr.Message)
	}
	return time.Since(start), nil
}
