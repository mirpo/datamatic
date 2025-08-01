package config

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/mirpo/datamatic/llm"
	"github.com/stretchr/testify/assert"
)

func TestIsValidFName(t *testing.T) {
	tests := []struct {
		name      string
		inputName string
		wantErr   bool
		errMsg    string
	}{
		{"Valid Simple", "myfile.txt", false, ""},
		{"Valid With Dot", "my.file.name", false, ""},
		{"Valid With Hyphen Underscore", "my-file_1.name", false, ""},
		{"Valid Just Dot", ".", false, ""},
		{"Valid Long Name", strings.Repeat("a", 255), false, ""},
		{"Empty", "", true, "filename cannot be empty"},
		{"Too Long", strings.Repeat("a", 256), true, "filename exceeds the maximum length of 255 characters"},
		{"Contains <", "my<file", true, "filename contains invalid characters"},
		{"Contains >", "my>file", true, "filename contains invalid characters"},
		{"Contains :", "my:file", true, "filename contains invalid characters"},
		{"Contains \"", "my\"file", true, "filename contains invalid characters"},
		{"Contains /", "my/file", true, "filename contains invalid characters"},
		{"Contains \\", "my\\file", true, "filename contains invalid characters"},
		{"Contains |", "my|file", true, "filename contains invalid characters"},
		{"Contains ?", "my?file", true, "filename contains invalid characters"},
		{"Contains *", "my*file", true, "filename contains invalid characters"},
		{"Ends with Space", "myfile ", true, "filename cannot end with a space or a period"},
		{"Ends with Period", "myfile.", true, "filename cannot end with a space or a period"},
		{"Ends with Period and Space", "myfile .", true, "filename cannot end with a space or a period"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := isValidName(tt.inputName)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetStepType(t *testing.T) {
	tests := []struct {
		name         string
		step         Step
		expectedType StepType
		wantErr      bool
		errMsg       string
	}{
		{"Prompt Only", Step{Prompt: "a prompt"}, PromptStepType, false, ""},
		{"Cmd Only", Step{Cmd: "a command"}, CliStepType, false, ""},
		{"Both", Step{Prompt: "a prompt", Cmd: "a command"}, UnknownStepType, true, "either 'prompt' or 'cmd' should be defined, not both"},
		{"Neither", Step{}, UnknownStepType, true, "either 'prompt' or 'cmd' must be defined"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stepType, err := getStepType(tt.step)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Equal(t, UnknownStepType, stepType)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedType, stepType)
			}
		})
	}
}

func TestGetModelDetails(t *testing.T) {
	tests := []struct {
		name             string
		step             Step
		expectedProvider llm.ProviderType
		expectedName     string
		wantErr          bool
		errMsg           string
	}{
		{"Valid Ollama", Step{Model: "ollama:llama3.1"}, llm.ProviderOllama, "llama3.1", false, ""},
		{"Valid LmStudio", Step{Model: "lmstudio:meta-llama/Llama-3-8B-Instruct"}, llm.ProviderLmStudio, "meta-llama/Llama-3-8B-Instruct", false, ""},
		{"Valid OpenAI", Step{Model: "openai:gpt-4"}, llm.ProviderOpenAI, "gpt-4", false, ""},
		{"Empty Model", Step{Model: ""}, llm.ProviderUnknown, "", true, "model definition can't be empty"},
		{"Missing Colon", Step{Model: "ollamallama3.1"}, llm.ProviderUnknown, "", true, "model should follow pattern"},
		{"Extra Colon", Step{Model: "ollama:model:extra"}, llm.ProviderOllama, "model:extra", false, ""},
		{"Empty Model Name", Step{Model: "ollama:"}, llm.ProviderUnknown, "", true, "model name can't be empty"},
		{"Unsupported Provider", Step{Model: "unsupported:model"}, llm.ProviderUnknown, "", true, "unsupported provider"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, name, err := getModelDetails(tt.step)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Equal(t, llm.ProviderUnknown, provider)
				assert.Empty(t, name)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedProvider, provider)
				assert.Equal(t, tt.expectedName, name)
			}
		})
	}
}

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

func TestValidateAndAbsOutputFolder(t *testing.T) {
	tests := []struct {
		name            string
		outputFolder    string
		expectedAbsPath string
		wantErr         bool
		errMsg          string
	}{
		{"Valid Folder", "output", "", false, ""},
		{"Valid Folder With Dot", "output.folder", "", false, ""},
		{"Valid Folder Just Dot", ".", "", false, ""},
		{"Empty", "", "", true, "output folder is required"},
		{"Invalid Chars", "output<folder>", "", true, "invalid output folder name"},
		{"Ends With Space", "output ", "", true, "invalid output folder name"},
	}

	for i := range tests {
		if !tests[i].wantErr {
			absPath, err := filepath.Abs(tests[i].outputFolder)
			assert.NoError(t, err, "Failed to calculate absolute path for test case setup")
			tests[i].expectedAbsPath = filepath.Clean(absPath)
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			absPath, err := validateAndAbsOutputFolder(tt.outputFolder)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Empty(t, absPath, "Should return empty string on error")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedAbsPath, filepath.Clean(absPath), "Returned path should be the expected absolute path")
			}
		})
	}
}

func TestGetFullOutputPath(t *testing.T) {
	baseOutputFolder := "test_output"
	absBaseOutputFolder, err := filepath.Abs(baseOutputFolder)
	assert.NoError(t, err, "failed to get absolute path for base test folder")
	absDotFolder, err := filepath.Abs(".")
	assert.NoError(t, err, "failed to get absolute path for '.' folder")

	tests := []struct {
		name                 string
		step                 Step
		absoluteOutputFolder string
		expectedAbsPath      string
		wantErr              bool
		errMsg               string
	}{
		{"Default Filename No Extension", Step{Name: "step1"}, absBaseOutputFolder, filepath.Join(absBaseOutputFolder, "step1.jsonl"), false, ""},
		{"Default Filename With Extension", Step{Name: "step2.jsonl"}, absBaseOutputFolder, filepath.Join(absBaseOutputFolder, "step2.jsonl"), false, ""},
		{"Explicit Filename No Extension", Step{Name: "step3_name", OutputFilename: "explicit_file"}, absBaseOutputFolder, filepath.Join(absBaseOutputFolder, "explicit_file.jsonl"), false, ""},
		{"Explicit Filename With Extension", Step{Name: "step4_name", OutputFilename: "explicit_file.jsonl"}, absBaseOutputFolder, filepath.Join(absBaseOutputFolder, "explicit_file.jsonl"), false, ""},
		{"Output Folder Is Root (Simulated)", Step{Name: "step5"}, string(filepath.Separator), filepath.Join(string(filepath.Separator), "step5.jsonl"), false, ""},
		{"Output Folder Is Dot Absolute", Step{Name: "step6"}, absDotFolder, filepath.Join(absDotFolder, "step6.jsonl"), false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expected := filepath.Clean(tt.expectedAbsPath)

			absPath, err := getFullOutputPath(tt.step, tt.absoluteOutputFolder)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Empty(t, absPath)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, expected, filepath.Clean(absPath))
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
			Steps: []Step{
				{
					Name:   "step1",
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
					Name:   "step2_cli",
					Model:  "ollama:dummy",
					Prompt: "Generate new.",
				},
				{
					Name:       "step3_default_maxresults",
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
		assert.Equal(t, llm.ProviderOllama, cfg.Steps[0].ModelConfig.ModelProvider)
		assert.Equal(t, "llama3.1", cfg.Steps[0].ModelConfig.ModelName)
		assert.Equal(t, 10, cfg.Steps[0].MaxResults)

		assert.Equal(t, PromptStepType, cfg.Steps[1].Type)
		assert.Equal(t, llm.ProviderOllama, cfg.Steps[1].ModelConfig.ModelProvider)
		assert.Equal(t, "dummy", cfg.Steps[1].ModelConfig.ModelName)
		assert.Equal(t, DefaultStepMinMaxResults, cfg.Steps[1].MaxResults)

		assert.Equal(t, PromptStepType, cfg.Steps[2].Type)
		assert.Equal(t, llm.ProviderLmStudio, cfg.Steps[2].ModelConfig.ModelProvider)
		assert.Equal(t, "dummy", cfg.Steps[2].ModelConfig.ModelName)
		assert.Equal(t, 1, cfg.Steps[2].MaxResults)

		absOutputFolder, _ := filepath.Abs("my_output")
		assert.Equal(t, filepath.Join(absOutputFolder, "step1.jsonl"), cfg.Steps[0].OutputFilename)
		assert.Equal(t, filepath.Join(absOutputFolder, "step2_cli.jsonl"), cfg.Steps[1].OutputFilename)
		assert.Equal(t, filepath.Join(absOutputFolder, "step3_default_maxresults.jsonl"), cfg.Steps[2].OutputFilename)
	})

	t.Run("Empty Steps", func(t *testing.T) {
		cfg := validConfig()
		cfg.Steps = []Step{}
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least one step is required")
	})

	t.Run("Duplicate Step Names", func(t *testing.T) {
		cfg := validConfig()
		cfg.Steps = append(cfg.Steps, cfg.Steps[0])
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate step name found: 'step1'")
	})

	t.Run("Step With Empty Name", func(t *testing.T) {
		cfg := validConfig()
		cfg.Steps[0].Name = ""
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "step at index 0: name can't be empty")
	})

	t.Run("Step With Both Prompt and Cmd", func(t *testing.T) {
		cfg := validConfig()
		cfg.Steps[0].Cmd = "a command"
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "step 'step1': either 'prompt' or 'cmd' should be defined, not both")
	})

	t.Run("Step With Neither Prompt nor Cmd", func(t *testing.T) {
		cfg := validConfig()
		cfg.Steps[0].Prompt = ""
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "step 'step1': either 'prompt' or 'cmd' must be defined")
	})

	t.Run("Step With Empty Model", func(t *testing.T) {
		cfg := validConfig()
		cfg.Steps[0].Model = ""
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "step 'step1': model definition can't be empty")
	})

	t.Run("Step With Invalid Model Format", func(t *testing.T) {
		cfg := validConfig()
		cfg.Steps[0].Model = "invalid-model-string"
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "step 'step1': model should follow pattern")
	})

	t.Run("Step With Unsupported Provider", func(t *testing.T) {
		cfg := validConfig()
		cfg.Steps[0].Model = "unsupported:model"
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "step 'step1': unsupported provider")
	})

	t.Run("Step With Invalid Model Config", func(t *testing.T) {
		cfg := validConfig()
		tempNeg := -0.1
		cfg.Steps[0].ModelConfig.Temperature = &tempNeg
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "step 'step1': model config validation failed: temperature must be between 0 and 1")
	})

	t.Run("Step With Invalid Output Filename", func(t *testing.T) {
		cfg := validConfig()
		cfg.Steps[0].OutputFilename = "invalid<filename>"
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "step 'step1': invalid output filename 'invalid<filename>': filename contains invalid characters")
	})

	t.Run("Invalid Output Folder", func(t *testing.T) {
		cfg := validConfig()
		cfg.OutputFolder = ""
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "output folder is required")
	})

	t.Run("MaxResults Defaults Correctly", func(t *testing.T) {
		cfg := validConfig()

		cfg.Steps[0].MaxResults = 10
		cfg.Steps[1].MaxResults = nil
		cfg.Steps[2].MaxResults = -5

		err := cfg.Validate()
		assert.NoError(t, err)

		assert.Equal(t, 10, cfg.Steps[0].MaxResults, "step with MaxResults > 0 should keep its value")
		assert.Equal(t, DefaultStepMinMaxResults, cfg.Steps[1].MaxResults, "step with MaxResults = nil should default")
		assert.Equal(t, DefaultStepMinMaxResults, cfg.Steps[2].MaxResults, "step with MaxResults < 0 should default")
	})
}

func TestValidateAndSetMaxResults(t *testing.T) {
	stepNames := map[string]bool{"foo": true, "bar": true}
	defaultVal := DefaultStepMinMaxResults

	tests := []struct {
		name        string
		input       interface{}
		expected    interface{}
		expectError bool
	}{
		{"nil input", nil, defaultVal, false},
		{"empty string", "", defaultVal, false},
		{"valid step reference", "foo.$length", nil, false},
		{"invalid step reference", "unknown.$length", nil, true},
		{"invalid string", "invalid", nil, true},
		{"zero int", 0, defaultVal, false},
		{"positive int", 5, 5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := &Step{MaxResults: tt.input}
			err := validateAndSetMaxResults(step, stepNames)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.expected != nil {
					assert.Equal(t, tt.expected, step.MaxResults)
				}
			}
		})
	}
}
