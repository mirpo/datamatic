package jsonl

import (
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"github.com/mirpo/datamatic/promptbuilder"
)

type LineEntity struct {
	ID       string                              `json:"id"`
	Format   string                              `json:"format"`
	Prompt   string                              `json:"prompt"`
	Response interface{}                         `json:"response"`
	Values   map[string]promptbuilder.ValueShort `json:"values,omitempty"`
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

func NewLineEntity(response string, prompt string, isJSON bool, values map[string]promptbuilder.ValueShort) (LineEntity, error) {
	cleanedResponse := cleanResponse(response)

	format := "text"
	var parsedResponse interface{} = cleanedResponse

	if isJSON {
		format = "json"

		err := json.Unmarshal([]byte(cleanedResponse), &parsedResponse)
		if err != nil {
			return LineEntity{}, err
		}
	}

	return LineEntity{
		ID:       uuid.New().String(),
		Prompt:   cleanResponse(prompt),
		Response: parsedResponse,
		Format:   format,
		Values:   values,
	}, nil
}
