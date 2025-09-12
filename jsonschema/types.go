package jsonschema

import (
	"encoding/json"
	"fmt"
	"slices"

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
