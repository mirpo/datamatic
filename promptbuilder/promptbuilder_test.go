package promptbuilder

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetCompiledPrompt(t *testing.T) {
	testCases := []struct {
		name           string
		promptTemplate string
		values         map[string]interface{}
		expected       string
		expectErr      bool
	}{
		{
			name:           "Valid template with values",
			promptTemplate: "Hello, {{.Name}}!",
			values: map[string]interface{}{
				"Name": "World",
			},
			expected: "Hello, World!",
		},
		{
			name:           "Invalid template syntax",
			promptTemplate: "Hello, {{.Name}!}",
			values: map[string]interface{}{
				"Name": "World",
			},
			expectErr: true,
		},
		{
			name:           "Empty template",
			promptTemplate: "",
			values: map[string]interface{}{
				"Name": "World",
			},
			expected: "",
		},
		{
			name:           "Empty values map",
			promptTemplate: "Hello, {{.Name}}!",
			values:         map[string]interface{}{},
			expected:       "Hello, <no value>!",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := GetCompiledPrompt(tc.promptTemplate, tc.values)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, strings.TrimSpace(tc.expected), strings.TrimSpace(result))
			}
		})
	}
}
