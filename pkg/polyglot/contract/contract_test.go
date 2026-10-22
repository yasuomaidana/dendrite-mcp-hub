package contract_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/yasuomaidana/dendrite-mcp-hub/pkg/jetbrains"
	"github.com/yasuomaidana/dendrite-mcp-hub/pkg/polyglot/contract"
)

func TestDriftDetection(t *testing.T) {
	canonical := contract.ContractSchema{
		Name: "User",
		Fields: map[string]contract.FieldDefinition{
			"user_id": {Name: "user_id", NormalizedType: "string", SerializationTag: "user_id", Optional: false},
			"age":     {Name: "age", NormalizedType: "int32", SerializationTag: "age", Optional: false},
			"email":   {Name: "email", NormalizedType: "string", SerializationTag: "email", Optional: true},
		},
	}

	// Model with:
	// 1. Missing field 'email'
	// 2. Type mismatch for 'age' (string instead of int32)
	// 3. Tag mismatch for 'user_id' (serializes as 'userId')
	driftingModel := contract.ContractSchema{
		Name:     "User",
		Language: contract.LangRust,
		Fields: map[string]contract.FieldDefinition{
			"user_id": {Name: "id", NormalizedType: "string", SerializationTag: "userId", Optional: false},
			"age":     {Name: "age", NormalizedType: "string", SerializationTag: "age", Optional: false},
		},
	}

	violations := contract.CompareSchemas(canonical, driftingModel)
	if len(violations) < 3 {
		t.Fatalf("expected at least 3 violations, got %d: %+v", len(violations), violations)
	}

	vTypes := make(map[contract.DriftType]bool)
	for _, v := range violations {
		vTypes[v.Type] = true
	}

	if !vTypes[contract.DriftFieldMissing] {
		t.Errorf("missing expected DriftFieldMissing")
	}
	if !vTypes[contract.DriftTypeMismatch] {
		t.Errorf("missing expected DriftTypeMismatch")
	}
	if !vTypes[contract.DriftTagMismatch] {
		t.Errorf("missing expected DriftTagMismatch")
	}

	report := contract.CompareAll([]contract.ContractSchema{canonical}, []contract.ContractSchema{driftingModel})
	if !report.HasErrors || report.TotalViolations == 0 {
		t.Errorf("expected report to register errors")
	}
}

func TestContractMonitor_EndToEnd(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "monitor_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	schemaDir := filepath.Join(tmpDir, "proto")
	goDir := filepath.Join(tmpDir, "backend")
	_ = os.MkdirAll(schemaDir, 0755)
	_ = os.MkdirAll(goDir, 0755)

	protoFile := filepath.Join(schemaDir, "service.proto")
	protoData := `message Account { string account_id = 1; int64 balance = 2; }`
	_ = os.WriteFile(protoFile, []byte(protoData), 0644)

	goFile := filepath.Join(goDir, "account.go")
	goData := "package main\ntype Account struct {\n\tAccountID string `json:\"account_id\"`\n\tBalance int64 `json:\"balance\"`\n}\n"
	_ = os.WriteFile(goFile, []byte(goData), 0644)

	monitor := contract.NewContractMonitor(nil)

	// Clean match
	report, err := monitor.CheckDrift([]string{schemaDir}, map[contract.TargetLanguage]string{
		contract.LangGo: goDir,
	})
	if err != nil {
		t.Fatalf("check drift failed: %v", err)
	}
	if report.TotalViolations != 0 {
		t.Errorf("expected 0 violations for clean match, got %d: %+v", report.TotalViolations, report.Violations)
	}

	// Missing schemas error
	_, err = monitor.ScanSchemas([]string{filepath.Join(tmpDir, "missing")})
	if !errors.Is(err, contract.ErrNoSchemasFound) {
		t.Errorf("expected ErrNoSchemasFound, got %v", err)
	}

	// Test NotifyIDE with nil bridge
	if err := monitor.NotifyIDE(context.Background(), report); err != nil {
		t.Errorf("unexpected error on nil bridge notify: %v", err)
	}

	// Test NotifyIDE with mock bridge
	fakeClient, _ := jetbrains.NewClient(nil)
	fakeBridge := jetbrains.NewBridge(fakeClient)
	fakeMonitor := contract.NewContractMonitor(fakeBridge)
	_ = fakeMonitor.NotifyIDE(context.Background(), &contract.DriftReport{TotalViolations: 1})
}

func TestContractMonitor_QuadLanguageScan(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "monitor_quad_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	schemaDir := filepath.Join(tmpDir, "schemas")
	_ = os.MkdirAll(schemaDir, 0755)

	// Add JSON schema
	jsonSchema := filepath.Join(schemaDir, "item.json")
	_ = os.WriteFile(jsonSchema, []byte(`{"title":"Item","type":"object","properties":{"id":{"type":"string"}}}`), 0644)

	// Add OpenAPI YAML
	yamlSchema := filepath.Join(schemaDir, "api.yaml")
	_ = os.WriteFile(yamlSchema, []byte("openapi: 3.0.0\ncomponents:\n  schemas:\n    Widget:\n      type: object\n      properties:\n        name:\n          type: string\n"), 0644)

	// Add OpenAPI JSON
	openAPIJson := filepath.Join(schemaDir, "openapi.json")
	_ = os.WriteFile(openAPIJson, []byte(`{"openapi":"3.0.0","components":{"schemas":{"Gadget":{"type":"object","properties":{"cost":{"type":"number"}}}}}}`), 0644)

	// Create directories for all 4 IDE languages
	rustDir := filepath.Join(tmpDir, "rust")
	tsDir := filepath.Join(tmpDir, "ts")
	pyDir := filepath.Join(tmpDir, "python")
	_ = os.MkdirAll(rustDir, 0755)
	_ = os.MkdirAll(tsDir, 0755)
	_ = os.MkdirAll(pyDir, 0755)

	_ = os.WriteFile(filepath.Join(rustDir, "item.rs"), []byte("pub struct Item {\n    pub id: String,\n}"), 0644)
	_ = os.WriteFile(filepath.Join(tsDir, "widget.ts"), []byte("export interface Widget {\n    name: string;\n}"), 0644)
	_ = os.WriteFile(filepath.Join(pyDir, "gadget.py"), []byte("class Gadget(BaseModel):\n    cost: float\n"), 0644)

	monitor := contract.NewContractMonitor(nil)

	// Scan all schemas
	schemas, err := monitor.ScanSchemas([]string{schemaDir})
	if err != nil {
		t.Fatalf("ScanSchemas failed: %v", err)
	}
	if len(schemas) < 3 {
		t.Errorf("expected at least 3 schemas, got %d", len(schemas))
	}

	// Scan all code models
	models, err := monitor.ScanCodeModels(map[contract.TargetLanguage]string{
		contract.LangRust:       rustDir,
		contract.LangTypeScript: tsDir,
		contract.LangPython:     pyDir,
	})
	if err != nil {
		t.Fatalf("ScanCodeModels failed: %v", err)
	}
	if len(models) < 3 {
		t.Errorf("expected at least 3 code models, got %d", len(models))
	}

	report, err := monitor.CheckDrift([]string{schemaDir}, map[contract.TargetLanguage]string{
		contract.LangRust:       rustDir,
		contract.LangTypeScript: tsDir,
		contract.LangPython:     pyDir,
	})
	if err != nil {
		t.Fatalf("CheckDrift failed: %v", err)
	}
	if report.TotalViolations != 0 {
		for _, v := range report.Violations {
			t.Logf("Violation: %+v", v)
		}
		t.Errorf("expected 0 violations across clean quad-language models, got %d", report.TotalViolations)
	}

	// Test optional mismatch
	can := contract.ContractSchema{
		Name: "Req",
		Fields: map[string]contract.FieldDefinition{
			"f": {Name: "f", NormalizedType: "string", Optional: false},
		},
	}
	opt := contract.ContractSchema{
		Name: "Req",
		Fields: map[string]contract.FieldDefinition{
			"f": {Name: "f", NormalizedType: "string", Optional: true},
		},
	}
	vList := contract.CompareSchemas(can, opt)
	if len(vList) != 1 || vList[0].Type != contract.DriftOptionalError {
		t.Errorf("expected DriftOptionalError, got %+v", vList)
	}
}

func TestContractMonitor_NotifyIDEWithHTTPTransport(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req jetbrains.JSONRPCRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		res := jetbrains.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  json.RawMessage(`{"protocolVersion":"2024-11-05","serverInfo":{"name":"JetBrains","version":"2024.3"},"content":[{"type":"text","text":"ok"}]}`),
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(res)
	}))
	defer server.Close()

	transport := jetbrains.NewHTTPTransport(server.URL, server.Client())
	client, err := jetbrains.NewClient(transport)
	if err != nil {
		t.Fatalf("failed creating client: %v", err)
	}
	_, _ = client.Connect(context.Background())
	bridge := jetbrains.NewBridge(client)
	monitor := contract.NewContractMonitor(bridge)

	report := &contract.DriftReport{
		TotalViolations: 2,
		HasErrors:       true,
		Violations: []contract.DriftViolation{
			{SchemaName: "User", FieldName: "id", Message: "missing", TargetFile: "user.go", Line: 10},
		},
	}

	if err := monitor.NotifyIDE(context.Background(), report); err != nil {
		t.Errorf("expected clean NotifyIDE with http transport: %v", err)
	}
}
