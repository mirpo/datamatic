package utils

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mirpo/datamatic/config"
	"github.com/mirpo/datamatic/jsonschema"
	"github.com/mirpo/datamatic/llm"
)

// setStepType determines and sets the step type based on step configuration
func setStepType(step *config.Step) error {
	promptDefined := len(step.Prompt) > 0
	cmdDefined := len(step.Cmd) > 0

	if promptDefined && cmdDefined {
		return errors.New("either 'prompt' or 'cmd' should be defined, not both")
	}

	if !promptDefined && !cmdDefined {
		return errors.New("either 'prompt' or 'cmd' must be defined")
	}

	if promptDefined {
		step.Type = config.PromptStepType
	} else {
		step.Type = config.CliStepType
	}

	return nil
}

// PreprocessConfig handles initial config setup: sets step types and processes schemas
func PreprocessConfig(cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	if len(cfg.Steps) == 0 {
		return errors.New("at least one step is required")
	}

	if err := setRootOutputFolder(cfg); err != nil {
		return fmt.Errorf("setting root output folder: %w", err)
	}

	stepNames := make(map[string]bool, len(cfg.Steps))

	for i := range cfg.Steps {
		step := &cfg.Steps[i]

		// Step name checks
		if strings.TrimSpace(step.Name) == "" {
			return fmt.Errorf("step at index %d: name can't be empty", i)
		}
		if strings.ToUpper(step.Name) == "SYSTEM" {
			return fmt.Errorf("using 'SYSTEM' as step name is not allowed")
		}
		if stepNames[step.Name] {
			return fmt.Errorf("duplicate step name found: '%s'", step.Name)
		}
		stepNames[step.Name] = true

		// Step type (prompt vs cli)
		if err := setStepType(step); err != nil {
			return fmt.Errorf("step '%s': %w", step.Name, err)
		}

		// Prompt steps
		if step.Type == config.PromptStepType {
			// Require valid model definition
			if err := setModelDetails(step); err != nil {
				return fmt.Errorf("processing model details for step '%s': %w", step.Name, err)
			}

			// Load JSON schema if provided
			if step.JSONSchemaRaw != nil {
				schema, err := jsonschema.LoadSchema(step.JSONSchemaRaw)
				if err != nil {
					return fmt.Errorf("processing JSON schema for step '%s': %w", step.Name, err)
				}
				if schema != nil {
					step.JSONSchema = *schema
				}
			}
		}

		// CLI steps
		if step.Type == config.CliStepType {
			if step.OutputFilename == "" {
				return fmt.Errorf("step '%s': output filename is mandatory for CLI steps", step.Name)
			}
			if err := isValidName(step.OutputFilename); err != nil {
				return fmt.Errorf("step '%s': invalid output filename '%s': %w",
					step.Name, step.OutputFilename, err)
			}
		}

		// Normalize and validate output filename (all steps)
		if err := setOutputFilename(step, cfg.OutputFolder); err != nil {
			return fmt.Errorf("step '%s': %w", step.Name, err)
		}

		// Normalize image path if needed
		if step.HasImages() {
			if err := setImagePath(step, cfg.OutputFolder); err != nil {
				return fmt.Errorf("step '%s': %w", step.Name, err)
			}
		}

		// Apply MaxResults defaults
		if err := setMaxResultsDefaults(step); err != nil {
			return fmt.Errorf("step '%s': %w", step.Name, err)
		}
	}

	return nil
}

// setModelDetails extracts and sets provider and model details in step config
func setModelDetails(step *config.Step) error {
	if step.Model == "" {
		return errors.New("model definition can't be empty")
	}

	provider, model, found := strings.Cut(step.Model, ":")
	if !found {
		return fmt.Errorf("model should follow pattern 'provider:model', examples: 'ollama:llama3.2'")
	}

	if model == "" {
		return errors.New("model name can't be empty")
	}

	providerType := llm.ProviderType(provider)
	if !isValidProvider(providerType) {
		return fmt.Errorf("unsupported provider: %s", provider)
	}

	step.ModelConfig.ModelProvider = providerType
	step.ModelConfig.ModelName = model
	return nil
}

func isValidProvider(provider llm.ProviderType) bool {
	switch provider {
	case llm.ProviderOllama, llm.ProviderLmStudio, llm.ProviderOpenAI,
		llm.ProviderOpenRouter, llm.ProviderGemini:
		return true
	default:
		return false
	}
}

// getFullOutputPath constructs the full output path for a step
func getFullOutputPath(step config.Step, outputFolder string) (string, error) {
	extension := ".jsonl"

	filename := step.OutputFilename
	if len(filename) == 0 {
		filename = step.Name
	}

	if err := isValidName(filename); err != nil {
		return "", fmt.Errorf("invalid effective output filename '%s': %w", filename, err)
	}

	if !strings.HasSuffix(filename, extension) {
		filename = filename + extension
	}

	fullPath := filepath.Join(outputFolder, filename)

	return filepath.Clean(fullPath), nil
}

// setOutputFilename sets the full output path for a step
func setOutputFilename(step *config.Step, outputFolder string) error {
	fullOutputPath, err := getFullOutputPath(*step, outputFolder)
	if err != nil {
		return fmt.Errorf("failed to get full output path: %w", err)
	}
	step.OutputFilename = fullOutputPath
	return nil
}

// setImagePath processes and sets the image path for a step
func setImagePath(step *config.Step, outputFolder string) error {
	step.ImagePath = strings.TrimSpace(step.ImagePath)

	if !filepath.IsAbs(step.ImagePath) {
		step.ImagePath = filepath.Join(outputFolder, step.ImagePath)
	}

	return nil
}

// setMaxResultsDefaults sets default MaxResults for nil, empty string, and int <= 0 cases
func setMaxResultsDefaults(step *config.Step) error {
	switch v := step.MaxResults.(type) {
	case nil:
		step.MaxResults = config.DefaultStepMinMaxResults
		return nil

	case string:
		if v == "" {
			step.MaxResults = config.DefaultStepMinMaxResults
			return nil
		}
		// check dynamic strings (like "foo.$length") in validation phase
		return nil

	case int:
		if v <= 0 {
			step.MaxResults = config.DefaultStepMinMaxResults
		}
		// positive int values passed as-is
		return nil

	default:
		return nil
	}
}

// isValidName validates filename according to filesystem rules
func isValidName(name string) error {
	if len(name) == 0 {
		return errors.New("filename cannot be empty")
	}

	if len(name) > 255 {
		return errors.New("filename exceeds the maximum length of 255 characters")
	}

	illegalChars := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1F]`)
	if illegalChars.MatchString(name) {
		return errors.New("filename contains invalid characters")
	}

	if strings.HasSuffix(name, " ") || (len(name) > 1 && strings.HasSuffix(name, ".")) {
		return errors.New("filename cannot end with a space or a period (unless the name is just '.')")
	}

	return nil
}

func setRootOutputFolder(cfg *config.Config) error {
	if len(cfg.OutputFolder) == 0 {
		return errors.New("output folder is required")
	}

	absOutputFolder, err := filepath.Abs(cfg.OutputFolder)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for output folder '%s': %w", cfg.OutputFolder, err)
	}

	cfg.OutputFolder = absOutputFolder
	return nil
}
