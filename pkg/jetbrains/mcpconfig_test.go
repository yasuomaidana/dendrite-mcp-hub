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
	res := &SetupResult{FilesCreated: []string{}, FilesUpdated: []string{}}

	// First run creates the file
	noErr(t, configureMCPFile(targetPath, "/bin/dendrite", res))
	check(t, len(res.FilesCreated) == 1 && res.FilesCreated[0] == targetPath, "expected FilesCreated to have targetPath, got %+v", res.FilesCreated)
	check(t, len(res.FilesUpdated) == 0, "expected FilesUpdated to be empty, got %+v", res.FilesUpdated)

	// Second run updates the file
	noErr(t, configureMCPFile(targetPath, "/bin/dendrite", res))
	check(t, len(res.FilesCreated) == 1, "FilesCreated should still have 1 entry, got %d", len(res.FilesCreated))
	check(t, len(res.FilesUpdated) == 1 && res.FilesUpdated[0] == targetPath, "expected FilesUpdated to have targetPath, got %+v", res.FilesUpdated)

	// File content has the dendrite server
	var cfg MCPConfig
	readJSONFile(t, targetPath, &cfg)
	srv := cfg.MCPServers["dendrite"]
	check(t, srv.Command == "/bin/dendrite", "expected command '/bin/dendrite', got '%s'", srv.Command)
	check(t, strings.Join(srv.Args, " ") == "inter serve" && len(srv.Env) == 0, "expected 'inter serve' with no env, got %v %v", srv.Args, srv.Env)

	// Error case: targetPath is a directory
	wantErr(t, configureMCPFile(dir, "/bin/dendrite", res), "configureMCPFile to fail when targetPath is a directory")
}

func TestMCPConfig_StatAndReadErrors(t *testing.T) {
	dir := t.TempDir()
	entry := MCPServerEntry{Command: "x"}

	// Stat error (not IsNotExist): parent is a regular file
	parentFile := filepath.Join(dir, "file_as_dir")
	mustWrite(t, parentFile, "x", 0644)
	badStatPath := filepath.Join(parentFile, "mcp.json")
	_, err := UpsertMCPServer(badStatPath, "srv", entry)
	wantErr(t, err, "stat error in UpsertMCPServer")
	_, err = RemoveMCPServer(badStatPath, "srv")
	wantErr(t, err, "stat error in RemoveMCPServer")

	// ReadFile error: file with 0000 permissions
	unreadable := filepath.Join(dir, "unreadable.json")
	mustWrite(t, unreadable, "{}", 0000)
	t.Cleanup(func() { _ = os.Chmod(unreadable, 0644) })
	_, err = UpsertMCPServer(unreadable, "srv", entry)
	wantErr(t, err, "read error in UpsertMCPServer")
	_, err = RemoveMCPServer(unreadable, "srv")
	wantErr(t, err, "read error in RemoveMCPServer")
}

func TestMCPConfig_WriteErrors(t *testing.T) {
	dir := t.TempDir()

	// WriteFileAtomic error in UpsertMCPServer: read-only directory
	roDir := filepath.Join(dir, "ro_upsert")
	mustMkdir(t, roDir)
	lockDir(t, roDir)
	_, err := UpsertMCPServer(filepath.Join(roDir, "mcp.json"), "srv", MCPServerEntry{Command: "x"})
	wantErr(t, err, "write error in UpsertMCPServer on read-only dir")

	// WriteFileAtomic error in RemoveMCPServer: file exists with server, then dir becomes read-only
	roRemoveDir := filepath.Join(dir, "ro_remove")
	mustMkdir(t, roRemoveDir)
	roRemoveFile := filepath.Join(roRemoveDir, "mcp.json")
	mustWrite(t, roRemoveFile, `{"mcpServers":{"srv":{"command":"x"}}}`, 0644)
	lockDir(t, roRemoveDir)
	_, err = RemoveMCPServer(roRemoveFile, "srv")
	wantErr(t, err, "write error in RemoveMCPServer on read-only dir")
}

// Assertion helpers keep test bodies free of branching.

func check(t *testing.T, ok bool, format string, args ...interface{}) {
	t.Helper()
	if !ok {
		t.Errorf(format, args...)
	}
}

func noErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func wantErr(t *testing.T, err error, what string) {
	t.Helper()
	check(t, err != nil, "expected %s", what)
}

func mustWrite(t *testing.T, path, content string, perm os.FileMode) {
	t.Helper()
	noErr(t, os.WriteFile(path, []byte(content), perm))
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	noErr(t, os.MkdirAll(path, 0755))
}

// lockDir makes dir read-only until the test ends.
func lockDir(t *testing.T, dir string) {
	t.Helper()
	noErr(t, os.Chmod(dir, 0555))
	t.Cleanup(func() { _ = os.Chmod(dir, 0755) })
}

func decodeJSON(t *testing.T, data []byte, v interface{}) {
	t.Helper()
	noErr(t, json.Unmarshal(data, v))
}

func readJSONFile(t *testing.T, path string, v interface{}) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	noErr(t, err)
	decodeJSON(t, data, v)
	return data
}
