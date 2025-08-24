package jsonl

import (
	"encoding/json"
	"fmt"
	"slices"
)

type JSONSchema struct {
	Type                 string              `yaml:"type" json:"type"`
	Properties           map[string]Property `yaml:"properties" json:"properties,omitempty"`
	Required             []string            `yaml:"required" json:"required,omitempty"`
	AdditionalProperties *bool               `yaml:"additionalProperties,omitempty" json:"additionalProperties,omitempty"`
}

type Property struct {
	Type                 string              `yaml:"type" json:"type"`
	Description          string              `yaml:"description,omitempty" json:"description,omitempty"`
	Minimum              *float64            `yaml:"minimum,omitempty" json:"minimum,omitempty"`
	Maximum              *float64            `yaml:"maximum,omitempty" json:"maximum,omitempty"`
	Items                *Property           `yaml:"items,omitempty" json:"items,omitempty"`
	Properties           map[string]Property `yaml:"properties" json:"properties,omitempty"`
	Required             []string            `yaml:"required" json:"required,omitempty"`
	MinLength            *int                `yaml:"minLength,omitempty" json:"minLength,omitempty"`
	MaxLength            *int                `yaml:"maxLength,omitempty" json:"maxLength,omitempty"`
	Pattern              string              `yaml:"pattern,omitempty" json:"pattern,omitempty"`
	Enum                 []interface{}       `yaml:"enum,omitempty" json:"enum,omitempty"`
	AdditionalProperties *bool               `yaml:"additionalProperties,omitempty" json:"additionalProperties,omitempty"`
}

func (j *JSONSchema) MarshalToJSONText() (string, error) {
	jsonSchemaBytes, err := json.MarshalIndent(j, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JsonSchema struct to JSON: %w", err)
	}

	return string(jsonSchemaBytes), nil
}

func (j *JSONSchema) MarshalJSON() ([]byte, error) {
	return json.Marshal(*j)
}

func (j *JSONSchema) HasSchemaDefinition() bool {
	return len(j.Properties) > 0 || len(j.Required) > 0
}

func (j *JSONSchema) ValidateRequiredProperties() bool {
	if !j.HasSchemaDefinition() {
		return false
	}

	if len(j.Required) != len(j.Properties) {
		return false
	}

	for _, requiredField := range j.Required {
		if _, ok := j.Properties[requiredField]; !ok {
			return false
		}
	}

	return true
}

func (j *JSONSchema) ValidateJSONText(input string) error {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(input), &data); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	return j.validateAgainstSchema(data)
}

func (j *JSONSchema) validateAgainstSchema(data map[string]interface{}) error {
	for _, req := range j.Required {
		if _, ok := data[req]; !ok {
			return fmt.Errorf("missing required field: %s", req)
		}
	}

	for key, prop := range j.Properties {
		val, exists := data[key]
		if !exists {
			continue
		}

		if err := validateType(val, prop); err != nil {
			return fmt.Errorf("invalid type for field '%s': %v", key, err)
		}
	}

	if j.HasSchemaDefinition() && (j.AdditionalProperties == nil || !*j.AdditionalProperties) {
		for key := range data {
			if _, exists := j.Properties[key]; !exists {
				if len(j.Properties) == 0 && slices.Contains(j.Required, key) {
					continue
				}
				return fmt.Errorf("unknown property: %s", key)
			}
		}
	}

	return nil
}

func (j *JSONSchema) HasRequiredProperty(name string) bool {
	_, exist := j.Properties[name]
	if !exist {
		return false
	}

	return slices.Contains(j.Required, name)
}

func validateType(value interface{}, schema Property) error {
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
				if err := validateType(item, *schema.Items); err != nil {
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
		if err := subSchema.validateAgainstSchema(obj); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported type: %s", schema.Type)
	}

	return nil
}
