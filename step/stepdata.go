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

// getSourceDataFromLine extracts the data and record ID from a step line
// Shell steps: full line is an unknown JSON
// Prompt steps: line contains JSON created with datamatic
func getSourceDataFromLine(step config.Step, line string) (interface{}, string, error) {
	switch step.Type {
	case config.ShellStepType:
		var decoded map[string]interface{}
		if err := json.Unmarshal([]byte(line), &decoded); err != nil {
			return nil, "", fmt.Errorf("shell step: failed to parse JSON: %w", err)
		}
		return decoded, "", nil

	case config.PromptStepType:
		var decoded jsonl.LineEntity
		if err := json.Unmarshal([]byte(line), &decoded); err != nil {
			return nil, "", fmt.Errorf("prompt step: failed to parse JSON: %w", err)
		}
		return decoded.Response, decoded.ID, nil

	default:
		return nil, "", fmt.Errorf("unsupported step type '%s'", step.Type)
	}
}

var readLineFromFile = fs.ReadLineFromFile

// readStepValuesBatch reads multiple field values from a step in one operation
func readStepValuesBatch(step config.Step, outputFolder string, lineNumber int, fieldPaths []string) (map[string]promptbuilder.StepValue, error) {
	line, err := readLineFromFile(step.OutputFilename, lineNumber)
	if err != nil {
		return nil, err
	}

	sourceData, recordID, err := getSourceDataFromLine(step, line)
	if err != nil {
		return nil, fmt.Errorf("line %d: %w", lineNumber, err)
	}

	result := make(map[string]promptbuilder.StepValue)
	for _, fieldPath := range fieldPaths {
		var value string

		if step.Type == config.PromptStepType && !step.JSONSchema.HasSchemaDefinition() {
			str, ok := sourceData.(string)
			if !ok {
				return nil, fmt.Errorf("prompt step: expected string response, got %T", sourceData)
			}
			value = str
		} else {
			value, err = jsonschema.ExtractFieldByPathAsString(sourceData, fieldPath)
			if err != nil {
				return nil, fmt.Errorf("failed to extract field '%s': %w", fieldPath, err)
			}
		}

		fieldID := recordID
		if fieldID == "" { // Shell case
			fieldID = uuidFromString(value)
		}

		result[fieldPath] = promptbuilder.StepValue{
			ID:      fieldID,
			Content: value,
		}
	}

	return result, nil
}
