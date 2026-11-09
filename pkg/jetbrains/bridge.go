package jetbrains

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Bridge provides high-level strongly-typed IDE operations over JetBrains MCP.
type Bridge struct {
	client *Client
}

// NewBridge creates a new JetBrains IDE bridge around an MCP client.
func NewBridge(client *Client) *Bridge {
	return &Bridge{client: client}
}

// GetClient returns underlying MCP client.
func (b *Bridge) GetClient() *Client {
	return b.client
}

// firstText returns the text of the first content item, or "" when there is none.
func firstText(res *CallToolResult) string {
	if len(res.Content) == 0 {
		return ""
	}
	return res.Content[0].Text
}

// decodeList decodes a JSON array; empty or non-JSON text yields an empty (non-nil) slice.
func decodeList[T any](text string) []T {
	var items []T
	if text == "" || json.Unmarshal([]byte(text), &items) != nil {
		return []T{}
	}
	return items
}

// callList calls an IDE tool whose text result is a JSON array of T.
func callList[T any](ctx context.Context, b *Bridge, tool, what string, args map[string]interface{}) ([]T, error) {
	res, err := b.client.CallTool(ctx, tool, args)
	if err != nil {
		return nil, fmt.Errorf("%s failed: %w", what, err)
	}
	return decodeList[T](firstText(res)), nil
}

// OpenFile instructs JetBrains IDE to open a file and place the cursor at line and column.
func (b *Bridge) OpenFile(ctx context.Context, filePath string, line, col int) error {
	if strings.TrimSpace(filePath) == "" {
		return ErrEmptyFilePath
	}
	if line < 1 {
		line = 1
	}
	if col < 1 {
		col = 1
	}

	args := map[string]interface{}{
		"file_path": filePath,
		"line":      line,
		"column":    col,
	}

	_, err := b.client.CallTool(ctx, "open_file_in_editor", args)
	if err != nil {
		return fmt.Errorf("open file failed: %w", err)
	}
	return nil
}

// GetFileProblems retrieves syntax errors, linter warnings, and inspection issues from IDE.
func (b *Bridge) GetFileProblems(ctx context.Context, filePath string) ([]Problem, error) {
	if strings.TrimSpace(filePath) == "" {
		return nil, ErrEmptyFilePath
	}

	args := map[string]interface{}{
		"file_path": filePath,
	}

	res, err := b.client.CallTool(ctx, "get_file_problems", args)
	if err != nil {
		return nil, fmt.Errorf("get problems failed: %w", err)
	}

	text := firstText(res)
	if text == "" {
		return []Problem{}, nil
	}

	var problems []Problem
	if err := json.Unmarshal([]byte(text), &problems); err != nil {
		// Non-JSON plain text fallback
		return []Problem{
			{
				File:        filePath,
				Description: text,
				Severity:    "info",
			},
		}, nil
	}

	return problems, nil
}

// SearchSymbol searches for functions, structs, interfaces, and variables across project PSI index.
func (b *Bridge) SearchSymbol(ctx context.Context, query string) ([]Symbol, error) {
	return callList[Symbol](ctx, b, "search_symbol", "search symbol", map[string]interface{}{"query": query})
}

// BuildProject triggers IDE background compilation and reports status.
func (b *Bridge) BuildProject(ctx context.Context) (*BuildResult, error) {
	start := time.Now()
	res, err := b.client.CallTool(ctx, "build_project", nil)
	duration := time.Since(start)

	if err != nil {
		return &BuildResult{
			Success:  false,
			Errors:   []string{err.Error()},
			Duration: duration,
		}, err
	}

	return &BuildResult{
		Success:  !res.IsError,
		Duration: duration,
	}, nil
}

// ExecuteRunConfiguration runs an IDE configuration (such as a unit test suite or application runner).
func (b *Bridge) ExecuteRunConfiguration(ctx context.Context, configName string) (string, error) {
	if strings.TrimSpace(configName) == "" {
		return "", fmt.Errorf("configuration name cannot be empty")
	}

	args := map[string]interface{}{
		"configuration_name": configName,
	}

	res, err := b.client.CallTool(ctx, "execute_run_configuration", args)
	if err != nil {
		return "", fmt.Errorf("execute run configuration failed: %w", err)
	}

	if len(res.Content) > 0 {
		return res.Content[0].Text, nil
	}
	return "", nil
}

// LintFiles lints multiple files using JetBrains inspections.
func (b *Bridge) LintFiles(ctx context.Context, files []string) ([]Problem, error) {
	return callList[Problem](ctx, b, "lint_files", "lint files", map[string]interface{}{"files": files})
}
