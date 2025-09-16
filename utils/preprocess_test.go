package utils

import (
	"strings"
	"testing"

	"github.com/mirpo/datamatic/config"
	"github.com/mirpo/datamatic/llm"
	"github.com/stretchr/testify/assert"
)

func TestIsValidName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"Valid simple", "file.txt", false},
		{"Valid dot", ".", false},
		{"Valid long name", strings.Repeat("a", 255), false},
		{"Empty", "", true},
		{"Too long", strings.Repeat("a", 256), true},
		{"Invalid char <", "bad<name", true},
		{"Ends with space", "bad ", true},
		{"Ends with period", "bad.", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := isValidName(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsValidProvider(t *testing.T) {
	assert.True(t, isValidProvider(llm.ProviderOllama))
	assert.True(t, isValidProvider(llm.ProviderOpenAI))
	assert.False(t, isValidProvider(llm.ProviderUnknown))
	assert.False(t, isValidProvider(llm.ProviderType("INVALID")))
}

func TestPreprocessConfig_Success(t *testing.T) {
	cfg := &config.Config{
		OutputFolder: "/tmp/test",
		Steps: []config.Step{
			{
				Name:           "prompt1",
				Model:          "ollama:llama3.2",
				Prompt:         "Generate something",
				OutputFilename: "custom",
				ImagePath:      "images/photo.jpg",
				MaxResults:     nil, // should default
			},
			{
				Name:       "cli1",
				Cmd:        "echo hi",
				MaxResults: -1, // should default
			},
			{
				Name:       "prompt2",
				Model:      "openai:gpt-4",
				Prompt:     "More text",
				MaxResults: 5,
			},
			{
				Name:       "prompt3",
				Model:      "gemini:gemini-pro",
				Prompt:     "Dynamic",
				MaxResults: "prompt1.$length",
			},
		},
	}

	err := PreprocessConfig(cfg)
	assert.NoError(t, err)

	// Step types
	assert.Equal(t, config.PromptStepType, cfg.Steps[0].Type)
	assert.Equal(t, config.CliStepType, cfg.Steps[1].Type)

	// Providers + models
	assert.Equal(t, llm.ProviderOllama, cfg.Steps[0].ModelConfig.ModelProvider)
	assert.Equal(t, "llama3.2", cfg.Steps[0].ModelConfig.ModelName)
	assert.Equal(t, llm.ProviderOpenAI, cfg.Steps[2].ModelConfig.ModelProvider)
	assert.Equal(t, "gpt-4", cfg.Steps[2].ModelConfig.ModelName)

	// Filenames
	assert.Equal(t, "/tmp/test/custom.jsonl", cfg.Steps[0].OutputFilename)
	assert.Equal(t, "/tmp/test/cli1.jsonl", cfg.Steps[1].OutputFilename)

	// Image path
	assert.Equal(t, "/tmp/test/images/photo.jpg", cfg.Steps[0].ImagePath)

	// MaxResults
	assert.Equal(t, config.DefaultStepMinMaxResults, cfg.Steps[0].MaxResults)
	assert.Equal(t, config.DefaultStepMinMaxResults, cfg.Steps[1].MaxResults)
	assert.Equal(t, 5, cfg.Steps[2].MaxResults)
	assert.Equal(t, "prompt1.$length", cfg.Steps[3].MaxResults)
}

func TestPreprocessConfig_Failures(t *testing.T) {
	tests := []struct {
		name   string
		config *config.Config
		errMsg string
	}{
		{
			"Both prompt and cmd",
			&config.Config{OutputFolder: "/tmp", Steps: []config.Step{
				{Name: "bad", Prompt: "p", Cmd: "c"},
			}},
			"either 'prompt' or 'cmd' should be defined",
		},
		{
			"Missing provider colon",
			&config.Config{OutputFolder: "/tmp", Steps: []config.Step{
				{Name: "bad", Prompt: "p", Model: "invalidmodel"},
			}},
			"model should follow pattern",
		},
		{
			"Invalid filename",
			&config.Config{OutputFolder: "/tmp", Steps: []config.Step{
				{Name: "bad<name>", Prompt: "p", Model: "ollama:llama3.2"},
			}},
			"filename contains invalid characters",
		},
		{
			"Empty step name",
			&config.Config{OutputFolder: "/tmp", Steps: []config.Step{
				{Name: "", Prompt: "p", Model: "ollama:llama3.2"},
			}},
			"name can't be empty",
		},
		{
			"Reserved name SYSTEM",
			&config.Config{OutputFolder: "/tmp", Steps: []config.Step{
				{Name: "SYSTEM", Prompt: "p", Model: "ollama:llama3.2"},
			}},
			"not allowed",
		},
		{
			"Duplicate step names",
			&config.Config{OutputFolder: "/tmp", Steps: []config.Step{
				{Name: "dup", Prompt: "p", Model: "ollama:llama3.2"},
				{Name: "dup", Prompt: "q", Model: "openai:gpt-4"},
			}},
			"duplicate step name",
		},
		{
			"CLI without output filename",
			&config.Config{OutputFolder: "/tmp", Steps: []config.Step{
				{Name: "cli1", Cmd: "echo hi"},
			}},
			"output filename is mandatory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := PreprocessConfig(tt.config)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}
