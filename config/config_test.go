package config

import (
	"testing"

	"github.com/mirpo/datamatic/llm"
	"github.com/stretchr/testify/assert"
)

func floatPtr(f float64) *float64 {
	return &f
}

func intPtr(i int) *int {
	return &i
}

func TestNewConfig(t *testing.T) {
	cfg := NewConfig()

	assert.NotNil(t, cfg)
	assert.Equal(t, "", cfg.ConfigFile)
	assert.False(t, cfg.Verbose)
	assert.True(t, cfg.LogPretty)
	assert.Equal(t, "dataset", cfg.OutputFolder)
	assert.Equal(t, 300, cfg.HTTPTimeout)
	assert.Equal(t, "", cfg.Version)
	assert.Nil(t, cfg.Steps)
}

func TestGetProviderConfig(t *testing.T) {
	tests := []struct {
		name        string
		step        Step
		httpTimeout int
		expected    llm.ProviderConfig
	}{
		{
			name: "Full config with temperature and max tokens",
			step: Step{
				ModelConfig: ModelConfig{
					BaseURL:       "http://example.com/api",
					ModelProvider: llm.ProviderOllama,
					ModelName:     "llama3",
					Temperature:   floatPtr(0.7),
					MaxTokens:     intPtr(1000),
				},
			},
			httpTimeout: 30,
			expected: llm.ProviderConfig{
				ProviderType: llm.ProviderOllama,
				BaseURL:      "http://example.com/api",
				ModelName:    "llama3",
				AuthToken:    "token",
				Temperature:  floatPtr(0.7),
				MaxTokens:    intPtr(1000),
				HTTPTimeout:  30,
			},
		},
		{
			name: "Config without temperature and max tokens",
			step: Step{
				ModelConfig: ModelConfig{
					BaseURL:       "http://another-api.com",
					ModelProvider: llm.ProviderOllama,
					ModelName:     "command",
					Temperature:   nil,
					MaxTokens:     nil,
				},
			},
			httpTimeout: 60,
			expected: llm.ProviderConfig{
				ProviderType: llm.ProviderOllama,
				BaseURL:      "http://another-api.com",
				ModelName:    "command",
				AuthToken:    "token",
				Temperature:  nil,
				MaxTokens:    nil,
				HTTPTimeout:  60,
			},
		},
		{
			name: "Different http timeout",
			step: Step{
				ModelConfig: ModelConfig{
					BaseURL:       "http://api3.org",
					ModelProvider: llm.ProviderLmStudio,
					ModelName:     "gpt-3.5-turbo",
					Temperature:   floatPtr(0.1),
					MaxTokens:     intPtr(500),
				},
			},
			httpTimeout: 10,
			expected: llm.ProviderConfig{
				ProviderType: llm.ProviderLmStudio,
				BaseURL:      "http://api3.org",
				ModelName:    "gpt-3.5-turbo",
				AuthToken:    "token",
				Temperature:  floatPtr(0.1),
				MaxTokens:    intPtr(500),
				HTTPTimeout:  10,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.step.GetProviderConfig(tt.httpTimeout)
			assert.Equal(t, tt.expected, actual, "ProviderConfig should match expected")
		})
	}
}
