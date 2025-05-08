package config

import (
	"fmt"
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

func TestGetStepByName(t *testing.T) {
	step1 := Step{Name: "step1"}
	step2 := Step{Name: "step2"}
	config := &Config{Steps: []Step{step1, step2}}

	t.Run("Step exists", func(t *testing.T) {
		step := config.GetStepByName("step1")
		assert.NotNil(t, step)
		assert.Equal(t, "step1", step.Name)
	})

	t.Run("Step does not exist", func(t *testing.T) {
		step := config.GetStepByName("nonexistent")
		assert.Nil(t, step)
	})
}

func TestConvertJSONValueToStringReflected(t *testing.T) {
	tests := []struct {
		value    interface{}
		expected string
	}{
		{nil, ""},
		{"string", "string"},
		{3.14, "3.14"},
		{3, "3"},
		{true, "true"},
		{[]interface{}{"a", "b", "c"}, "a, b, c"},
		{map[string]interface{}{"key": "value"}, `{"key":"value"}`},
		{map[string]interface{}{"foo": 42, "bar": true}, `{"bar":true,"foo":42}`},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt.value), func(t *testing.T) {
			result := convertJSONValueToStringReflected(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetFieldAsString(t *testing.T) {
	tests := []struct {
		data     map[string]interface{}
		key      string
		expected string
		err      error
	}{
		{map[string]interface{}{"key1": "value1"}, "key1", "value1", nil},
		{map[string]interface{}{"key1": 42}, "key1", "42", nil},
		{map[string]interface{}{"key1": true}, "key1", "true", nil},
		{map[string]interface{}{}, "nonexistent", "", fmt.Errorf("key 'nonexistent' not found")},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("Key %s", tt.key), func(t *testing.T) {
			result, err := GetFieldAsString(tt.data, tt.key)
			if tt.err != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
