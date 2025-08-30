package step

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/mirpo/datamatic/config"
	"github.com/mirpo/datamatic/fs"
	"github.com/mirpo/datamatic/jsonl"
)

type LineValue struct {
	ID       string `json:"id"`
	Response string `json:"response"`
}

func convertJSONValueToStringReflected(value interface{}) string {
	if value == nil {
		return ""
	}

	val := reflect.ValueOf(value)

	switch val.Kind() {
	case reflect.String:
		return val.String()
	case reflect.Float64:
		v := val.Float()
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10)
		}
		return strconv.FormatFloat(v, 'f', 2, 64)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(val.Int(), 10)
	case reflect.Bool:
		return strconv.FormatBool(val.Bool())
	case reflect.Slice:
		var elements []string
		for i := range val.Len() {
			elements = append(elements, convertJSONValueToStringReflected(val.Index(i).Interface()))
		}
		return strings.Join(elements, ", ")
	case reflect.Map:
		data, err := json.Marshal(value)
		if err != nil {
			return fmt.Sprintf("error marshalling map: %v", err)
		}
		return string(data)
	default:
		return fmt.Sprintf("%v", value)
	}
}

func uuidFromString(input string) string {
	return uuid.NewMD5(uuid.NameSpaceOID, []byte(input)).String()
}

func getFieldAsString(data map[string]interface{}, key string) (string, error) {
	value, exists := data[key]
	if !exists {
		return "", fmt.Errorf("key '%s' not found", key)
	}
	return convertJSONValueToStringReflected(value), nil
}

func readStepValue(step config.Step, outputFolder string, lineNumber int, attrKey string) (*LineValue, error) {
	line, err := fs.ReadLineFromFile(step.OutputFilename, lineNumber)
	if err != nil {
		return nil, err
	}

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

		return &LineValue{
			ID:       uuidFromString(value),
			Response: value,
		}, nil

	case config.PromptStepType:
		var decoded jsonl.LineEntity
		if err := json.Unmarshal([]byte(line), &decoded); err != nil {
			return nil, fmt.Errorf("prompt step: failed to parse JSON from line %d: %w", lineNumber, err)
		}

		var value string
		if !step.JSONSchema.HasSchemaDefinition() {
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

		return &LineValue{
			ID:       decoded.ID,
			Response: value,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported step type '%s'", step.Type)
	}
}
