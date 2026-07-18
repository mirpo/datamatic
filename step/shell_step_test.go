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

func shellStep(t *testing.T, run string) (config.Step, string) {
	t.Helper()
	dir := t.TempDir()
	return config.Step{
		Name:           "gen",
		Type:           config.ShellStepType,
		Run:            run,
		WorkDir:        dir,
		OutputFilename: filepath.Join(dir, "out.jsonl"),
	}, dir
}

func TestShellStepRun_PassesWhenOutputCreated(t *testing.T) {
	step, dir := shellStep(t, "echo hi > out.jsonl")

	err := (&ShellStep{}).Run(context.Background(), config.NewConfig(), step, dir)

	require.NoError(t, err)
	_, statErr := os.Stat(step.OutputFilename)
	assert.NoError(t, statErr, "the declared output file exists")
}

func TestShellStepRun_FailsWhenOutputNotCreated(t *testing.T) {
	// the command succeeds but never writes the declared outputFilename
	// (e.g. a typo in the run command) — the step must fail, not silently pass
	step, dir := shellStep(t, "echo hi")

	err := (&ShellStep{}).Run(context.Background(), config.NewConfig(), step, dir)

	require.Error(t, err, "must not silently pass when the output file is missing")
	assert.Contains(t, err.Error(), "gen", "error names the step")
	assert.Contains(t, err.Error(), "out.jsonl", "error names the missing file")
}

func TestShellStepRun_FailsWhenCommandWritesDifferentFile(t *testing.T) {
	// typo: command writes out.json but the step declared out.jsonl
	step, dir := shellStep(t, "echo hi > out.json")

	err := (&ShellStep{}).Run(context.Background(), config.NewConfig(), step, dir)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "out.jsonl")
}
