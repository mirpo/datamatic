package jsonschema

import (
	"encoding/json"
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

func (j *JSONSchema) MarshalJSON() ([]byte, error) {
	return json.Marshal(*j)
}
