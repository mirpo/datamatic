package jsonschema

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigValidator_HasSchemaDefinition(t *testing.T) {
	validator := NewConfigValidator()

	tests := []struct {
		name     string
		schema   JSONSchema
		expected bool
	}{
		{
			name:     "empty schema",
			schema:   JSONSchema{},
			expected: false,
		},
		{
			name: "schema with properties",
			schema: JSONSchema{
				Properties: map[string]Property{
					"name": {Type: "string"},
				},
			},
			expected: true,
		},
		{
			name: "schema with required",
			schema: JSONSchema{
				Required: []string{"name"},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.HasSchemaDefinition(tt.schema)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfigValidator_ValidateRequiredProperties(t *testing.T) {
	validator := NewConfigValidator()

	tests := []struct {
		name     string
		schema   JSONSchema
		expected bool
	}{
		{
			name:     "empty schema",
			schema:   JSONSchema{},
			expected: false,
		},
		{
			name: "valid schema",
			schema: JSONSchema{
				Properties: map[string]Property{
					"name": {Type: "string"},
				},
				Required: []string{"name"},
			},
			expected: true,
		},
		{
			name: "missing property in required",
			schema: JSONSchema{
				Properties: map[string]Property{
					"name": {Type: "string"},
				},
				Required: []string{"name", "missing"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateRequiredProperties(tt.schema)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfigValidator_HasRequiredProperty(t *testing.T) {
	validator := NewConfigValidator()
	schema := JSONSchema{
		Properties: map[string]Property{
			"name": {Type: "string"},
			"age":  {Type: "number"},
		},
		Required: []string{"name"},
	}

	tests := []struct {
		name     string
		property string
		expected bool
	}{
		{"existing required property", "name", true},
		{"existing non-required property", "age", false},
		{"non-existing property", "missing", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.HasRequiredProperty(schema, tt.property)
			assert.Equal(t, tt.expected, result)
		})
	}
}
