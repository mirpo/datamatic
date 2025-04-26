package config

import "github.com/mirpo/datamatic/llm"

const (
	DefaultStepMinMaxResults = 3
)

func NewConfig() *Config {
	return &Config{
		ConfigFile:   "",
		Verbose:      false,
		LogPretty:    true,
		OutputFolder: "dataset",
		HTTPTimeout:  300,
	}
}

type Config struct {
	ConfigFile   string
	Verbose      bool
	LogPretty    bool
	OutputFolder string
	HTTPTimeout  int
	Version      string `yaml:"version"`
	Steps        []Step `yaml:"steps"`
}

type StepType string

const (
	PromptStepType  StepType = "prompt"
	CliStepType     StepType = "cli"
	UnknownStepType StepType = "unknown"
)

type Step struct {
	Type           StepType
	Name           string      `yaml:"name"`
	Model          string      `yaml:"model"`
	Prompt         string      `yaml:"prompt"`
	Cmd            string      `yaml:"cmd"`
	SystemPrompt   string      `yaml:"systemPrompt"`
	MaxResults     int         `yaml:"maxResults"`
	ModelConfig    ModelConfig `yaml:"modelConfig"`
	OutputFilename string      `yaml:"outputFilename"`
}

type ModelConfig struct {
	ModelProvider llm.ProviderType
	ModelName     string
	BaseURL       string   `yaml:"baseUrl"`
	Temperature   *float64 `yaml:"temperature"`
	MaxTokens     *int     `yaml:"maxTokens"`
}

func (s *Step) GetProviderConfig(httpTimeout int) llm.ProviderConfig {
	providerConfig := llm.ProviderConfig{
		BaseURL:      s.ModelConfig.BaseURL,
		ProviderType: s.ModelConfig.ModelProvider,
		ModelName:    s.ModelConfig.ModelName,
		AuthToken:    "token",
		HTTPTimeout:  httpTimeout,
		Temperature:  s.ModelConfig.Temperature,
		MaxTokens:    s.ModelConfig.MaxTokens,
	}

	return providerConfig
}
