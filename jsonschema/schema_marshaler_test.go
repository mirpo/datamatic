package jsonschema

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSchemaMarshaler_MarshalToJSONText(t *testing.T) {
	marshaler := NewSchemaMarshaler()

	schema := JSONSchema{
		Type: "object",
		Properties: map[string]Property{
			"name": {Type: "string"},
			"age":  {Type: "number"},
		},
		Required: []string{"name"},
	}

	result, err := marshaler.MarshalToJSONText(schema)
	assert.NoError(t, err)

	var unmarshaled JSONSchema
	err = json.Unmarshal([]byte(result), &unmarshaled)
	assert.NoError(t, err)

	assert.Equal(t, schema.Type, unmarshaled.Type)
	assert.Equal(t, len(schema.Properties), len(unmarshaled.Properties))
	assert.Equal(t, len(schema.Required), len(unmarshaled.Required))
}
