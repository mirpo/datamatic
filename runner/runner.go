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

func (r *Runner) Run() error {
	if err := r.PrepareOutputDirectory(); err != nil {
		return err
	}

	for _, stepConfig := range r.cfg.Steps {
		log.Info().Msgf("Starting step: '%s' (type: '%s')", stepConfig.Name, stepConfig.Type)

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
