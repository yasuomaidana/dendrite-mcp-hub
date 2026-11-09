package jetbrains_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/yasuomaidana/dendrite-mcp-hub/pkg/jetbrains"
)

func connectedBridge(t *testing.T) (*jetbrains.Bridge, *jetbrains.Client) {
	t.Helper()
	_, transport := newScriptedServer(t)
	client, err := jetbrains.NewClient(transport)
	requireOK(t, err)
	_, err = client.Connect(context.Background())
	requireOK(t, err)
	return jetbrains.NewBridge(client), client
}

func TestBridge_OpenFile(t *testing.T) {
	ctx := context.Background()
	bridge, client := connectedBridge(t)
	expect(t, bridge.GetClient() == client, "bridge.GetClient() mismatch")

	requireOK(t, bridge.OpenFile(ctx, "pkg/jetbrains/bridge.go", 15, 2))
	requireOK(t, bridge.OpenFile(ctx, "pkg/jetbrains/bridge.go", 0, 0)) // line/col clamped to 1
	requireIs(t, bridge.OpenFile(ctx, "", 1, 1), jetbrains.ErrEmptyFilePath)
}

func TestBridge_GetFileProblemsAndSearch(t *testing.T) {
	ctx := context.Background()
	bridge, _ := connectedBridge(t)

	problems, err := bridge.GetFileProblems(ctx, "main.go")
	requireOK(t, err)
	expect(t, len(problems) > 0 && problems[0].Severity == "warning", "unexpected problems: %+v", problems)
	_, err = bridge.GetFileProblems(ctx, "   ")
	requireIs(t, err, jetbrains.ErrEmptyFilePath)

	symbols, err := bridge.SearchSymbol(ctx, "NewClient")
	requireOK(t, err)
	expect(t, len(symbols) > 0 && symbols[0].Name == "NewClient", "unexpected symbols result: %+v", symbols)
}

func TestBridge_BuildRunLint(t *testing.T) {
	ctx := context.Background()
	bridge, _ := connectedBridge(t)

	buildRes, err := bridge.BuildProject(ctx)
	requireOK(t, err)
	expect(t, buildRes.Success, "expected build to succeed")

	output, err := bridge.ExecuteRunConfiguration(ctx, "UnitTestSuite")
	requireOK(t, err)
	expect(t, output != "", "expected non-empty output from run configuration")
	_, err = bridge.ExecuteRunConfiguration(ctx, "")
	requireErr(t, err)

	lintProblems, err := bridge.LintFiles(ctx, []string{"main.go"})
	requireOK(t, err)
	expect(t, len(lintProblems) == 0, "expected 0 lint problems, got %d", len(lintProblems))
}

// toolReply answers a tools/call request with a text (or empty) result.
func toolReply(req *jetbrains.JSONRPCRequest, items ...jetbrains.ContentItem) *jetbrains.JSONRPCResponse {
	b, _ := json.Marshal(jetbrains.CallToolResult{Content: items})
	return &jetbrains.JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: b}
}

func text(s string) jetbrains.ContentItem { return jetbrains.ContentItem{Type: "text", Text: s} }

func toolName(req *jetbrains.JSONRPCRequest) string {
	paramsBytes, _ := json.Marshal(req.Params)
	var callParams jetbrains.CallToolParams
	_ = json.Unmarshal(paramsBytes, &callParams)
	return callParams.Name
}

func TestBridge_FallbackProblems(t *testing.T) {
	ctx := context.Background()
	srv, transport := newScriptedServer(t)
	srv.RegisterHandler("tools/call", func(req *jetbrains.JSONRPCRequest) (*jetbrains.JSONRPCResponse, error) {
		return toolReply(req, text("Syntax error on line 5")), nil
	})
	client, err := jetbrains.NewClient(transport)
	requireOK(t, err)
	_, _ = client.Connect(ctx)

	probs, err := jetbrains.NewBridge(client).GetFileProblems(ctx, "test.go")
	requireOK(t, err)
	expect(t, len(probs) == 1 && probs[0].Description == "Syntax error on line 5", "unexpected fallback problems: %+v", probs)
}

func TestBridge_EmptyContent(t *testing.T) {
	ctx := context.Background()
	srv, transport := newScriptedServer(t)
	srv.RegisterHandler("tools/call", func(req *jetbrains.JSONRPCRequest) (*jetbrains.JSONRPCResponse, error) {
		if toolName(req) == "search_symbol" {
			return toolReply(req), nil // no content items at all
		}
		return toolReply(req, text("")), nil
	})
	client, err := jetbrains.NewClient(transport)
	requireOK(t, err)
	_, _ = client.Connect(ctx)
	bridge := jetbrains.NewBridge(client)

	syms, err := bridge.SearchSymbol(ctx, "Missing")
	requireOK(t, err)
	expect(t, len(syms) == 0, "expected 0 symbols, got %d", len(syms))

	lints, err := bridge.LintFiles(ctx, []string{"test.go"})
	requireOK(t, err)
	expect(t, len(lints) == 0, "expected 0 lints, got %d", len(lints))
}

func TestBridge_DisconnectedErrors(t *testing.T) {
	ctx := context.Background()
	_, transport := newScriptedServer(t)
	disconnectedClient, _ := jetbrains.NewClient(transport)
	b := jetbrains.NewBridge(disconnectedClient)

	requireErr(t, b.OpenFile(ctx, "main.go", 1, 1))
	_, err := b.GetFileProblems(ctx, "main.go")
	requireErr(t, err)
	_, err = b.SearchSymbol(ctx, "Sym")
	requireErr(t, err)
	res, err := b.BuildProject(ctx)
	requireErr(t, err)
	expect(t, res != nil && !res.Success, "expected Success=false when disconnected")
	_, err = b.ExecuteRunConfiguration(ctx, "Run")
	requireErr(t, err)
	_, err = b.LintFiles(ctx, []string{"main.go"})
	requireErr(t, err)
}

func TestBridge_EmptyAndInvalidResults(t *testing.T) {
	ctx := context.Background()
	rt := &recordedTransport{
		sendFn: func(_ context.Context, req *jetbrains.JSONRPCRequest) (*jetbrains.JSONRPCResponse, error) {
			if req.Method == "initialize" || req.Method == "notifications/initialized" {
				return &jetbrains.JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: json.RawMessage(`{"protocolVersion":"2024-11-05"}`)}, nil
			}
			switch toolName(req) {
			case "get_file_problems", "execute_run_configuration":
				return toolReply(req), nil
			case "search_symbol", "lint_files":
				return toolReply(req, text("not-json")), nil
			}
			return nil, errors.New("unexpected")
		},
	}
	c, _ := jetbrains.NewClient(rt)
	_, _ = c.Connect(ctx)
	b := jetbrains.NewBridge(c)

	probs, err := b.GetFileProblems(ctx, "main.go")
	expect(t, err == nil && len(probs) == 0, "expected empty problems, got %+v, err=%v", probs, err)
	out, err := b.ExecuteRunConfiguration(ctx, "Run")
	expect(t, err == nil && out == "", "expected empty output, got %q, err=%v", out, err)
	syms, err := b.SearchSymbol(ctx, "Sym")
	expect(t, err == nil && len(syms) == 0, "expected empty symbols on non-JSON, got %+v, err=%v", syms, err)
	lints, err := b.LintFiles(ctx, []string{"main.go"})
	expect(t, err == nil && len(lints) == 0, "expected empty lints on non-JSON, got %+v, err=%v", lints, err)
}
