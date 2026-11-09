package jetbrains_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yasuomaidana/dendrite-mcp-hub/pkg/jetbrains"
)

func readConfig(t *testing.T, path string) jetbrains.MCPConfig {
	t.Helper()
	data, err := os.ReadFile(path)
	requireOK(t, err)
	var cfg jetbrains.MCPConfig
	requireOK(t, json.Unmarshal(data, &cfg))
	return cfg
}

func TestSetupJetBrainsToolset_Full(t *testing.T) {
	tempProject := t.TempDir()
	globalAIPath := filepath.Join(t.TempDir(), ".ai", "mcp", "mcp.json")
	// Pre-create .idea folder to simulate an IDE project
	requireOK(t, os.MkdirAll(filepath.Join(tempProject, ".idea"), 0755))

	opts := jetbrains.SetupOptions{
		ProjectDir:             tempProject,
		BinaryPath:             filepath.Join(tempProject, "bin", "dendrite"),
		ConfigureProjectMCP:    true,
		ConfigureGlobalMCP:     true,
		ConfigureRunConfigs:    true,
		ConfigureExternalTools: true,
		GlobalAIPathOverride:   globalAIPath,
	}

	res, err := jetbrains.SetupJetBrainsToolset(opts)
	requireOK(t, err)
	expect(t, len(res.FilesCreated) > 0, "expected created files, got 0")

	projCfg := readConfig(t, filepath.Join(tempProject, ".ai", "mcp", "mcp.json"))
	expect(t, projCfg.MCPServers["dendrite"].Command == opts.BinaryPath, "unexpected project command: %+v", projCfg.MCPServers)
	globalCfg := readConfig(t, globalAIPath)
	expect(t, globalCfg.MCPServers["dendrite"].Command == opts.BinaryPath, "unexpected global command: %+v", globalCfg.MCPServers)

	entries, err := os.ReadDir(filepath.Join(tempProject, ".idea", "runConfigurations"))
	requireOK(t, err)
	expect(t, len(entries) >= 5, "expected at least 5 run configurations, found: %d", len(entries))

	_, err = os.Stat(filepath.Join(tempProject, ".idea", "tools", "External_Tools.xml"))
	requireOK(t, err)

	// Idempotence: running again updates existing files without errors
	res2, err := jetbrains.SetupJetBrainsToolset(opts)
	requireOK(t, err)
	expect(t, len(res2.FilesUpdated) > 0, "expected updated files on re-run, got 0")
}

func TestSetupJetBrainsToolset_DefaultPaths(t *testing.T) {
	tempProject := t.TempDir()
	opts := jetbrains.SetupOptions{
		ProjectDir:          tempProject,
		ConfigureProjectMCP: true,
	}

	res, err := jetbrains.SetupJetBrainsToolset(opts)
	if err != nil {
		t.Fatalf("setup with defaults failed: %v", err)
	}
	if len(res.FilesCreated) != 1 {
		t.Fatalf("expected 1 created file, got %d", len(res.FilesCreated))
	}
}

func TestDefaultBinaryPathNeverInsideProject(t *testing.T) {
	proj := t.TempDir()
	res, err := jetbrains.SetupJetBrainsToolset(jetbrains.SetupOptions{ProjectDir: proj, ConfigureProjectMCP: true, GlobalAIPathOverride: filepath.Join(proj, "g.json")})
	if err != nil {
		t.Fatal(err)
	}
	_ = res
	if _, err := os.Stat(filepath.Join(proj, "bin")); err == nil {
		t.Fatal("setup created or referenced a project-local bin/ directory")
	}
	data, _ := os.ReadFile(filepath.Join(proj, ".ai", "mcp", "mcp.json"))
	if strings.Contains(string(data), filepath.Join(proj, "bin")) {
		t.Fatalf("mcp.json points into the project bin/: %s", data)
	}
}

func TestSetupJetBrainsToolset_EmptyProjectDirUsesWd(t *testing.T) {
	origWd, err := os.Getwd()
	requireOK(t, err)
	requireOK(t, os.Chdir(t.TempDir()))
	t.Cleanup(func() { _ = os.Chdir(origWd) })

	// PATH without dendrite so defaultBinaryPath falls back to "dendrite"
	t.Setenv("PATH", t.TempDir())
	res, err := jetbrains.SetupJetBrainsToolset(jetbrains.SetupOptions{ConfigureProjectMCP: true})
	expect(t, err == nil && len(res.FilesCreated) == 1, "setup with empty ProjectDir failed: %v, res=%+v", err, res)
}

func TestSetupJetBrainsToolset_BinaryOnPath(t *testing.T) {
	binDir := t.TempDir()
	requireOK(t, os.WriteFile(filepath.Join(binDir, "dendrite"), []byte("#!/bin/sh\n"), 0755))
	t.Setenv("PATH", binDir)
	_, err := jetbrains.SetupJetBrainsToolset(jetbrains.SetupOptions{ProjectDir: t.TempDir(), ConfigureProjectMCP: true})
	requireOK(t, err)
}

func TestSetupJetBrainsToolset_GlobalMCPHome(t *testing.T) {
	proj := t.TempDir()
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)
	_, err := jetbrains.SetupJetBrainsToolset(jetbrains.SetupOptions{ProjectDir: proj, ConfigureGlobalMCP: true})
	requireOK(t, err)
	_, err = os.Stat(filepath.Join(fakeHome, ".ai", "mcp", "mcp.json"))
	expect(t, err == nil, "expected global mcp.json in fakeHome: %v", err)

	// HOME empty -> os.UserHomeDir() error
	t.Setenv("HOME", "")
	_, err = jetbrains.SetupJetBrainsToolset(jetbrains.SetupOptions{ProjectDir: proj, ConfigureGlobalMCP: true})
	expect(t, err != nil, "expected error when HOME is empty and GlobalAIPathOverride is unset")
}

// projectWith builds a project dir holding a regular file at rel, or a read-only dir when readOnly.
func projectWith(t *testing.T, rel string, readOnly bool) string {
	t.Helper()
	proj := t.TempDir()
	path := filepath.Join(proj, rel)
	if !readOnly {
		requireOK(t, os.WriteFile(path, []byte("file"), 0644))
		return proj
	}
	requireOK(t, os.MkdirAll(path, 0755))
	requireOK(t, os.Chmod(path, 0555))
	t.Cleanup(func() { _ = os.Chmod(path, 0755) })
	return proj
}

func TestSetupJetBrainsToolset_Errors(t *testing.T) {
	cases := []struct {
		name string
		opts jetbrains.SetupOptions
	}{
		{"project mcp: .ai is a file", jetbrains.SetupOptions{ProjectDir: projectWith(t, ".ai", false), ConfigureProjectMCP: true}},
		{"global mcp: override is a directory", jetbrains.SetupOptions{ProjectDir: t.TempDir(), ConfigureGlobalMCP: true, GlobalAIPathOverride: t.TempDir()}},
		{"run configs: .idea is a file", jetbrains.SetupOptions{ProjectDir: projectWith(t, ".idea", false), ConfigureRunConfigs: true}},
		{"run configs: dir read-only", jetbrains.SetupOptions{ProjectDir: projectWith(t, filepath.Join(".idea", "runConfigurations"), true), ConfigureRunConfigs: true}},
		{"external tools: .idea is a file", jetbrains.SetupOptions{ProjectDir: projectWith(t, ".idea", false), ConfigureExternalTools: true}},
		{"external tools: dir read-only", jetbrains.SetupOptions{ProjectDir: projectWith(t, filepath.Join(".idea", "tools"), true), ConfigureExternalTools: true}},
	}
	for _, tc := range cases {
		_, err := jetbrains.SetupJetBrainsToolset(tc.opts)
		expect(t, err != nil, "%s: expected an error", tc.name)
	}
}
