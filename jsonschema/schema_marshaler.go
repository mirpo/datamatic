package jsonschema

import (
	"encoding/json"
	"fmt"
)

type SchemaMarshaler struct{}

func NewSchemaMarshaler() *SchemaMarshaler {
	return &SchemaMarshaler{}
}

func (m *SchemaMarshaler) MarshalToJSONText(schema JSONSchema) (string, error) {
	jsonSchemaBytes, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JsonSchema struct to JSON: %w", err)
	}

	return string(jsonSchemaBytes), nil
}
