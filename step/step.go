package step

import (
	"context"
	"errors"

	"github.com/mirpo/datamatic/config"
)

type StepRunner interface {
	Run(ctx context.Context, cfg *config.Config, step config.Step, outputFolder string) error
}

func NewStepRunner(step config.Step) (StepRunner, error) {
	switch step.Type {
	case config.PromptStepType:
		return &PromptStep{}, nil
	case config.CliStepType:
		return &CliStep{}, nil
	default:
		return nil, errors.New("unsupported step type")
	}
}
