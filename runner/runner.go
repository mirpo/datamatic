package runner

import (
	"context"
	"fmt"

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

// resolveIterations sets how many rows a prompt step produces: forEach source
// row count, image-glob match count, explicit count, or the generator default.
// This is the single place the iteration-source decision lives.
func (r *Runner) resolveIterations(step *config.Step) error {
	switch {
	case step.ForEach != "":
		refStep := r.cfg.GetStepByName(step.ForEach)
		if refStep == nil {
			return fmt.Errorf("forEach references unknown step '%s'", step.ForEach)
		}

		lines, err := fs.CachedLineCount(refStep.OutputFilename)
		if err != nil {
			return fmt.Errorf("failed to count rows of step '%s': %w", step.ForEach, err)
		}

		step.ResolvedCount = lines
		log.Debug().Msgf("Resolved iterations for step '%s' to %d from forEach: %s", step.Name, lines, step.ForEach)

	case step.HasImages() && step.Count == 0:
		images, err := fs.CountFiles(step.ImagePath)
		if err != nil {
			return fmt.Errorf("failed to count images matching '%s': %w", step.ImagePath, err)
		}

		step.ResolvedCount = images
		log.Debug().Msgf("Resolved iterations for step '%s' to %d from imagePath: %s", step.Name, images, step.ImagePath)

	case step.Count == 0:
		step.ResolvedCount = config.DefaultStepCount

	default:
		step.ResolvedCount = step.Count
	}

	return nil
}

func (r *Runner) Run(ctx context.Context) error {
	if err := r.PrepareOutputDirectory(); err != nil {
		return err
	}

	for _, stepConfig := range r.cfg.Steps {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("run cancelled: %w", err)
		}

		log.Info().Msgf("Starting step: '%s' (type: '%s')", stepConfig.Name, stepConfig.Type)

		if stepConfig.Type == config.PromptStepType {
			if err := r.resolveIterations(&stepConfig); err != nil {
				return fmt.Errorf("failed to resolve iterations for step '%s': %w", stepConfig.Name, err)
			}
		}

		runner, err := step.NewStepRunner(stepConfig)
		if err != nil {
			log.Error().Err(err).Msg("failed to create step runner")
			return err
		}

		err = runner.Run(ctx, r.cfg, stepConfig, r.cfg.OutputFolder)
		if err != nil {
			log.Error().Err(err).Msgf("step '%s' failed", stepConfig.Name)
			return fmt.Errorf("step '%s': %w", stepConfig.Name, err)
		}

		log.Info().Msgf("Completed step: %s", stepConfig.Name)
	}

	return nil
}
