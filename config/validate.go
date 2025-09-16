package config

import (
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mirpo/datamatic/promptbuilder"
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

func validateAndAbsOutputFolder(outputFolder string) (string, error) {
	if len(outputFolder) == 0 {
		return "", errors.New("output folder is required")
	}

	if err := isValidName(outputFolder); err != nil {
		return "", fmt.Errorf("invalid output folder name: %w", err)
	}

	absOutputFolder, err := filepath.Abs(outputFolder)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path for output folder '%s': %w", outputFolder, err)
	}

	return absOutputFolder, nil
}

func validateMaxResults(step *Step, stepNames map[string]bool) error {
	switch v := step.MaxResults.(type) {
	case nil, int:
		// These cases are handled in preprocessing
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

	default:
		return fmt.Errorf("maxResults unsupported type '%T'", step.MaxResults)
	}
}

func (c *Config) Validate() error {
	if err := validateVersion(c.Version); err != nil {
		return err
	}

	absOutputFolder, err := validateAndAbsOutputFolder(c.OutputFolder)
	if err != nil {
		return err
	}
	c.OutputFolder = absOutputFolder

	if len(c.Steps) == 0 {
		return errors.New("at least one step is required")
	}

	stepNames := map[string]bool{}
	cliCalls := []string{}

	for index := range c.Steps {
		step := &c.Steps[index]

		if len(step.Name) == 0 {
			return fmt.Errorf("step at index %d: name can't be empty", index)
		}

		if strings.ToUpper(step.Name) == "SYSTEM" {
			return errors.New("using 'SYSTEM as step name is not allowed")
		}

		if stepNames[step.Name] {
			return fmt.Errorf("duplicate step name found: '%s'", step.Name)
		}
		stepNames[step.Name] = true

		stepType := step.Type

		if stepType == CliStepType {
			cliCalls = append(cliCalls, fmt.Sprintf("- %s", step.Cmd))

			if step.OutputFilename == "" {
				return fmt.Errorf("step '%s': output filename is mandatory for external CLI", step.Name)
			}

			if err := isValidName(step.OutputFilename); err != nil {
				return fmt.Errorf("step '%s': invalid output filename '%s': %w", step.Name, step.OutputFilename, err)
			}

			if !strings.Contains(step.Cmd, step.OutputFilename) {
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

					// JSON key
					if len(val.Key) > 0 {
						if strings.Contains(val.Key, ".") {
							return fmt.Errorf("placeholders currently support only one level of nesting, step: %s, placeholder: %+v", step.Name, val)
						}

						refStep := c.GetStepByName(val.Step)
						if refStep.Type == PromptStepType {
							if !refStep.JSONSchema.HasSchemaDefinition() {
								return fmt.Errorf("step %s must have JSON schema, key: %s", val.Step, val.Key)
							}

							if !refStep.JSONSchema.HasRequiredProperty(val.Key) {
								return fmt.Errorf("'%s' key must be defined in step %s in JSON schema as a property and required", val.Key, val.Step)
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

			if len(step.OutputFilename) > 0 {
				if err := isValidName(step.OutputFilename); err != nil {
					return fmt.Errorf("step '%s': invalid output filename '%s': %w", step.Name, step.OutputFilename, err)
				}
			}
		}
	}

	if !c.SkipCliWarning && len(cliCalls) > 0 {
		fmt.Printf("⚠️ WARNING: External application call detected! The author assumes no responsibility for execution results. Please verify all external calls before proceeding. Use at your own risk.\n\nCalls: \n%s\n\nPress Enter to continue", strings.Join(cliCalls, "\n"))
		fmt.Scanln() //nolint:golint,errcheck
	}

	return nil
}
