package mcpserver

// FilterTools returns a server with the identity of src that holds only the tools of src for which keep
// returns true. The handlers are shared with src, not copied. Hosts use it to expose a smaller tool
// surface to agents (every listed tool costs context in every request) without registering tools twice.
func FilterTools(src *Server, keep func(name string) bool) *Server {
	dst := NewServer(ServerConfig(src.Info()))
	for _, t := range src.ListTools() {
		if !keep(t.Name) {
			continue
		}
		if h, ok := src.Handler(t.Name); ok {
			dst.RegisterTool(t, h)
		}
	}
	return dst
}
