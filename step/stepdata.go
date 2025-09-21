package step

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/mirpo/datamatic/config"
	"github.com/mirpo/datamatic/fs"
	"github.com/mirpo/datamatic/jsonl"
	"github.com/mirpo/datamatic/jsonschema"
	"github.com/mirpo/datamatic/promptbuilder"
)

// uuidFromString generates a UUID based on the input string using MD5 hashing
// mainly used to create fake ID for external data where no ID is provided
func uuidFromString(input string) string {
	return uuid.NewMD5(uuid.NameSpaceOID, []byte(input)).String()
}

// readStepValuesBatch reads multiple field values from a step in one operation
func readStepValuesBatch(step config.Step, outputFolder string, lineNumber int, fieldPaths []string) (map[string]promptbuilder.StepValue, error) {
	line, err := fs.ReadLineFromFile(step.OutputFilename, lineNumber)
	if err != nil {
		return nil, err
	}

	result := make(map[string]promptbuilder.StepValue)

	switch step.Type {
	case config.CliStepType:
		var decoded map[string]interface{}
		if err := json.Unmarshal([]byte(line), &decoded); err != nil {
			return nil, fmt.Errorf("CLI step: failed to parse JSON from line %d: %w", lineNumber, err)
		}

		for _, fieldPath := range fieldPaths {
			value, err := jsonschema.ExtractFieldByPathAsString(decoded, fieldPath)
			if err != nil {
				return nil, fmt.Errorf("prompt step: failed to extract field '%s': %w", fieldPath, err)
			}

			result[fieldPath] = promptbuilder.StepValue{
				ID:      uuidFromString(value),
				Content: value,
			}
		}

	case config.PromptStepType:
		var decoded jsonl.LineEntity
		if err := json.Unmarshal([]byte(line), &decoded); err != nil {
			return nil, fmt.Errorf("prompt step: failed to parse JSON from line %d: %w", lineNumber, err)
		}

		for _, fieldPath := range fieldPaths {
			var value string
			if !step.JSONSchema.HasSchemaDefinition() {
				str, ok := decoded.Response.(string)
				if !ok {
					return nil, fmt.Errorf("prompt step: expected string response, got %T", decoded.Response)
				}
				value = str
			} else {
				value, err = jsonschema.ExtractFieldByPathAsString(decoded.Response, fieldPath)
				if err != nil {
					return nil, fmt.Errorf("prompt step: failed to extract field '%s': %w", fieldPath, err)
				}
			}

			result[fieldPath] = promptbuilder.StepValue{
				ID:      decoded.ID,
				Content: value,
			}
		}

	default:
		return nil, fmt.Errorf("unsupported step type '%s'", step.Type)
	}

	return result, nil
}
