package runner

import (
	"context"
	"fmt"
	"strings"

	"github.com/mirpo/datamatic/config"
	"github.com/mirpo/datamatic/fs"
	"github.com/mirpo/datamatic/step"
	"github.com/rs/zerolog/log"
)

type Runner struct {
	cfg *config.Config
}

func NewRunner(cfg *config.Config) *Runner {
	return &Runner{
		cfg: cfg,
	}
}

func (r *Runner) PrepareOutputDirectory() error {
	log.Debug().Msgf("Preparing root output folder: %s", r.cfg.OutputFolder)

	if err := fs.CreateVersionedFolder(r.cfg.OutputFolder); err != nil {
		return fmt.Errorf("failed to prepare output directory: %w", err)
	}

	log.Debug().Msg("Root output folder prepared successfully.")
	return nil
}

func (r *Runner) resolveMaxResults(step *config.Step) error {
	switch v := step.MaxResults.(type) {
	case int:
		step.ResolvedMaxResults = v
		return nil

	case string:
		if strings.HasSuffix(v, ".$length") {
			refStepName := strings.TrimSuffix(v, ".$length")
			refStep := r.cfg.GetStepByName(refStepName)

			if refStep == nil {
				return fmt.Errorf("reference step '%s' not found", refStepName)
			}

			isImageStep := step.HasImages()
			if isImageStep {
				imagesCount, err := fs.CountFiles(step.ImagePath)
				if err != nil {
					return fmt.Errorf("failed to count images in folder '%s': %w", step.ImagePath, err)
				}
				step.ResolvedMaxResults = imagesCount
				log.Debug().Msgf("Resolved MaxResults for step '%s' to %d from refStep: %s", step.Name, imagesCount, refStepName)
			} else {
				lines, err := fs.CountLinesInFile(refStep.OutputFilename)
				if err != nil {
					return fmt.Errorf("failed to count lines in '%s': %w", refStep.OutputFilename, err)
				}

				step.ResolvedMaxResults = lines
				log.Debug().Msgf("Resolved MaxResults for step '%s' to %d from refStep: %s", step.Name, lines, refStepName)
			}

			return nil
		}
	}

	return fmt.Errorf("unexpected MaxResults value: %v", step.MaxResults)
}

func (r *Runner) Run() error {
	if err := r.PrepareOutputDirectory(); err != nil {
		return err
	}

	for _, stepConfig := range r.cfg.Steps {
		log.Info().Msgf("Starting step: '%s' (type: '%s')", stepConfig.Name, stepConfig.Type)

		if stepConfig.Type == config.PromptStepType {
			if err := r.resolveMaxResults(&stepConfig); err != nil {
				return fmt.Errorf("failed to resolve MaxResults for step '%s', err: %w", stepConfig.Name, err)
			}
		}

		runner, err := step.NewStepRunner(stepConfig)
		if err != nil {
			log.Error().Err(err).Msg("failed to create step runner")
			return err
		}

		err = runner.Run(context.Background(), r.cfg, stepConfig, r.cfg.OutputFolder)
		if err != nil {
			log.Error().Err(err).Msgf("step '%s' failed", stepConfig.Name)
			return err
		}

		log.Info().Msgf("Completed step: %s", stepConfig.Name)
	}

	return nil
}
