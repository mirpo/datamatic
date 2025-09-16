package utils

import (
	"testing"

	"github.com/mirpo/datamatic/config"
	"github.com/stretchr/testify/assert"
)

func TestSetStepType(t *testing.T) {
	tests := []struct {
		name         string
		step         config.Step
		expectedType config.StepType
		wantErr      bool
		errMsg       string
	}{
		{"Prompt Only", config.Step{Prompt: "a prompt"}, config.PromptStepType, false, ""},
		{"Cmd Only", config.Step{Cmd: "a command"}, config.CliStepType, false, ""},
		{"Both", config.Step{Prompt: "a prompt", Cmd: "a command"}, config.UnknownStepType, true, "either 'prompt' or 'cmd' should be defined, not both"},
		{"Neither", config.Step{}, config.UnknownStepType, true, "either 'prompt' or 'cmd' must be defined"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := tt.step
			err := setStepType(&step)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedType, step.Type)
			}
		})
	}
}

func TestPreprocessConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *config.Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid config with prompt step",
			config: &config.Config{
				Steps: []config.Step{
					{Name: "test", Prompt: "test prompt", Model: "ollama:llama3.2"},
				},
			},
			wantErr: false,
		},
		{
			name: "Valid config with cli step",
			config: &config.Config{
				Steps: []config.Step{
					{Name: "test", Cmd: "test command"},
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid step type",
			config: &config.Config{
				Steps: []config.Step{
					{Name: "test"}, // No prompt or cmd
				},
			},
			wantErr: true,
			errMsg:  "either 'prompt' or 'cmd' must be defined",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := PreprocessConfig(tt.config, false)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
				// Verify step types are set
				for _, step := range tt.config.Steps {
					assert.NotEqual(t, config.UnknownStepType, step.Type)
				}
			}
		})
	}
}
