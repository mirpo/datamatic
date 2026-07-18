package step

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mirpo/datamatic/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runReadStep(t *testing.T, read, format string, files map[string]string) []string {
	t.Helper()
	src := t.TempDir()
	for name, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(src, name), []byte(content), 0o644))
	}
	out := t.TempDir()
	step := config.Step{
		Name:           "src",
		Type:           config.ReadStepType,
		Read:           filepath.Join(src, read),
		Format:         format,
		OutputFilename: filepath.Join(out, "src.jsonl"),
	}
	err := (&ReadStep{}).Run(context.Background(), config.NewConfig(), step, out)
	require.NoError(t, err)
	return readOutput(t, step.OutputFilename)
}

func TestReadStepRun_Files(t *testing.T) {
	lines := runReadStep(t, "*.md", config.ReadFormatFiles, map[string]string{
		"b.md": "beta",
		"a.md": "alpha",
	})
	require.Len(t, lines, 2)
	// sorted ascending: a before b
	assert.Contains(t, lines[0], `"name":"a.md"`)
	assert.Contains(t, lines[0], "alpha")
	assert.Contains(t, lines[1], `"name":"b.md"`)
	assert.Contains(t, lines[1], "beta")
}

func TestReadStepRun_CSV(t *testing.T) {
	lines := runReadStep(t, "leads.csv", config.ReadFormatCSV, map[string]string{
		"leads.csv": "company,city\nAcme,NYC\nGlobex,LA\n",
	})
	require.Len(t, lines, 2)
	assert.Contains(t, lines[0], `"company":"Acme"`)
	assert.Contains(t, lines[0], `"city":"NYC"`)
	assert.Contains(t, lines[1], `"company":"Globex"`)
}

func TestReadStepRun_JSONL(t *testing.T) {
	lines := runReadStep(t, "seed.jsonl", config.ReadFormatJSONL, map[string]string{
		"seed.jsonl": `{"x":1}` + "\n" + `{"x":2}` + "\n",
	})
	require.Len(t, lines, 2)
	assert.Contains(t, lines[0], `"x":1`)
	assert.Contains(t, lines[1], `"x":2`)
}

func TestReadStepRun_EmptyGlobFails(t *testing.T) {
	out := t.TempDir()
	step := config.Step{
		Name:           "src",
		Type:           config.ReadStepType,
		Read:           filepath.Join(t.TempDir(), "*.md"),
		Format:         config.ReadFormatFiles,
		OutputFilename: filepath.Join(out, "src.jsonl"),
	}
	err := (&ReadStep{}).Run(context.Background(), config.NewConfig(), step, out)
	require.Error(t, err)
}
