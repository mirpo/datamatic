package utils

import (
	"errors"
	"fmt"

	"github.com/mirpo/datamatic/config"
	"github.com/mirpo/datamatic/jsonschema"
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

	return nil
}
