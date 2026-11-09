package contract_test

import (
	"errors"
	"testing"

	"github.com/yasuomaidana/dendrite-mcp-hub/pkg/polyglot/contract"
)

func TestNormalizeType(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"int", "int32"},
		{"bytes", "bytes"},
		{"byte[]", "bytes"},
		{"Vec<u8>", "bytes"},
		{"int32", "int32"},
		{"i32", "int32"},
		{"integer", "int32"},
		{"int64", "int64"},
		{"i64", "int64"},
		{"long", "int64"},
		{"float", "float32"},
		{"float32", "float32"},
		{"f32", "float32"},
		{"double", "float64"},
		{"float64", "float64"},
		{"f64", "float64"},
		{"number", "float64"},
		{"string", "string"},
		{"str", "string"},
		{"text", "string"},
		{"bool", "bool"},
		{"boolean", "bool"},
		{"bytes", "bytes"},
		{"byte[]", "bytes"},
		{"vec<u8>", "bytes"},
		{"repeated string", "array"},
		{"[]string", "array"},
		{"vec<i32>", "array"},
		{"list[str]", "array"},
		{"array<string>", "array"},
		{"string[]", "array"},
		{"map<string, int>", "map"},
		{"map[string]int", "map"},
		{"dict[str, int]", "map"},
		{"hashmap<String, i32>", "map"},
		{"Option<String>", "string"},
		{"Optional[int]", "int32"},
		{"*string", "string"},
		{"CustomType", "customtype"},
	}

	for _, c := range cases {
		got := contract.NormalizeType(c.input)
		if got != c.want {
			t.Errorf("NormalizeType(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestParseProtobuf(t *testing.T) {
	protoContent := `
syntax = "proto3";

// User account message
message User {
  string user_id = 1;
  int32 age = 2;
  repeated string roles = 3;
  optional string email = 4;
  // nested unclosed before next message
message Unclosed {
  string x = 1;
message Order {
  int64 order_id = 1;
`
	schemas, err := contract.ParseProtobuf(protoContent, "user.proto")
	if err != nil {
		t.Fatalf("unexpected error parsing protobuf: %v", err)
	}
	if len(schemas) != 3 {
		t.Fatalf("expected 3 schemas, got %d", len(schemas))
	}

	user := schemas[0]
	if user.Name != "User" || len(user.Fields) != 4 {
		t.Errorf("unexpected User schema: %+v", user)
	}
	if user.Fields["roles"].NormalizedType != "array" {
		t.Errorf("expected roles to be array, got %s", user.Fields["roles"].NormalizedType)
	}
	if !user.Fields["email"].Optional {
		t.Errorf("expected email to be optional")
	}

	// Normal closed messages
	closedSchemas, _ := contract.ParseProtobuf("message Closed {\n  string id = 1;\n}\n", "c.proto")
	if len(closedSchemas) != 1 || closedSchemas[0].Name != "Closed" {
		t.Errorf("unexpected closedSchemas: %+v", closedSchemas)
	}
}

func TestParseJSONSchema(t *testing.T) {
	validJSON := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"title": "UserProfile",
		"type": "object",
		"required": ["id", "username"],
		"properties": {
			"id": { "type": "string" },
			"username": { "type": "string" },
			"bio": { "type": "string" }
		}
	}`

	schemas, err := contract.ParseJSONSchema(validJSON, "user.json")
	if err != nil {
		t.Fatalf("unexpected error parsing json schema: %v", err)
	}
	if len(schemas) != 1 {
		t.Fatalf("expected 1 schema, got %d", len(schemas))
	}

	up := schemas[0]
	if up.Name != "UserProfile" || len(up.Fields) != 3 {
		t.Errorf("unexpected schema: %+v", up)
	}
	if up.Fields["bio"].Optional != true {
		t.Errorf("expected bio to be optional")
	}
	if up.Fields["id"].Optional != false {
		t.Errorf("expected id to be required")
	}

}

func TestParseJSONSchema_Fallbacks(t *testing.T) {
	// Untitled schema defaults to RootSchema
	untitled, err := contract.ParseJSONSchema(`{"properties":{"x":{"type":"int"}}}`, "untitled.json")
	if err != nil || len(untitled) != 1 || untitled[0].Name != "RootSchema" {
		t.Errorf("expected RootSchema, got %+v (err=%v)", untitled, err)
	}

	// Invalid JSON
	_, err = contract.ParseJSONSchema("{bad-json", "bad.json")
	if !errors.Is(err, contract.ErrInvalidSchemaContent) {
		t.Errorf("expected ErrInvalidSchemaContent on malformed json, got %v", err)
	}

	// Schema without properties
	emptyProps, _ := contract.ParseJSONSchema(`{"title": "Empty"}`, "empty.json")
	if len(emptyProps) != 0 {
		t.Errorf("expected 0 schemas for missing properties")
	}
}

func TestParseOpenAPI(t *testing.T) {
	yamlContent := `
openapi: 3.0.0
info:
  title: Sample API
  version: 0.1.0
components:
  schemas:
    Product:
      type: object
      required:
        - sku
      properties:
        sku:
          type: string
        price:
          type: number
        stock:
          type: integer
          format: int64
        count:
          type: integer
`
	schemas, err := contract.ParseOpenAPI(yamlContent, "api.yaml")
	if err != nil {
		t.Fatalf("unexpected error parsing openapi: %v", err)
	}
	if len(schemas) != 1 {
		t.Fatalf("expected 1 schema, got %d", len(schemas))
	}

	prod := schemas[0]
	if prod.Name != "Product" || len(prod.Fields) != 4 {
		t.Errorf("unexpected Product schema: %+v", prod)
	}
	if prod.Fields["stock"].NormalizedType != "int64" {
		t.Errorf("expected stock int64, got %s", prod.Fields["stock"].NormalizedType)
	}
	if prod.Fields["count"].NormalizedType != "int32" {
		t.Errorf("expected count int32, got %s", prod.Fields["count"].NormalizedType)
	}
	if prod.Fields["price"].NormalizedType != "float64" {
		t.Errorf("expected price float64, got %s", prod.Fields["price"].NormalizedType)
	}

}

func TestParseOpenAPI_Fallbacks(t *testing.T) {
	// Missing components or schemas
	noComp, _ := contract.ParseOpenAPI("openapi: 3.0.0\n", "nocomp.yaml")
	if len(noComp) != 0 {
		t.Errorf("expected 0 schemas when components missing")
	}
	noSchemas, _ := contract.ParseOpenAPI("openapi: 3.0.0\ncomponents:\n  responses: {}\n", "noschemas.yaml")
	if len(noSchemas) != 0 {
		t.Errorf("expected 0 schemas when components.schemas missing")
	}

	// Invalid YAML
	_, err := contract.ParseOpenAPI(": bad yaml [", "bad.yaml")
	if !errors.Is(err, contract.ErrInvalidSchemaContent) {
		t.Errorf("expected ErrInvalidSchemaContent, got %v", err)
	}
}

func TestParseGoModels(t *testing.T) {
	// 1. Go Struct (including unclosed struct before next struct and unclosed at EOF)
	goCode := `
package models

// User account
type User struct {
	// comment inside struct
	ID    string   ` + "`json:\"user_id\"`" + `
	Age   int      ` + "`json:\"age\"`" + `
	Roles []string ` + "`json:\"roles\"`" + `
	Email *string  ` + "`json:\"email,omitempty\"`" + `
}

type UnclosedGo struct {
	A string
type EOFGo struct {
	B int
`
	goSchemas, err := contract.ParseGoStructs(goCode, "user.go")
	if err != nil || len(goSchemas) != 3 {
		t.Fatalf("failed parsing Go struct: %v, schemas=%+v", err, goSchemas)
	}
	if goSchemas[0].Fields["user_id"].NormalizedType != "string" {
		t.Errorf("unexpected Go user_id type")
	}
	if !goSchemas[0].Fields["email"].Optional {
		t.Errorf("expected pointer/omitempty to be optional")
	}

}

func TestParseRustModels(t *testing.T) {
	// 2. Rust Struct (including comment, unclosed before next, unclosed at EOF)
	rsCode := `
#[derive(Serialize, Deserialize)]
pub struct User {
    // comment in rust struct
    #[serde(rename = "user_id")]
    pub id: String,
    pub age: i32,
    pub roles: Vec<String>,
    pub email: Option<String>,
}

struct UnclosedRust {
    a: i32,
struct EOFRust {
    b: String,
`
	rsSchemas, err := contract.ParseRustSerdeStructs(rsCode, "user.rs")
	if err != nil || len(rsSchemas) != 3 {
		t.Fatalf("failed parsing Rust struct: %v", err)
	}
	if rsSchemas[0].Fields["user_id"].SerializationTag != "user_id" {
		t.Errorf("expected serde rename 'user_id', got %s", rsSchemas[0].Fields["user_id"].SerializationTag)
	}
	if !rsSchemas[0].Fields["email"].Optional {
		t.Errorf("expected Option<String> to be optional")
	}

}

func TestParseTypeScriptModels(t *testing.T) {
	// 3. TypeScript Interface (including comment, unclosed before next, unclosed at EOF)
	tsCode := `
export interface User {
  // comment in ts interface
  user_id: string;
  age: number;
  roles: string[];
  email?: string;
}

interface UnclosedTS {
  a: string;
interface EOFTS {
  b: number;
`
	tsSchemas, err := contract.ParseTypeScriptInterfaces(tsCode, "user.ts")
	if err != nil || len(tsSchemas) != 3 {
		t.Fatalf("failed parsing TS interface: %v", err)
	}
	if tsSchemas[0].Fields["age"].NormalizedType != "float64" {
		t.Errorf("expected number to normalize to float64")
	}
	if !tsSchemas[0].Fields["email"].Optional {
		t.Errorf("expected ? to be optional")
	}

}

func TestParsePythonModels(t *testing.T) {
	// 4. Python Pydantic Model (including comment, back-to-back classes, top-level statement after class)
	pyCode := `
from pydantic import BaseModel, Field
from typing import Optional, List

# top level comment
class First(BaseModel):
    # field comment
    x: int
class User(BaseModel):
    id: str = Field(alias="user_id")
    age: int
    roles: List[str]
    email: Optional[str]

END_CONST = 1
`
	pySchemas, err := contract.ParsePythonPydanticModels(pyCode, "user.py")
	if err != nil || len(pySchemas) != 2 {
		t.Fatalf("failed parsing Python model: %v, len=%d", err, len(pySchemas))
	}
	if pySchemas[1].Fields["user_id"].SerializationTag != "user_id" {
		t.Errorf("expected alias 'user_id', got %s", pySchemas[1].Fields["user_id"].SerializationTag)
	}
	if !pySchemas[1].Fields["email"].Optional {
		t.Errorf("expected Optional[str] to be optional")
	}

	// Python class at EOF without trailing top-level line
	pyEOF, _ := contract.ParsePythonPydanticModels("class Tail(BaseModel):\n    id: str\n", "tail.py")
	if len(pyEOF) != 1 || pyEOF[0].Name != "Tail" {
		t.Errorf("expected Tail schema at EOF, got %+v", pyEOF)
	}
}
