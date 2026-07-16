package step

import (
	"context"
	"fmt"

	"github.com/mirpo/datamatic/config"
	"github.com/mirpo/datamatic/fs"
	"github.com/mirpo/datamatic/jsonl"
	"github.com/mirpo/datamatic/llm"
	"github.com/mirpo/datamatic/promptbuilder"
	"github.com/mirpo/datamatic/retry"
	"github.com/rs/zerolog/log"
)

func newProviderConfigFromStep(step config.Step, httpTimeout int) llm.ProviderConfig {
	return llm.ProviderConfig{
		BaseURL:      step.ModelConfig.BaseURL,
		ProviderType: step.ModelConfig.ModelProvider,
		ModelName:    step.ModelConfig.ModelName,
		AuthToken:    "token",
		HTTPTimeout:  httpTimeout,
		Temperature:  step.ModelConfig.Temperature,
		MaxTokens:    step.ModelConfig.MaxTokens,
	}
}

type PromptStep struct{}

func (p *PromptStep) retryLLMGeneration(ctx context.Context, cfg *config.Config, provider llm.Provider, req llm.GenerateRequest, response **llm.GenerateResponse) error {
	return retry.Do(ctx, cfg.RetryConfig, func() error {
		resp, err := provider.Generate(ctx, req)
		if err == nil {
			*response = resp
			return nil
		}
		return err
	}, retry.ShouldRetryHTTPError)
}

func (p *PromptStep) Run(ctx context.Context, cfg *config.Config, step config.Step, outputFolder string) error {
	maxResult := step.ResolvedCount
	i := 0

	// registerInvalid tracks consecutive invalid LLM responses; it returns a
	// terminal error once the budget (retryConfig.maxAttempts) is exhausted.
	invalidAttempts := 0
	registerInvalid := func(cause error, responseText string) error {
		invalidAttempts++
		log.Warn().Err(cause).Msgf("invalid LLM response (attempt %d/%d): %s",
			invalidAttempts, cfg.RetryConfig.MaxAttempts, responseText)
		if invalidAttempts >= cfg.RetryConfig.MaxAttempts {
			return fmt.Errorf("row %d: LLM returned invalid response %d times in a row: %w", i, invalidAttempts, cause)
		}
		return nil
	}

	writer, err := jsonl.NewWriter(step.OutputFilename)
	if err != nil {
		return fmt.Errorf("failed to create JSONL writer: %w", err)
	}
	defer writer.Close()

	provider, err := llm.NewProvider(newProviderConfigFromStep(step, cfg.HTTPTimeout))
	if err != nil {
		return fmt.Errorf("failed to create LLM provider: %w", err)
	}

	for i < maxResult {
		log.Info().
			Str("step_name", step.Name).
			Str("step_type", string(step.Type)).
			Int("iteration", i).
			Msg("Running step")

		promptBuilder := promptbuilder.NewPromptBuilder(step.Prompt)
		hasSchemaSchema := step.JSONSchema.HasSchemaDefinition()

		if hasSchemaSchema {
			jsonSchemaAsText := step.JSONSchema.ToJSONString()
			promptBuilder.AddValue("-", "SYSTEM", "JSON_SCHEMA", jsonSchemaAsText)
		}

		if promptBuilder.HasPlaceholders() {
			stepGroups := promptBuilder.GroupPlaceholdersByStep()

			for stepName, fieldPaths := range stepGroups {
				refStep := cfg.GetStepByName(stepName)
				if refStep == nil {
					return fmt.Errorf("prompt references unknown step '%s'", stepName)
				}

				stepValues, err := readStepValuesBatch(*refStep, outputFolder, i, fieldPaths)
				if err != nil {
					return fmt.Errorf("failed to read values from step '%s': %w", stepName, err)
				}
				for fieldPath, stepValue := range stepValues {
					log.Debug().Msgf("step: %s, field: %s, value: %s", stepName, fieldPath, stepValue.Content)
				}

				promptBuilder.AddStepValues(stepName, stepValues)
			}
		}

		userPrompt, err := promptBuilder.BuildPrompt()
		if err != nil {
			return fmt.Errorf("failed to build prompt: %w", err)
		}

		var base64Image string
		if step.HasImages() {
			imagePath, err := fs.PickImageFile(step.ImagePath, i)
			if err != nil {
				return fmt.Errorf("failed to find images by pattern '%s': %w", step.ImagePath, err)
			}

			base64Image, err = fs.ImageToBase64(imagePath)
			if err != nil {
				return fmt.Errorf("failed to encode image '%s': %w", imagePath, err)
			}

			promptBuilder.AddValue(base64Image[:15], step.Name, "image", imagePath)
		}

		var response *llm.GenerateResponse
		err = p.retryLLMGeneration(ctx, cfg, provider, llm.GenerateRequest{
			UserMessage:   userPrompt,
			SystemMessage: step.SystemPrompt,
			IsJSON:        hasSchemaSchema,
			JSONSchema:    step.JSONSchema,
			Base64Image:   base64Image,
		}, &response)
		if err != nil {
			return fmt.Errorf("failed to get response from LLM after retries: %w", err)
		}

		if cfg.ValidateResponse && hasSchemaSchema {
			log.Debug().Msg("Validating response from LLM using JSON schema")
			if err := step.JSONSchema.ValidateJSONText(response.Text); err != nil {
				if failErr := registerInvalid(err, response.Text); failErr != nil {
					return failErr
				}
				continue
			}
		}

		log.Info().Msgf("Response from LLM: '%s'", response.Text)

		lineEntity, err := jsonl.NewLineEntity(response.Text, userPrompt, hasSchemaSchema, promptBuilder.GetValues())
		if err != nil {
			if failErr := registerInvalid(err, response.Text); failErr != nil {
				return failErr
			}
			continue
		}

		err = writer.WriteLine(lineEntity)
		if err != nil {
			return fmt.Errorf("failed to write output line: %w", err)
		}

		invalidAttempts = 0
		i++
	}

	return nil
}
