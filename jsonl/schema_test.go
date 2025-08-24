package jsonl

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONSchemaMarshalToJson(t *testing.T) {
	float64Ptr := func(f float64) *float64 { return &f }
	intPtr := func(i int) *int { return &i }
	boolPtr := func(b bool) *bool { return &b }

	tests := []struct {
		name               string
		inputStruct        JSONSchema
		expectedJSONOutput string
	}{
		{
			name: "Simple Schema",
			inputStruct: JSONSchema{
				Type: "object",
				Properties: map[string]Property{
					"name": {Type: "string"},
					"age":  {Type: "integer"},
				},
				Required: []string{"name"},
			},
			expectedJSONOutput: `{
  "type": "object",
  "properties": {
    "age": {
      "type": "integer"
    },
    "name": {
      "type": "string"
    }
  },
  "required": [
    "name"
  ]
}`,
		},
		{
			name: "Schema with Numbers and String Lengths",
			inputStruct: JSONSchema{
				Type: "object",
				Properties: map[string]Property{
					"count": {Type: "number", Minimum: float64Ptr(0.5), Maximum: float64Ptr(100.0)},
					"code":  {Type: "string", MinLength: intPtr(5), MaxLength: intPtr(10)},
				},
				Required: nil,
			},
			expectedJSONOutput: `{
  "type": "object",
  "properties": {
    "code": {
      "type": "string",
      "minLength": 5,
      "maxLength": 10
    },
    "count": {
      "type": "number",
      "minimum": 0.5,
      "maximum": 100
    }
  }
}`,
		},
		{
			name: "Schema with Nested Object and Array",
			inputStruct: JSONSchema{
				Type: "object",
				Properties: map[string]Property{
					"address": {
						Type: "object",
						Properties: map[string]Property{
							"street": {Type: "string"},
							"city":   {Type: "string"},
						},
						Required: []string{"street", "city"},
					},
					"tags": {
						Type:  "array",
						Items: &Property{Type: "string"},
					},
				},
				Required: nil,
			},
			expectedJSONOutput: `{
  "type": "object",
  "properties": {
    "address": {
      "type": "object",
      "properties": {
        "city": {
          "type": "string"
        },
        "street": {
          "type": "string"
        }
      },
      "required": [
        "street",
        "city"
      ]
    },
    "tags": {
      "type": "array",
      "items": {
        "type": "string"
      }
    }
  }
}`,
		},
		{
			name: "Schema with AdditionalProperties (true and false)",
			inputStruct: JSONSchema{
				Type: "object",
				Properties: map[string]Property{
					"prop1": {Type: "string"},
				},
				Required:             nil,
				AdditionalProperties: boolPtr(false),
			},
			expectedJSONOutput: `{
  "type": "object",
  "properties": {
    "prop1": {
      "type": "string"
    }
  },
  "additionalProperties": false
}`,
		},
		{
			name: "Schema with AdditionalProperties (true)",
			inputStruct: JSONSchema{
				Type: "object",
				Properties: map[string]Property{
					"prop1": {Type: "string"},
				},
				Required:             nil,
				AdditionalProperties: boolPtr(true),
			},
			expectedJSONOutput: `{
  "type": "object",
  "properties": {
    "prop1": {
      "type": "string"
    }
  },
  "additionalProperties": true
}`,
		},
		{
			name: "Empty Schema (only Type)",
			inputStruct: JSONSchema{
				Type:                 "object",
				Properties:           nil,
				Required:             nil,
				AdditionalProperties: nil,
			},
			expectedJSONOutput: `{
  "type": "object"
}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actualJSONBytes, err := tc.inputStruct.MarshalToJSONText()

			require.NoError(t, err, "JSON marshalling should not fail")

			assert.JSONEq(t, tc.expectedJSONOutput, actualJSONBytes, "Marshalled JSON output mismatch")
		})
	}
}

func TestJSONSchemaValidationHelpers(t *testing.T) {
	tests := []struct {
		name                     string
		inputStruct              JSONSchema
		expectedHasSchema        bool
		expectedValidateRequired bool
	}{
		{
			name: "Schema with Properties and Required (Valid)",
			inputStruct: JSONSchema{
				Type: "object",
				Properties: map[string]Property{
					"field1": {Type: "string"},
					"field2": {Type: "integer"},
				},
				Required: []string{"field1"},
			},
			expectedHasSchema:        true,
			expectedValidateRequired: false,
		},
		{
			name: "Schema with Properties and Required (Invalid - Missing)",
			inputStruct: JSONSchema{
				Type: "object",
				Properties: map[string]Property{
					"field1": {Type: "string"},
				},
				Required: []string{"field1", "field3"},
			},
			expectedHasSchema:        true,
			expectedValidateRequired: false,
		},
		{
			name: "Schema with Only Properties (No Required)",
			inputStruct: JSONSchema{
				Type: "object",
				Properties: map[string]Property{
					"field1": {Type: "string"},
				},
				Required: nil,
			},
			expectedHasSchema:        true,
			expectedValidateRequired: false,
		},
		{
			name: "Schema with Only Required (No Properties)",
			inputStruct: JSONSchema{
				Type:       "object",
				Properties: nil,
				Required:   []string{"field1"},
			},
			expectedHasSchema:        true,
			expectedValidateRequired: false,
		},
		{
			name: "Empty Schema (Only Type)",
			inputStruct: JSONSchema{
				Type:       "object",
				Properties: nil,
				Required:   nil,
			},
			expectedHasSchema:        false,
			expectedValidateRequired: false,
		},
		{
			name: "Schema with Empty Properties Map and Empty Required Slice",
			inputStruct: JSONSchema{
				Type:       "object",
				Properties: map[string]Property{},
				Required:   []string{},
			},
			expectedHasSchema:        false,
			expectedValidateRequired: false,
		},
		{
			name: "Schema with Properties, Empty Required Slice",
			inputStruct: JSONSchema{
				Type: "object",
				Properties: map[string]Property{
					"field1": {Type: "string"},
				},
				Required: []string{},
			},
			expectedHasSchema:        true,
			expectedValidateRequired: false,
		},
		{
			name: "Schema with Empty Properties Map, Required Fields",
			inputStruct: JSONSchema{
				Type:       "object",
				Properties: map[string]Property{},
				Required:   []string{"field1"},
			},
			expectedHasSchema:        true,
			expectedValidateRequired: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actualHasSchema := tc.inputStruct.HasSchemaDefinition()
			assert.Equal(t, tc.expectedHasSchema, actualHasSchema, "HasSchemaDefinition mismatch")

			actualValidateRequired := tc.inputStruct.ValidateRequiredProperties()
			assert.Equal(t, tc.expectedValidateRequired, actualValidateRequired, "ValidateRequiredProperties mismatch")
		})
	}
}

func TestValidateJSONText(t *testing.T) {
	obj := JSONSchema{}
	err := obj.ValidateJSONText(`{=}`)
	assert.Error(t, err)

	err = obj.ValidateJSONText(`{"a": 1}`)
	assert.NoError(t, err)

	obj = JSONSchema{
		Type: "object",
		Properties: map[string]Property{
			"address": {
				Type: "object",
				Properties: map[string]Property{
					"street": {Type: "string"},
					"city":   {Type: "string"},
					"index":  {Type: "integer"},
				},
				Required: []string{"street", "city"},
			},
			"tags": {
				Type:  "array",
				Items: &Property{Type: "string"},
			},
		},
		Required: []string{"tags"},
	}
	err = obj.ValidateJSONText(`{"address": {"street": "123 Main St", "city": "New York", "index": 12345}, "tags": ["tag1", "tag2"]}`)
	assert.NoError(t, err)

	err = obj.ValidateJSONText(`{"address": {"street": "123 Main St", "city": "New York", "index": 12345}, "tags1": ["tag1", "tag2"]}`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing required field: tags")

	err = obj.ValidateJSONText(`{"address": {"street": "123 Main St", "city": "New York", "index": "12345"}, "tags": ["tag1", "tag2"]}`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), `invalid type for field 'address': invalid type for field 'index': expected integer`)

	err = obj.ValidateJSONText(`{"address": {"street": "123 Main St", "city": "New York"}, "tags": ["tag1", "tag2"]}`)
	assert.NoError(t, err, "Missing optional 'index' property should be allowed")

	medicalSchema := JSONSchema{
		Type: "object",
		Properties: map[string]Property{
			"vital_signs": {
				Type: "object",
				Properties: map[string]Property{
					"blood_pressure":    {Type: "string"},
					"heart_rate":        {Type: "string"},
					"temperature":       {Type: "string"},
					"respiratory_rate":  {Type: "string"},
					"oxygen_saturation": {Type: "string"},
				},
			},
		},
		Required: []string{"vital_signs"},
	}
	err = medicalSchema.ValidateJSONText(`{"vital_signs": {"blood_pressure": "98/60 mmHg", "heart_rate": "130 bpm", "temperature": "38.5°C"}}`)
	assert.NoError(t, err, "Missing optional nested properties should be allowed")

	err = medicalSchema.ValidateJSONText(`{"other_field": "value"}`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing required field: vital_signs")
}

func TestValidateAdditionalProperties(t *testing.T) {
	// additionalProperties: false - should reject extra fields
	boolFalse := false
	strictSchema := JSONSchema{
		Type: "object",
		Properties: map[string]Property{
			"name": {Type: "string"},
		},
		Required:             []string{"name"},
		AdditionalProperties: &boolFalse,
	}
	err := strictSchema.ValidateJSONText(`{"name": "John", "extra_field": "should_fail"}`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown property: extra_field")

	// additionalProperties: true - should allow extra fields
	boolTrue := true
	flexibleSchema := JSONSchema{
		Type: "object",
		Properties: map[string]Property{
			"name": {Type: "string"},
		},
		Required:             []string{"name"},
		AdditionalProperties: &boolTrue,
	}
	err = flexibleSchema.ValidateJSONText(`{"name": "John", "extra_field": "allowed"}`)
	assert.NoError(t, err, "Extra fields should be allowed with additionalProperties: true")

	// default (nil) additionalProperties - should reject extra fields
	defaultSchema := JSONSchema{
		Type: "object",
		Properties: map[string]Property{
			"name": {Type: "string"},
		},
		Required: []string{"name"},
	}
	err = defaultSchema.ValidateJSONText(`{"name": "John", "extra_field": "should_fail"}`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown property: extra_field")

	// edge case: schema with only required fields (no properties defined)
	requiredOnlySchema := JSONSchema{
		Type:     "object",
		Required: []string{"field1"},
	}
	err = requiredOnlySchema.ValidateJSONText(`{"field1": "value", "extra_field": "should_fail"}`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown property: extra_field")
}
