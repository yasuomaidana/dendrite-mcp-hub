package mcpserver

import (
	"encoding/json"
	"fmt"
)

// IndentJSON renders v as indented JSON. Tool results are plain data (structs, maps and slices of
// strings and numbers), which always marshal.
func IndentJSON(v interface{}) []byte {
	data, _ := json.MarshalIndent(v, "", "  ")
	return data
}

// TextResult is a successful tool result holding one text item.
func TextResult(text string) *CallToolResult {
	return &CallToolResult{Content: []ContentItem{{Type: "text", Text: text}}}
}

// JsonResult renders v as an indented JSON text result, or the marshal error.
func JsonResult(v interface{}) (*CallToolResult, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return TextResult(string(data)), nil
}

// ErrorResult renders a tool-level failure the agent can read and react to.
func ErrorResult(format string, a ...interface{}) *CallToolResult {
	return &CallToolResult{IsError: true, Content: []ContentItem{{Type: "text", Text: fmt.Sprintf(format, a...)}}}
}

// ResultOrError renders a successful result as indented JSON text, or the failure under failFormat.
func ResultOrError(result interface{}, err error, failFormat string) *CallToolResult {
	if err != nil {
		return ErrorResult(failFormat, err)
	}
	return TextResult(string(IndentJSON(result)))
}
