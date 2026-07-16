package step

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/mirpo/datamatic/config"
	"github.com/mirpo/datamatic/fs"
	"github.com/mirpo/datamatic/internal/llmtest"
	"github.com/mirpo/datamatic/jsonschema"
	"github.com/mirpo/datamatic/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const titleSchema = `{
	"type": "object",
	"properties": {"title": {"type": "string"}},
	"required": ["title"],
	"additionalProperties": false
}`

func testSchema(t *testing.T, raw string) jsonschema.Schema {
	t.Helper()
	s, err := jsonschema.LoadSchema(raw)
	require.NoError(t, err)
	return *s
}

func promptStepConfig(t *testing.T, srvURL string) (*config.Config, config.Step, string) {
	t.Helper()
	dir := t.TempDir()

	cfg := config.NewConfig()
	cfg.OutputFolder = dir

	step := config.Step{
		Name:           "gen",
		Type:           config.PromptStepType,
		Prompt:         "generate something",
		ResolvedCount:  3,
		OutputFilename: filepath.Join(dir, "gen.jsonl"),
		ModelConfig: config.ModelConfig{
			ModelProvider: llm.ProviderOllama,
			ModelName:     "test-model",
			BaseURL:       srvURL,
		},
	}
	return cfg, step, dir
}

func countLines(t *testing.T, path string) int {
	t.Helper()
	lines, err := fs.CountLinesInFile(path)
	require.NoError(t, err)
	return lines
}

func TestPromptStepRun_WritesResolvedCountLines(t *testing.T) {
	srv := llmtest.NewServer(t, "hello world")
	cfg, step, dir := promptStepConfig(t, srv.URL)

	err := (&PromptStep{}).Run(context.Background(), cfg, step, dir)

	require.NoError(t, err)
	assert.Equal(t, 3, countLines(t, step.OutputFilename))
	assert.Equal(t, 3, srv.CallCount())
}

func TestPromptStepRun_UnknownRefStepReturnsError(t *testing.T) {
	srv := llmtest.NewServer(t, "never reached")
	cfg, step, dir := promptStepConfig(t, srv.URL)
	step.Prompt = "use {{.ghost.field}}" // no step named "ghost" in cfg.Steps

	err := (&PromptStep{}).Run(context.Background(), cfg, step, dir)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "ghost")
	assert.Equal(t, 0, srv.CallCount())
}

func TestPromptStepRun_RefValuesReadFailureFailsStep(t *testing.T) {
	srv := llmtest.NewServer(t, "never reached")
	cfg, step, dir := promptStepConfig(t, srv.URL)

	// "prev" exists in config but its output file does not exist on disk
	cfg.Steps = []config.Step{
		{Name: "prev", Type: config.PromptStepType, OutputFilename: filepath.Join(dir, "missing.jsonl")},
	}
	step.Prompt = "use {{.prev}}"

	err := (&PromptStep{}).Run(context.Background(), cfg, step, dir)

	require.Error(t, err, "must fail, not send '<no value>' prompts to the LLM")
	assert.Contains(t, err.Error(), "prev")
	assert.Equal(t, 0, srv.CallCount(), "no LLM call with a broken prompt")
}

func TestPromptStepRun_MissingImagesFailsStep(t *testing.T) {
	srv := llmtest.NewServer(t, "never reached")
	cfg, step, dir := promptStepConfig(t, srv.URL)
	step.ImagePath = filepath.Join(dir, "no-such-dir", "*.jpg")

	err := (&PromptStep{}).Run(context.Background(), cfg, step, dir)

	require.Error(t, err)
	assert.Equal(t, 0, srv.CallCount())
}

func TestPromptStepRun_FailsAfterRepeatedInvalidResponses(t *testing.T) {
	// server always returns JSON that violates the schema (missing "title")
	srv := llmtest.NewServer(t, `{"wrong": true}`)
	cfg, step, dir := promptStepConfig(t, srv.URL)
	step.JSONSchema = testSchema(t, titleSchema)

	err := (&PromptStep{}).Run(context.Background(), cfg, step, dir)

	require.Error(t, err, "must not loop forever on a persistently invalid model")
	assert.Contains(t, err.Error(), "invalid response")
	assert.Equal(t, cfg.RetryConfig.MaxAttempts, srv.CallCount(),
		"one LLM call per allowed attempt, then stop")
}

func TestPromptStepRun_RecoversAfterTransientInvalidResponse(t *testing.T) {
	// first response invalid, then valid ones — step must succeed
	srv := llmtest.NewServer(t, `{"wrong": true}`, `{"title": "ok"}`, `{"title": "ok2"}`, `{"title": "ok3"}`)
	cfg, step, dir := promptStepConfig(t, srv.URL)
	step.JSONSchema = testSchema(t, titleSchema)

	err := (&PromptStep{}).Run(context.Background(), cfg, step, dir)

	require.NoError(t, err)
	assert.Equal(t, 3, countLines(t, step.OutputFilename))
	assert.Equal(t, 4, srv.CallCount()) // 1 failed + 3 good
}
