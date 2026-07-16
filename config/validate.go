package config

import (
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/mirpo/datamatic/retry"
	"github.com/rs/zerolog/log"
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
		if *step.Temperature < 0 || *step.Temperature > 2 {
			return errors.New("temperature must be between 0 and 2")
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

func (c *Config) Validate() error {
	if err := validateVersion(c.Version); err != nil {
		return err
	}

	if err := validateRetryConfig(c.RetryConfig); err != nil {
		return fmt.Errorf("retry config validation failed: %w", err)
	}

	for index := range c.Steps {
		step := &c.Steps[index]

		stepType := step.Type

		if stepType == ShellStepType {
			filename := filepath.Base(step.OutputFilename)
			if !strings.Contains(step.Run, filename) {
				log.Warn().Msgf("step '%s': output filename '%s' not found in run command — make sure the command actually creates this file",
					step.Name, filename)
			}
		}

		if stepType == PromptStepType {
			if step.JSONSchema.HasSchemaDefinition() {
				if err := step.JSONSchema.EnsureAllPropertiesRequired(); err != nil {
					return fmt.Errorf("step '%s': %w", step.Name, err)
				}
				for _, issue := range step.JSONSchema.StrictCompatibilityIssues() {
					log.Warn().Msgf("step '%s': schema is not strict-mode compatible (may be rejected by OpenAI): %s", step.Name, issue)
				}
			}

			if err := validateModelConfig(step.ModelConfig); err != nil {
				return fmt.Errorf("step '%s': model config validation failed: %w", step.Name, err)
			}
		}
	}

	return nil
}

// ShellCommands returns the run commands of all shell steps, used by the CLI
// to warn the user before executing external applications.
func (c *Config) ShellCommands() []string {
	var commands []string
	for _, step := range c.Steps {
		if step.Type == ShellStepType {
			commands = append(commands, step.Run)
		}
	}
	return commands
}
