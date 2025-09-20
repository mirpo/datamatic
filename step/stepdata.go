package step

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/mirpo/datamatic/config"
	"github.com/mirpo/datamatic/fs"
	"github.com/mirpo/datamatic/jsonl"
	"github.com/mirpo/datamatic/jsonschema"
	"github.com/mirpo/datamatic/promptbuilder"
)

func uuidFromString(input string) string {
	return uuid.NewMD5(uuid.NameSpaceOID, []byte(input)).String()
}

// getFieldAsString extracts a field from data using a path (supports nested fields)
// For CLI steps, we don't have schema, so we use a schema-less approach
func getFieldAsString(data map[string]interface{}, key string) (string, error) {
	parts := strings.Split(key, ".")
	current := interface{}(data)

	for i, part := range parts {
		switch v := current.(type) {
		case map[string]interface{}:
			value, exists := v[part]
			if !exists {
				return "", fmt.Errorf("field '%s' not found at path '%s'", part, strings.Join(parts[:i+1], "."))
			}
			current = value
		default:
			return "", fmt.Errorf("cannot traverse field '%s' on non-object type %T at path '%s'", part, current, strings.Join(parts[:i], "."))
		}
	}

	return convertToString(current), nil
}

// convertToString converts various types to their string representation
// This is similar to the one in jsonschema but optimized for step data
func convertToString(value interface{}) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return v
	case bool:
		if v {
			return "true"
		}
		return "false"
	case float64:
		if v == float64(int64(v)) {
			return fmt.Sprintf("%.0f", v)
		}
		return fmt.Sprintf("%g", v)
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case []interface{}:
		parts := make([]string, len(v))
		for i, item := range v {
			parts[i] = convertToString(item)
		}
		return strings.Join(parts, ", ")
	default:
		// For complex objects, return JSON representation
		if jsonBytes, err := json.Marshal(value); err == nil {
			return string(jsonBytes)
		}
		return fmt.Sprintf("%v", value)
	}
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
			value, err := getFieldAsString(decoded, fieldPath)
			if err != nil {
				return nil, fmt.Errorf("CLI step: missing or invalid '%s' field: %w", fieldPath, err)
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
