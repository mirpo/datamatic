package step

import (
	"testing"

	"github.com/mirpo/datamatic/config"
	"github.com/mirpo/datamatic/llm"
	"github.com/stretchr/testify/assert"
)

func TestNewStepRunner_PromptStep(t *testing.T) {
	step := config.Step{Type: config.PromptStepType}
	runner, err := NewStepRunner(step)

	assert.NoError(t, err)
	assert.IsType(t, &PromptStep{}, runner)
}

func TestNewStepRunner_ShellStep(t *testing.T) {
	step := config.Step{Type: config.ShellStepType}
	runner, err := NewStepRunner(step)

	assert.NoError(t, err)
	assert.IsType(t, &ShellStep{}, runner)
}

func TestNewStepRunner_UnsupportedStep(t *testing.T) {
	step := config.Step{Type: "unknown_type"}
	runner, err := NewStepRunner(step)

	assert.Error(t, err)
	assert.Nil(t, runner)
	assert.EqualError(t, err, "unsupported step type")
}

func TestNewProviderConfigFromStep(t *testing.T) {
	temp := 0.7
	maxTokens := 1000

	step := config.Step{
		ModelConfig: config.ModelConfig{
			BaseURL:       "http://example.com/api",
			ModelProvider: llm.ProviderOllama,
			ModelName:     "llama3",
			Temperature:   &temp,
			MaxTokens:     &maxTokens,
		},
	}

	result := newProviderConfigFromStep(step, 30)

	assert.Equal(t, "http://example.com/api", result.BaseURL)
	assert.Equal(t, llm.ProviderOllama, result.ProviderType)
	assert.Equal(t, "llama3", result.ModelName)
	assert.Equal(t, "token", result.AuthToken)
	assert.Equal(t, 30, result.HTTPTimeout)
	assert.Equal(t, &temp, result.Temperature)
	assert.Equal(t, &maxTokens, result.MaxTokens)
}
