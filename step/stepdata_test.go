package step

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvertToString(t *testing.T) {
	tests := []struct {
		value    interface{}
		expected string
	}{
		{nil, ""},
		{"string", "string"},
		{3.14, "3.14"},
		{3.0, "3"},
		{3, "3"},
		{int64(42), "42"},
		{true, "true"},
		{false, "false"},
		{[]interface{}{"a", "b", "c"}, "a, b, c"},
		{map[string]interface{}{"key": "value"}, `{"key":"value"}`},
		{map[string]interface{}{"foo": 42, "bar": true}, `{"bar":true,"foo":42}`},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt.value), func(t *testing.T) {
			result := convertToString(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetFieldAsString(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]interface{}
		key      string
		expected string
		hasError bool
		errorMsg string
	}{
		{
			name:     "Simple field access",
			data:     map[string]interface{}{"key1": "value1"},
			key:      "key1",
			expected: "value1",
			hasError: false,
		},
		{
			name:     "Integer field",
			data:     map[string]interface{}{"key1": 42},
			key:      "key1",
			expected: "42",
			hasError: false,
		},
		{
			name:     "Boolean field",
			data:     map[string]interface{}{"key1": true},
			key:      "key1",
			expected: "true",
			hasError: false,
		},
		{
			name:     "Nested field access",
			data:     map[string]interface{}{"user": map[string]interface{}{"profile": map[string]interface{}{"name": "John"}}},
			key:      "user.profile.name",
			expected: "John",
			hasError: false,
		},
		{
			name:     "Nested field with numbers",
			data:     map[string]interface{}{"stats": map[string]interface{}{"score": 95}},
			key:      "stats.score",
			expected: "95",
			hasError: false,
		},
		{
			name:     "Nonexistent top-level field",
			data:     map[string]interface{}{},
			key:      "nonexistent",
			expected: "",
			hasError: true,
			errorMsg: "field 'nonexistent' not found at path 'nonexistent'",
		},
		{
			name:     "Nonexistent nested field",
			data:     map[string]interface{}{"user": map[string]interface{}{}},
			key:      "user.missing",
			expected: "",
			hasError: true,
			errorMsg: "field 'missing' not found at path 'user.missing'",
		},
		{
			name:     "Cannot traverse non-object",
			data:     map[string]interface{}{"value": "string"},
			key:      "value.field",
			expected: "",
			hasError: true,
			errorMsg: "cannot traverse field 'field' on non-object type string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := getFieldAsString(tt.data, tt.key)
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

func TestUUIDFromString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"example", "16eda250-39c7-3c5d-b2f8-7eb6dfedff40"},
		{"", "596b79dc-00dd-3991-a72f-d3696c38c64f"},
	}

	for _, tt := range tests {
		got := uuidFromString(tt.input)
		if got != tt.expected {
			t.Errorf("UUIDFromString(%q) = %v; want %v", tt.input, got, tt.expected)
		}
	}
}
