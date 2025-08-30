package jsonschema

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

type SchemaMarshaler struct{}

func (m *SchemaMarshaler) MarshalToJSONText(schema JSONSchema) (string, error) {
	jsonSchemaBytes, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JsonSchema struct to JSON: %w", err)
	}

	return string(jsonSchemaBytes), nil
}

func (m *SchemaMarshaler) ParseSchemaFromInterface(schemaInput interface{}) (JSONSchema, error) {
	if schemaInput == nil {
		return JSONSchema{}, nil
	}

	switch v := schemaInput.(type) {
	case string:
		// Handle string input - could be JSON or YAML
		return m.parseStringSchema(v)
	case map[string]interface{}:
		// Handle YAML object - convert to JSONSchema struct
		return m.convertMapToJSONSchema(v)
	case JSONSchema:
		// Already a JSONSchema struct
		return v, nil
	default:
		return JSONSchema{}, fmt.Errorf("unsupported schema type: %T", v)
	}
}

func (m *SchemaMarshaler) parseStringSchema(schemaStr string) (JSONSchema, error) {
	// Use YAML parser to handle both JSON and YAML (since YAML is a superset of JSON)
	var schemaMap map[string]interface{}
	if err := yaml.Unmarshal([]byte(schemaStr), &schemaMap); err != nil {
		return JSONSchema{}, fmt.Errorf("failed to parse schema string as YAML/JSON: %w", err)
	}

	// Convert map to JSONSchema struct
	return m.convertMapToJSONSchema(schemaMap)
}

func (m *SchemaMarshaler) convertMapToJSONSchema(schemaMap map[string]interface{}) (JSONSchema, error) {
	// Convert the map to JSON bytes and then unmarshal to JSONSchema struct
	jsonBytes, err := json.Marshal(schemaMap)
	if err != nil {
		return JSONSchema{}, fmt.Errorf("failed to marshal schema map to JSON: %w", err)
	}

	var schema JSONSchema
	err = json.Unmarshal(jsonBytes, &schema)
	if err != nil {
		return JSONSchema{}, fmt.Errorf("failed to unmarshal JSON to JSONSchema struct: %w", err)
	}

	return schema, nil
}
