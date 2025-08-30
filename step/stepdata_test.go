package step

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvertJSONValueToStringReflected(t *testing.T) {
	tests := []struct {
		value    interface{}
		expected string
	}{
		{nil, ""},
		{"string", "string"},
		{3.14, "3.14"},
		{3, "3"},
		{true, "true"},
		{[]interface{}{"a", "b", "c"}, "a, b, c"},
		{map[string]interface{}{"key": "value"}, `{"key":"value"}`},
		{map[string]interface{}{"foo": 42, "bar": true}, `{"bar":true,"foo":42}`},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt.value), func(t *testing.T) {
			result := convertJSONValueToStringReflected(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetFieldAsString(t *testing.T) {
	tests := []struct {
		data     map[string]interface{}
		key      string
		expected string
		err      error
	}{
		{map[string]interface{}{"key1": "value1"}, "key1", "value1", nil},
		{map[string]interface{}{"key1": 42}, "key1", "42", nil},
		{map[string]interface{}{"key1": true}, "key1", "true", nil},
		{map[string]interface{}{}, "nonexistent", "", fmt.Errorf("key 'nonexistent' not found")},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("Key %s", tt.key), func(t *testing.T) {
			result, err := getFieldAsString(tt.data, tt.key)
			if tt.err != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.err.Error())
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
