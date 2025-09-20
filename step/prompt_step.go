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
	maxResult := step.ResolvedMaxResults
	i := 0

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
			placeholders := promptBuilder.GetPlaceholders()
			for _, placeholder := range placeholders {
				refStep := cfg.GetStepByName(placeholder.Step)
				lineValue, err := readStepValue(*refStep, outputFolder, i, placeholder.Key)
				if err != nil {
					log.Error().Err(err).Msg("failed to read value from the ref step")
					break
				}

				log.Debug().Msgf("placeholder: %+v, read line: '%s'", placeholder, lineValue)
				promptBuilder.AddValue(lineValue.ID, placeholder.Step, placeholder.Key, lineValue.Response)
			}
		}

		userPrompt, err := promptBuilder.BuildPrompt()
		if err != nil {
			log.Error().Err(err).Msg("failed to build user prompt")
			break
		}

		var base64Image string
		if step.HasImages() {
			imagePath, err := fs.PickImageFile(step.ImagePath, i)
			if err != nil {
				log.Error().Err(err).Msgf("failed to find images by pattern: %s, err: %s", step.ImagePath, err)
				break
			}

			base64Image, err = fs.ImageToBase64(imagePath)
			if err != nil {
				log.Error().Err(err).Msgf("failed to get base64 of image: %s, err: %s", imagePath, err)
				break
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
			err := step.JSONSchema.ValidateJSONText(response.Text)
			if err != nil {
				log.Error().Msgf("JSON response: %s, not following JSON schema. List of errors: %s, retrying", response.Text, err.Error())
				continue
			}
		}

		log.Info().Msgf("Response from LLM: '%s'", response.Text)

		lineEntity, err := jsonl.NewLineEntity(response.Text, userPrompt, hasSchemaSchema, promptBuilder.GetValues())
		if err != nil {
			log.Err(err).Msgf("got invalid JSON: %+v", response.Text)
			continue
		}

		err = writer.WriteLine(lineEntity)
		if err != nil {
			log.Error().Err(err).Msg("failed to write jsonl line to file")
			break
		}

		i++
	}

	return nil
}
