package jetbrains

import "github.com/yasuomaidana/dendrite-mcp-hub/pkg/mcpclient"

// The generic MCP client lives in pkg/mcpclient; JetBrains is just one server it talks to.
type (
	Client         = mcpclient.Client
	Option         = mcpclient.Option
	Transport      = mcpclient.Transport
	HTTPTransport  = mcpclient.HTTPTransport
	StdioTransport = mcpclient.StdioTransport
)

// Constructors and options of the generic client.
var (
	NewHTTPTransport    = mcpclient.NewHTTPTransport
	NewStdioTransport   = mcpclient.NewStdioTransport
	WithClientInfo      = mcpclient.WithClientInfo
	WithProtocolVersion = mcpclient.WithProtocolVersion
)

// NewClient initializes a client for the JetBrains MCP server; options override its identity.
func NewClient(transport Transport, opts ...Option) (*Client, error) {
	identity := WithClientInfo("dendrite-jetbrains-bridge", "0.2.0")
	return mcpclient.NewClient(transport, append([]Option{identity}, opts...)...)
}
