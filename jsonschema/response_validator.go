package jsonschema

import (
	"encoding/json"
	"fmt"
	"slices"
)

type ResponseValidator struct{}

func NewResponseValidator() *ResponseValidator {
	return &ResponseValidator{}
}

func (v *ResponseValidator) ValidateJSONText(schema JSONSchema, input string) error {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(input), &data); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	return v.validateAgainstSchema(schema, data)
}

func (v *ResponseValidator) validateAgainstSchema(schema JSONSchema, data map[string]interface{}) error {
	for _, req := range schema.Required {
		if _, ok := data[req]; !ok {
			return fmt.Errorf("missing required field: %s", req)
		}
	}

	for key, prop := range schema.Properties {
		val, exists := data[key]
		if !exists {
			continue
		}

		if err := v.validateType(val, prop); err != nil {
			return fmt.Errorf("invalid type for field '%s': %v", key, err)
		}
	}

	if len(schema.Properties) > 0 && (schema.AdditionalProperties == nil || !*schema.AdditionalProperties) {
		for key := range data {
			if _, exists := schema.Properties[key]; !exists {
				if len(schema.Properties) == 0 && slices.Contains(schema.Required, key) {
					continue
				}
				return fmt.Errorf("unknown property: %s", key)
			}
		}
	}

	return nil
}

func (v *ResponseValidator) validateType(value interface{}, schema Property) error {
	if len(schema.Enum) > 0 {
		found := false
		for _, enumValue := range schema.Enum {
			if value == enumValue {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("value %v is not in enum %v", value, schema.Enum)
		}
	}

	switch schema.Type {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("expected string")
		}
	case "number":
		if _, ok := value.(float64); !ok {
			return fmt.Errorf("expected number")
		}
	case "integer":
		if f, ok := value.(float64); !ok || f != float64(int(f)) {
			return fmt.Errorf("expected integer")
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("expected boolean")
		}
	case "array":
		arr, ok := value.([]interface{})
		if !ok {
			return fmt.Errorf("expected array")
		}
		if schema.Items != nil {
			for i, item := range arr {
				if err := v.validateType(item, *schema.Items); err != nil {
					return fmt.Errorf("array item %d: %v", i, err)
				}
			}
		}
	case "object":
		obj, ok := value.(map[string]interface{})
		if !ok {
			return fmt.Errorf("expected object")
		}
		subSchema := JSONSchema{
			Type:                 schema.Type,
			Properties:           schema.Properties,
			Required:             schema.Required,
			AdditionalProperties: schema.AdditionalProperties,
		}
		if err := v.validateAgainstSchema(subSchema, obj); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported type: %s", schema.Type)
	}

	return nil
}
