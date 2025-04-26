package jsonl

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCleanResponse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes surrounding quotes",
			input:    `"hello world"`,
			expected: "hello world",
		},
		{
			name:     "removes ```json prefix and ``` suffix",
			input:    "```json\n{\"key\":\"value\"}\n```",
			expected: `{"key":"value"}`,
		},
		{
			name:     "removes ``` prefix and suffix",
			input:    "```\nplain text\n```",
			expected: "plain text",
		},
		{
			name:     "trims extra spaces",
			input:    "   hello    ",
			expected: "hello",
		},
		{
			name:     "no change if clean already",
			input:    "already clean",
			expected: "already clean",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleaned := cleanResponse(tt.input)
			assert.Equal(t, tt.expected, cleaned)
		})
	}
}

func TestNewLineEntity(t *testing.T) {
	response := "```json\n{\"foo\":\"bar\"}\n```"
	prompt := "  What is foo?   "

	entity := NewLineEntity(response, prompt)

	assert.NotEmpty(t, entity.ID)
	assert.True(t, isValidUUID(entity.ID))

	assert.Equal(t, "text", entity.Format)

	assert.Equal(t, "What is foo?", entity.Prompt)
	assert.Equal(t, `{"foo":"bar"}`, entity.Response)
}

func isValidUUID(u string) bool {
	_, err := uuid.Parse(u)
	return err == nil
}
