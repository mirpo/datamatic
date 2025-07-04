package jsonl

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewWriter(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test*.jsonl")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	writer, err := NewWriter(tmpFile.Name())
	assert.NoError(t, err)
	assert.NotNil(t, writer)

	err = writer.Close()
	assert.NoError(t, err)
}

func TestWriteLine(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test*.jsonl")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	writer, err := NewWriter(tmpFile.Name())
	assert.NoError(t, err)
	defer writer.Close()

	entity := LineEntity{
		ID:       "123",
		Format:   "json",
		Prompt:   "Say hello",
		Response: map[string]string{"text": "hello"},
	}

	err = writer.WriteLine(entity)
	assert.NoError(t, err)

	content, err := os.ReadFile(tmpFile.Name())
	assert.NoError(t, err)

	var decoded LineEntity
	err = json.Unmarshal(content, &decoded)
	assert.NoError(t, err)

	assert.Equal(t, entity.ID, decoded.ID)
	assert.Equal(t, entity.Format, decoded.Format)
	assert.Equal(t, entity.Prompt, decoded.Prompt)

	respMap, ok := decoded.Response.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "hello", respMap["text"])
}

func TestWriteLine_ErrorOnMarshal(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test*.jsonl")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	writer, err := NewWriter(tmpFile.Name())
	assert.NoError(t, err)
	defer writer.Close()

	entity := LineEntity{
		ID:       "123",
		Format:   "json",
		Prompt:   "Say hello",
		Response: make(chan int),
	}

	err = writer.WriteLine(entity)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to marshal line entity")
}
