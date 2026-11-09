package contract

import (
	"regexp"
	"strings"
)

var (
	goStructRegex = regexp.MustCompile(`^\s*type\s+([A-Za-z0-9_]+)\s+struct\s*\{`)
	goFieldRegex  = regexp.MustCompile(`^\s*([A-Za-z0-9_]+)\s+([*]?[A-Za-z0-9_\[\]]+)(\s+` + "`" + `([^` + "`" + `]+)` + "`" + `)?`)
	goJSONTagRe   = regexp.MustCompile(`json:"([^",]+)`)

	rustStructRegex = regexp.MustCompile(`^\s*(pub\s+)?struct\s+([A-Za-z0-9_]+)\s*\{`)
	rustFieldRegex  = regexp.MustCompile(`^\s*(pub\s+)?([A-Za-z0-9_]+)\s*:\s*([^,;]+)`)
	rustRenameRegex = regexp.MustCompile(`#\[serde\(rename\s*=\s*"([^"]+)"\)\]`)

	tsInterfaceRegex = regexp.MustCompile(`^\s*(export\s+)?interface\s+([A-Za-z0-9_]+)\s*\{`)
	tsFieldRegex     = regexp.MustCompile(`^\s*([A-Za-z0-9_]+)(\?)?:\s*([^;]+);?`)

	pyClassRegex = regexp.MustCompile(`^\s*class\s+([A-Za-z0-9_]+)\((?:BaseModel|Schema)\):\s*`)
	pyFieldRegex = regexp.MustCompile(`^\s*([A-Za-z0-9_]+)\s*:\s*([^=]+)(?:=\s*Field\([^)]*alias="([^"]+)"[^)]*\))?`)
)

// blockParser is a line-oriented scanner for languages that declare a named
// block (struct, interface, class, message) containing one field per line.
type blockParser struct {
	comment string
	tmpl    ContractSchema
	begin   func(line string) (string, bool)
	end     func(line, trimmed string) bool
	field   func(line string, lineNo int) (string, FieldDefinition, bool)
}

func closeBrace(_, trimmed string) bool { return trimmed == "}" }

func newField(name, rawType string, optional bool, tag string, lineNo int) FieldDefinition {
	return FieldDefinition{
		Name:             name,
		NormalizedType:   NormalizeType(rawType),
		RawType:          rawType,
		Optional:         optional,
		SerializationTag: tag,
		LineNumber:       lineNo,
	}
}

func (p blockParser) parse(content string) []ContractSchema {
	var schemas []ContractSchema
	var cur *ContractSchema

	flush := func() {
		if cur != nil {
			schemas = append(schemas, *cur)
			cur = nil
		}
	}

	for i, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, p.comment) {
			continue
		}
		if name, ok := p.begin(line); ok {
			flush()
			s := p.tmpl
			s.Name = name
			s.Fields = make(map[string]FieldDefinition)
			cur = &s
			continue
		}
		if cur == nil {
			continue
		}
		if p.end(line, trimmed) {
			flush()
			continue
		}
		if key, def, ok := p.field(line, i+1); ok {
			cur.Fields[key] = def
		}
	}
	flush()
	return schemas
}

func submatchBegin(re *regexp.Regexp, group int) func(string) (string, bool) {
	return func(line string) (string, bool) {
		m := re.FindStringSubmatch(line)
		if len(m) <= group {
			return "", false
		}
		return m[group], true
	}
}

func goField(line string, lineNo int) (string, FieldDefinition, bool) {
	m := goFieldRegex.FindStringSubmatch(line)
	if len(m) <= 2 {
		return "", FieldDefinition{}, false
	}
	name, rawType, tagContent := m[1], m[2], m[4]
	tag := name
	if j := goJSONTagRe.FindStringSubmatch(tagContent); len(j) > 1 {
		tag = j[1]
	}
	optional := strings.HasPrefix(rawType, "*") || strings.Contains(tagContent, "omitempty")
	return tag, newField(name, rawType, optional, tag, lineNo), true
}

// ParseGoStructs parses Go source files and extracts struct fields and `json:"..."` tags.
func ParseGoStructs(content, filename string) ([]ContractSchema, error) {
	return blockParser{"//", ContractSchema{Language: LangGo, SourceFile: filename}, submatchBegin(goStructRegex, 1), closeBrace, goField}.parse(content), nil
}

// ParseRustSerdeStructs parses Rust structs and extracts serde tags and types.
func ParseRustSerdeStructs(content, filename string) ([]ContractSchema, error) {
	pendingRename := ""
	field := func(line string, lineNo int) (string, FieldDefinition, bool) {
		if s := rustRenameRegex.FindStringSubmatch(line); len(s) > 1 {
			pendingRename = s[1]
			return "", FieldDefinition{}, false
		}
		m := rustFieldRegex.FindStringSubmatch(line)
		if len(m) <= 3 {
			return "", FieldDefinition{}, false
		}
		name, rawType := m[2], strings.TrimSpace(m[3])
		tag := name
		if pendingRename != "" {
			tag, pendingRename = pendingRename, ""
		}
		return tag, newField(name, rawType, strings.HasPrefix(rawType, "Option<"), tag, lineNo), true
	}
	return blockParser{"//", ContractSchema{Language: LangRust, SourceFile: filename}, submatchBegin(rustStructRegex, 2), closeBrace, field}.parse(content), nil
}

func tsField(line string, lineNo int) (string, FieldDefinition, bool) {
	m := tsFieldRegex.FindStringSubmatch(line)
	if len(m) <= 3 {
		return "", FieldDefinition{}, false
	}
	name := m[1]
	return name, newField(name, strings.TrimSpace(m[3]), m[2] == "?", name, lineNo), true
}

// ParseTypeScriptInterfaces extracts interface fields from TypeScript files.
func ParseTypeScriptInterfaces(content, filename string) ([]ContractSchema, error) {
	return blockParser{"//", ContractSchema{Language: LangTypeScript, SourceFile: filename}, submatchBegin(tsInterfaceRegex, 2), closeBrace, tsField}.parse(content), nil
}

func pyField(line string, lineNo int) (string, FieldDefinition, bool) {
	m := pyFieldRegex.FindStringSubmatch(line)
	if len(m) <= 2 {
		return "", FieldDefinition{}, false
	}
	name, rawType := m[1], strings.TrimSpace(m[2])
	tag := name
	if len(m) > 3 && m[3] != "" {
		tag = m[3]
	}
	return tag, newField(name, rawType, strings.HasPrefix(rawType, "Optional["), tag, lineNo), true
}

func pyDedent(line, trimmed string) bool {
	return len(line) > 0 && line[0] != ' ' && line[0] != '\t' && trimmed != ""
}

// ParsePythonPydanticModels extracts Pydantic class models from Python files.
func ParsePythonPydanticModels(content, filename string) ([]ContractSchema, error) {
	return blockParser{"#", ContractSchema{Language: LangPython, SourceFile: filename}, submatchBegin(pyClassRegex, 1), pyDedent, pyField}.parse(content), nil
}
