package jsonl

import (
	"strings"

	"github.com/google/uuid"
)

type LineEntity struct {
	ID       string      `json:"id"`
	Format   string      `json:"format"`
	Prompt   string      `json:"prompt"`
	Response interface{} `json:"response"`
}

func cleanResponse(input string) string {
	input = strings.TrimPrefix(input, "\"")
	input = strings.TrimSuffix(input, "\"")

	input = strings.TrimPrefix(input, "```json")
	input = strings.TrimPrefix(input, "```")

	input = strings.TrimSuffix(input, "```")

	input = strings.TrimSpace(input)

	return input
}

func NewLineEntity(response string, prompt string) LineEntity {
	cleanedResponse := cleanResponse(response)

	format := "text"
	var parsedResponse interface{} = cleanedResponse

	return LineEntity{
		ID:       uuid.New().String(),
		Prompt:   cleanResponse(prompt),
		Response: parsedResponse,
		Format:   format,
	}
}
