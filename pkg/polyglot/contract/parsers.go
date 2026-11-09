package contract

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var typeAliases = map[string]string{
	"bytes": "bytes", "byte[]": "bytes", "vec<u8>": "bytes",
	"int": "int32", "int32": "int32", "i32": "int32", "integer": "int32",
	"int64": "int64", "i64": "int64", "long": "int64",
	"float": "float32", "float32": "float32", "f32": "float32",
	"double": "float64", "float64": "float64", "f64": "float64", "number": "float64",
	"string": "string", "str": "string", "text": "string",
	"bool": "bool", "boolean": "bool",
}

var (
	arrayPrefixes = []string{"repeated ", "[]", "vec<", "list[", "array<"}
	mapPrefixes   = []string{"map<", "map[", "dict[", "hashmap<"}
	// wrapperAffixes maps an optional wrapper prefix to its closing suffix.
	wrapperAffixes = [][2]string{{"option<", ">"}, {"optional[", "]"}}
)

func hasAnyPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

// unwrapOptional returns the inner type of Option<T> / Optional[T].
func unwrapOptional(s string) (string, bool) {
	for _, w := range wrapperAffixes {
		if strings.HasPrefix(s, w[0]) && strings.HasSuffix(s, w[1]) {
			return s[len(w[0]) : len(s)-len(w[1])], true
		}
	}
	return "", false
}

// NormalizeType standardizes type representations across Protobuf, OpenAPI, Go, Rust, TS, and Python.
func NormalizeType(raw string) string {
	cleaned := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(raw)), "*") // Pointer in Go

	if cleaned == "bytes" || cleaned == "byte[]" || cleaned == "vec<u8>" {
		return "bytes"
	}
	if hasAnyPrefix(cleaned, arrayPrefixes) || strings.HasSuffix(cleaned, "[]") {
		return "array"
	}
	if hasAnyPrefix(cleaned, mapPrefixes) {
		return "map"
	}
	if inner, ok := unwrapOptional(cleaned); ok {
		return NormalizeType(inner)
	}
	if canonical, ok := typeAliases[cleaned]; ok {
		return canonical
	}
	return cleaned
}

var (
	protoMsgRegex   = regexp.MustCompile(`^\s*message\s+([A-Za-z0-9_]+)\s*\{`)
	protoFieldRegex = regexp.MustCompile(`^\s*(optional\s+|repeated\s+)?([A-Za-z0-9_<>]+)\s+([A-Za-z0-9_]+)\s*=\s*\d+;`)
)

func protoField(line string, lineNo int) (string, FieldDefinition, bool) {
	m := protoFieldRegex.FindStringSubmatch(line)
	if len(m) <= 3 {
		return "", FieldDefinition{}, false
	}
	modifier, rawType, name := strings.TrimSpace(m[1]), m[2], m[3]
	if modifier == "repeated" {
		rawType = "repeated " + rawType
	}
	return name, newField(name, rawType, modifier == "optional", name, lineNo), true
}

// ParseProtobuf extracts messages and field definitions from .proto files.
func ParseProtobuf(content, filename string) ([]ContractSchema, error) {
	return blockParser{"//", ContractSchema{Type: ContractProtobuf, SourceFile: filename}, submatchBegin(protoMsgRegex, 1), closeBrace, protoField}.parse(content), nil
}

func extractRequiredSet(m map[string]interface{}) map[string]bool {
	reqMap := make(map[string]bool)
	if requiredList, hasReq := m["required"].([]interface{}); hasReq {
		for _, r := range requiredList {
			if rStr, ok := r.(string); ok {
				reqMap[rStr] = true
			}
		}
	}
	return reqMap
}

// ParseJSONSchema extracts object schemas from JSON Schema definitions.
func ParseJSONSchema(content, filename string) ([]ContractSchema, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidSchemaContent, err)
	}

	title, _ := raw["title"].(string)
	if title == "" {
		title = "RootSchema"
	}

	props, ok := raw["properties"].(map[string]interface{})
	if !ok {
		return nil, nil
	}

	reqMap := extractRequiredSet(raw)

	fields := make(map[string]FieldDefinition)
	for propName, propVal := range props {
		propMap, _ := propVal.(map[string]interface{})
		pType, _ := propMap["type"].(string)

		fields[propName] = newField(propName, pType, !reqMap[propName], propName, 0)
	}

	return []ContractSchema{
		{
			Name:       title,
			Type:       ContractJSONSchema,
			SourceFile: filename,
			Fields:     fields,
		},
	}, nil
}

// openAPIRawType resolves an OpenAPI property's type, mapping integer formats to int32/int64.
func openAPIRawType(prop map[string]interface{}) string {
	rawT, _ := prop["type"].(string)
	if rawT != "integer" {
		return rawT
	}
	if format, _ := prop["format"].(string); format == "int64" {
		return "int64"
	}
	return "int32"
}

// ParseOpenAPI extracts component schemas from OpenAPI 3.x specifications (JSON/YAML).
func ParseOpenAPI(content, filename string) ([]ContractSchema, error) {
	var root map[string]interface{}
	if err := yaml.Unmarshal([]byte(content), &root); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidSchemaContent, err)
	}

	components, ok := root["components"].(map[string]interface{})
	if !ok {
		return nil, nil
	}

	schemasObj, ok := components["schemas"].(map[string]interface{})
	if !ok {
		return nil, nil
	}

	var results []ContractSchema
	for schemaName, schemaVal := range schemasObj {
		sMap, _ := schemaVal.(map[string]interface{})
		props, _ := sMap["properties"].(map[string]interface{})

		reqMap := extractRequiredSet(sMap)

		fields := make(map[string]FieldDefinition)
		for pName, pVal := range props {
			pMap, _ := pVal.(map[string]interface{})
			fields[pName] = newField(pName, openAPIRawType(pMap), !reqMap[pName], pName, 0)
		}

		results = append(results, ContractSchema{
			Name:       schemaName,
			Type:       ContractOpenAPI,
			SourceFile: filename,
			Fields:     fields,
		})
	}

	return results, nil
}
