package config

import (
	"fmt"
	"log/slog"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mirpo/datamatic/defaults"
	"github.com/mirpo/datamatic/jsonschema"
	"github.com/mirpo/datamatic/llm"
	"github.com/mirpo/datamatic/promptbuilder"
)

func validateVersion(version string) error {
	if version == "" {
		return fmt.Errorf("validating config version: %w", ErrVersionRequired)
	}

	if version != defaults.SupportedConfigVersion {
		return fmt.Errorf("validating config version '%s': %w", version, ErrUnsupportedVersion)
	}

	return nil
}

func isValidName(name string) error {
	if len(name) == 0 {
		return fmt.Errorf("validating filename: %w", ErrFilenameEmpty)
	}

	if len(name) > 255 {
		return fmt.Errorf("validating filename '%s': %w", name, ErrFilenameTooLong)
	}

	illegalChars := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1F]`)
	if illegalChars.MatchString(name) {
		return fmt.Errorf("validating filename '%s': %w", name, ErrFilenameInvalidChars)
	}

	if strings.HasSuffix(name, " ") || (len(name) > 1 && strings.HasSuffix(name, ".")) {
		return fmt.Errorf("validating filename '%s': %w", name, ErrFilenameInvalidSuffix)
	}

	return nil
}

func getStepType(step Step) (StepType, error) {
	promptDefined := len(step.Prompt) > 0
	cmdDefined := len(step.Cmd) > 0

	if promptDefined && cmdDefined {
		return UnknownStepType, fmt.Errorf("validating step type: %w", ErrBothPromptAndCmd)
	}

	if !promptDefined && !cmdDefined {
		return UnknownStepType, fmt.Errorf("validating step type: %w", ErrNeitherPromptNorCmd)
	}

	if promptDefined {
		return PromptStepType, nil
	}

	return CliStepType, nil
}

func getModelDetails(step Step) (llm.ProviderType, string, error) {
	if step.Model == "" {
		return llm.ProviderUnknown, "", fmt.Errorf("parsing model definition: %w", ErrModelEmpty)
	}

	result := strings.SplitN(step.Model, ":", 2)
	if len(result) != 2 {
		return llm.ProviderUnknown, "", fmt.Errorf("parsing model '%s': %w", step.Model, ErrModelInvalidFormat)
	}

	providerStr := result[0]
	modelName := result[1]

	providerType := llm.ProviderType(providerStr)
	switch providerType {
	case llm.ProviderOllama, llm.ProviderLmStudio, llm.ProviderOpenAI, llm.ProviderOpenRouter, llm.ProviderGemini:
	default:
		return llm.ProviderUnknown, "", fmt.Errorf("unsupported provider: %s", providerStr)
	}

	if len(modelName) == 0 {
		return llm.ProviderUnknown, "", fmt.Errorf("parsing model '%s': %w", step.Model, ErrModelNameEmpty)
	}

	return providerType, modelName, nil
}

func validateURL(input string) error {
	parsedURL, err := url.ParseRequestURI(input)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return fmt.Errorf("validating URL '%s': missing scheme or host", input)
	}

	return nil
}

func validateModelConfig(step ModelConfig) error {
	if step.Temperature != nil {
		if *step.Temperature < 0 || *step.Temperature > 1 {
			return fmt.Errorf("validating temperature %f: %w", *step.Temperature, ErrTemperatureOutOfRange)
		}
	}

	if step.MaxTokens != nil {
		if *step.MaxTokens <= 0 {
			return fmt.Errorf("validating maxTokens %d: %w", *step.MaxTokens, ErrMaxTokensInvalid)
		}
	}

	if step.BaseURL != "" {
		if err := validateURL(step.BaseURL); err != nil {
			return fmt.Errorf("invalid baseUrl: %w", err)
		}
	}

	return nil
}

func getFullOutputPath(step Step, outputFolder string) (string, error) {
	extension := defaults.FileExtension

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

func validateAndAbsOutputFolder(outputFolder string) (string, error) {
	if len(outputFolder) == 0 {
		return "", fmt.Errorf("validating output folder: %w", ErrOutputFolderRequired)
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

func validateAndSetMaxResults(step *Step, stepNames map[string]bool) error {
	switch v := step.MaxResults.(type) {
	case nil:
		step.MaxResults = DefaultStepMinMaxResults
		return nil

	case string:
		if v == "" {
			step.MaxResults = DefaultStepMinMaxResults
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

	case int:
		if v <= 0 {
			step.MaxResults = DefaultStepMinMaxResults
		} else {
			step.MaxResults = v
		}
		return nil
	}

	return fmt.Errorf("maxResults unsupported type '%T'", step.MaxResults)
}

func (c *Config) Validate() error {
	slog.Debug("start config validation")

	if err := validateVersion(c.Version); err != nil {
		return err
	}

	absOutputFolder, err := validateAndAbsOutputFolder(c.OutputFolder)
	if err != nil {
		return err
	}
	c.OutputFolder = absOutputFolder

	if len(c.Steps) == 0 {
		return fmt.Errorf("validating config steps: %w", ErrAtLeastOneStepRequired)
	}

	stepNames := map[string]bool{}
	cliCalls := []string{}
	configValidator := jsonschema.NewConfigValidator()

	for index := range c.Steps {
		step := &c.Steps[index]

		if len(step.Name) == 0 {
			return fmt.Errorf("step at index %d: %w", index, ErrStepNameEmpty)
		}

		if strings.ToUpper(step.Name) == defaults.SystemStepName {
			return fmt.Errorf("validating step name '%s': %w", step.Name, ErrSystemStepNameNotAllowed)
		}

		if stepNames[step.Name] {
			return fmt.Errorf("validating step '%s': %w", step.Name, ErrDuplicateStepName)
		}
		stepNames[step.Name] = true

		stepType, err := getStepType(*step)
		if err != nil {
			return fmt.Errorf("step '%s': %w", step.Name, err)
		}
		step.Type = stepType

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
			if configValidator.HasSchemaDefinition(step.JSONSchema) {
				if !configValidator.ValidateRequiredProperties(step.JSONSchema) {
					return fmt.Errorf("step '%s': invalid schema validation, properties or required are not matching", step.Name)
				}
			}

			if strings.Contains(step.Prompt, "{{.SYSTEM.JSON_SCHEMA}}") && !configValidator.ValidateRequiredProperties(step.JSONSchema) {
				return fmt.Errorf("step '%s': JSON schema is required when using '{{.SYSTEM.JSON_SCHEMA}}' in the prompt", step.Name)
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
							if !configValidator.HasSchemaDefinition(refStep.JSONSchema) {
								return fmt.Errorf("step %s must have JSON schema, key: %s", val.Step, val.Key)
							}

							if !configValidator.HasRequiredProperty(refStep.JSONSchema, val.Key) {
								return fmt.Errorf("'%s' key must be defined in step %s in JSON schema as a property and required", val.Key, val.Step)
							}
						}
					}
				}
			}

			llmProvider, modelName, err := getModelDetails(*step)
			if err != nil {
				return fmt.Errorf("step '%s': %w", step.Name, err)
			}
			step.ModelConfig.ModelProvider = llmProvider
			step.ModelConfig.ModelName = modelName

			if err := validateModelConfig(step.ModelConfig); err != nil {
				return fmt.Errorf("step '%s': model config validation failed: %w", step.Name, err)
			}

			if err := validateAndSetMaxResults(step, stepNames); err != nil {
				return fmt.Errorf("step '%s': maxResults validation failed: %w", step.Name, err)
			}

			if len(step.OutputFilename) > 0 {
				if err := isValidName(step.OutputFilename); err != nil {
					return fmt.Errorf("step '%s': invalid output filename '%s': %w", step.Name, step.OutputFilename, err)
				}
			}
		}

		fullOutputPath, err := getFullOutputPath(*step, c.OutputFolder)
		if err != nil {
			return fmt.Errorf("step '%s': failed to get full output path: %w", step.Name, err)
		}
		step.OutputFilename = fullOutputPath

		if step.HasImages() {
			step.ImagePath = strings.TrimSpace(step.ImagePath)

			if !filepath.IsAbs(step.ImagePath) {
				step.ImagePath = filepath.Join(c.OutputFolder, step.ImagePath)
			}
		}
	}

	if !c.SkipCliWarning && len(cliCalls) > 0 {
		fmt.Printf("⚠️ WARNING: External application call detected! The author assumes no responsibility for execution results. Please verify all external calls before proceeding. Use at your own risk.\n\nCalls: \n%s\n\nPress Enter to continue", strings.Join(cliCalls, "\n"))
		fmt.Scanln() //nolint:golint,errcheck
	}

	slog.Debug("config validation successful")
	return nil
}
