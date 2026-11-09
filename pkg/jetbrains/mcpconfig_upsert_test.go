package jetbrains

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpsertMCPServer_NewFile(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "sub", "mcp.json")
	entry := MCPServerEntry{
		Command: "/usr/local/bin/dendrite",
		Args:    []string{"mcp", "serve"},
		Env:     map[string]string{"DENDRITE_ENV": "development"},
	}

	created, err := UpsertMCPServer(targetPath, "dendrite", entry)
	noErr(t, err)
	check(t, created, "expected created to be true for new file")

	var cfg MCPConfig
	data := readJSONFile(t, targetPath, &cfg)
	check(t, strings.HasSuffix(string(data), "\n"), "expected trailing newline")
	srv := cfg.MCPServers["dendrite"]
	check(t, srv.Command == entry.Command, "expected command %s, got %s", entry.Command, srv.Command)
	check(t, len(srv.Args) == len(entry.Args) && srv.Args[0] == "mcp", "unexpected args: %v", srv.Args)
	check(t, srv.Env["DENDRITE_ENV"] == "development", "unexpected env: %v", srv.Env)
}

func TestUpsertMCPServer_ExistingFile_CreatedFalse(t *testing.T) {
	dir := t.TempDir()
	targetPath := filepath.Join(dir, "mcp.json")

	entry1 := MCPServerEntry{
		Command: "bin/v1",
	}
	created1, err := UpsertMCPServer(targetPath, "dendrite", entry1)
	if err != nil || !created1 {
		t.Fatalf("first upsert failed: created=%v, err=%v", created1, err)
	}

	entry2 := MCPServerEntry{
		Command: "bin/v2",
	}
	created2, err := UpsertMCPServer(targetPath, "dendrite", entry2)
	if err != nil {
		t.Fatalf("second upsert failed: %v", err)
	}
	if created2 {
		t.Fatalf("expected created=false on existing file")
	}

	data, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	var cfg MCPConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if cfg.MCPServers["dendrite"].Command != "bin/v2" {
		t.Errorf("expected updated command 'bin/v2', got '%s'", cfg.MCPServers["dendrite"].Command)
	}
}

func TestUpsertMCPServer_PreservesForeignServerAndTopLevelInputs(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "mcp.json")
	mustWrite(t, targetPath, `{
  "inputs": {"token_limit": 4000, "mode": "auto"},
  "mcpServers": {
    "foreign-sse": {"url": "http://x", "type": "sse", "headers": {"Authorization": "Bearer token123"}}
  }
}`, 0644)

	created, err := UpsertMCPServer(targetPath, "dendrite", MCPServerEntry{
		Command: "/bin/dendrite",
		Args:    []string{"mcp", "serve"},
		Env:     map[string]string{"KEY": "VAL"},
	})
	noErr(t, err)
	check(t, !created, "expected created to be false for existing file")

	var topLevel map[string]json.RawMessage
	readJSONFile(t, targetPath, &topLevel)

	// top-level "inputs" key is intact
	var inputs struct {
		Mode       string `json:"mode"`
		TokenLimit int    `json:"token_limit"`
	}
	decodeJSON(t, topLevel["inputs"], &inputs)
	check(t, inputs.Mode == "auto" && inputs.TokenLimit == 4000, "inputs content corrupted: %+v", inputs)

	// mcpServers has both "foreign-sse" and "dendrite"
	var servers map[string]json.RawMessage
	decodeJSON(t, topLevel["mcpServers"], &servers)

	var foreign map[string]interface{}
	decodeJSON(t, servers["foreign-sse"], &foreign)
	check(t, foreign["url"] == "http://x" && foreign["type"] == "sse", "foreign server lost or changed: %v", foreign)

	var dendrite MCPServerEntry
	decodeJSON(t, servers["dendrite"], &dendrite)
	check(t, dendrite.Command == "/bin/dendrite", "unexpected dendrite command: %s", dendrite.Command)
}

func TestUpsertMCPServer_EmptyAndWhitespaceFile(t *testing.T) {
	dir := t.TempDir()

	entry := MCPServerEntry{Command: "/bin/tool"}

	// Empty file
	emptyPath := filepath.Join(dir, "empty.json")
	if err := os.WriteFile(emptyPath, []byte(""), 0644); err != nil {
		t.Fatalf("failed to write empty file: %v", err)
	}
	created, err := UpsertMCPServer(emptyPath, "srv", entry)
	if err != nil {
		t.Fatalf("UpsertMCPServer on empty file failed: %v", err)
	}
	if created {
		t.Errorf("expected created=false because empty file existed")
	}
	data, _ := os.ReadFile(emptyPath)
	var top map[string]json.RawMessage
	if err := json.Unmarshal(data, &top); err != nil || top["mcpServers"] == nil {
		t.Fatalf("failed to create valid json in empty file: %s", string(data))
	}

	// Whitespace file
	wsPath := filepath.Join(dir, "whitespace.json")
	if err := os.WriteFile(wsPath, []byte("  \t\r\n  \n"), 0644); err != nil {
		t.Fatalf("failed to write whitespace file: %v", err)
	}
	created, err = UpsertMCPServer(wsPath, "srv", entry)
	if err != nil {
		t.Fatalf("UpsertMCPServer on whitespace file failed: %v", err)
	}
	if created {
		t.Errorf("expected created=false because whitespace file existed")
	}
}

func TestUpsertMCPServer_InvalidJSON_Untouched(t *testing.T) {
	dir := t.TempDir()
	targetPath := filepath.Join(dir, "corrupt.json")
	invalidContent := "{\n  \"invalid\": json,\n  corrupt\n"

	if err := os.WriteFile(targetPath, []byte(invalidContent), 0644); err != nil {
		t.Fatalf("failed to write invalid file: %v", err)
	}

	entry := MCPServerEntry{Command: "bin/test"}
	created, err := UpsertMCPServer(targetPath, "dendrite", entry)
	if err == nil {
		t.Fatalf("expected error on invalid JSON, got nil")
	}
	if created {
		t.Errorf("expected created=false on error")
	}

	data, readErr := os.ReadFile(targetPath)
	if readErr != nil {
		t.Fatalf("failed to read back corrupt file: %v", readErr)
	}
	if string(data) != invalidContent {
		t.Fatalf("file content was modified despite invalid JSON:\n%s", string(data))
	}
}

func TestUpsertMCPServer_InvalidMCPServers(t *testing.T) {
	dir := t.TempDir()
	targetPath := filepath.Join(dir, "bad_servers.json")
	content := "{\"mcpServers\": \"not-an-object\"}"
	if err := os.WriteFile(targetPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	entry := MCPServerEntry{Command: "bin/test"}
	_, err := UpsertMCPServer(targetPath, "dendrite", entry)
	if err == nil {
		t.Fatalf("expected error when mcpServers is not an object")
	}

	data, _ := os.ReadFile(targetPath)
	if string(data) != content {
		t.Fatalf("file was modified on error")
	}
}

func TestUpsertMCPServer_EmptyServerName(t *testing.T) {
	dir := t.TempDir()
	targetPath := filepath.Join(dir, "test.json")
	_, err := UpsertMCPServer(targetPath, "", MCPServerEntry{Command: "x"})
	if err == nil {
		t.Fatalf("expected error for empty server name")
	}
}

func TestUpsertMCPServer_DirectoryTargetError(t *testing.T) {
	dir := t.TempDir()
	_, err := UpsertMCPServer(dir, "srv", MCPServerEntry{Command: "x"})
	if err == nil {
		t.Fatalf("expected error targeting a directory")
	}
}
