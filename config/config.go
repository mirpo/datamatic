package config

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mirpo/datamatic/fs"
	"github.com/mirpo/datamatic/jsonl"
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
	Name               string                `yaml:"name"`
	Model              string                `yaml:"model"`
	Prompt             string                `yaml:"prompt"`
	Cmd                string                `yaml:"cmd"`
	SystemPrompt       string                `yaml:"systemPrompt"`
	MaxResults         interface{}           `yaml:"maxResults"`
	ModelConfig        ModelConfig           `yaml:"modelConfig"`
	OutputFilename     string                `yaml:"outputFilename"`
	JSONSchema         jsonschema.JSONSchema `yaml:"jsonSchema"`
	ImagePath          string                `yaml:"imagePath"`
	ResolvedMaxResults int
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

type LineValue struct {
	ID       string `json:"id"`
	Response string `json:"response"`
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

func (c *Config) GetStepByName(name string) *Step {
	for _, step := range c.Steps {
		if step.Name == name {
			return &step
		}
	}
	return nil
}

func convertJSONValueToStringReflected(value interface{}) string {
	if value == nil {
		return ""
	}

	val := reflect.ValueOf(value)

	switch val.Kind() {
	case reflect.String:
		return val.String()
	case reflect.Float64:
		v := val.Float()
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10)
		}
		return strconv.FormatFloat(v, 'f', 2, 64)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(val.Int(), 10)
	case reflect.Bool:
		return strconv.FormatBool(val.Bool())
	case reflect.Slice:
		var elements []string
		for i := range val.Len() {
			elements = append(elements, convertJSONValueToStringReflected(val.Index(i).Interface()))
		}
		return strings.Join(elements, ", ")
	case reflect.Map:
		data, err := json.Marshal(value)
		if err != nil {
			return fmt.Sprintf("error marshalling map: %v", err)
		}
		return string(data)
	default:
		return fmt.Sprintf("%v", value)
	}
}

func uuidFromString(input string) string {
	return uuid.NewMD5(uuid.NameSpaceOID, []byte(input)).String()
}

func GetFieldAsString(data map[string]interface{}, key string) (string, error) {
	value, exists := data[key]
	if !exists {
		return "", fmt.Errorf("key '%s' not found", key)
	}
	return convertJSONValueToStringReflected(value), nil
}

func (s *Step) GetValue(outputFolder string, lineNumber int, attrKey string) (*LineValue, error) {
	line, err := fs.ReadLineFromFile(s.OutputFilename, lineNumber)
	if err != nil {
		return nil, err
	}

	configValidator := jsonschema.NewConfigValidator()

	switch s.Type {
	case CliStepType:
		var decoded map[string]interface{}
		if err := json.Unmarshal([]byte(line), &decoded); err != nil {
			return nil, fmt.Errorf("CLI step: failed to parse JSON from line %d: %w", lineNumber, err)
		}

		value, err := GetFieldAsString(decoded, attrKey)
		if err != nil {
			return nil, fmt.Errorf("CLI step: missing or invalid '%s' field: %w", attrKey, err)
		}

		return &LineValue{
			ID:       uuidFromString(value),
			Response: value,
		}, nil

	case PromptStepType:
		var decoded jsonl.LineEntity
		if err := json.Unmarshal([]byte(line), &decoded); err != nil {
			return nil, fmt.Errorf("prompt step: failed to parse JSON from line %d: %w", lineNumber, err)
		}

		var value string
		if !configValidator.HasSchemaDefinition(s.JSONSchema) {
			str, ok := decoded.Response.(string)
			if !ok {
				return nil, fmt.Errorf("prompt step: expected string response, got %T", decoded.Response)
			}
			value = str
		} else {
			data, ok := decoded.Response.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("prompt step: expected map response, got %T", decoded.Response)
			}

			value, err = GetFieldAsString(data, attrKey)
			if err != nil {
				return nil, fmt.Errorf("prompt step: missing or invalid '%s' field", attrKey)
			}
		}

		return &LineValue{
			ID:       decoded.ID,
			Response: value,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported step type '%s'", s.Type)
	}
}

func (s *Step) HasImages() bool {
	return len(s.ImagePath) > 0
}
