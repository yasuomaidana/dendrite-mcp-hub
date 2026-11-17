package mcpserver

// TypedProp is a JSON-schema property of the given type with a description.
func TypedProp(typ, description string) map[string]interface{} {
	return map[string]interface{}{"type": typ, "description": description}
}

// ListProp is a JSON-schema array property whose items have the given type.
func ListProp(itemType, description string) map[string]interface{} {
	prop := TypedProp("array", description)
	prop["items"] = map[string]interface{}{"type": itemType}
	return prop
}

// PropsSchema is an object schema with the given properties and no required keys.
func PropsSchema(props map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{"type": "object", "properties": props}
}

// ObjectSchema is an object schema with the given properties and required keys.
func ObjectSchema(required []string, props map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{"type": "object", "properties": props, "required": required}
}

// AddTool registers a tool from its name, description, input schema and handler.
func AddTool(s *Server, name, description string, schema map[string]interface{}, handler ToolHandler) {
	s.RegisterTool(Tool{Name: name, Description: description, InputSchema: schema}, handler)
}
