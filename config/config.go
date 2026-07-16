package config

import (
	"github.com/mirpo/datamatic/jq"
	"github.com/mirpo/datamatic/jsonschema"
	"github.com/mirpo/datamatic/llm"
	"github.com/mirpo/datamatic/retry"
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
		RetryConfig:      retry.NewDefaultConfig(),
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
	Version          string       `yaml:"version"`
	EnvVars          []string     `yaml:"envVars"`
	Steps            []Step       `yaml:"steps"`
	RetryConfig      retry.Config `yaml:"retryConfig"`
}

type StepType string

const (
	PromptStepType    StepType = "prompt"
	ShellStepType     StepType = "shell"
	TransformStepType StepType = "transform"
	UnknownStepType   StepType = "unknown"
)

type Step struct {
	Type               StepType    `yaml:"type,omitempty"`
	Name               string      `yaml:"name"`
	Model              string      `yaml:"model"`
	Prompt             string      `yaml:"prompt"`
	Run                string      `yaml:"run"`
	JQ                 string      `yaml:"jq"`    // transform steps: jq program
	From               string      `yaml:"from"`  // transform steps: source step name
	Limit              int         `yaml:"limit"` // transform steps: cap output rows (0 = no cap)
	WorkDir            string      `yaml:"workDir,omitempty"`
	SystemPrompt       string      `yaml:"systemPrompt"`
	MaxResults         interface{} `yaml:"maxResults"`
	ModelConfig        ModelConfig `yaml:"modelConfig"`
	OutputFilename     string      `yaml:"outputFilename"`
	JSONSchemaRaw      interface{} `yaml:"jsonSchema"`
	ImagePath          string      `yaml:"imagePath"`
	ResolvedMaxResults int
	JSONSchema         jsonschema.Schema
	// JQProgram holds the compiled jq program (set during preprocessing)
	JQProgram *jq.Program
}

type ModelConfig struct {
	ModelProvider llm.ProviderType
	ModelName     string
	BaseURL       string   `yaml:"baseUrl"`
	Temperature   *float64 `yaml:"temperature"`
	MaxTokens     *int     `yaml:"maxTokens"`
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
