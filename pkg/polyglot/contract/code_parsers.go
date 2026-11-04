package contract

import (
	"regexp"
	"strings"
)

// ParseGoStructs parses Go source files and extracts struct fields and `json:"..."` tags.
func ParseGoStructs(content, filename string) ([]ContractSchema, error) {
	lines := strings.Split(content, "\n")
	var schemas []ContractSchema
	var currentSchema *ContractSchema

	structRegex := regexp.MustCompile(`^\s*type\s+([A-Za-z0-9_]+)\s+struct\s*\{`)
	fieldRegex := regexp.MustCompile(`^\s*([A-Za-z0-9_]+)\s+([*]?[A-Za-z0-9_\[\]]+)(\s+` + "`" + `([^` + "`" + `]+)` + "`" + `)?`)
	jsonTagRegex := regexp.MustCompile(`json:"([^",]+)`)

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			continue
		}

		if matches := structRegex.FindStringSubmatch(line); len(matches) > 1 {
			if currentSchema != nil {
				schemas = append(schemas, *currentSchema)
			}
			currentSchema = &ContractSchema{
				Name:       matches[1],
				Language:   LangGo,
				SourceFile: filename,
				Fields:     make(map[string]FieldDefinition),
			}
			continue
		}

		if currentSchema != nil && trimmed == "}" {
			schemas = append(schemas, *currentSchema)
			currentSchema = nil
			continue
		}

		if currentSchema != nil {
			if fMatches := fieldRegex.FindStringSubmatch(line); len(fMatches) > 2 {
				name := fMatches[1]
				rawType := fMatches[2]
				tagContent := fMatches[4]

				tag := name
				if jMatches := jsonTagRegex.FindStringSubmatch(tagContent); len(jMatches) > 1 {
					tag = jMatches[1]
				}

				isOptional := strings.HasPrefix(rawType, "*") || strings.Contains(tagContent, "omitempty")

				currentSchema.Fields[tag] = FieldDefinition{
					Name:             name,
					NormalizedType:   NormalizeType(rawType),
					RawType:          rawType,
					Optional:         isOptional,
					SerializationTag: tag,
					LineNumber:       i + 1,
				}
			}
		}
	}

	if currentSchema != nil {
		schemas = append(schemas, *currentSchema)
	}

	return schemas, nil
}

// ParseRustSerdeStructs parses Rust structs and extracts serde tags and types.
func ParseRustSerdeStructs(content, filename string) ([]ContractSchema, error) {
	lines := strings.Split(content, "\n")
	var schemas []ContractSchema
	var currentSchema *ContractSchema

	structRegex := regexp.MustCompile(`^\s*(pub\s+)?struct\s+([A-Za-z0-9_]+)\s*\{`)
	fieldRegex := regexp.MustCompile(`^\s*(pub\s+)?([A-Za-z0-9_]+)\s*:\s*([^,;]+)`)
	serdeRenameRegex := regexp.MustCompile(`#\[serde\(rename\s*=\s*"([^"]+)"\)\]`)

	pendingRename := ""

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			continue
		}

		if sMatches := serdeRenameRegex.FindStringSubmatch(line); len(sMatches) > 1 {
			pendingRename = sMatches[1]
			continue
		}

		if matches := structRegex.FindStringSubmatch(line); len(matches) > 2 {
			if currentSchema != nil {
				schemas = append(schemas, *currentSchema)
			}
			currentSchema = &ContractSchema{
				Name:       matches[2],
				Language:   LangRust,
				SourceFile: filename,
				Fields:     make(map[string]FieldDefinition),
			}
			continue
		}

		if currentSchema != nil && trimmed == "}" {
			schemas = append(schemas, *currentSchema)
			currentSchema = nil
			continue
		}

		if currentSchema != nil {
			if fMatches := fieldRegex.FindStringSubmatch(line); len(fMatches) > 3 {
				fieldName := fMatches[2]
				rawType := strings.TrimSpace(fMatches[3])

				tag := fieldName
				if pendingRename != "" {
					tag = pendingRename
					pendingRename = ""
				}

				isOptional := strings.HasPrefix(rawType, "Option<")

				currentSchema.Fields[tag] = FieldDefinition{
					Name:             fieldName,
					NormalizedType:   NormalizeType(rawType),
					RawType:          rawType,
					Optional:         isOptional,
					SerializationTag: tag,
					LineNumber:       i + 1,
				}
			}
		}
	}

	if currentSchema != nil {
		schemas = append(schemas, *currentSchema)
	}

	return schemas, nil
}

// ParseTypeScriptInterfaces extracts interface fields from TypeScript files.
func ParseTypeScriptInterfaces(content, filename string) ([]ContractSchema, error) {
	lines := strings.Split(content, "\n")
	var schemas []ContractSchema
	var currentSchema *ContractSchema

	interfaceRegex := regexp.MustCompile(`^\s*(export\s+)?interface\s+([A-Za-z0-9_]+)\s*\{`)
	fieldRegex := regexp.MustCompile(`^\s*([A-Za-z0-9_]+)(\?)?:\s*([^;]+);?`)

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			continue
		}

		if matches := interfaceRegex.FindStringSubmatch(line); len(matches) > 2 {
			if currentSchema != nil {
				schemas = append(schemas, *currentSchema)
			}
			currentSchema = &ContractSchema{
				Name:       matches[2],
				Language:   LangTypeScript,
				SourceFile: filename,
				Fields:     make(map[string]FieldDefinition),
			}
			continue
		}

		if currentSchema != nil && trimmed == "}" {
			schemas = append(schemas, *currentSchema)
			currentSchema = nil
			continue
		}

		if currentSchema != nil {
			if fMatches := fieldRegex.FindStringSubmatch(line); len(fMatches) > 3 {
				name := fMatches[1]
				isOptional := fMatches[2] == "?"
				rawType := strings.TrimSpace(fMatches[3])

				currentSchema.Fields[name] = FieldDefinition{
					Name:             name,
					NormalizedType:   NormalizeType(rawType),
					RawType:          rawType,
					Optional:         isOptional,
					SerializationTag: name,
					LineNumber:       i + 1,
				}
			}
		}
	}

	if currentSchema != nil {
		schemas = append(schemas, *currentSchema)
	}

	return schemas, nil
}

// ParsePythonPydanticModels extracts Pydantic class models from Python files.
func ParsePythonPydanticModels(content, filename string) ([]ContractSchema, error) {
	lines := strings.Split(content, "\n")
	var schemas []ContractSchema
	var currentSchema *ContractSchema

	classRegex := regexp.MustCompile(`^\s*class\s+([A-Za-z0-9_]+)\((?:BaseModel|Schema)\):\s*`)
	fieldRegex := regexp.MustCompile(`^\s*([A-Za-z0-9_]+)\s*:\s*([^=]+)(?:=\s*Field\([^)]*alias="([^"]+)"[^)]*\))?`)

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		if matches := classRegex.FindStringSubmatch(line); len(matches) > 1 {
			if currentSchema != nil {
				schemas = append(schemas, *currentSchema)
			}
			currentSchema = &ContractSchema{
				Name:       matches[1],
				Language:   LangPython,
				SourceFile: filename,
				Fields:     make(map[string]FieldDefinition),
			}
			continue
		}

		if currentSchema != nil && len(line) > 0 && line[0] != ' ' && line[0] != '\t' && trimmed != "" {
			schemas = append(schemas, *currentSchema)
			currentSchema = nil
		}

		if currentSchema != nil {
			if fMatches := fieldRegex.FindStringSubmatch(line); len(fMatches) > 2 {
				name := fMatches[1]
				rawType := strings.TrimSpace(fMatches[2])
				alias := ""
				if len(fMatches) > 3 {
					alias = fMatches[3]
				}

				tag := name
				if alias != "" {
					tag = alias
				}

				isOptional := strings.HasPrefix(rawType, "Optional[")

				currentSchema.Fields[tag] = FieldDefinition{
					Name:             name,
					NormalizedType:   NormalizeType(rawType),
					RawType:          rawType,
					Optional:         isOptional,
					SerializationTag: tag,
					LineNumber:       i + 1,
				}
			}
		}
	}

	if currentSchema != nil {
		schemas = append(schemas, *currentSchema)
	}

	return schemas, nil
}
