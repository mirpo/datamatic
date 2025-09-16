package utils

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mirpo/datamatic/config"
	"github.com/mirpo/datamatic/jsonschema"
	"github.com/mirpo/datamatic/llm"
	"github.com/rs/zerolog/log"
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
func PreprocessConfig(cfg *config.Config, verbose bool) error {
	// Set step types for all steps
	for i := range cfg.Steps {
		step := &cfg.Steps[i]
		if err := setStepType(step); err != nil {
			return fmt.Errorf("step '%s': %w", step.Name, err)
		}
	}

	// Process JSON schemas for prompt steps
	for i := range cfg.Steps {
		step := &cfg.Steps[i]
		if verbose {
			log.Debug().Msgf("Checking step '%s': Type=%s, JSONSchemaRaw=%v", step.Name, step.Type, step.JSONSchemaRaw != nil)
		}
		if step.Type != config.PromptStepType || step.JSONSchemaRaw == nil {
			continue
		}
		if verbose {
			log.Debug().Msgf("Processing JSON schema for step '%s'", step.Name)
		}
		schema, err := jsonschema.LoadSchema(step.JSONSchemaRaw)
		if err != nil {
			return fmt.Errorf("processing JSON schema for step '%s': %w", step.Name, err)
		}
		if schema != nil {
			step.JSONSchema = *schema
			if verbose {
				log.Debug().Msgf("Successfully loaded JSON schema for step '%s'", step.Name)
			}
		}
	}

	// Process model details for prompt steps
	for i := range cfg.Steps {
		step := &cfg.Steps[i]
		if step.Type != config.PromptStepType {
			continue
		}
		if verbose {
			log.Debug().Msgf("Processing model details for step '%s'", step.Name)
		}
		if err := setModelDetails(step); err != nil {
			return fmt.Errorf("processing model details for step '%s': %w", step.Name, err)
		}
		if verbose {
			log.Debug().Msgf("Successfully set model provider '%s' and model '%s' for step '%s'", step.ModelConfig.ModelProvider, step.ModelConfig.ModelName, step.Name)
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
