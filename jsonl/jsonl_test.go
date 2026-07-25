package jsonl

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// TestNewWriter_TruncatesExistingFile pins the contract a rerun depends on: a
// step's output file holds only the rows of the run that produced it. Appending
// would silently stack a new run's rows on top of the previous one's.
func TestNewWriter_TruncatesExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "step.jsonl")
	require.NoError(t, os.WriteFile(path, []byte(`{"stale":"previous run"}`+"\n"), 0o644))

	writer, err := NewWriter(path)
	require.NoError(t, err)
	require.NoError(t, writer.WriteJSON(map[string]string{"fresh": "current run"}))
	require.NoError(t, writer.Close())

	content, err := os.ReadFile(path)
	require.NoError(t, err)

	assert.Equal(t, `{"fresh":"current run"}`+"\n", string(content),
		"reopening a step's output must replace the previous run's rows, not append to them")
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
	assert.Contains(t, err.Error(), "failed to marshal value")
}

func TestWriteJSON_RawValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.jsonl")
	w, err := NewWriter(path)
	require.NoError(t, err)

	require.NoError(t, w.WriteJSON(map[string]interface{}{"a": 1}))
	require.NoError(t, w.WriteJSON("scalar"))
	require.NoError(t, w.Close())

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "{\"a\":1}\n\"scalar\"\n", string(data))
}
