package config

import (
	"time"

	"github.com/mirpo/datamatic/jsonschema"
	"github.com/mirpo/datamatic/llm"
)

const (
	DefaultStepMinMaxResults = 3
)

func NewConfig() *Config {
	return &Config{
		ConfigFile:       "",
		Verbose:          false,
		LogPretty:        true,
		OutputFolder:     "dataset",
		HTTPTimeout:      300,
		ValidateResponse: true,
		SkipCliWarning:   false,
		RetryConfig:      NewDefaultRetryConfig(),
	}
}

type Config struct {
	ConfigFile       string
	Verbose          bool
	LogPretty        bool
	OutputFolder     string
	HTTPTimeout      int
	ValidateResponse bool
	SkipCliWarning   bool
	Version          string      `yaml:"version"`
	Steps            []Step      `yaml:"steps"`
	RetryConfig      RetryConfig `yaml:"retryConfig"`
}

type StepType string

const (
	PromptStepType  StepType = "prompt"
	CliStepType     StepType = "cli"
	UnknownStepType StepType = "unknown"
)

type Step struct {
	Type               StepType
	Name               string      `yaml:"name"`
	Model              string      `yaml:"model"`
	Prompt             string      `yaml:"prompt"`
	Cmd                string      `yaml:"cmd"`
	SystemPrompt       string      `yaml:"systemPrompt"`
	MaxResults         interface{} `yaml:"maxResults"`
	ModelConfig        ModelConfig `yaml:"modelConfig"`
	OutputFilename     string      `yaml:"outputFilename"`
	JSONSchemaRaw      interface{} `yaml:"jsonSchema"`
	ImagePath          string      `yaml:"imagePath"`
	ResolvedMaxResults int
	JSONSchema         jsonschema.Schema
}

type ModelConfig struct {
	ModelProvider llm.ProviderType
	ModelName     string
	BaseURL       string   `yaml:"baseUrl"`
	Temperature   *float64 `yaml:"temperature"`
	MaxTokens     *int     `yaml:"maxTokens"`
}

type RetryConfig struct {
	MaxAttempts       int           `yaml:"maxAttempts"`
	InitialDelay      time.Duration `yaml:"initialDelay"`
	MaxDelay          time.Duration `yaml:"maxDelay"`
	BackoffMultiplier float64       `yaml:"backoffMultiplier"`
	Enabled           bool          `yaml:"enabled"`
}

func NewDefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:       3,
		InitialDelay:      1 * time.Second,
		MaxDelay:          10 * time.Second,
		BackoffMultiplier: 2.0,
		Enabled:           true,
	}
}

func (c *Config) GetStepByName(name string) *Step {
	for _, step := range c.Steps {
		if step.Name == name {
			return &step
		}
	}
	return nil
}

func (s *Step) HasImages() bool {
	return len(s.ImagePath) > 0
}
