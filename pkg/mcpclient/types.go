// Package mcpclient is a generic outbound MCP client: JSON-RPC 2.0 over stdio or HTTP, the initialize
// handshake, tools/list, tools/call and ping. It knows nothing about any one server; pkg/jetbrains wraps it.
package mcpclient

import (
	"encoding/json"
	"errors"
)

// Standard domain errors of the client.
var (
	ErrNotConnected            = errors.New("mcp client is not connected")
	ErrAlreadyConnected        = errors.New("mcp client is already connected")
	ErrNilTransport            = errors.New("transport cannot be nil")
	ErrToolNotFound            = errors.New("requested tool was not found on the mcp server")
	ErrExecutionFailed         = errors.New("mcp tool execution failed")
	ErrInvalidResponse         = errors.New("received invalid json-rpc response")
	ErrEmptyToolName           = errors.New("tool name cannot be empty")
	ErrProtocolVersionMismatch = errors.New("unsupported protocol version returned by server")
)

// DefaultProtocolVersion defines the standard MCP specification version.
const DefaultProtocolVersion = "2024-11-05"

// JSONRPCRequest represents a JSON-RPC 2.0 request envelope. A request without an ID is a notification.
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response envelope.
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

// JSONRPCError represents standard JSON-RPC 2.0 error object.
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ClientInfo describes the MCP client identity.
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// InitializeParams holds parameters sent during the MCP initialize handshake.
type InitializeParams struct {
	ProtocolVersion string             `json:"protocolVersion"`
	ClientInfo      ClientInfo         `json:"clientInfo"`
	Capabilities    ClientCapabilities `json:"capabilities"`
}

// ClientCapabilities declares optional capabilities supported by the client.
type ClientCapabilities struct {
	Roots    *RootsCapability    `json:"roots,omitempty"`
	Sampling *SamplingCapability `json:"sampling,omitempty"`
}

// RootsCapability indicates root folder tracking support.
type RootsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// SamplingCapability indicates LLM sampling support.
type SamplingCapability struct{}

// ServerInfo describes the remote server identity.
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ServerCapabilities lists capabilities supported by the remote server.
type ServerCapabilities struct {
	Tools     *ToolsCapability     `json:"tools,omitempty"`
	Resources *ResourcesCapability `json:"resources,omitempty"`
	Prompts   *PromptsCapability   `json:"prompts,omitempty"`
}

// ToolsCapability indicates tool calling support.
type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ResourcesCapability indicates resource reading support.
type ResourcesCapability struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

// PromptsCapability indicates prompt template support.
type PromptsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// InitializeResult holds server info and capabilities after handshake.
type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	ServerInfo      ServerInfo         `json:"serverInfo"`
	Capabilities    ServerCapabilities `json:"capabilities"`
}

// Tool represents a tool exposed by the remote server.
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"inputSchema,omitempty"`
}

// ListToolsResult represents the payload returned by tools/list.
type ListToolsResult struct {
	Tools []Tool `json:"tools"`
}

// CallToolParams holds arguments for tools/call.
type CallToolParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// ContentItem holds a single content payload (text, image, or resource) returned by a tool.
type ContentItem struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	Data     string `json:"data,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
}

// CallToolResult represents the response payload for tools/call.
type CallToolResult struct {
	Content []ContentItem `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}
