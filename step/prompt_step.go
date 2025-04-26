package step

import (
	"context"
	"errors"
	"os"

	"github.com/mirpo/datamatic/config"
	"github.com/mirpo/datamatic/httpclient"
	"github.com/mirpo/datamatic/jsonl"
	"github.com/mirpo/datamatic/llm"
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

		response, err := provider.Generate(ctx, llm.GenerateRequest{
			UserMessage:   step.Prompt,
			SystemMessage: step.SystemPrompt,
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

		log.Info().Msgf("Response from LLM: '%s'", response.Text)

		lineEntity := jsonl.NewLineEntity(response.Text, step.Prompt)

		err = writer.WriteLine(lineEntity)
		if err != nil {
			log.Error().Err(err).Msg("failed to write jsonl line to file")
			break
		}

		i++
	}

	return nil
}
