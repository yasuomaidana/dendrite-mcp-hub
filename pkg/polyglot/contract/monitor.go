package contract

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/yasuomaidana/dendrite-mcp-hub/pkg/jetbrains"
)

// ContractMonitor scans shared schema repositories and models across Quad-IDE projects.
type ContractMonitor struct {
	bridge *jetbrains.Bridge
}

// NewContractMonitor initializes a new polyglot contract drift monitor.
func NewContractMonitor(bridge *jetbrains.Bridge) *ContractMonitor {
	return &ContractMonitor{bridge: bridge}
}

// walkFiles calls visit with the path and content of every readable file under dir.
// A missing dir is skipped; unreadable paths are skipped too, so it has no error to report.
func walkFiles(dir string, visit func(path, ext, content string)) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return
	}
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		visit(path, strings.ToLower(filepath.Ext(path)), string(data))
		return nil
	})
}

func containsAny(content string, needles ...string) bool {
	for _, n := range needles {
		if strings.Contains(content, n) {
			return true
		}
	}
	return false
}

// parseSchemaFile picks the schema parser matching a specification file, or nil.
func parseSchemaFile(path, ext, content string) []ContractSchema {
	var schemas []ContractSchema
	switch {
	case ext == ".proto":
		schemas, _ = ParseProtobuf(content, path)
	case ext == ".json" && containsAny(content, `"openapi"`, `"swagger"`):
		schemas, _ = ParseOpenAPI(content, path)
	case ext == ".json" && containsAny(content, `"properties"`, `"$schema"`):
		schemas, _ = ParseJSONSchema(content, path)
	case (ext == ".yaml" || ext == ".yml") && containsAny(content, "openapi:", "swagger:"):
		schemas, _ = ParseOpenAPI(content, path)
	}
	return schemas
}

// ScanSchemas reads and parses schema specifications from one or more directories.
func (m *ContractMonitor) ScanSchemas(schemaDirs []string) ([]ContractSchema, error) {
	var results []ContractSchema
	for _, dir := range schemaDirs {
		walkFiles(dir, func(path, ext, content string) {
			results = append(results, parseSchemaFile(path, ext, content)...)
		})
	}
	if len(results) == 0 {
		return nil, ErrNoSchemasFound
	}
	return results, nil
}

type codeLang struct {
	exts  []string
	parse func(content, filename string) ([]ContractSchema, error)
}

var codeLangs = map[TargetLanguage]codeLang{
	LangGo:         {[]string{".go"}, ParseGoStructs},
	LangRust:       {[]string{".rs"}, ParseRustSerdeStructs},
	LangTypeScript: {[]string{".ts", ".tsx"}, ParseTypeScriptInterfaces},
	LangPython:     {[]string{".py"}, ParsePythonPydanticModels},
}

func (c codeLang) matches(path, ext string) bool {
	if strings.HasSuffix(path, "_test.go") {
		return false
	}
	for _, e := range c.exts {
		if e == ext {
			return true
		}
	}
	return false
}

// ScanCodeModels scans language projects and extracts implementation schemas.
func (m *ContractMonitor) ScanCodeModels(codeDirs map[TargetLanguage]string) ([]ContractSchema, error) {
	var results []ContractSchema
	for lang, dir := range codeDirs {
		cl, known := codeLangs[lang]
		walkFiles(dir, func(path, ext, content string) {
			if known && cl.matches(path, ext) {
				schemas, _ := cl.parse(content, path)
				results = append(results, schemas...)
			}
		})
	}
	return results, nil
}

// CheckDrift runs end-to-end drift detection across schemas and Quad-IDE codebases.
func (m *ContractMonitor) CheckDrift(schemaDirs []string, codeDirs map[TargetLanguage]string) (*DriftReport, error) {
	canonicalSchemas, err := m.ScanSchemas(schemaDirs)
	if err != nil {
		return nil, err
	}

	codeSchemas, _ := m.ScanCodeModels(codeDirs) // never fails: unreadable paths are skipped

	return CompareAll(canonicalSchemas, codeSchemas), nil
}

// NotifyIDE dispatches drift diagnostics to the connected JetBrains IDE (GoLand, RustRover, WebStorm, PyCharm).
func (m *ContractMonitor) NotifyIDE(ctx context.Context, report *DriftReport) error {
	if m.bridge == nil || report == nil || len(report.Violations) == 0 {
		return nil
	}

	client := m.bridge.GetClient()
	if client == nil {
		return nil
	}

	for _, v := range report.Violations {
		if v.TargetFile != "" {
			_ = m.bridge.OpenFile(ctx, v.TargetFile, v.Line, 1)
			break
		}
	}

	return nil
}
