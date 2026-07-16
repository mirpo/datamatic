package step

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mirpo/datamatic/config"
	"github.com/mirpo/datamatic/jq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustCompile(t *testing.T, program string) *jq.Program {
	t.Helper()
	p, err := jq.Compile(program)
	require.NoError(t, err)
	return p
}

// transformFixture writes source lines to disk and returns a ready cfg + step.
func transformFixture(t *testing.T, sourceType config.StepType, sourceLines string, program string, limit int) (*config.Config, config.Step) {
	t.Helper()
	dir := t.TempDir()

	srcPath := filepath.Join(dir, "src.jsonl")
	require.NoError(t, os.WriteFile(srcPath, []byte(sourceLines), 0o644))

	cfg := config.NewConfig()
	cfg.OutputFolder = dir
	cfg.Steps = []config.Step{
		{Name: "src", Type: sourceType, OutputFilename: srcPath},
	}

	step := config.Step{
		Name:           "tr",
		Type:           config.TransformStepType,
		From:           "src",
		JQ:             program,
		JQProgram:      mustCompile(t, program),
		Limit:          limit,
		OutputFilename: filepath.Join(dir, "tr.jsonl"),
	}
	return cfg, step
}

func readOutput(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
}

func TestTransformStepRun_FilterAndReshape(t *testing.T) {
	cfg, step := transformFixture(t, config.ShellStepType,
		`{"keep":true,"v":1}`+"\n"+`{"keep":false,"v":2}`+"\n"+`{"keep":true,"v":3}`+"\n",
		`select(.keep) | {value: .v}`, 0)

	err := (&TransformStep{}).Run(context.Background(), cfg, step, cfg.OutputFolder)

	require.NoError(t, err)
	assert.Equal(t, []string{`{"value":1}`, `{"value":3}`}, readOutput(t, step.OutputFilename))
}

func TestTransformStepRun_FanOut(t *testing.T) {
	cfg, step := transformFixture(t, config.ShellStepType,
		`{"items":["a","b"]}`+"\n"+`{"items":["c"]}`+"\n",
		`.items[]`, 0)

	err := (&TransformStep{}).Run(context.Background(), cfg, step, cfg.OutputFolder)

	require.NoError(t, err)
	assert.Equal(t, []string{`"a"`, `"b"`, `"c"`}, readOutput(t, step.OutputFilename))
}

func TestTransformStepRun_LimitCapsOutput(t *testing.T) {
	cfg, step := transformFixture(t, config.ShellStepType,
		`{"items":[1,2,3,4,5]}`+"\n",
		`.items[]`, 2)

	err := (&TransformStep{}).Run(context.Background(), cfg, step, cfg.OutputFolder)

	require.NoError(t, err)
	assert.Len(t, readOutput(t, step.OutputFilename), 2)
}

func TestTransformStepRun_PromptSourceUsesResponse(t *testing.T) {
	// prompt-step lines carry the LineEntity envelope; jq must see only .response
	cfg, step := transformFixture(t, config.PromptStepType,
		`{"id":"1","format":"json","prompt":"p","response":{"title":"hello"}}`+"\n",
		`.title`, 0)

	err := (&TransformStep{}).Run(context.Background(), cfg, step, cfg.OutputFolder)

	require.NoError(t, err)
	assert.Equal(t, []string{`"hello"`}, readOutput(t, step.OutputFilename))
}

func TestTransformStepRun_JQErrorFailsStep(t *testing.T) {
	cfg, step := transformFixture(t, config.ShellStepType,
		`{"a":"str"}`+"\n",
		`.a + 1`, 0)

	err := (&TransformStep{}).Run(context.Background(), cfg, step, cfg.OutputFolder)

	assert.Error(t, err)
}

func TestTransformStepRun_UnknownFromStepFails(t *testing.T) {
	cfg, step := transformFixture(t, config.ShellStepType, `{}`+"\n", `.`, 0)
	step.From = "ghost"

	err := (&TransformStep{}).Run(context.Background(), cfg, step, cfg.OutputFolder)

	assert.ErrorContains(t, err, "ghost")
}
