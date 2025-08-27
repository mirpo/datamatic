package jsonschema

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResponseValidator_ValidateJSONText(t *testing.T) {
	validator := NewResponseValidator()

	schema := JSONSchema{
		Type: "object",
		Properties: map[string]Property{
			"name": {Type: "string"},
			"age":  {Type: "number"},
		},
		Required: []string{"name"},
	}

	tests := []struct {
		name      string
		jsonInput string
		wantError bool
	}{
		{
			name:      "valid JSON",
			jsonInput: `{"name": "John", "age": 30}`,
			wantError: false,
		},
		{
			name:      "missing required field",
			jsonInput: `{"age": 30}`,
			wantError: true,
		},
		{
			name:      "invalid JSON",
			jsonInput: `{invalid}`,
			wantError: true,
		},
		{
			name:      "wrong type",
			jsonInput: `{"name": 123, "age": 30}`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateJSONText(schema, tt.jsonInput)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestResponseValidator_ValidateEnums(t *testing.T) {
	validator := NewResponseValidator()

	schema := JSONSchema{
		Type: "object",
		Properties: map[string]Property{
			"status": {
				Type: "string",
				Enum: []interface{}{"active", "inactive"},
			},
		},
		Required: []string{"status"},
	}

	tests := []struct {
		name      string
		jsonInput string
		wantError bool
	}{
		{
			name:      "valid enum value",
			jsonInput: `{"status": "active"}`,
			wantError: false,
		},
		{
			name:      "invalid enum value",
			jsonInput: `{"status": "pending"}`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateJSONText(schema, tt.jsonInput)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
