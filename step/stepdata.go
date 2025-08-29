package step

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/mirpo/datamatic/config"
	"github.com/mirpo/datamatic/fs"
	"github.com/mirpo/datamatic/jsonl"
	"github.com/mirpo/datamatic/jsonschema"
)

func uuidFromString(input string) string {
	return uuid.NewMD5(uuid.NameSpaceOID, []byte(input)).String()
}

func getFieldAsString(data map[string]interface{}, key string) (string, error) {
	value, exists := data[key]
	if !exists {
		return "", fmt.Errorf("getting field '%s' from data: %w", key, ErrKeyNotFound)
	}
	return jsonschema.ConvertValueToString(value), nil
}

func ReadStepValue(step config.Step, outputFolder string, lineNumber int, attrKey string) (*config.LineValue, error) {
	line, err := fs.ReadLineFromFile(step.OutputFilename, lineNumber)
	if err != nil {
		return nil, err
	}

	configValidator := jsonschema.NewConfigValidator()

	switch step.Type {
	case config.CliStepType:
		var decoded map[string]interface{}
		if err := json.Unmarshal([]byte(line), &decoded); err != nil {
			return nil, fmt.Errorf("CLI step: failed to parse JSON from line %d: %w", lineNumber, err)
		}

		value, err := getFieldAsString(decoded, attrKey)
		if err != nil {
			return nil, fmt.Errorf("CLI step: missing or invalid '%s' field: %w", attrKey, err)
		}

		return &config.LineValue{
			ID:       uuidFromString(value),
			Response: value,
		}, nil

	case config.PromptStepType:
		var decoded jsonl.LineEntity
		if err := json.Unmarshal([]byte(line), &decoded); err != nil {
			return nil, fmt.Errorf("prompt step: failed to parse JSON from line %d: %w", lineNumber, err)
		}

		var value string
		if !configValidator.HasSchemaDefinition(step.JSONSchema) {
			str, ok := decoded.Response.(string)
			if !ok {
				return nil, fmt.Errorf("prompt step: expected string response, got %T", decoded.Response)
			}
			value = str
		} else {
			data, ok := decoded.Response.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("prompt step: expected map response, got %T", decoded.Response)
			}

			value, err = getFieldAsString(data, attrKey)
			if err != nil {
				return nil, fmt.Errorf("prompt step: missing or invalid '%s' field", attrKey)
			}
		}

		return &config.LineValue{
			ID:       decoded.ID,
			Response: value,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported step type: %s", step.Type)
	}
}
