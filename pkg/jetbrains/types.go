package jetbrains

import (
	"errors"
	"time"

	"github.com/yasuomaidana/dendrite-mcp-hub/pkg/mcpclient"
)

// Domain errors of the client are those of pkg/mcpclient, so errors.Is matches either name.
var (
	ErrNotConnected            = mcpclient.ErrNotConnected
	ErrAlreadyConnected        = mcpclient.ErrAlreadyConnected
	ErrNilTransport            = mcpclient.ErrNilTransport
	ErrToolNotFound            = mcpclient.ErrToolNotFound
	ErrExecutionFailed         = mcpclient.ErrExecutionFailed
	ErrInvalidResponse         = mcpclient.ErrInvalidResponse
	ErrEmptyToolName           = mcpclient.ErrEmptyToolName
	ErrProtocolVersionMismatch = mcpclient.ErrProtocolVersionMismatch
	ErrEmptyFilePath           = errors.New("file path cannot be empty")
)

// DefaultProtocolVersion defines the standard MCP specification version.
const DefaultProtocolVersion = mcpclient.DefaultProtocolVersion

// Wire types shared with pkg/mcpclient.
type (
	JSONRPCRequest      = mcpclient.JSONRPCRequest
	JSONRPCResponse     = mcpclient.JSONRPCResponse
	JSONRPCError        = mcpclient.JSONRPCError
	ClientInfo          = mcpclient.ClientInfo
	InitializeParams    = mcpclient.InitializeParams
	ClientCapabilities  = mcpclient.ClientCapabilities
	RootsCapability     = mcpclient.RootsCapability
	SamplingCapability  = mcpclient.SamplingCapability
	ServerInfo          = mcpclient.ServerInfo
	ServerCapabilities  = mcpclient.ServerCapabilities
	ToolsCapability     = mcpclient.ToolsCapability
	ResourcesCapability = mcpclient.ResourcesCapability
	PromptsCapability   = mcpclient.PromptsCapability
	InitializeResult    = mcpclient.InitializeResult
	Tool                = mcpclient.Tool
	ListToolsResult     = mcpclient.ListToolsResult
	CallToolParams      = mcpclient.CallToolParams
	ContentItem         = mcpclient.ContentItem
	CallToolResult      = mcpclient.CallToolResult
)

// Problem represents an IDE diagnostic or inspection problem.
type Problem struct {
	File        string `json:"file"`
	Line        int    `json:"line"`
	Column      int    `json:"column"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
}

// Symbol represents an IDE code symbol found across project index.
type Symbol struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	File      string `json:"file"`
	Line      int    `json:"line"`
	Container string `json:"container,omitempty"`
}

// BuildResult represents IDE project build execution summary.
type BuildResult struct {
	Success  bool          `json:"success"`
	Errors   []string      `json:"errors,omitempty"`
	Duration time.Duration `json:"duration"`
}
