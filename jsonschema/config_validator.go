package jsonschema

import "slices"

type ConfigValidator struct{}

func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{}
}

func (v *ConfigValidator) HasSchemaDefinition(schema JSONSchema) bool {
	return len(schema.Properties) > 0 || len(schema.Required) > 0
}

func (v *ConfigValidator) ValidateRequiredProperties(schema JSONSchema) bool {
	if !v.HasSchemaDefinition(schema) {
		return false
	}

	if len(schema.Required) != len(schema.Properties) {
		return false
	}

	for _, requiredField := range schema.Required {
		if _, ok := schema.Properties[requiredField]; !ok {
			return false
		}
	}

	return true
}

func (v *ConfigValidator) HasRequiredProperty(schema JSONSchema, name string) bool {
	_, exist := schema.Properties[name]
	if !exist {
		return false
	}

	return slices.Contains(schema.Required, name)
}
