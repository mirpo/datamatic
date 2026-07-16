package config

import (
	"testing"
	"time"

	"github.com/mirpo/datamatic/retry"
	"github.com/stretchr/testify/assert"
)

func TestValidateUrl(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		errMsg  string
	}{
		{"Valid HTTP", "http://localhost:11434", false, ""},
		{"Valid HTTPS", "https://api.example.com", false, ""},
		{"Valid With Path", "http://example.com/api/v1", false, ""},
		{"Valid With Query", "http://example.com?query=test", false, ""},
		{"Invalid Format", "not a url", true, "invalid URL: parse"},
		{"Missing Scheme", "localhost:11434", true, "invalid URL: missing scheme or host"},
		{"Missing Host", "http://", true, "invalid URL: missing scheme or host"},
		{"Empty", "", true, "invalid URL: parse"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateURL(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateModelConfig(t *testing.T) {
	temp0_5 := 0.5
	temp1_0 := 1.0
	temp1_5 := 1.5
	temp2_0 := 2.0
	tempNeg := -0.1
	tempOver := 2.1

	maxTokens100 := 100
	maxTokensNeg := -1
	maxTokensLarge := 9999

	tests := []struct {
		name    string
		config  ModelConfig
		wantErr bool
		errMsg  string
	}{
		{"Valid Mid", ModelConfig{Temperature: &temp0_5, MaxTokens: &maxTokens100, BaseURL: "http://example.com"}, false, ""},
		{"Valid Max", ModelConfig{Temperature: &temp1_0, MaxTokens: &maxTokensLarge, BaseURL: "https://test.org/api"}, false, ""},
		{"Valid Above 1 (OpenAI range)", ModelConfig{Temperature: &temp1_5, MaxTokens: &maxTokens100}, false, ""},
		{"Valid Upper Bound 2.0", ModelConfig{Temperature: &temp2_0, MaxTokens: &maxTokens100}, false, ""},
		{"Valid Nil Temp", ModelConfig{Temperature: nil, MaxTokens: &maxTokens100, BaseURL: ""}, false, ""},
		{"Invalid Temp Below 0", ModelConfig{Temperature: &tempNeg, MaxTokens: &maxTokens100}, true, "temperature must be between 0 and 2"},
		{"Invalid Temp Above 2", ModelConfig{Temperature: &tempOver, MaxTokens: &maxTokens100}, true, "temperature must be between 0 and 2"},
		{"Invalid MaxTokens Negative", ModelConfig{Temperature: &temp0_5, MaxTokens: &maxTokensNeg}, true, "maxTokens must be > 0"},
		{"Invalid BaseUrl", ModelConfig{Temperature: &temp0_5, MaxTokens: &maxTokens100, BaseURL: "not a url"}, true, "invalid baseUrl"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateModelConfig(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	validConfig := func() Config {
		temp0_5 := 0.5
		maxTokens500 := 500

		return Config{
			Version:      "1.0",
			OutputFolder: "my_output",
			RetryConfig:  retry.NewDefaultConfig(),
			Steps: []Step{
				{
					Name:   "step1",
					Type:   PromptStepType,
					Model:  "ollama:llama3.1",
					Prompt: "Generate something.",
					ModelConfig: ModelConfig{
						Temperature: &temp0_5,
						MaxTokens:   &maxTokens500,
						BaseURL:     "http://localhost:11434",
					},
					Count: 10,
				},
				{
					Name:   "step2_cli",
					Type:   PromptStepType,
					Model:  "ollama:dummy",
					Prompt: "Generate new.",
					Count:  DefaultStepCount,
				},
				{
					Name:   "step3_count_one",
					Type:   PromptStepType,
					Model:  "lmstudio:dummy",
					Prompt: "Another prompt.",
					Count:  1,
				},
			},
		}
	}

	t.Run("Valid Config", func(t *testing.T) {
		cfg := validConfig()
		err := cfg.Validate()
		assert.NoError(t, err)

		assert.Equal(t, PromptStepType, cfg.Steps[0].Type)
		assert.Equal(t, 10, cfg.Steps[0].Count)

		assert.Equal(t, PromptStepType, cfg.Steps[1].Type)
		assert.Equal(t, DefaultStepCount, cfg.Steps[1].Count)

		assert.Equal(t, PromptStepType, cfg.Steps[2].Type)
		assert.Equal(t, 1, cfg.Steps[2].Count)
	})

	t.Run("Step With Invalid Model Config", func(t *testing.T) {
		cfg := validConfig()
		tempNeg := -0.1
		cfg.Steps[0].ModelConfig.Temperature = &tempNeg
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "step 'step1': model config validation failed: temperature must be between 0 and 2")
	})
}

func TestCLIFilenameValidation_PostPreprocessing(t *testing.T) {
	t.Run("CLI step with exact filename in command passes", func(t *testing.T) {
		cfg := &Config{
			Version:     "1.0",
			RetryConfig: retry.NewDefaultConfig(),
			Steps: []Step{
				{
					Name:           "convert_to_json",
					Type:           ShellStepType,
					Run:            `echo '{"test": "data"}' > output.json`,
					OutputFilename: "output.json", // exact match
				},
			},
		}

		err := cfg.Validate()
		assert.NoError(t, err, "Should pass because 'output.json' is found in command")
	})

	t.Run("Shell step with relative path in command passes", func(t *testing.T) {
		cfg := &Config{
			Version:     "1.0",
			RetryConfig: retry.NewDefaultConfig(),
			Steps: []Step{
				{
					Name:           "jq_filter",
					Type:           ShellStepType,
					Run:            `jq -c 'select(.test)' input.json > ./results.jsonl`,
					OutputFilename: "./results.jsonl",
				},
			},
		}

		err := cfg.Validate()
		assert.NoError(t, err, "Should pass because './results.jsonl' is found in command")
	})

	t.Run("Shell step with absolute path in command works", func(t *testing.T) {
		cfg := &Config{
			Version:     "1.0",
			RetryConfig: retry.NewDefaultConfig(),
			Steps: []Step{
				{
					Name:           "full_path_cmd",
					Type:           ShellStepType,
					Run:            `duckdb -c "COPY (...) TO '/abs/path/to/output/data.json' (FORMAT JSON);"`,
					OutputFilename: "/abs/path/to/output/data.json",
				},
			},
		}

		err := cfg.Validate()
		assert.NoError(t, err, "Should pass because full path matches exactly")
	})

	t.Run("Shell step without filename in command warns but passes", func(t *testing.T) {
		cfg := &Config{
			Version:        "1.0",
			RetryConfig:    retry.NewDefaultConfig(),
			SkipCliWarning: true,
			Steps: []Step{
				{
					Name:           "download_only",
					Type:           ShellStepType,
					Run:            `./scripts/fetch.sh`, // writes expected.json internally
					OutputFilename: "/abs/path/to/output/expected.json",
				},
			},
		}

		err := cfg.Validate()
		assert.NoError(t, err, "filename-in-command is a heuristic; must warn, not fail")
	})
}

func TestValidateRetryConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  retry.Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid Config",
			config: retry.Config{
				MaxAttempts:       3,
				InitialDelay:      1 * time.Second,
				MaxDelay:          10 * time.Second,
				BackoffMultiplier: 2.0,
				Enabled:           true,
			},
			wantErr: false,
		},
		{
			name: "Zero MaxAttempts",
			config: retry.Config{
				MaxAttempts:       0,
				InitialDelay:      1 * time.Second,
				MaxDelay:          10 * time.Second,
				BackoffMultiplier: 2.0,
				Enabled:           true,
			},
			wantErr: true,
			errMsg:  "maxAttempts must be greater than 0",
		},
		{
			name: "Negative MaxAttempts",
			config: retry.Config{
				MaxAttempts:       -1,
				InitialDelay:      1 * time.Second,
				MaxDelay:          10 * time.Second,
				BackoffMultiplier: 2.0,
				Enabled:           true,
			},
			wantErr: true,
			errMsg:  "maxAttempts must be greater than 0",
		},
		{
			name: "Zero InitialDelay",
			config: retry.Config{
				MaxAttempts:       3,
				InitialDelay:      0,
				MaxDelay:          10 * time.Second,
				BackoffMultiplier: 2.0,
				Enabled:           true,
			},
			wantErr: true,
			errMsg:  "initialDelay must be greater than 0",
		},
		{
			name: "Negative InitialDelay",
			config: retry.Config{
				MaxAttempts:       3,
				InitialDelay:      -1 * time.Second,
				MaxDelay:          10 * time.Second,
				BackoffMultiplier: 2.0,
				Enabled:           true,
			},
			wantErr: true,
			errMsg:  "initialDelay must be greater than 0",
		},
		{
			name: "MaxDelay Less Than InitialDelay",
			config: retry.Config{
				MaxAttempts:       3,
				InitialDelay:      10 * time.Second,
				MaxDelay:          5 * time.Second,
				BackoffMultiplier: 2.0,
				Enabled:           true,
			},
			wantErr: true,
			errMsg:  "maxDelay must be greater than or equal to initialDelay",
		},
		{
			name: "BackoffMultiplier Less Than 1",
			config: retry.Config{
				MaxAttempts:       3,
				InitialDelay:      1 * time.Second,
				MaxDelay:          10 * time.Second,
				BackoffMultiplier: 0.5,
				Enabled:           true,
			},
			wantErr: true,
			errMsg:  "backoffMultiplier must be greater than or equal to 1.0",
		},
		{
			name: "Edge Case - Equal Delays",
			config: retry.Config{
				MaxAttempts:       1,
				InitialDelay:      5 * time.Second,
				MaxDelay:          5 * time.Second,
				BackoffMultiplier: 1.0,
				Enabled:           false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRetryConfig(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigValidateRetryConfig(t *testing.T) {
	t.Run("Config with invalid retry config fails validation", func(t *testing.T) {
		cfg := &Config{
			Version: "1.0",
			RetryConfig: retry.Config{
				MaxAttempts:       3,                // Valid
				InitialDelay:      -1 * time.Second, // Invalid - negative delay
				MaxDelay:          10 * time.Second,
				BackoffMultiplier: 2.0,
				Enabled:           true,
			},
			Steps: []Step{},
		}

		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "retry config validation failed: initialDelay must be greater than 0")
	})

	t.Run("Config with valid retry config passes validation", func(t *testing.T) {
		cfg := &Config{
			Version:     "1.0",
			RetryConfig: retry.NewDefaultConfig(),
			Steps:       []Step{},
		}

		err := cfg.Validate()
		assert.NoError(t, err)
	})
}
