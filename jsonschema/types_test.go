package jsonschema

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadSchema(t *testing.T) {
	tests := []struct {
		name        string
		input       interface{}
		wantErr     bool
		errContains string
	}{
		{
			name:    "nil input",
			input:   nil,
			wantErr: false,
		},
		{
			name:    "valid JSON string",
			input:   `{"type": "object", "properties": {"name": {"type": "string"}}}`,
			wantErr: false,
		},
		{
			name:        "invalid JSON string",
			input:       `{"type": "object", "properties":}`,
			wantErr:     true,
			errContains: "invalid JSON",
		},
		{
			name: "valid map object",
			input: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{"type": "string"},
					"age":  map[string]interface{}{"type": "integer"},
				},
			},
			wantErr: false,
		},
		{
			name:        "unsupported type",
			input:       42,
			wantErr:     true,
			errContains: "unsupported schema type",
		},
		{
			name:        "unmarshalable map",
			input:       map[string]interface{}{"key": make(chan int)},
			wantErr:     true,
			errContains: "failed to marshal map",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, err := LoadSchema(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, schema)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			assert.NoError(t, err)
			if tt.input == nil {
				assert.Nil(t, schema)
			} else {
				assert.NotNil(t, schema)
			}
		})
	}
}

func TestEnsureAllPropertiesRequired(t *testing.T) {
	tests := []struct {
		name        string
		schema      *Schema
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid schema with matching required properties",
			schema: func() *Schema {
				s, _ := LoadSchema(map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"name": map[string]interface{}{"type": "string"},
						"age":  map[string]interface{}{"type": "integer"},
					},
					"required": []interface{}{"name", "age"},
				})
				return s
			}(),
			expectError: false,
		},
		{
			name: "invalid schema with no required fields",
			schema: func() *Schema {
				s, _ := LoadSchema(map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"name": map[string]interface{}{"type": "string"},
					},
				})
				return s
			}(),
			expectError: true,
			errorMsg:    "property \"name\" is not marked as required",
		},
		{
			name: "invalid schema with no properties",
			schema: func() *Schema {
				s, _ := LoadSchema(map[string]interface{}{
					"type":     "object",
					"required": []interface{}{"name"},
				})
				return s
			}(),
			expectError: true,
			errorMsg:    "schema has no properties defined",
		},
		{
			name: "invalid schema with required field not in properties",
			schema: func() *Schema {
				s, _ := LoadSchema(map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"name": map[string]interface{}{"type": "string"},
					},
					"required": []interface{}{"name", "age"},
				})
				return s
			}(),
			expectError: true,
			errorMsg:    "required field \"age\" not defined in properties",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.schema.EnsureAllPropertiesRequired()
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestHasRequiredProperty(t *testing.T) {
	schema, err := LoadSchema(map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{"type": "string"},
			"age":  map[string]interface{}{"type": "integer"},
		},
		"required": []interface{}{"name"},
	})
	assert.NoError(t, err)
	assert.NotNil(t, schema)

	assert.True(t, schema.HasRequiredProperty("name"))
	assert.False(t, schema.HasRequiredProperty("age"))
	assert.False(t, schema.HasRequiredProperty("email"))
}

func TestToJSONString(t *testing.T) {
	// Test with real schema using LoadSchema
	schemaJSON := `{
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer"}
		},
		"required": ["name"]
	}`
	schema, err := LoadSchema(schemaJSON)
	assert.NoError(t, err)
	assert.NotNil(t, schema)

	result := schema.ToJSONString()

	assert.Contains(t, result, `{`)
	assert.Contains(t, result, ` "properties": {`)
	assert.Contains(t, result, `  "name": {`)
	assert.Contains(t, result, `   "type": "string"`)
	assert.Contains(t, result, `  },`)
	assert.Contains(t, result, `  "age": {`)
	assert.Contains(t, result, `   "type": "integer"`)
	assert.Contains(t, result, ` },`)
	assert.Contains(t, result, `"type": "object"`)
	assert.Contains(t, result, ` "required": [`)
	assert.Contains(t, result, `  "name"`)
	assert.Contains(t, result, ` ]`)
	assert.Contains(t, result, `}`)
}

func TestValidateJSONText(t *testing.T) {
	schemaJSON := `{
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer"},
			"email": {"type": "string"}
		},
		"required": ["name", "age", "email"]
	}`
	schema, err := LoadSchema(schemaJSON)
	assert.NoError(t, err)
	assert.NotNil(t, schema)

	// no errors
	validJSON := `{"name": "John", "age": 30, "email": "john@example.com"}`
	err = schema.ValidateJSONText(validJSON)
	assert.NoError(t, err)

	// one error
	singleErrorJSON := `{"name": "John", "age": "thirty", "email": "john@example.com"}`
	err = schema.ValidateJSONText(singleErrorJSON)
	assert.Error(t, err)

	var validationErr *ValidationError
	assert.True(t, errors.As(err, &validationErr))
	assert.Equal(t, "validation failed: Property 'age' does not match the schema", err.Error())

	// multiple errors
	multiErrorJSON := `{"age": "thirty"}`
	err = schema.ValidateJSONText(multiErrorJSON)
	assert.Error(t, err)

	assert.True(t, errors.As(err, &validationErr))
	assert.Len(t, validationErr.Errors, 2)

	// Check formatted error message for multiple errors
	errorMsg := err.Error()
	assert.Contains(t, errorMsg, "validation failed with 2 errors:")
	assert.Contains(t, errorMsg, "- ")
	assert.Contains(t, errorMsg, "Required properties 'name', 'email' are missing")
	assert.Contains(t, errorMsg, "Properties 'age', 'email', 'name' do not match their schemas")
}

func TestHasFieldPath(t *testing.T) {
	schema, err := LoadSchema(map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{"type": "string"},
			"user": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"profile": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"age":  map[string]interface{}{"type": "integer"},
							"city": map[string]interface{}{"type": "string"},
						},
					},
					"settings": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"theme": map[string]interface{}{"type": "string"},
						},
					},
				},
			},
			"tags": map[string]interface{}{
				"type":  "array",
				"items": map[string]interface{}{"type": "string"},
			},
		},
	})
	assert.NoError(t, err)
	assert.NotNil(t, schema)

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"Top-level field", "name", true},
		{"Top-level array field", "tags", true},
		{"Nested field", "user.profile.age", true},
		{"Deep nested field", "user.profile.city", true},
		{"Another nested field", "user.settings.theme", true},
		{"Non-existent top-level", "missing", false},
		{"Non-existent nested", "user.missing", false},
		{"Non-existent deep nested", "user.profile.missing", false},
		{"Traverse non-object", "tags.length", false},
		{"Empty path", "", false},
		{"Path to object itself", "user", true},
		{"Path to nested object", "user.profile", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := schema.HasFieldPath(tt.path)
			assert.Equal(t, tt.expected, result, "HasFieldPath(%q) = %v, want %v", tt.path, result, tt.expected)
		})
	}
}

func TestHasFieldPath_NoSchema(t *testing.T) {
	var schema *Schema
	assert.False(t, schema.HasFieldPath("any.path"))

	schema = &Schema{}
	assert.False(t, schema.HasFieldPath("any.path"))
}

func TestExtractFieldByPathAsString(t *testing.T) {
	testData := map[string]interface{}{
		"name": "John",
		"user": map[string]interface{}{
			"profile": map[string]interface{}{
				"age":  30,
				"tags": []interface{}{"developer", "golang"},
			},
		},
	}

	tests := []struct {
		name     string
		path     string
		expected string
		hasError bool
		errorMsg string
	}{
		{"Extract top-level string", "name", "John", false, ""},
		{"Extract nested integer", "user.profile.age", "30", false, ""},
		{"Extract nested array", "user.profile.tags", "developer, golang", false, ""},
		{"Extract nested object", "user.profile", `{"age":30,"tags":["developer","golang"]}`, false, ""},
		{"Non-existent field", "missing", "", true, "field 'missing' not found at path 'missing'"},
		{"Non-existent nested field", "user.missing", "", true, "field 'missing' not found at path 'user.missing'"},
		{"Traverse non-object", "name.field", "", true, "cannot traverse field 'field' on non-object type string"},
		{"Empty path", "", "", true, "path cannot be empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExtractFieldByPathAsString(testData, tt.path)
			if tt.hasError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
