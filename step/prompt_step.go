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

type PromptStep struct{}

func (p *PromptStep) retryLLMGeneration(ctx context.Context, cfg *config.Config, provider llm.Provider, req llm.GenerateRequest, response **llm.GenerateResponse) error {
	retryConfig := retry.Config{
		Enabled:           cfg.RetryConfig.Enabled,
		MaxAttempts:       cfg.RetryConfig.MaxAttempts,
		InitialDelay:      cfg.RetryConfig.InitialDelay,
		MaxDelay:          cfg.RetryConfig.MaxDelay,
		BackoffMultiplier: cfg.RetryConfig.BackoffMultiplier,
	}

	return retry.Do(ctx, retryConfig, func() error {
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

	provider, err := llm.NewProvider(step.GetProviderConfig(cfg.HTTPTimeout))
	if err != nil {
		return fmt.Errorf("failed to create LLM provider: %w", err)
	}

	for i < maxResult {
		log.Info().Msgf("Running step '%s' (type: '%s'), iteration [%d]", step.Name, step.Type, i)

		promptBuilder := promptbuilder.NewPromptBuilder(step.Prompt)
		hasSchemaSchema := step.JSONSchema.HasSchemaDefinition()

		if hasSchemaSchema {
			jsonSchemaAsText, err := step.JSONSchema.MarshalToJSONText()
			if err != nil {
				log.Error().Err(err).Msg("failed to marshal JSON schema to text")
				break
			}

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
				log.Error().Msgf("JSON response: %s, not following JSON schema: %+v, retrying", response.Text, step.JSONSchema)
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
