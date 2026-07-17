package utils

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mirpo/datamatic/config"
	"github.com/mirpo/datamatic/jq"
	"github.com/mirpo/datamatic/jsonschema"
	"github.com/mirpo/datamatic/llm"
	"github.com/mirpo/datamatic/promptbuilder"
	"github.com/mirpo/datamatic/retry"
)

// setStepType determines and sets the step type based on step configuration
func setStepType(step *config.Step) error {
	switch step.Type {
	case "", config.PromptStepType, config.ShellStepType, config.TransformStepType:
	default:
		return fmt.Errorf("unknown step type '%s' (expected 'prompt', 'shell' or 'transform')", step.Type)
	}

	var inferred config.StepType
	var sourceField string
	count := 0
	if step.Prompt != "" {
		inferred, sourceField, count = config.PromptStepType, "prompt", count+1
	}
	if step.Run != "" {
		inferred, sourceField, count = config.ShellStepType, "run", count+1
	}
	if step.JQ != "" {
		inferred, sourceField, count = config.TransformStepType, "jq", count+1
	}
	if count != 1 {
		return errors.New("exactly one of 'prompt', 'run' or 'jq' must be defined")
	}

	if step.Type != "" && step.Type != inferred {
		return fmt.Errorf("explicit type '%s' does not match step definition (inferred '%s' from '%s' field)",
			step.Type, inferred, sourceField)
	}

	step.Type = inferred
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

	// retryConfig not set in YAML (zero values) falls back to defaults
	if cfg.RetryConfig.MaxAttempts == 0 {
		cfg.RetryConfig = retry.NewDefaultConfig()
	}

	stepNames := make(map[string]bool, len(cfg.Steps))
	stepByName := make(map[string]*config.Step, len(cfg.Steps))

	for i := range cfg.Steps {
		step := &cfg.Steps[i]

		// Step name checks
		if strings.TrimSpace(step.Name) == "" {
			return fmt.Errorf("step at index %d: name can't be empty", i)
		}
		if strings.ToUpper(step.Name) == "SYSTEM" {
			return fmt.Errorf("using 'SYSTEM' as step name is not allowed (reserved)")
		}
		if step.Name == promptbuilder.ItemAliasName {
			return fmt.Errorf("using '%s' as step name is not allowed (reserved for forEach references)", promptbuilder.ItemAliasName)
		}
		if stepNames[step.Name] {
			return fmt.Errorf("duplicate step name found: '%s'", step.Name)
		}

		// Step type (prompt vs shell vs transform)
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

		// Shell steps
		if step.Type == config.ShellStepType {
			if step.OutputFilename == "" {
				return fmt.Errorf("step '%s': output filename is mandatory for shell steps", step.Name)
			}
			if err := isValidName(step.OutputFilename); err != nil {
				return fmt.Errorf("step '%s': invalid output filename '%s': %w",
					step.Name, step.OutputFilename, err)
			}

			// Set workDir first (before OutputFilename)
			if err := setWorkDir(step, cfg.OutputFolder); err != nil {
				return fmt.Errorf("step '%s': %w", step.Name, err)
			}

			// Join OutputFilename with workDir (not outputFolder)
			step.OutputFilename = filepath.Join(step.WorkDir, step.OutputFilename)
		}

		// Prompt steps
		if step.Type == config.PromptStepType {
			if err := setOutputFilename(step, cfg.OutputFolder); err != nil {
				return fmt.Errorf("step '%s': %w", step.Name, err)
			}
		}

		// Transform steps
		if step.Type == config.TransformStepType {
			if step.From == "" {
				return fmt.Errorf("step '%s': 'from' is required for transform steps", step.Name)
			}
			if err := requireEarlierStep(stepNames, "from", step.From); err != nil {
				return fmt.Errorf("step '%s': %w", step.Name, err)
			}
			if step.Limit < 0 {
				return fmt.Errorf("step '%s': limit must be >= 0", step.Name)
			}

			program, err := jq.Compile(step.JQ)
			if err != nil {
				return fmt.Errorf("step '%s': %w", step.Name, err)
			}
			step.JQProgram = program

			if err := setOutputFilename(step, cfg.OutputFolder); err != nil {
				return fmt.Errorf("step '%s': %w", step.Name, err)
			}
		}

		// Normalize image path if needed
		if step.HasImages() {
			if err := setImagePath(step, cfg.OutputFolder); err != nil {
				return fmt.Errorf("step '%s': %w", step.Name, err)
			}
		}

		if err := validateIterationSettings(step, stepNames); err != nil {
			return fmt.Errorf("step '%s': %w", step.Name, err)
		}

		if step.Type == config.PromptStepType {
			if err := validatePromptPlaceholders(step, stepByName); err != nil {
				return fmt.Errorf("step '%s': %w", step.Name, err)
			}
		}

		stepNames[step.Name] = true
		stepByName[step.Name] = step
	}

	return nil
}

// validatePromptPlaceholders checks every {{.step.field}} reference in the
// prompt against earlier steps: the step must exist ({{.item}} aliases the
// forEach source), field references into prompt steps must match their JSON
// schema, and a step may not be referenced both as a whole and by field in
// one prompt.
func validatePromptPlaceholders(step *config.Step, stepByName map[string]*config.Step) error {
	builder, err := promptbuilder.NewPromptBuilder(step.Prompt, step.ForEach)
	if err != nil {
		return err
	}
	if !builder.HasPlaceholders() {
		return nil
	}

	keysByStep := map[string]map[bool]bool{} // step -> {isWhole -> seen}

	for _, ref := range builder.GetPlaceholders() {
		refStep, ok := stepByName[ref.Step]
		if !ok {
			return fmt.Errorf("prompt references unknown step '%s' (must be an earlier step)", ref.Step)
		}

		if keysByStep[ref.Step] == nil {
			keysByStep[ref.Step] = map[bool]bool{}
		}
		keysByStep[ref.Step][ref.Key == ""] = true

		if ref.Key != "" && refStep.Type == config.PromptStepType {
			if !refStep.JSONSchema.HasSchemaDefinition() {
				return fmt.Errorf("step '%s' must have a JSON schema to reference field '%s'", ref.Step, ref.Key)
			}
			if !refStep.JSONSchema.HasFieldPath(ref.Key) {
				return fmt.Errorf("field path '%s' not found in step '%s' JSON schema", ref.Key, ref.Step)
			}
		}
	}

	for name, kinds := range keysByStep {
		if kinds[true] && kinds[false] {
			return fmt.Errorf("step '%s' is referenced both as a whole ({{.%s}}) and by field — use one style", name, name)
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

// requireEarlierStep checks that a cross-step reference points at an already
// defined step. stepNames must hold earlier steps only.
func requireEarlierStep(stepNames map[string]bool, field, name string) error {
	if !stepNames[name] {
		return fmt.Errorf("'%s' references unknown step '%s' (must be an earlier step)", field, name)
	}
	return nil
}

// validateIterationSettings checks count/forEach consistency; iteration
// counts themselves are resolved at runtime by the runner.
func validateIterationSettings(step *config.Step, stepNames map[string]bool) error {
	if step.Type != config.PromptStepType {
		if step.Count != 0 || step.ForEach != "" {
			return fmt.Errorf("'count' and 'forEach' are only valid on prompt steps")
		}
		return nil
	}

	if step.Count < 0 {
		return fmt.Errorf("count must be >= 0")
	}
	if step.Count > 0 && step.ForEach != "" {
		return fmt.Errorf("either 'count' or 'forEach' may be set, not both")
	}
	if step.ForEach != "" {
		if err := requireEarlierStep(stepNames, "forEach", step.ForEach); err != nil {
			return err
		}
	}

	return nil
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

// setWorkDir sets and normalizes the working directory for shell steps
func setWorkDir(step *config.Step, outputFolder string) error {
	if step.WorkDir == "" {
		step.WorkDir = outputFolder
		return nil
	}

	if !filepath.IsAbs(step.WorkDir) {
		step.WorkDir = filepath.Join(outputFolder, step.WorkDir)
	}

	absPath, err := filepath.Abs(step.WorkDir)
	if err != nil {
		return fmt.Errorf("invalid workDir path: %w", err)
	}

	step.WorkDir = absPath
	return nil
}
