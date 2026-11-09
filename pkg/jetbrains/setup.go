package jetbrains

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// MCPConfig represents the standard JetBrains .ai/mcp/mcp.json structure.
type MCPConfig struct {
	MCPServers map[string]MCPServerEntry `json:"mcpServers"`
}

// MCPServerEntry represents a single MCP server configuration.
type MCPServerEntry struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env,omitempty"`
}

// SetupOptions controls what gets configured for JetBrains.
type SetupOptions struct {
	ProjectDir             string
	BinaryPath             string
	ConfigureProjectMCP    bool
	ConfigureGlobalMCP     bool
	ConfigureRunConfigs    bool
	ConfigureExternalTools bool
	GlobalAIPathOverride   string // Optional override for tests
	JetBrainsHomeOverride  string // Optional override for tests
}

// SetupResult captures the files generated or updated.
type SetupResult struct {
	FilesCreated []string `json:"files_created"`
	FilesUpdated []string `json:"files_updated"`
	ToolsAdded   []string `json:"tools_added"`
}

// getwd and absPath resolve the project directory; tests replace them to make resolution fail.
var (
	getwd   = os.Getwd
	absPath = filepath.Abs
)

// record notes path as created or updated depending on whether it existed before.
func (r *SetupResult) record(path string, existed bool) {
	if existed {
		r.FilesUpdated = append(r.FilesUpdated, path)
	} else {
		r.FilesCreated = append(r.FilesCreated, path)
	}
}

// resolveProjectDir returns the absolute project directory, defaulting to the working directory.
func resolveProjectDir(dir string) (string, error) {
	if dir == "" {
		cwd, err := getwd()
		if err != nil {
			return "", fmt.Errorf("failed to determine project directory: %w", err)
		}
		dir = cwd
	}
	abs, err := absPath(dir)
	if err != nil {
		return "", fmt.Errorf("invalid project directory path: %w", err)
	}
	return abs, nil
}

// globalMCPPath is where the application-level MCP configuration lives.
func globalMCPPath(override string) (string, error) {
	if override != "" {
		return override, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to determine user home directory: %w", err)
	}
	return filepath.Join(home, ".ai", "mcp", "mcp.json"), nil
}

// configureGlobalMCP configures the global / application-level MCP file (~/.ai/mcp/mcp.json).
func configureGlobalMCP(opts SetupOptions, res *SetupResult) error {
	path, err := globalMCPPath(opts.GlobalAIPathOverride)
	if err != nil {
		return err
	}
	if err := configureMCPFile(path, opts.BinaryPath, res); err != nil {
		return fmt.Errorf("failed to configure global MCP: %w", err)
	}
	return nil
}

// writeIDEFile writes content (trimmed, newline-terminated) and records it in res.
func writeIDEFile(path, content string, res *SetupResult) error {
	existed := fileExists(path)
	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)+"\n"), 0644); err != nil {
		return err
	}
	res.record(path, existed)
	return nil
}

// writeRunConfigs generates the JetBrains Run Configurations (.idea/runConfigurations/).
func writeRunConfigs(projectDir string, res *SetupResult) error {
	dir := filepath.Join(projectDir, ".idea", "runConfigurations")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create runConfigurations dir: %w", err)
	}
	for name, xmlContent := range getRunConfigurationTemplates() {
		if err := writeIDEFile(filepath.Join(dir, name), xmlContent, res); err != nil {
			return fmt.Errorf("failed to write run configuration %s: %w", name, err)
		}
	}
	return nil
}

// writeExternalTools generates the JetBrains External Tools (.idea/tools/External_Tools.xml).
func writeExternalTools(projectDir string, res *SetupResult) error {
	dir := filepath.Join(projectDir, ".idea", "tools")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create tools dir: %w", err)
	}
	if err := writeIDEFile(filepath.Join(dir, "External_Tools.xml"), getExternalToolsXML(), res); err != nil {
		return fmt.Errorf("failed to write external tools file: %w", err)
	}
	res.ToolsAdded = append(res.ToolsAdded,
		"Dendrite: Run Local Agent Triage on Active File",
		"Dendrite: Run Aider on Active File",
		"Dendrite: Run Antigravity on Active File",
		"Dendrite: Probe MCP Diagnostics",
		"Dendrite: List Registered MCP Tools",
		"Dendrite: Telemetry Snapshot",
	)
	return nil
}

// SetupJetBrainsToolset configures JetBrains IDEs to natively recognize Dendrite MCP and autonomous agents.
func SetupJetBrainsToolset(opts SetupOptions) (*SetupResult, error) {
	projectDir, err := resolveProjectDir(opts.ProjectDir)
	if err != nil {
		return nil, err
	}
	if opts.BinaryPath == "" {
		opts.BinaryPath = defaultBinaryPath()
	}

	res := &SetupResult{
		FilesCreated: []string{},
		FilesUpdated: []string{},
		ToolsAdded:   []string{},
	}

	steps := []struct {
		enabled bool
		run     func() error
	}{
		{opts.ConfigureProjectMCP, func() error {
			if err := configureMCPFile(filepath.Join(projectDir, ".ai", "mcp", "mcp.json"), opts.BinaryPath, res); err != nil {
				return fmt.Errorf("failed to configure project MCP: %w", err)
			}
			return nil
		}},
		{opts.ConfigureGlobalMCP, func() error { return configureGlobalMCP(opts, res) }},
		{opts.ConfigureRunConfigs, func() error { return writeRunConfigs(projectDir, res) }},
		{opts.ConfigureExternalTools, func() error { return writeExternalTools(projectDir, res) }},
	}
	for _, step := range steps {
		if !step.enabled {
			continue
		}
		if err := step.run(); err != nil {
			return nil, err
		}
	}

	return res, nil
}

func configureMCPFile(targetPath, binaryPath string, res *SetupResult) error {
	entry := MCPServerEntry{
		Command: binaryPath,
		Args:    []string{"inter", "serve"},
	}

	created, err := UpsertMCPServer(targetPath, "dendrite", entry)
	if err != nil {
		return err
	}

	res.record(targetPath, !created)
	return nil
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func getRunConfigurationTemplates() map[string]string {
	return map[string]string{
		"Dendrite_Local_Agent_Triage.xml": `<component name="ProjectRunConfigurationManager">
  <configuration default="false" name="Dendrite: Local Agent Triage" type="GoApplicationRunConfiguration" factoryName="Go Application">
    <module name="Dendrite" />
    <working_directory value="$PROJECT_DIR$" />
    <parameters value="inhibitory-triage &quot;Evaluate task complexity, select skills, and optimize token spend&quot;" />
    <kind value="PACKAGE" />
    <package value="Dendrite/cmd/dendrite" />
    <directory value="$PROJECT_DIR$" />
    <filePath value="$PROJECT_DIR$/cmd/dendrite/main.go" />
    <method v="2" />
  </configuration>
</component>`,
		"Dendrite_MCP_Server.xml": `<component name="ProjectRunConfigurationManager">
  <configuration default="false" name="Dendrite: MCP Server" type="GoApplicationRunConfiguration" factoryName="Go Application">
    <module name="Dendrite" />
    <working_directory value="$PROJECT_DIR$" />
    <parameters value="inter serve" />
    <kind value="PACKAGE" />
    <package value="Dendrite/cmd/dendrite" />
    <directory value="$PROJECT_DIR$" />
    <filePath value="$PROJECT_DIR$/cmd/dendrite/main.go" />
    <method v="2" />
  </configuration>
</component>`,
		"Dendrite_Agent_Aider.xml": `<component name="ProjectRunConfigurationManager">
  <configuration default="false" name="Dendrite: Agent Runner (Aider)" type="GoApplicationRunConfiguration" factoryName="Go Application">
    <module name="Dendrite" />
    <working_directory value="$PROJECT_DIR$" />
    <parameters value="bouton run aider --ticket DEN-050 --task &quot;Autonomous implementation loop&quot;" />
    <kind value="PACKAGE" />
    <package value="Dendrite/cmd/dendrite" />
    <directory value="$PROJECT_DIR$" />
    <filePath value="$PROJECT_DIR$/cmd/dendrite/main.go" />
    <method v="2" />
  </configuration>
</component>`,
		"Dendrite_Agent_Antigravity.xml": `<component name="ProjectRunConfigurationManager">
  <configuration default="false" name="Dendrite: Agent Runner (Antigravity)" type="GoApplicationRunConfiguration" factoryName="Go Application">
    <module name="Dendrite" />
    <working_directory value="$PROJECT_DIR$" />
    <parameters value="bouton run agy --task &quot;Strategic architecture synthesis&quot;" />
    <kind value="PACKAGE" />
    <package value="Dendrite/cmd/dendrite" />
    <directory value="$PROJECT_DIR$" />
    <filePath value="$PROJECT_DIR$/cmd/dendrite/main.go" />
    <method v="2" />
  </configuration>
</component>`,
		"Dendrite_Web_UI.xml": `<component name="ProjectRunConfigurationManager">
  <configuration default="false" name="Dendrite: Web UI &amp; Dashboard" type="GoApplicationRunConfiguration" factoryName="Go Application">
    <module name="Dendrite" />
    <working_directory value="$PROJECT_DIR$" />
    <parameters value="cortex serve --port 3456" />
    <kind value="PACKAGE" />
    <package value="Dendrite/cmd/dendrite" />
    <directory value="$PROJECT_DIR$" />
    <filePath value="$PROJECT_DIR$/cmd/dendrite/main.go" />
    <method v="2" />
  </configuration>
</component>`,
		"Dendrite_Doctor.xml": `<component name="ProjectRunConfigurationManager">
  <configuration default="false" name="Dendrite: Pre-flight Doctor" type="GoApplicationRunConfiguration" factoryName="Go Application">
    <module name="Dendrite" />
    <working_directory value="$PROJECT_DIR$" />
    <parameters value="soma diagnose" />
    <kind value="PACKAGE" />
    <package value="Dendrite/cmd/dendrite" />
    <directory value="$PROJECT_DIR$" />
    <filePath value="$PROJECT_DIR$/cmd/dendrite/main.go" />
    <method v="2" />
  </configuration>
</component>`,
		"Dendrite_JetBrains_Probe.xml": `<component name="ProjectRunConfigurationManager">
  <configuration default="false" name="Dendrite: Probe JetBrains MCP" type="GoApplicationRunConfiguration" factoryName="Go Application">
    <module name="Dendrite" />
    <working_directory value="$PROJECT_DIR$" />
    <parameters value="bridge probe" />
    <kind value="PACKAGE" />
    <package value="Dendrite/cmd/dendrite" />
    <directory value="$PROJECT_DIR$" />
    <filePath value="$PROJECT_DIR$/cmd/dendrite/main.go" />
    <method v="2" />
  </configuration>
</component>`,
	}
}

func getExternalToolsXML() string {
	return `<toolSet name="External Tools">
  <tool name="Dendrite: Run Local Agent Triage on Active File" showInMainMenu="true" showInEditor="true" showInProject="true" showInSearchPopup="true" disabled="false" useConsole="true" showConsoleOnStdOut="true" showConsoleOnStdErr="true" synchronizeAfterRun="false">
    <exec>
      <option name="COMMAND" value="$ProjectFileDir$/bin/dendrite" />
      <option name="PARAMETERS" value="inhibitory-triage --file $FilePath$ &quot;Optimize context and triage task&quot;" />
      <option name="WORKING_DIRECTORY" value="$ProjectFileDir$" />
    </exec>
  </tool>
  <tool name="Dendrite: Run Aider on Active File" showInMainMenu="true" showInEditor="true" showInProject="true" showInSearchPopup="true" disabled="false" useConsole="true" showConsoleOnStdOut="true" showConsoleOnStdErr="true" synchronizeAfterRun="true">
    <exec>
      <option name="COMMAND" value="$ProjectFileDir$/bin/dendrite" />
      <option name="PARAMETERS" value="bouton run aider --task &quot;Refactor and improve $FileName$&quot;" />
      <option name="WORKING_DIRECTORY" value="$ProjectFileDir$" />
    </exec>
  </tool>
  <tool name="Dendrite: Run Antigravity on Active File" showInMainMenu="true" showInEditor="true" showInProject="true" showInSearchPopup="true" disabled="false" useConsole="true" showConsoleOnStdOut="true" showConsoleOnStdErr="true" synchronizeAfterRun="true">
    <exec>
      <option name="COMMAND" value="$ProjectFileDir$/bin/dendrite" />
      <option name="PARAMETERS" value="bouton run agy --task &quot;Analyze architecture and specs for $FilePath$&quot;" />
      <option name="WORKING_DIRECTORY" value="$ProjectFileDir$" />
    </exec>
  </tool>
  <tool name="Dendrite: Probe MCP Diagnostics" showInMainMenu="true" showInEditor="true" showInProject="true" showInSearchPopup="true" disabled="false" useConsole="true" showConsoleOnStdOut="true" showConsoleOnStdErr="true" synchronizeAfterRun="true">
    <exec>
      <option name="COMMAND" value="$ProjectFileDir$/bin/dendrite" />
      <option name="PARAMETERS" value="bridge problems $FilePath$" />
      <option name="WORKING_DIRECTORY" value="$ProjectFileDir$" />
    </exec>
  </tool>
  <tool name="Dendrite: List Registered MCP Tools" showInMainMenu="true" showInEditor="true" showInProject="true" showInSearchPopup="true" disabled="false" useConsole="true" showConsoleOnStdOut="true" showConsoleOnStdErr="true" synchronizeAfterRun="false">
    <exec>
      <option name="COMMAND" value="$ProjectFileDir$/bin/dendrite" />
      <option name="PARAMETERS" value="inter tools" />
      <option name="WORKING_DIRECTORY" value="$ProjectFileDir$" />
    </exec>
  </tool>
  <tool name="Dendrite: Telemetry Snapshot" showInMainMenu="true" showInEditor="true" showInProject="true" showInSearchPopup="true" disabled="false" useConsole="true" showConsoleOnStdOut="true" showConsoleOnStdErr="true" synchronizeAfterRun="false">
    <exec>
      <option name="COMMAND" value="$ProjectFileDir$/bin/dendrite" />
      <option name="PARAMETERS" value="sensory snapshot" />
      <option name="WORKING_DIRECTORY" value="$ProjectFileDir$" />
    </exec>
  </tool>
</toolSet>`
}

// defaultBinaryPath returns the dendrite binary the generated MCP configs
// should launch: the running executable, else the one on PATH, else the bare
// name. It never points inside the target project, so setup cannot make the
// project depend on (or be polluted by) a project-local bin/ directory.
// currentExecutable names the running program; tests replace it.
var currentExecutable = os.Executable

func defaultBinaryPath() string {
	if exe, err := currentExecutable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		if strings.EqualFold(strings.TrimSuffix(filepath.Base(exe), ".exe"), "dendrite") {
			return exe
		}
	}
	if p, err := exec.LookPath("dendrite"); err == nil {
		return p
	}
	return "dendrite"
}
