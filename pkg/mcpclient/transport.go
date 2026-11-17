package mcpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Transport abstracts the physical JSON-RPC 2.0 medium.
//
// Send writes a request and waits for the response with the same id. Notify writes a notification and
// returns at once: a server never answers one, so waiting for a reply would hang forever.
type Transport interface {
	Send(ctx context.Context, req *JSONRPCRequest) (*JSONRPCResponse, error)
	Notify(ctx context.Context, req *JSONRPCRequest) error
	Close() error
}

// DefaultHTTPURL is the endpoint NewHTTPTransport falls back to: the JetBrains built-in MCP web server.
const DefaultHTTPURL = "http://localhost:63342/api/mcp"

// HTTPTransport implements Transport as JSON-RPC over HTTP POST.
type HTTPTransport struct {
	client  *http.Client
	baseURL string
}

// NewHTTPTransport points a transport at baseURL; a nil client gets a 10 s timeout, an empty URL DefaultHTTPURL.
func NewHTTPTransport(baseURL string, client *http.Client) *HTTPTransport {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	if baseURL == "" {
		baseURL = DefaultHTTPURL
	}
	return &HTTPTransport{client: client, baseURL: baseURL}
}

// post sends the payload and returns the open response, which the caller closes.
func (h *HTTPTransport) post(ctx context.Context, req *JSONRPCRequest) (*http.Response, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, h.baseURL, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	res, err := h.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute http request: %w", err)
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		defer func() { _ = res.Body.Close() }()
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("mcp http error status %d: %s", res.StatusCode, string(body))
	}
	return res, nil
}

// Send posts a request and decodes the JSON-RPC response.
func (h *HTTPTransport) Send(ctx context.Context, req *JSONRPCRequest) (*JSONRPCResponse, error) {
	res, err := h.post(ctx, req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()
	var rpcRes JSONRPCResponse
	if err := json.NewDecoder(res.Body).Decode(&rpcRes); err != nil {
		return nil, fmt.Errorf("failed to decode json-rpc response: %w", err)
	}
	return &rpcRes, nil
}

// Notify posts a notification; the body of the reply, if any, is ignored.
func (h *HTTPTransport) Notify(ctx context.Context, req *JSONRPCRequest) error {
	res, err := h.post(ctx, req)
	if err != nil {
		return err
	}
	return res.Body.Close()
}

// Close cleans up the HTTP transport.
func (h *HTTPTransport) Close() error { return nil }
