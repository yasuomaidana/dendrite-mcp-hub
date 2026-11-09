package jetbrains

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpsertMCPServerAcceptsNullDocuments(t *testing.T) {
	for _, doc := range []string{`null`, `{"mcpServers": null}`} {
		path := filepath.Join(t.TempDir(), "mcp.json")
		if err := os.WriteFile(path, []byte(doc), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := UpsertMCPServer(path, "dendrite", MCPServerEntry{Command: "dendrite"}); err != nil {
			t.Fatalf("%s: %v", doc, err)
		}
		data, _ := os.ReadFile(path)
		if !strings.Contains(string(data), `"dendrite"`) {
			t.Fatalf("%s: server missing from %s", doc, data)
		}
	}
}

func TestRemoveMCPServerReportsAnUnreadableFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mcp.json")
	if err := os.WriteFile(path, []byte(`{"mcpServers":{"a":{}}}`), 0o000); err != nil {
		t.Fatal(err)
	}
	if _, err := RemoveMCPServer(path, "a"); err == nil && os.Geteuid() != 0 {
		t.Fatal("an unreadable file was reported as fine")
	}
}

func TestDisconnectAndCloseTolerateMissingParts(t *testing.T) {
	if err := (&Client{connected: true}).Disconnect(); err != nil {
		t.Fatalf("Disconnect without a transport: %v", err)
	}
	for _, tr := range []*StdioTransport{{}, {cmd: &exec.Cmd{}}} {
		if err := tr.Close(); err != nil {
			t.Fatalf("Close = %v", err)
		}
	}
}

func TestSetupFailsWhenTheProjectDirectoryCannotBeResolved(t *testing.T) {
	boom := errors.New("cannot resolve")
	swap(t, &getwd, func() (string, error) { return "", boom })
	if _, err := SetupJetBrainsToolset(SetupOptions{}); !errors.Is(err, boom) {
		t.Fatalf("getwd failure = %v", err)
	}
	swap(t, &getwd, os.Getwd)
	swap(t, &absPath, func(string) (string, error) { return "", boom })
	if _, err := SetupJetBrainsToolset(SetupOptions{ProjectDir: "x"}); !errors.Is(err, boom) {
		t.Fatalf("abs failure = %v", err)
	}
}

func TestRemoveMCPServerTreatsAVanishedFileAsNothingToRemove(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mcp.json")
	if err := os.WriteFile(path, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	swap(t, &readFile, func(string) ([]byte, error) { return nil, os.ErrNotExist })
	if removed, err := RemoveMCPServer(path, "a"); removed || err != nil {
		t.Fatalf("removed=%v err=%v", removed, err)
	}
}

func TestDefaultBinaryPathPrefersTheRunningDendrite(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "dendrite")
	if err := os.WriteFile(exe, nil, 0o755); err != nil {
		t.Fatal(err)
	}
	swap(t, &currentExecutable, func() (string, error) { return exe, nil })
	want, _ := filepath.EvalSymlinks(exe)
	if got := defaultBinaryPath(); got != want {
		t.Fatalf("defaultBinaryPath = %q, want %q", got, want)
	}
}

func TestNewStdioTransportReportsPipeFailures(t *testing.T) {
	boom := errors.New("pipe failed")
	swap(t, &pipeStdin, func(*exec.Cmd) (io.WriteCloser, error) { return nil, boom })
	if _, err := NewStdioTransport("true"); !errors.Is(err, boom) {
		t.Fatalf("stdin failure = %v", err)
	}
	swap(t, &pipeStdin, (*exec.Cmd).StdinPipe)
	swap(t, &pipeStdout, func(*exec.Cmd) (io.ReadCloser, error) { return nil, boom })
	if _, err := NewStdioTransport("true"); !errors.Is(err, boom) {
		t.Fatalf("stdout failure = %v", err)
	}
}

// swap replaces a package-level seam for the length of the test.
func swap[T any](t *testing.T, target *T, value T) {
	t.Helper()
	old := *target
	t.Cleanup(func() { *target = old })
	*target = value
}
