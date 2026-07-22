package fs

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteCSV(t *testing.T) {
	p := filepath.Join(t.TempDir(), "out.csv")
	rows := []map[string]interface{}{
		{"name": "Acme", "score": float64(9)},
		{"name": "Globex", "tags": []interface{}{"a", "b"}}, // differing keys + nested
	}

	require.NoError(t, WriteCSV(p, rows))

	f, err := os.Open(p)
	require.NoError(t, err)
	defer f.Close()
	recs, err := csv.NewReader(f).ReadAll()
	require.NoError(t, err)

	assert.Equal(t, []string{"name", "score", "tags"}, recs[0], "header = sorted union of keys")
	assert.Equal(t, []string{"Acme", "9", ""}, recs[1], "missing key → empty; number verbatim")
	assert.Equal(t, []string{"Globex", "", `["a","b"]`}, recs[2], "nested value JSON-encoded")
}

func TestWriteJSONArray(t *testing.T) {
	p := filepath.Join(t.TempDir(), "out.json")
	rows := []interface{}{
		map[string]interface{}{"a": float64(1)},
		map[string]interface{}{"a": float64(2)},
	}

	require.NoError(t, WriteJSONArray(p, rows))

	data, err := os.ReadFile(p)
	require.NoError(t, err)
	var back []map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &back))
	require.Len(t, back, 2)
	assert.Equal(t, float64(1), back[0]["a"])
	assert.Contains(t, string(data), "\n", "pretty-printed")
}

func TestWriteMarkdownTable(t *testing.T) {
	p := filepath.Join(t.TempDir(), "out.md")
	rows := []map[string]interface{}{
		{"name": "Acme", "score": float64(9)},
		{"name": "Globex", "score": float64(4)},
	}

	require.NoError(t, WriteMarkdownTable(p, rows))

	data, err := os.ReadFile(p)
	require.NoError(t, err)
	got := string(data)
	lines := strings.Split(strings.TrimSpace(got), "\n")
	assert.Equal(t, "| name | score |", lines[0])
	assert.Equal(t, "| --- | --- |", lines[1])
	assert.Contains(t, got, "| Acme | 9 |")
}
