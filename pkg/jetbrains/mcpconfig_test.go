package jetbrains

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemoveMCPServer_Present(t *testing.T) {
	dir := t.TempDir()
	targetPath := filepath.Join(dir, "mcp.json")

	content := `{
  "inputs": "preserved",
  "mcpServers": {
    "serverA": {
      "command": "cmdA"
    },
    "serverB": {
      "command": "cmdB"
    }
  }
}`
	if err := os.WriteFile(targetPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	removed, err := RemoveMCPServer(targetPath, "serverA")
	if err != nil {
		t.Fatalf("RemoveMCPServer failed: %v", err)
	}
	if !removed {
		t.Fatalf("expected removed=true")
	}

	data, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	var topLevel map[string]json.RawMessage
	if err := json.Unmarshal(data, &topLevel); err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if string(topLevel["inputs"]) != `"preserved"` {
		t.Errorf("inputs lost or altered: %s", string(topLevel["inputs"]))
	}

	var servers map[string]json.RawMessage
	if err := json.Unmarshal(topLevel["mcpServers"], &servers); err != nil {
		t.Fatalf("parse servers failed: %v", err)
	}
	if _, ok := servers["serverA"]; ok {
		t.Errorf("serverA was not removed")
	}
	if _, ok := servers["serverB"]; !ok {
		t.Errorf("serverB was unintentionally removed")
	}
}

func TestRemoveMCPServer_Absent(t *testing.T) {
	dir := t.TempDir()
	targetPath := filepath.Join(dir, "mcp.json")

	content := `{
  "mcpServers": {
    "serverB": {
      "command": "cmdB"
    }
  }
}`
	if err := os.WriteFile(targetPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	removed, err := RemoveMCPServer(targetPath, "serverA")
	if err != nil {
		t.Fatalf("RemoveMCPServer failed: %v", err)
	}
	if removed {
		t.Fatalf("expected removed=false for absent server")
	}
}

func TestRemoveMCPServer_MissingFile(t *testing.T) {
	dir := t.TempDir()
	targetPath := filepath.Join(dir, "nonexistent.json")

	removed, err := RemoveMCPServer(targetPath, "serverA")
	if err != nil {
		t.Fatalf("RemoveMCPServer returned error on missing file: %v", err)
	}
	if removed {
		t.Fatalf("expected removed=false for missing file")
	}
}

func TestRemoveMCPServer_NoOpContents(t *testing.T) {
	cases := []struct {
		name    string
		content string
	}{
		{"EmptyFile", ""},
		{"NoMCPServersKey", `{"other": true}`},
		{"NullMCPServersKey", `{"mcpServers": null}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			targetPath := filepath.Join(dir, "mcp.json")
			if err := os.WriteFile(targetPath, []byte(tc.content), 0644); err != nil {
				t.Fatalf("failed to write file: %v", err)
			}

			removed, err := RemoveMCPServer(targetPath, "serverA")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if removed {
				t.Fatalf("expected removed=false")
			}
		})
	}
}

func TestRemoveMCPServer_InvalidContents(t *testing.T) {
	cases := []struct {
		name    string
		content string
	}{
		{"InvalidJSON", "{bad"},
		{"InvalidMCPServers", `{"mcpServers": 123}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			targetPath := filepath.Join(dir, "invalid.json")
			if err := os.WriteFile(targetPath, []byte(tc.content), 0644); err != nil {
				t.Fatalf("failed to write file: %v", err)
			}

			removed, err := RemoveMCPServer(targetPath, "serverA")
			if err == nil {
				t.Fatalf("expected error on invalid content")
			}
			if removed {
				t.Fatalf("expected removed=false on error")
			}
		})
	}
}

func TestRemoveMCPServer_EmptyName(t *testing.T) {
	dir := t.TempDir()
	targetPath := filepath.Join(dir, "mcp.json")
	_ = os.WriteFile(targetPath, []byte("{}"), 0644)

	removed, err := RemoveMCPServer(targetPath, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if removed {
		t.Fatalf("expected removed=false for empty name")
	}
}

func TestRemoveMCPServer_DirectoryTargetError(t *testing.T) {
	dir := t.TempDir()
	_, err := RemoveMCPServer(dir, "srv")
	if err == nil {
		t.Fatalf("expected error targeting a directory")
	}
}

func TestConfigureMCPFile_RecordsCreatedVsUpdated(t *testing.T) {
	dir := t.TempDir()
	targetPath := filepath.Join(dir, ".ai", "mcp", "mcp.json")

	res := &SetupResult{
		FilesCreated: []string{},
		FilesUpdated: []string{},
	}

	// 1. First run creates the file
	err := configureMCPFile(targetPath, "/bin/dendrite", res)
	if err != nil {
		t.Fatalf("first configureMCPFile failed: %v", err)
	}
	if len(res.FilesCreated) != 1 || res.FilesCreated[0] != targetPath {
		t.Fatalf("expected FilesCreated to have targetPath, got %+v", res.FilesCreated)
	}
	if len(res.FilesUpdated) != 0 {
		t.Fatalf("expected FilesUpdated to be empty, got %+v", res.FilesUpdated)
	}

	// 2. Second run updates the file
	err = configureMCPFile(targetPath, "/bin/dendrite", res)
	if err != nil {
		t.Fatalf("second configureMCPFile failed: %v", err)
	}
	if len(res.FilesCreated) != 1 {
		t.Fatalf("FilesCreated should still have 1 entry, got %d", len(res.FilesCreated))
	}
	if len(res.FilesUpdated) != 1 || res.FilesUpdated[0] != targetPath {
		t.Fatalf("expected FilesUpdated to have targetPath, got %+v", res.FilesUpdated)
	}

	// Verify file content has dendrite server
	data, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("failed to read mcp.json: %v", err)
	}
	var cfg MCPConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	srv, ok := cfg.MCPServers["dendrite"]
	if !ok {
		t.Fatalf("missing dendrite server in mcpServers")
	}
	if srv.Command != "/bin/dendrite" {
		t.Errorf("expected command '/bin/dendrite', got '%s'", srv.Command)
	}
	if strings.Join(srv.Args, " ") != "inter serve" || len(srv.Env) != 0 {
		t.Errorf("expected 'inter serve' with no env, got %v %v", srv.Args, srv.Env)
	}

	// 3. Error case: targetPath is a directory
	if err := configureMCPFile(dir, "/bin/dendrite", res); err == nil {
		t.Fatalf("expected configureMCPFile to fail when targetPath is a directory")
	}
}

func TestMCPConfig_FilesystemErrors(t *testing.T) {
	dir := t.TempDir()

	// Stat error (not IsNotExist): parent is a regular file
	parentFile := filepath.Join(dir, "file_as_dir")
	if err := os.WriteFile(parentFile, []byte("x"), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	badStatPath := filepath.Join(parentFile, "mcp.json")
	if _, err := UpsertMCPServer(badStatPath, "srv", MCPServerEntry{Command: "x"}); err == nil {
		t.Errorf("expected stat error in UpsertMCPServer")
	}
	if _, err := RemoveMCPServer(badStatPath, "srv"); err == nil {
		t.Errorf("expected stat error in RemoveMCPServer")
	}

	// ReadFile error: file with 0000 permissions
	unreadable := filepath.Join(dir, "unreadable.json")
	if err := os.WriteFile(unreadable, []byte("{}"), 0000); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	defer func() { _ = os.Chmod(unreadable, 0644) }()
	if _, err := UpsertMCPServer(unreadable, "srv", MCPServerEntry{Command: "x"}); err == nil {
		t.Errorf("expected read error in UpsertMCPServer")
	}
	if _, err := RemoveMCPServer(unreadable, "srv"); err == nil {
		t.Errorf("expected read error in RemoveMCPServer")
	}

	// WriteFileAtomic error in UpsertMCPServer: read-only directory
	roDir := filepath.Join(dir, "ro_upsert")
	if err := os.MkdirAll(roDir, 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.Chmod(roDir, 0555); err != nil {
		t.Fatalf("chmod failed: %v", err)
	}
	defer func() { _ = os.Chmod(roDir, 0755) }()
	if _, err := UpsertMCPServer(filepath.Join(roDir, "mcp.json"), "srv", MCPServerEntry{Command: "x"}); err == nil {
		t.Errorf("expected write error in UpsertMCPServer on read-only dir")
	}

	// WriteFileAtomic error in RemoveMCPServer: file exists with server, then dir becomes read-only
	roRemoveDir := filepath.Join(dir, "ro_remove")
	if err := os.MkdirAll(roRemoveDir, 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	roRemoveFile := filepath.Join(roRemoveDir, "mcp.json")
	if err := os.WriteFile(roRemoveFile, []byte(`{"mcpServers":{"srv":{"command":"x"}}}`), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	if err := os.Chmod(roRemoveDir, 0555); err != nil {
		t.Fatalf("chmod failed: %v", err)
	}
	defer func() { _ = os.Chmod(roRemoveDir, 0755) }()
	if _, err := RemoveMCPServer(roRemoveFile, "srv"); err == nil {
		t.Errorf("expected write error in RemoveMCPServer on read-only dir")
	}
}
