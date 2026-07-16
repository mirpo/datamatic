package runner_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mirpo/datamatic/config"
	"github.com/mirpo/datamatic/fs"
	"github.com/mirpo/datamatic/internal/llmtest"
	"github.com/mirpo/datamatic/runner"
	"github.com/mirpo/datamatic/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestValidateConfigFromExamples(t *testing.T) {
	matches, err := filepath.Glob("../examples/**/**/*.yaml")
	assert.NoError(t, err, "Failed to glob YAML files")

	assert.NotEmpty(t, matches, "No YAML files found in examples directory")

	for _, path := range matches {
		t.Logf("Testing file: %s", path)

		testName := filepath.ToSlash(path)

		t.Run(testName, func(t *testing.T) {
			setupTestEnvVars(t, path)

			data, err := os.ReadFile(path)
			assert.NoError(t, err, "Failed to read file: %s", path)

			expandedYaml, err := utils.ExpandEnv(string(data), nil)
			assert.NoError(t, err, "Failed to expand env vars for file: %s", path)

			var cfg config.Config
			err = yaml.Unmarshal([]byte(expandedYaml), &cfg)
			assert.NoError(t, err, "Failed to unmarshal YAML: %s", path)

			cfg.OutputFolder = "test"
			cfg.SkipCliWarning = true

			err = utils.PreprocessConfig(&cfg)
			assert.NoError(t, err, "Preprocessing failed for file: %s", path)

			err = cfg.Validate()
			assert.NoError(t, err, "Validation failed for file: %s", path)
		})
	}
}

// setupTestEnvVars sets up environment variables needed for specific test configs
func setupTestEnvVars(t *testing.T, path string) {
	// For the workdir-multi-stage-pipeline example that uses env vars
	if filepath.Base(filepath.Dir(path)) == "18. workdir-multi-stage-pipeline" {
		os.Setenv("REQUIRED_FILE", "prompts.csv")
		os.Setenv("DOWNLOAD_DIR", "downloads")
		os.Setenv("PROVIDER", "ollama")
		os.Setenv("MODEL", "llama3.2")
		t.Cleanup(func() {
			os.Unsetenv("REQUIRED_FILE")
			os.Unsetenv("DOWNLOAD_DIR")
			os.Unsetenv("PROVIDER")
			os.Unsetenv("MODEL")
		})
	}
}

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
			Name:       "seed",
			Model:      "ollama:test-model",
			MaxResults: 3,
			Prompt:     "Suggest a topic",
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
			Name:       "describe",
			Model:      "ollama:test-model",
			MaxResults: "picked.$length",
			Prompt:     "Describe {{.picked}}",
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
