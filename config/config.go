package config

import (
	"bytes"
	"fmt"

	"github.com/mirpo/datamatic/jq"
	"github.com/mirpo/datamatic/jsonschema"
	"github.com/mirpo/datamatic/llm"
	"github.com/mirpo/datamatic/retry"
	"gopkg.in/yaml.v3"
)

const (
	DefaultStepCount = 3
)

func NewConfig() *Config {
	return &Config{
		ConfigFile:       "",
		Verbose:          false,
		LogPretty:        true,
		OutputFolder:     "dataset",
		HTTPTimeout:      300,
		ValidateResponse: true,
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
	ReadStepType      StepType = "read"
	UnknownStepType   StepType = "unknown"
)

const (
	SourceFormatJSONL = "jsonl" // one JSON value per line (default)
	SourceFormatJSON  = "json"  // the whole file is a single JSON value
)

const (
	ReadFormatFiles = "files" // one row per file: {path, name, content}
	ReadFormatCSV   = "csv"   // one row per record; columns become fields
	ReadFormatJSONL = "jsonl" // one row per line (parsed JSON)
)

type Step struct {
	Type           StepType    `yaml:"type,omitempty"`
	Name           string      `yaml:"name"`
	Model          string      `yaml:"model"`
	Prompt         string      `yaml:"prompt"`
	Run            string      `yaml:"run"`
	JQ             string      `yaml:"jq"`           // transform steps: jq program
	Read           string      `yaml:"read"`         // read steps: file/glob/dir to load as rows
	Format         string      `yaml:"format"`       // read steps: "files" | "csv" | "jsonl" (default: by extension)
	From           string      `yaml:"from"`         // transform steps: source step name
	Limit          int         `yaml:"limit"`        // transform steps: cap output rows (0 = no cap)
	Collect        bool        `yaml:"collect"`      // transform steps: jq sees an array of ALL source rows (fan-in)
	SourceFormat   string      `yaml:"sourceFormat"` // transform steps: "jsonl" (default, line per row) or "json" (whole file is one value)
	WorkDir        string      `yaml:"workDir,omitempty"`
	SystemPrompt   string      `yaml:"systemPrompt"`
	Count          int         `yaml:"count"`       // generator steps: how many rows to produce (default 3)
	ForEach        string      `yaml:"forEach"`     // iterate once per row of an earlier step
	Concurrency    int         `yaml:"concurrency"` // prompt steps: rows to generate in parallel (default 1)
	ModelConfig    ModelConfig `yaml:"modelConfig"`
	OutputFilename string      `yaml:"outputFilename"`
	JSONSchemaRaw  interface{} `yaml:"jsonSchema"`
	ImagePath      string      `yaml:"imagePath"`
	ResolvedCount  int
	JSONSchema     jsonschema.Schema
	// JQProgram holds the compiled jq program (set during preprocessing);
	// UsesParent records whether it references the $parent variable
	JQProgram  *jq.Program
	UsesParent bool
}

type ModelConfig struct {
	ModelProvider llm.ProviderType
	ModelName     string
	BaseURL       string   `yaml:"baseUrl"`
	Temperature   *float64 `yaml:"temperature"`
	MaxTokens     *int     `yaml:"maxTokens"`
}

// ParseYAML decodes a config strictly: unknown keys (typos, removed syntax
// like maxResults) are errors instead of being silently ignored.
func ParseYAML(data []byte, cfg *Config) error {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(cfg); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}
	return nil
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
