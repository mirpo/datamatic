package step

import (
	"testing"
)

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
