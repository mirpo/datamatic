package step

import (
	"context"
	"fmt"

	"github.com/mirpo/datamatic/config"
	"github.com/mirpo/datamatic/defaults"
	"github.com/mirpo/datamatic/executor"
)

type CliStep struct{}

func (p *CliStep) Run(ctx context.Context, cfg *config.Config, step config.Step, outputFolder string) error {
	if err := executor.ExecuteCommand(ctx, step.Cmd, outputFolder, defaults.CmdTimeout); err != nil {
		return fmt.Errorf("failed to execute external application: %w", err)
	}
	return nil
}
