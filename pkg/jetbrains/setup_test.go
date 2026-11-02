package jetbrains_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yasuomaidana/dendrite-mcp-hub/pkg/jetbrains"
)

func TestSetupJetBrainsToolset_Full(t *testing.T) {
	tempProject := t.TempDir()
	tempGlobal := t.TempDir()
	globalAIPath := filepath.Join(tempGlobal, ".ai", "mcp", "mcp.json")

	// Pre-create .idea folder to simulate an IDE project
	if err := os.MkdirAll(filepath.Join(tempProject, ".idea"), 0755); err != nil {
		t.Fatalf("failed to create .idea: %v", err)
	}

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
	if err != nil {
		t.Fatalf("SetupJetBrainsToolset failed: %v", err)
	}

	if len(res.FilesCreated) == 0 {
		t.Fatal("expected created files, got 0")
	}

	// 1. Verify project MCP file
	projMCPPath := filepath.Join(tempProject, ".ai", "mcp", "mcp.json")
	data, err := os.ReadFile(projMCPPath)
	if err != nil {
		t.Fatalf("failed to read project mcp.json: %v", err)
	}
	var cfg jetbrains.MCPConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("invalid json in project mcp.json: %v", err)
	}
	dendriteServer, ok := cfg.MCPServers["dendrite"]
	if !ok {
		t.Fatal("expected 'dendrite' server in mcpServers")
	}
	if dendriteServer.Command != opts.BinaryPath {
		t.Errorf("expected command %s, got %s", opts.BinaryPath, dendriteServer.Command)
	}

	// 2. Verify global MCP file
	data, err = os.ReadFile(globalAIPath)
	if err != nil {
		t.Fatalf("failed to read global mcp.json: %v", err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("invalid json in global mcp.json: %v", err)
	}
	if _, ok := cfg.MCPServers["dendrite"]; !ok {
		t.Fatal("expected 'dendrite' server in global mcp.json")
	}

	// 3. Verify Run Configurations
	runConfigDir := filepath.Join(tempProject, ".idea", "runConfigurations")
	entries, err := os.ReadDir(runConfigDir)
	if err != nil || len(entries) < 5 {
		t.Fatalf("expected at least 5 run configurations, found: %d", len(entries))
	}

	// 4. Verify External Tools
	extToolsPath := filepath.Join(tempProject, ".idea", "tools", "External_Tools.xml")
	if _, err := os.Stat(extToolsPath); err != nil {
		t.Fatalf("expected External_Tools.xml to exist: %v", err)
	}

	// 5. Test idempotence: running again updates existing files without errors
	res2, err := jetbrains.SetupJetBrainsToolset(opts)
	if err != nil {
		t.Fatalf("second run failed: %v", err)
	}
	if len(res2.FilesUpdated) == 0 {
		t.Fatal("expected updated files on re-run, got 0")
	}
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

func TestSetupJetBrainsToolset_BranchesAndErrors(t *testing.T) {
	// 1. Empty ProjectDir uses os.Getwd() (chdir into TempDir first)
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd failed: %v", err)
	}
	tmpWd := t.TempDir()
	if err := os.Chdir(tmpWd); err != nil {
		t.Fatalf("Chdir failed: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWd) })

	// Also test PATH without dendrite so defaultBinaryPath falls back to "dendrite"
	t.Setenv("PATH", t.TempDir())
	res, err := jetbrains.SetupJetBrainsToolset(jetbrains.SetupOptions{
		ConfigureProjectMCP: true,
	})
	if err != nil || len(res.FilesCreated) != 1 {
		t.Fatalf("SetupJetBrainsToolset with empty ProjectDir failed: %v, res=%+v", err, res)
	}

	// 2. Test PATH with an executable named dendrite so exec.LookPath succeeds
	binDir := t.TempDir()
	dendriteBin := filepath.Join(binDir, "dendrite")
	if err := os.WriteFile(dendriteBin, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatalf("write dendrite bin failed: %v", err)
	}
	t.Setenv("PATH", binDir)
	proj2 := t.TempDir()
	if _, err := jetbrains.SetupJetBrainsToolset(jetbrains.SetupOptions{
		ProjectDir:          proj2,
		ConfigureProjectMCP: true,
	}); err != nil {
		t.Fatalf("setup with dendrite on PATH failed: %v", err)
	}

	// 3. ConfigureGlobalMCP without GlobalAIPathOverride (uses HOME from t.Setenv)
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)
	if _, err := jetbrains.SetupJetBrainsToolset(jetbrains.SetupOptions{
		ProjectDir:         proj2,
		ConfigureGlobalMCP: true,
	}); err != nil {
		t.Fatalf("ConfigureGlobalMCP with fake HOME failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(fakeHome, ".ai", "mcp", "mcp.json")); err != nil {
		t.Errorf("expected global mcp.json in fakeHome: %v", err)
	}

	// 4. ConfigureGlobalMCP when HOME is empty -> os.UserHomeDir() error
	t.Setenv("HOME", "")
	if _, err := jetbrains.SetupJetBrainsToolset(jetbrains.SetupOptions{
		ProjectDir:         proj2,
		ConfigureGlobalMCP: true,
	}); err == nil {
		t.Errorf("expected error when HOME is empty and GlobalAIPathOverride is unset")
	}
	t.Setenv("HOME", fakeHome)

	// 5. ConfigureProjectMCP error (when .ai is a regular file)
	projBadMCP := t.TempDir()
	_ = os.WriteFile(filepath.Join(projBadMCP, ".ai"), []byte("file"), 0644)
	if _, err := jetbrains.SetupJetBrainsToolset(jetbrains.SetupOptions{
		ProjectDir:          projBadMCP,
		ConfigureProjectMCP: true,
	}); err == nil {
		t.Errorf("expected error when .ai is a file")
	}

	// 6. ConfigureGlobalMCP error (when GlobalAIPathOverride is a directory)
	if _, err := jetbrains.SetupJetBrainsToolset(jetbrains.SetupOptions{
		ProjectDir:           proj2,
		ConfigureGlobalMCP:   true,
		GlobalAIPathOverride: t.TempDir(),
	}); err == nil {
		t.Errorf("expected error when GlobalAIPathOverride is a directory")
	}

	// 7. ConfigureRunConfigs MkdirAll error (.idea is a file)
	projBadIdea := t.TempDir()
	_ = os.WriteFile(filepath.Join(projBadIdea, ".idea"), []byte("file"), 0644)
	if _, err := jetbrains.SetupJetBrainsToolset(jetbrains.SetupOptions{
		ProjectDir:          projBadIdea,
		ConfigureRunConfigs: true,
	}); err == nil {
		t.Errorf("expected error when .idea is a file for RunConfigs")
	}

	// 8. ConfigureRunConfigs WriteFile error (runConfigurations is read-only)
	projRORun := t.TempDir()
	roRunDir := filepath.Join(projRORun, ".idea", "runConfigurations")
	_ = os.MkdirAll(roRunDir, 0755)
	_ = os.Chmod(roRunDir, 0555)
	defer func() { _ = os.Chmod(roRunDir, 0755) }()
	if _, err := jetbrains.SetupJetBrainsToolset(jetbrains.SetupOptions{
		ProjectDir:          projRORun,
		ConfigureRunConfigs: true,
	}); err == nil {
		t.Errorf("expected error when runConfigurations dir is read-only")
	}

	// 9. ConfigureExternalTools MkdirAll error (.idea is a file)
	if _, err := jetbrains.SetupJetBrainsToolset(jetbrains.SetupOptions{
		ProjectDir:             projBadIdea,
		ConfigureExternalTools: true,
	}); err == nil {
		t.Errorf("expected error when .idea is a file for ExternalTools")
	}

	// 10. ConfigureExternalTools WriteFile error (tools dir is read-only)
	projROTools := t.TempDir()
	roToolsDir := filepath.Join(projROTools, ".idea", "tools")
	_ = os.MkdirAll(roToolsDir, 0755)
	_ = os.Chmod(roToolsDir, 0555)
	defer func() { _ = os.Chmod(roToolsDir, 0755) }()
	if _, err := jetbrains.SetupJetBrainsToolset(jetbrains.SetupOptions{
		ProjectDir:             projROTools,
		ConfigureExternalTools: true,
	}); err == nil {
		t.Errorf("expected error when tools dir is read-only")
	}
}
