package step

import (
	"context"
	"fmt"
	"time"

	"github.com/mirpo/datamatic/config"
	"github.com/mirpo/datamatic/executor"
)

type CliStep struct{}

const defaultCmdTimeout = 1 * time.Hour

func (p *CliStep) Run(ctx context.Context, cfg *config.Config, step config.Step, outputFolder string) error {
	if err := executor.ExecuteCommand(ctx, step.Cmd, outputFolder, defaultCmdTimeout); err != nil {
		return fmt.Errorf("failed to execute external application: %w", err)
	}
	return nil
}
