package jetbrains_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/yasuomaidana/dendrite-mcp-hub/pkg/jetbrains"
)

func TestBridge_HighLevelMethods(t *testing.T) {
	ctx := context.Background()
	_, transport := newScriptedServer(t)
	client, err := jetbrains.NewClient(transport)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	if _, err := client.Connect(ctx); err != nil {
		t.Fatalf("failed to connect: %v", err)
	}

	bridge := jetbrains.NewBridge(client)
	if bridge.GetClient() != client {
		t.Error("bridge.GetClient() mismatch")
	}

	// 1. OpenFile
	if err := bridge.OpenFile(ctx, "pkg/jetbrains/bridge.go", 15, 2); err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	// Line and col clamping (< 1)
	if err := bridge.OpenFile(ctx, "pkg/jetbrains/bridge.go", 0, 0); err != nil {
		t.Fatalf("OpenFile with line/col 0 failed: %v", err)
	}
	if err := bridge.OpenFile(ctx, "", 1, 1); !errors.Is(err, jetbrains.ErrEmptyFilePath) {
		t.Errorf("expected ErrEmptyFilePath, got: %v", err)
	}

	// 2. GetFileProblems
	problems, err := bridge.GetFileProblems(ctx, "main.go")
	if err != nil {
		t.Fatalf("GetFileProblems failed: %v", err)
	}
	if len(problems) == 0 {
		t.Fatal("expected problems for main.go")
	}
	if problems[0].Severity != "warning" {
		t.Errorf("expected severity 'warning', got %s", problems[0].Severity)
	}
	if _, err := bridge.GetFileProblems(ctx, "   "); !errors.Is(err, jetbrains.ErrEmptyFilePath) {
		t.Errorf("expected ErrEmptyFilePath, got: %v", err)
	}

	// 3. SearchSymbol
	symbols, err := bridge.SearchSymbol(ctx, "NewClient")
	if err != nil {
		t.Fatalf("SearchSymbol failed: %v", err)
	}
	if len(symbols) == 0 || symbols[0].Name != "NewClient" {
		t.Errorf("unexpected symbols result: %+v", symbols)
	}

	// 4. BuildProject
	buildRes, err := bridge.BuildProject(ctx)
	if err != nil {
		t.Fatalf("BuildProject failed: %v", err)
	}
	if !buildRes.Success {
		t.Error("expected build to succeed")
	}

	// 5. ExecuteRunConfiguration
	output, err := bridge.ExecuteRunConfiguration(ctx, "UnitTestSuite")
	if err != nil {
		t.Fatalf("ExecuteRunConfiguration failed: %v", err)
	}
	if output == "" {
		t.Error("expected non-empty output from run configuration")
	}
	if _, err := bridge.ExecuteRunConfiguration(ctx, ""); err == nil {
		t.Error("expected error for empty configuration name")
	}

	// 6. LintFiles
	lintProblems, err := bridge.LintFiles(ctx, []string{"main.go"})
	if err != nil {
		t.Fatalf("LintFiles failed: %v", err)
	}
	if len(lintProblems) != 0 {
		t.Errorf("expected 0 lint problems, got %d", len(lintProblems))
	}
}

func TestBridge_EmptyAndFallbackContent(t *testing.T) {
	ctx := context.Background()
	srv, transport := newScriptedServer(t)

	srv.RegisterHandler("tools/call", func(req *jetbrains.JSONRPCRequest) (*jetbrains.JSONRPCResponse, error) {
		paramsBytes, _ := json.Marshal(req.Params)
		var callParams jetbrains.CallToolParams
		_ = json.Unmarshal(paramsBytes, &callParams)

		switch callParams.Name {
		case "get_file_problems":
			// Plaintext non-JSON fallback
			res := jetbrains.CallToolResult{
				Content: []jetbrains.ContentItem{{Type: "text", Text: "Syntax error on line 5"}},
			}
			b, _ := json.Marshal(res)
			return &jetbrains.JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: b}, nil
		case "search_symbol":
			// Empty content
			res := jetbrains.CallToolResult{Content: []jetbrains.ContentItem{}}
			b, _ := json.Marshal(res)
			return &jetbrains.JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: b}, nil
		case "lint_files":
			// Non-JSON or empty
			res := jetbrains.CallToolResult{Content: []jetbrains.ContentItem{{Type: "text", Text: ""}}}
			b, _ := json.Marshal(res)
			return &jetbrains.JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: b}, nil
		default:
			return nil, errors.New("unhandled")
		}
	})

	client, err := jetbrains.NewClient(transport)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, _ = client.Connect(ctx)
	bridge := jetbrains.NewBridge(client)

	// Plain text problems fallback
	probs, err := bridge.GetFileProblems(ctx, "test.go")
	if err != nil {
		t.Fatalf("GetFileProblems failed: %v", err)
	}
	if len(probs) != 1 || probs[0].Description != "Syntax error on line 5" {
		t.Errorf("unexpected fallback problems: %+v", probs)
	}

	// Empty symbols
	syms, err := bridge.SearchSymbol(ctx, "Missing")
	if err != nil {
		t.Fatalf("SearchSymbol failed: %v", err)
	}
	if len(syms) != 0 {
		t.Errorf("expected 0 symbols, got %d", len(syms))
	}

	// Empty lint files
	lints, err := bridge.LintFiles(ctx, []string{"test.go"})
	if err != nil {
		t.Fatalf("LintFiles failed: %v", err)
	}
	if len(lints) != 0 {
		t.Errorf("expected 0 lints, got %d", len(lints))
	}
}
