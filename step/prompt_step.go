package step

import (
	"context"
	"errors"
	"os"

	"github.com/mirpo/datamatic/config"
	"github.com/mirpo/datamatic/httpclient"
	"github.com/mirpo/datamatic/jsonl"
	"github.com/mirpo/datamatic/llm"
	"github.com/mirpo/datamatic/promptbuilder"
	"github.com/rs/zerolog/log"
)

type PromptStep struct{}

func (p *PromptStep) Run(ctx context.Context, cfg *config.Config, step config.Step, outputFolder string) error {
	maxResult := step.MaxResults
	i := 0

	writer, err := jsonl.NewWriter(step.OutputFilename)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create JSONL writer")
		return err
	}
	defer writer.Close()

	provider, err := llm.NewProvider(step.GetProviderConfig(cfg.HTTPTimeout))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create LLM provider")
	}

	for i < maxResult {
		log.Info().Msgf("Running step '%s' (type: '%s'), iteration [%d]", step.Name, step.Type, i)

		prompt := step.Prompt
		hasSchemaSchema := step.JSONSchema.HasSchemaDefinition()
		placeholderValues := map[string]interface{}{}

		if hasSchemaSchema {
			jsonSchemaAsText, err := step.JSONSchema.MarshalToJSONText()
			if err != nil {
				log.Error().Err(err).Msg("failed to marshal JSON schema to text")
				break
			}

			placeholderValues["SYSTEM"] = map[string]interface{}{
				"JSON_SCHEMA": jsonSchemaAsText,
			}

			prompt, err = promptbuilder.GetCompiledPrompt(prompt, placeholderValues)
			if err != nil {
				log.Error().Err(err).Msg("failed to get compiled prompt")
				break
			}
		}

		response, err := provider.Generate(ctx, llm.GenerateRequest{
			UserMessage:   prompt,
			SystemMessage: step.SystemPrompt,
			IsJSON:        hasSchemaSchema,
			JSONSchema:    step.JSONSchema,
		})
		if err != nil {
			var errCustom *httpclient.HTTPError
			if errors.As(err, &errCustom) {
				log.Error().Err(err).Msgf("model %s is not found, please check %s provider config", step.ModelConfig.ModelName, step.ModelConfig.ModelProvider)
				os.Exit(1)
			}

			// TODO add max retries based on the error, model not found - exit
			log.Error().Err(err).Msg("failed to get response from LLM")
			break
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

		lineEntity, err := jsonl.NewLineEntity(response.Text, step.Prompt, hasSchemaSchema)
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
