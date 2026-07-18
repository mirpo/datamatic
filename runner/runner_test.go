package runner_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mirpo/datamatic/config"
	"github.com/mirpo/datamatic/fs"
	"github.com/mirpo/datamatic/internal/llmtest"
	"github.com/mirpo/datamatic/runner"
	"github.com/mirpo/datamatic/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun_CancelledContextStopsExecution(t *testing.T) {
	dir := t.TempDir()
	cfg := config.NewConfig()
	cfg.OutputFolder = dir
	cfg.Steps = []config.Step{
		{Name: "s1", Type: config.ShellStepType, Run: "sleep 5", WorkDir: dir, OutputFilename: filepath.Join(dir, "x.jsonl")},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled

	err := runner.NewRunner(cfg).Run(ctx)

	assert.Error(t, err)
}

func TestRun_TransformPipelineEndToEnd(t *testing.T) {
	// prompt(seed) -> transform -> prompt(describe); the mock server first
	// returns the 3 seed rows, then "analyzed" for every describe call
	srv := llmtest.NewServer(t,
		`{"topic":"go","keep":true}`,
		`{"topic":"js","keep":false}`,
		`{"topic":"rust","keep":true}`,
		"analyzed",
	)
	dir := t.TempDir()

	cfg := config.NewConfig()
	cfg.OutputFolder = dir
	cfg.Version = "1.0"
	cfg.Steps = []config.Step{
		{
			Name:   "seed", // no count: generator default (3) resolved at runtime
			Model:  "ollama:test-model",
			Prompt: "Suggest a topic",
			JSONSchemaRaw: `{
				"type": "object",
				"properties": {"topic": {"type": "string"}, "keep": {"type": "boolean"}},
				"required": ["topic", "keep"],
				"additionalProperties": false
			}`,
			ModelConfig: config.ModelConfig{
				BaseURL: srv.URL,
			},
		},
		{
			Name: "picked",
			JQ:   `select(.keep) | .topic`,
			From: "seed",
		},
		{
			Name:    "describe",
			Model:   "ollama:test-model",
			ForEach: "picked",
			Prompt:  "Describe {{.item}}",
			ModelConfig: config.ModelConfig{
				BaseURL: srv.URL,
			},
		},
	}

	require.NoError(t, utils.PreprocessConfig(cfg))
	require.NoError(t, cfg.Validate())
	require.NoError(t, runner.NewRunner(cfg).Run(context.Background()))

	lines, err := fs.CountLinesInFile(cfg.Steps[2].OutputFilename)
	require.NoError(t, err)
	assert.Equal(t, 2, lines, "2 of 3 seed rows survive the filter")
	assert.Equal(t, 5, srv.CallCount(), "3 seed generations + 2 describe calls")
}

func TestRun_ReadFilesPipelineEndToEnd(t *testing.T) {
	// read a folder of files -> prompt per file; the mock returns one response
	// per file, and the prompt must see each file's content
	srv := llmtest.NewServer(t)
	srv.EchoPrompt = true

	fileDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(fileDir, "a.md"), []byte("alpha-content"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(fileDir, "b.md"), []byte("beta-content"), 0o644))

	dir := t.TempDir()
	cfg := config.NewConfig()
	cfg.OutputFolder = dir
	cfg.Version = "1.0"
	cfg.Steps = []config.Step{
		{
			Name: "docs",
			Read: filepath.Join(fileDir, "*.md"),
		},
		{
			Name:        "summarize",
			Model:       "ollama:test-model",
			ForEach:     "docs",
			Prompt:      "Summarize: {{.item.content}}",
			ModelConfig: config.ModelConfig{BaseURL: srv.URL},
		},
	}

	require.NoError(t, utils.PreprocessConfig(cfg))
	require.NoError(t, cfg.Validate())
	require.NoError(t, runner.NewRunner(cfg).Run(context.Background()))

	out := readOutputLines(t, cfg.Steps[1].OutputFilename)
	require.Len(t, out, 2, "one row per file")
	// echo mode: response == prompt, so content flowed through in sorted order
	assert.Contains(t, out[0], "alpha-content")
	assert.Contains(t, out[1], "beta-content")
	assert.Equal(t, 2, srv.CallCount())
}

func readOutputLines(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var lines []string
	for _, l := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if l != "" {
			lines = append(lines, l)
		}
	}
	return lines
}
