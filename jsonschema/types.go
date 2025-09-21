package jsonschema

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/kaptinlin/jsonschema"
)

type Schema struct {
	schema   *jsonschema.Schema
	jsonText string
}

func LoadSchema(v interface{}) (*Schema, error) {
	if v == nil {
		return nil, nil
	}

	var (
		jsonBytes []byte
		err       error
	)

	switch val := v.(type) {
	case string:
		var rawData map[string]interface{}
		if err = json.Unmarshal([]byte(val), &rawData); err != nil {
			return nil, fmt.Errorf("invalid JSON: %w", err)
		}
		jsonBytes = []byte(val)
	case map[string]interface{}:
		if jsonBytes, err = json.Marshal(val); err != nil {
			return nil, fmt.Errorf("failed to marshal map: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported schema type: %T", v)
	}

	compiler := jsonschema.NewCompiler()
	schema, err := compiler.Compile(jsonBytes)
	if err != nil {
		return nil, fmt.Errorf("invalid JSON schema: %w", err)
	}

	jsonText, err := json.MarshalIndent(schema, "", " ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema to JSON: %w", err)
	}

	return &Schema{
		schema:   schema,
		jsonText: string(jsonText),
	}, nil
}

// HasSchemaDefinition checks if the schema has a valid definition
func (s *Schema) HasSchemaDefinition() bool {
	return s != nil && s.schema != nil
}

// EnsureAllPropertiesRequired checks if all properties are required and all required are in properties
func (s *Schema) EnsureAllPropertiesRequired() error {
	if s.schema.Properties == nil {
		return fmt.Errorf("schema has no properties defined")
	}

	props := *s.schema.Properties
	required := s.schema.Required

	propSet := make(map[string]struct{}, len(props))
	for name := range props {
		propSet[name] = struct{}{}
	}
	reqSet := make(map[string]struct{}, len(required))
	for _, name := range required {
		reqSet[name] = struct{}{}
	}

	// every property is required
	for name := range propSet {
		if _, ok := reqSet[name]; !ok {
			return fmt.Errorf("property %q is not marked as required", name)
		}
	}

	// every required exists in properties
	for name := range reqSet {
		if _, ok := propSet[name]; !ok {
			return fmt.Errorf("required field %q not defined in properties", name)
		}
	}

	return nil
}

// HasRequiredProperty checks if a specific property is defined as required
func (s *Schema) HasRequiredProperty(key string) bool {
	return slices.Contains(s.schema.Required, key)
}

// ToJSONString returns the pre-computed JSON string representation of the schema
func (s *Schema) ToJSONString() string {
	return s.jsonText
}

// ValidateJSONText validates a JSON text string against the schema
func (s *Schema) ValidateJSONText(input string) error {
	var data interface{}
	if err := json.Unmarshal([]byte(input), &data); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	result := s.schema.Validate(data)
	if !result.IsValid() {
		errors := []string{}
		for _, err := range result.Errors {
			errors = append(errors, err.Error())
		}

		return &ValidationError{Errors: errors}
	}

	return nil
}

// GetSchema returns the compiled schema for API usage
func (s *Schema) GetSchema() *jsonschema.Schema {
	return s.schema
}

// HasFieldPath checks if a field path exists in the schema.
func (s *Schema) HasFieldPath(path string) bool {
	if !s.HasSchemaDefinition() || path == "" {
		return false
	}

	current := s.schema
	for _, part := range strings.Split(path, ".") {
		if current.Properties == nil {
			return false
		}
		prop, ok := (*current.Properties)[part]
		if !ok {
			return false
		}
		current = prop
	}
	return true
}

// extractFieldByPath extracts a field from JSON data using a dot path.
func extractFieldByPath(data interface{}, path string) (interface{}, error) {
	if path == "" {
		return nil, fmt.Errorf("path cannot be empty")
	}

	current := data
	parts := strings.Split(path, ".")

	for i, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("cannot traverse field '%s' on non-object type %T at path '%s'", part, current, strings.Join(parts[:i], "."))
		}

		val, ok := m[part]
		if !ok {
			return nil, fmt.Errorf("field '%s' not found at path '%s'", part, strings.Join(parts[:i+1], "."))
		}

		current = val
	}

	return current, nil
}

// ExtractFieldByPathAsString extracts a field and converts it to string.
func ExtractFieldByPathAsString(data interface{}, path string) (string, error) {
	val, err := extractFieldByPath(data, path)
	if err != nil {
		return "", err
	}
	return convertToString(val), nil
}

// convertToString converts values into a test-friendly string form.
func convertToString(value interface{}) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case bool:
		if v {
			return "true"
		}
		return "false"
	case float64:
		// Make ints look like ints, not floats
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%g", v)
	case int, int64:
		return fmt.Sprintf("%d", v)
	case []interface{}:
		parts := make([]string, len(v))
		for i, item := range v {
			parts[i] = convertToString(item)
		}
		return strings.Join(parts, ", ")
	case map[string]interface{}:
		// Marshal to compact JSON
		if b, err := json.Marshal(v); err == nil {
			return string(b)
		}
		return fmt.Sprintf("%v", v)
	default:
		// Fallback: try JSON, else fmt
		if b, err := json.Marshal(v); err == nil {
			return string(b)
		}
		return fmt.Sprint(v)
	}
}
