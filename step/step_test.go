package step

import (
	"testing"

	"github.com/mirpo/datamatic/config"
	"github.com/stretchr/testify/assert"
)

func TestNewStepRunner_PromptStep(t *testing.T) {
	step := config.Step{Type: config.PromptStepType}
	runner, err := NewStepRunner(step)

	assert.NoError(t, err)
	assert.IsType(t, &PromptStep{}, runner)
}

func TestNewStepRunner_CliStep(t *testing.T) {
	step := config.Step{Type: config.CliStepType}
	runner, err := NewStepRunner(step)

	assert.NoError(t, err)
	assert.IsType(t, &CliStep{}, runner)
}

func TestNewStepRunner_UnsupportedStep(t *testing.T) {
	step := config.Step{Type: "unknown_type"}
	runner, err := NewStepRunner(step)

	assert.Error(t, err)
	assert.Nil(t, runner)
	assert.EqualError(t, err, "creating step runner for type unknown_type: unsupported step type")
}
