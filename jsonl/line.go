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
	input = strings.TrimSpace(input)

	if strings.HasPrefix(input, "```") {
		input = strings.TrimPrefix(input, "```json")
		input = strings.TrimPrefix(input, "```")
		input = strings.TrimSuffix(strings.TrimSpace(input), "```")
		input = strings.TrimSpace(input)
	}

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
		Prompt:   strings.TrimSpace(prompt),
		Response: parsedResponse,
		Format:   format,
		Values:   values,
	}, nil
}

// UnfoldLineage turns lineage keys (".step.field.path") into a nested plain
// map so jq programs can reach them naturally: $parent.step.field.path.
// Plain map[string]interface{} nodes are required by gojq (it type-switches
// on exactly that type), so this deliberately does not reuse promptbuilder's
// Object-building helpers. Returns untyped nil when there is no lineage.
func UnfoldLineage(values map[string]promptbuilder.ValueShort) interface{} {
	if len(values) == 0 {
		return nil
	}

	parent := make(map[string]interface{})
	for key, v := range values {
		parts := strings.Split(strings.TrimPrefix(key, "."), ".")
		current := parent
		for _, part := range parts[:len(parts)-1] {
			next, ok := current[part].(map[string]interface{})
			if !ok {
				next = make(map[string]interface{})
				current[part] = next
			}
			current = next
		}
		current[parts[len(parts)-1]] = v.Value
	}
	return parent
}
