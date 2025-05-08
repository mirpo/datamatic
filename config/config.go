package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/mirpo/datamatic/fs"
	"github.com/mirpo/datamatic/jsonl"
	"github.com/mirpo/datamatic/llm"
	"github.com/rs/zerolog/log"
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
	}
}

type Config struct {
	ConfigFile       string
	Verbose          bool
	LogPretty        bool
	OutputFolder     string
	HTTPTimeout      int
	ValidateResponse bool
	Version          string `yaml:"version"`
	Steps            []Step `yaml:"steps"`
}

type StepType string

const (
	PromptStepType  StepType = "prompt"
	CliStepType     StepType = "cli"
	UnknownStepType StepType = "unknown"
)

type Step struct {
	Type           StepType
	Name           string           `yaml:"name"`
	Model          string           `yaml:"model"`
	Prompt         string           `yaml:"prompt"`
	Cmd            string           `yaml:"cmd"`
	SystemPrompt   string           `yaml:"systemPrompt"`
	MaxResults     int              `yaml:"maxResults"`
	ModelConfig    ModelConfig      `yaml:"modelConfig"`
	OutputFilename string           `yaml:"outputFilename"`
	JSONSchema     jsonl.JSONSchema `yaml:"jsonSchema"`
}

type ModelConfig struct {
	ModelProvider llm.ProviderType
	ModelName     string
	BaseURL       string   `yaml:"baseUrl"`
	Temperature   *float64 `yaml:"temperature"`
	MaxTokens     *int     `yaml:"maxTokens"`
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

	if s.Type == PromptStepType {
		var decodedLine jsonl.LineEntity
		err = json.Unmarshal([]byte(line), &decodedLine)
		if err != nil {
			log.Err(err).Msgf("failed to parse data JSON from line '%s'", line)
			return nil, err
		}

		var val string
		hasSchemaSchema := s.JSONSchema.HasSchemaDefinition()
		if hasSchemaSchema {
			data, ok := decodedLine.Response.(map[string]interface{})
			if !ok {
				return nil, errors.New("failed to cast Response to JSON")
			}

			val, err = GetFieldAsString(data, attrKey)
			if err != nil {
				return nil, fmt.Errorf("failed to get %s attrKey as text", attrKey)
			}
		} else {
			val = decodedLine.Response.(string)
		}

		return &LineValue{
			ID:       decodedLine.ID,
			Response: val,
		}, nil
	}

	return nil, nil
}
