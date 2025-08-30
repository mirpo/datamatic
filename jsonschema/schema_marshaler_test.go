package jsonschema

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSchemaMarshaler_MarshalToJSONText(t *testing.T) {
	marshaler := &SchemaMarshaler{}

	schema := JSONSchema{
		Type: "object",
		Properties: map[string]Property{
			"name": {Type: "string"},
			"age":  {Type: "number"},
		},
		Required: []string{"name"},
	}

	result, err := marshaler.MarshalToJSONText(schema)
	assert.NoError(t, err)

	var unmarshaled JSONSchema
	err = json.Unmarshal([]byte(result), &unmarshaled)
	assert.NoError(t, err)

	assert.Equal(t, schema.Type, unmarshaled.Type)
	assert.Equal(t, len(schema.Properties), len(unmarshaled.Properties))
	assert.Equal(t, len(schema.Required), len(unmarshaled.Required))
}

func TestSchemaMarshaler_ParseSchemaFromInterface(t *testing.T) {
	marshaler := &SchemaMarshaler{}

	tests := []struct {
		name        string
		input       interface{}
		expected    JSONSchema
		expectError bool
	}{
		{
			name:        "nil input",
			input:       nil,
			expected:    JSONSchema{},
			expectError: false,
		},
		{
			name: "JSONSchema struct passthrough",
			input: JSONSchema{
				Type: "object",
				Properties: map[string]Property{
					"name": {Type: "string"},
				},
				Required: []string{"name"},
			},
			expected: JSONSchema{
				Type: "object",
				Properties: map[string]Property{
					"name": {Type: "string"},
				},
				Required: []string{"name"},
			},
			expectError: false,
		},
		{
			name: "valid JSON string",
			input: `{
				"type": "object",
				"properties": {
					"name": {"type": "string"},
					"age": {"type": "integer", "minimum": 0, "maximum": 150}
				},
				"required": ["name", "age"]
			}`,
			expected: JSONSchema{
				Type: "object",
				Properties: map[string]Property{
					"name": {Type: "string"},
					"age":  {Type: "integer", Minimum: floatPtr(0), Maximum: floatPtr(150)},
				},
				Required: []string{"name", "age"},
			},
			expectError: false,
		},
		{
			name: "valid YAML string",
			input: `type: object
properties:
  name:
    type: string
  age:
    type: integer
    minimum: 18
    maximum: 100
required:
  - name
  - age`,
			expected: JSONSchema{
				Type: "object",
				Properties: map[string]Property{
					"name": {Type: "string"},
					"age":  {Type: "integer", Minimum: floatPtr(18), Maximum: floatPtr(100)},
				},
				Required: []string{"name", "age"},
			},
			expectError: false,
		},
		{
			name: "YAML object (map)",
			input: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"title": map[string]interface{}{
						"type": "string",
					},
				},
				"required": []interface{}{"title"},
			},
			expected: JSONSchema{
				Type: "object",
				Properties: map[string]Property{
					"title": {Type: "string"},
				},
				Required: []string{"title"},
			},
			expectError: false,
		},
		{
			name: "complex nested schema",
			input: `{
				"type": "object",
				"properties": {
					"user": {
						"type": "object",
						"properties": {
							"name": {"type": "string"},
							"tags": {
								"type": "array",
								"items": {"type": "string"}
							}
						},
						"required": ["name"]
					}
				},
				"required": ["user"]
			}`,
			expected: JSONSchema{
				Type: "object",
				Properties: map[string]Property{
					"user": {
						Type: "object",
						Properties: map[string]Property{
							"name": {Type: "string"},
							"tags": {
								Type:  "array",
								Items: &Property{Type: "string"},
							},
						},
						Required: []string{"name"},
					},
				},
				Required: []string{"user"},
			},
			expectError: false,
		},
		{
			name:        "invalid JSON/YAML string",
			input:       `{invalid json`,
			expected:    JSONSchema{},
			expectError: true,
		},
		{
			name:        "unsupported type",
			input:       42,
			expected:    JSONSchema{},
			expectError: true,
		},
		{
			name:        "empty string",
			input:       "",
			expected:    JSONSchema{},
			expectError: false, // Empty string parses to empty schema
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := marshaler.ParseSchemaFromInterface(tt.input)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expected.Type, result.Type)
			assert.Equal(t, len(tt.expected.Properties), len(result.Properties))
			assert.Equal(t, len(tt.expected.Required), len(result.Required))

			// Check properties match
			for key, expectedProp := range tt.expected.Properties {
				actualProp, exists := result.Properties[key]
				assert.True(t, exists, "Property %s should exist", key)
				assert.Equal(t, expectedProp.Type, actualProp.Type)

				if expectedProp.Minimum != nil {
					assert.Equal(t, *expectedProp.Minimum, *actualProp.Minimum)
				}
				if expectedProp.Maximum != nil {
					assert.Equal(t, *expectedProp.Maximum, *actualProp.Maximum)
				}

				// Check nested properties and items
				if expectedProp.Properties != nil {
					assert.Equal(t, len(expectedProp.Properties), len(actualProp.Properties))
				}
				if expectedProp.Items != nil {
					assert.NotNil(t, actualProp.Items)
					assert.Equal(t, expectedProp.Items.Type, actualProp.Items.Type)
				}
			}

			// Check required fields match
			assert.ElementsMatch(t, tt.expected.Required, result.Required)
		})
	}
}

// Helper function for test readability
func floatPtr(f float64) *float64 {
	return &f
}
