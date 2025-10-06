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
	tempNeg := -0.1
	tempOver := 1.1

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
		{"Valid Nil Temp", ModelConfig{Temperature: nil, MaxTokens: &maxTokens100, BaseURL: ""}, false, ""},
		{"Invalid Temp Below 0", ModelConfig{Temperature: &tempNeg, MaxTokens: &maxTokens100}, true, "temperature must be between 0 and 1"},
		{"Invalid Temp Above 1", ModelConfig{Temperature: &tempOver, MaxTokens: &maxTokens100}, true, "temperature must be between 0 and 1"},
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
					MaxResults: 10,
				},
				{
					Name:       "step2_cli",
					Type:       PromptStepType,
					Model:      "ollama:dummy",
					Prompt:     "Generate new.",
					MaxResults: DefaultStepMinMaxResults,
				},
				{
					Name:       "step3_default_maxresults",
					Type:       PromptStepType,
					Model:      "lmstudio:dummy",
					Prompt:     "Another prompt.",
					MaxResults: 1,
				},
			},
		}
	}

	t.Run("Valid Config", func(t *testing.T) {
		cfg := validConfig()
		err := cfg.Validate()
		assert.NoError(t, err)

		assert.Equal(t, PromptStepType, cfg.Steps[0].Type)
		assert.Equal(t, 10, cfg.Steps[0].MaxResults)

		assert.Equal(t, PromptStepType, cfg.Steps[1].Type)
		assert.Equal(t, DefaultStepMinMaxResults, cfg.Steps[1].MaxResults)

		assert.Equal(t, PromptStepType, cfg.Steps[2].Type)
		assert.Equal(t, 1, cfg.Steps[2].MaxResults)
	})

	t.Run("Step With Invalid Model Config", func(t *testing.T) {
		cfg := validConfig()
		tempNeg := -0.1
		cfg.Steps[0].ModelConfig.Temperature = &tempNeg
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "step 'step1': model config validation failed: temperature must be between 0 and 1")
	})

	t.Run("MaxResults Defaults Correctly", func(t *testing.T) {
		cfg := validConfig()

		err := cfg.Validate()
		assert.NoError(t, err)

		assert.Equal(t, 10, cfg.Steps[0].MaxResults, "step with MaxResults > 0 should keep its value")
		assert.Equal(t, DefaultStepMinMaxResults, cfg.Steps[1].MaxResults, "step with MaxResults = nil should default")
		assert.Equal(t, 1, cfg.Steps[2].MaxResults, "step with MaxResults < 0 should default")
	})
}

func TestValidateMaxResults(t *testing.T) {
	stepNames := map[string]bool{"foo": true, "bar": true}

	tests := []struct {
		name        string
		input       interface{}
		expectError bool
	}{
		{"nil input", nil, false},
		{"empty string", "", false},
		{"valid step reference", "foo.$length", false},
		{"invalid step reference", "unknown.$length", true},
		{"invalid string", "invalid", true},
		{"zero int", 0, false},
		{"positive int", 5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := &Step{MaxResults: tt.input}
			err := validateMaxResults(step, stepNames)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCLIFilenameValidation_PostPreprocessing(t *testing.T) {
	t.Run("CLI step with exact filename in command passes", func(t *testing.T) {
		cfg := &Config{
			Version:     "1.0",
			RetryConfig: retry.NewDefaultConfig(),
			Steps: []Step{
				{
					Name:           "convert_to_json",
					Type:           CliStepType,
					Cmd:            `echo '{"test": "data"}' > output.json`,
					OutputFilename: "output.json", // exact match
				},
			},
		}

		err := cfg.Validate()
		assert.NoError(t, err, "Should pass because 'output.json' is found in command")
	})

	t.Run("CLI step with relative path in command passes", func(t *testing.T) {
		cfg := &Config{
			Version:     "1.0",
			RetryConfig: retry.NewDefaultConfig(),
			Steps: []Step{
				{
					Name:           "jq_filter",
					Type:           CliStepType,
					Cmd:            `jq -c 'select(.test)' input.json > ./results.jsonl`,
					OutputFilename: "./results.jsonl",
				},
			},
		}

		err := cfg.Validate()
		assert.NoError(t, err, "Should pass because './results.jsonl' is found in command")
	})

	t.Run("CLI step with absolute path in command works", func(t *testing.T) {
		cfg := &Config{
			Version:     "1.0",
			RetryConfig: retry.NewDefaultConfig(),
			Steps: []Step{
				{
					Name:           "full_path_cmd",
					Type:           CliStepType,
					Cmd:            `duckdb -c "COPY (...) TO '/abs/path/to/output/data.json' (FORMAT JSON);"`,
					OutputFilename: "/abs/path/to/output/data.json",
				},
			},
		}

		err := cfg.Validate()
		assert.NoError(t, err, "Should pass because full path matches exactly")
	})

	t.Run("CLI step without filename in command fails", func(t *testing.T) {
		cfg := &Config{
			Version:     "1.0",
			RetryConfig: retry.NewDefaultConfig(),
			Steps: []Step{
				{
					Name:           "download_only",
					Type:           CliStepType,
					Cmd:            `curl -o different_name.json https://api.example.com/data`,
					OutputFilename: "/abs/path/to/output/expected.json",
				},
			},
		}

		err := cfg.Validate()
		assert.Error(t, err, "Should fail because neither 'expected.json' nor full path is in command")
		assert.Contains(t, err.Error(), "output filename should match output result of external CLI")
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
