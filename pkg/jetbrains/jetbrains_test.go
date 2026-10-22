package jetbrains_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/yasuomaidana/dendrite-mcp-hub/pkg/jetbrains"
)

type scriptedServer struct {
	server       *httptest.Server
	mu           sync.Mutex
	handlers     map[string]func(req *jetbrains.JSONRPCRequest) (*jetbrains.JSONRPCResponse, error)
	sentRequests []*jetbrains.JSONRPCRequest
}

func newScriptedServer(t *testing.T) (*scriptedServer, *jetbrains.HTTPTransport) {
	t.Helper()
	s := &scriptedServer{
		handlers:     make(map[string]func(req *jetbrains.JSONRPCRequest) (*jetbrains.JSONRPCResponse, error)),
		sentRequests: make([]*jetbrains.JSONRPCRequest, 0),
	}
	s.registerDefaultHandlers()
	s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req jetbrains.JSONRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		s.mu.Lock()
		s.sentRequests = append(s.sentRequests, &req)
		handler, exists := s.handlers[req.Method]
		s.mu.Unlock()

		if !exists {
			http.Error(w, fmt.Sprintf("unhandled method: %s", req.Method), http.StatusNotImplemented)
			return
		}
		resp, err := handler(&req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(func() {
		s.server.Close()
	})
	return s, jetbrains.NewHTTPTransport(s.server.URL, s.server.Client())
}

func (s *scriptedServer) RegisterHandler(method string, handler func(req *jetbrains.JSONRPCRequest) (*jetbrains.JSONRPCResponse, error)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[method] = handler
}

func (s *scriptedServer) registerDefaultHandlers() {
	s.handlers["initialize"] = func(req *jetbrains.JSONRPCRequest) (*jetbrains.JSONRPCResponse, error) {
		initResult := jetbrains.InitializeResult{
			ProtocolVersion: jetbrains.DefaultProtocolVersion,
			ServerInfo: jetbrains.ServerInfo{
				Name:    "JetBrains GoLand MCP Server",
				Version: "2024.3",
			},
			Capabilities: jetbrains.ServerCapabilities{
				Tools:     &jetbrains.ToolsCapability{ListChanged: true},
				Resources: &jetbrains.ResourcesCapability{Subscribe: false},
			},
		}
		resBytes, _ := json.Marshal(initResult)
		return &jetbrains.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  resBytes,
		}, nil
	}

	s.handlers["notifications/initialized"] = func(req *jetbrains.JSONRPCRequest) (*jetbrains.JSONRPCResponse, error) {
		return &jetbrains.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
		}, nil
	}

	s.handlers["ping"] = func(req *jetbrains.JSONRPCRequest) (*jetbrains.JSONRPCResponse, error) {
		return &jetbrains.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  json.RawMessage(`{}`),
		}, nil
	}

	s.handlers["tools/list"] = func(req *jetbrains.JSONRPCRequest) (*jetbrains.JSONRPCResponse, error) {
		listRes := jetbrains.ListToolsResult{
			Tools: []jetbrains.Tool{
				{
					Name:        "open_file_in_editor",
					Description: "Opens the specified file in the active JetBrains editor window.",
				},
				{
					Name:        "get_file_problems",
					Description: "Retrieves inspection problems and syntax errors for a file.",
				},
				{
					Name:        "search_symbol",
					Description: "Searches for symbols matching query across project index.",
				},
				{
					Name:        "build_project",
					Description: "Compiles and builds the currently open project in JetBrains.",
				},
				{
					Name:        "execute_run_configuration",
					Description: "Runs a specified run/test configuration inside the IDE.",
				},
				{
					Name:        "lint_files",
					Description: "Lints specified file paths using JetBrains inspections.",
				},
			},
		}
		resBytes, _ := json.Marshal(listRes)
		return &jetbrains.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  resBytes,
		}, nil
	}

	s.handlers["tools/call"] = func(req *jetbrains.JSONRPCRequest) (*jetbrains.JSONRPCResponse, error) {
		paramsBytes, _ := json.Marshal(req.Params)
		var callParams jetbrains.CallToolParams
		if err := json.Unmarshal(paramsBytes, &callParams); err != nil {
			return nil, err
		}

		var callRes jetbrains.CallToolResult
		switch callParams.Name {
		case "open_file_in_editor":
			callRes = jetbrains.CallToolResult{
				Content: []jetbrains.ContentItem{{Type: "text", Text: "File opened successfully"}},
			}
		case "get_file_problems":
			problems := []jetbrains.Problem{
				{File: "main.go", Line: 12, Column: 5, Severity: "warning", Description: "Unused variable: count"},
			}
			pBytes, _ := json.Marshal(problems)
			callRes = jetbrains.CallToolResult{
				Content: []jetbrains.ContentItem{{Type: "text", Text: string(pBytes)}},
			}
		case "search_symbol":
			symbols := []jetbrains.Symbol{
				{Name: "NewClient", Kind: "function", File: "pkg/jetbrains/client.go", Line: 30},
			}
			sBytes, _ := json.Marshal(symbols)
			callRes = jetbrains.CallToolResult{
				Content: []jetbrains.ContentItem{{Type: "text", Text: string(sBytes)}},
			}
		case "build_project":
			callRes = jetbrains.CallToolResult{
				Content: []jetbrains.ContentItem{{Type: "text", Text: "Build completed successfully in 1.45s"}},
			}
		case "execute_run_configuration":
			configName := "Test"
			if v, ok := callParams.Arguments["configuration_name"].(string); ok {
				configName = v
			}
			callRes = jetbrains.CallToolResult{
				Content: []jetbrains.ContentItem{{Type: "text", Text: fmt.Sprintf("Run configuration '%s' finished with exit code 0", configName)}},
			}
		case "lint_files":
			callRes = jetbrains.CallToolResult{
				Content: []jetbrains.ContentItem{{Type: "text", Text: "[]"}},
			}
		default:
			return &jetbrains.JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &jetbrains.JSONRPCError{
					Code:    -32601,
					Message: fmt.Sprintf("tool '%s' not found", callParams.Name),
				},
			}, nil
		}

		resBytes, _ := json.Marshal(callRes)
		return &jetbrains.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  resBytes,
		}, nil
	}
}
