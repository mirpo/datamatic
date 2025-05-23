package jsonl

import (
	"testing"

	"github.com/google/uuid"
	"github.com/mirpo/datamatic/promptbuilder"
	"github.com/stretchr/testify/assert"
)

func isValidUUID(u string) bool {
	_, err := uuid.Parse(u)
	return err == nil
}

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

func TestNewTextLineEntity(t *testing.T) {
	response := "```json\n{\"foo\":\"bar\"}\n```"
	prompt := "  What is foo?   "

	entity, _ := NewLineEntity(response, prompt, false, nil)

	assert.NotEmpty(t, entity.ID)
	assert.True(t, isValidUUID(entity.ID))

	assert.Equal(t, "text", entity.Format)

	assert.Equal(t, "What is foo?", entity.Prompt)
	assert.Equal(t, `{"foo":"bar"}`, entity.Response)
	assert.Nil(t, entity.Values)
}

func TestNewTextLineEntityWithValues(t *testing.T) {
	response := "```json\n{\"foo\":\"bar\"}\n```"
	prompt := "  What is foo?   "

	values := map[string]promptbuilder.ValueShort{
		"flatten_question_chunk.rate": {
			ID:    "123",
			Value: 9,
		},
		"flatten_question_chunk.question": {
			ID:    "456",
			Value: "Super question",
		},
	}

	entity, _ := NewLineEntity(response, prompt, false, values)

	assert.NotEmpty(t, entity.ID)
	assert.True(t, isValidUUID(entity.ID))

	assert.Equal(t, "text", entity.Format)

	assert.Equal(t, "What is foo?", entity.Prompt)
	assert.Equal(t, `{"foo":"bar"}`, entity.Response)
	assert.Equal(t, map[string]promptbuilder.ValueShort{
		"flatten_question_chunk.rate": {
			ID:    "123",
			Value: 9,
		},
		"flatten_question_chunk.question": {
			ID:    "456",
			Value: "Super question",
		},
	}, entity.Values)
}

func TestNewJSONLineEntity(t *testing.T) {
	response := "```json\n{\"foo\":\"bar\"}\n```"
	prompt := "  What is foo?   "

	entity, _ := NewLineEntity(response, prompt, true, nil)

	assert.NotEmpty(t, entity.ID)
	assert.True(t, isValidUUID(entity.ID))

	assert.Equal(t, "json", entity.Format)

	assert.Equal(t, "What is foo?", entity.Prompt)
	assert.Equal(t, map[string]interface{}{"foo": "bar"}, entity.Response)
}

func TestNewJSONLineEntityError(t *testing.T) {
	response := "```json\n{foo\":\"bar\"}\n```"
	prompt := "  What is foo?   "

	_, err := NewLineEntity(response, prompt, true, nil)

	assert.Error(t, err)
}
