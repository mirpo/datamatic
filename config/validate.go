package config

import (
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/mirpo/datamatic/promptbuilder"
	"github.com/mirpo/datamatic/retry"
)

func validateVersion(version string) error {
	if version == "" {
		return errors.New("version is required")
	}

	if version != "1.0" {
		return fmt.Errorf("version '%s' is unsupported", version)
	}

	return nil
}

func validateURL(input string) error {
	parsedURL, err := url.ParseRequestURI(input)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return errors.New("invalid URL: missing scheme or host")
	}

	return nil
}

func validateModelConfig(step ModelConfig) error {
	if step.Temperature != nil {
		if *step.Temperature < 0 || *step.Temperature > 1 {
			return errors.New("temperature must be between 0 and 1")
		}
	}

	if step.MaxTokens != nil {
		if *step.MaxTokens <= 0 {
			return errors.New("maxTokens must be > 0")
		}
	}

	if step.BaseURL != "" {
		if err := validateURL(step.BaseURL); err != nil {
			return fmt.Errorf("invalid baseUrl: %w", err)
		}
	}

	return nil
}

func validateRetryConfig(cfg retry.Config) error {
	if cfg.MaxAttempts <= 0 {
		return errors.New("maxAttempts must be greater than 0")
	}

	if cfg.InitialDelay <= 0 {
		return errors.New("initialDelay must be greater than 0")
	}

	if cfg.MaxDelay < cfg.InitialDelay {
		return errors.New("maxDelay must be greater than or equal to initialDelay")
	}

	if cfg.BackoffMultiplier < 1.0 {
		return errors.New("backoffMultiplier must be greater than or equal to 1.0")
	}

	return nil
}

func validateMaxResults(step *Step, stepNames map[string]bool) error {
	switch v := step.MaxResults.(type) {
	case nil, int:
		return nil
	case string:
		if v == "" {
			// Empty string case is handled in preprocessing
			return nil
		}

		if strings.HasSuffix(v, ".$length") {
			stepName := strings.TrimSuffix(v, ".$length")
			if !stepNames[stepName] {
				return fmt.Errorf("maxResults reference to unknown step '%s'", stepName)
			}
			return nil
		}

		return fmt.Errorf("invalid string format for maxResults: '%s'", v)
	}

	return fmt.Errorf("maxResults unsupported type '%T'", step.MaxResults)
}

func (c *Config) Validate() error {
	if err := validateVersion(c.Version); err != nil {
		return err
	}

	// If retryConfig is not set in YAML (has zero values), use defaults
	if c.RetryConfig.MaxAttempts == 0 {
		c.RetryConfig = retry.NewDefaultConfig()
	}

	if err := validateRetryConfig(c.RetryConfig); err != nil {
		return fmt.Errorf("retry config validation failed: %w", err)
	}

	stepNames := map[string]bool{}
	cliCalls := []string{}

	for index := range c.Steps {
		step := &c.Steps[index]

		stepNames[step.Name] = true

		stepType := step.Type

		if stepType == CliStepType {
			cliCalls = append(cliCalls, fmt.Sprintf("- %s", step.Cmd))

			basename := filepath.Base(step.OutputFilename)
			if !strings.Contains(step.Cmd, basename) && !strings.Contains(step.Cmd, step.OutputFilename) {
				return fmt.Errorf("step '%s': output filename should match output result of external CLI; cmd: [%s], output file: %s",
					step.Name, step.Cmd, step.OutputFilename)
			}
		}

		if stepType == PromptStepType {
			if step.JSONSchema.HasSchemaDefinition() {
				if err := step.JSONSchema.EnsureAllPropertiesRequired(); err != nil {
					return fmt.Errorf("step '%s': %w", step.Name, err)
				}
			}

			if strings.Contains(step.Prompt, "{{.SYSTEM.JSON_SCHEMA}}") {
				if err := step.JSONSchema.EnsureAllPropertiesRequired(); err != nil {
					return fmt.Errorf("step '%s': JSON schema validation failed when using '{{.SYSTEM.JSON_SCHEMA}}' in prompt: %w", step.Name, err)
				}
			}

			promptBuilder := promptbuilder.NewPromptBuilder(step.Prompt)
			if promptBuilder.HasPlaceholders() {
				placeholders := promptBuilder.GetPlaceholders()
				for _, val := range placeholders {
					if !stepNames[val.Step] {
						return fmt.Errorf("placeholder has a references to unknown or not previous steps, step: %s, placeholder: %+v", step.Name, val)
					}

					// JSON key - supports nested paths like "user.profile.name"
					if len(val.Key) > 0 {
						refStep := c.GetStepByName(val.Step)
						if refStep.Type == PromptStepType {
							if !refStep.JSONSchema.HasSchemaDefinition() {
								return fmt.Errorf("step %s must have JSON schema, key: %s", val.Step, val.Key)
							}

							if strings.Contains(val.Key, ".") {
								if !refStep.JSONSchema.HasFieldPath(val.Key) {
									return fmt.Errorf("field path '%s' not found in step %s JSON schema", val.Key, val.Step)
								}
							} else {
								// For single field, use existing validation
								if !refStep.JSONSchema.HasRequiredProperty(val.Key) {
									return fmt.Errorf("'%s' key must be defined in step %s in JSON schema as a property and required", val.Key, val.Step)
								}
							}
						}
					}
				}
			}

			if err := validateModelConfig(step.ModelConfig); err != nil {
				return fmt.Errorf("step '%s': model config validation failed: %w", step.Name, err)
			}

			if err := validateMaxResults(step, stepNames); err != nil {
				return fmt.Errorf("step '%s': maxResults validation failed: %w", step.Name, err)
			}
		}
	}

	if !c.SkipCliWarning && len(cliCalls) > 0 {
		fmt.Printf("⚠️ WARNING: External application call detected! The author assumes no responsibility for execution results. Please verify all external calls before proceeding. Use at your own risk.\n\nCalls: \n%s\n\nPress Enter to continue", strings.Join(cliCalls, "\n"))
		fmt.Scanln() //nolint:golint,errcheck
	}

	return nil
}
