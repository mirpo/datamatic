package step

import (
	"context"
	"fmt"
	"time"

	"github.com/mirpo/datamatic/config"
	"github.com/mirpo/datamatic/executor"
)

type ShellStep struct{}

const defaultCmdTimeout = 1 * time.Hour

func (p *ShellStep) Run(ctx context.Context, cfg *config.Config, step config.Step, outputFolder string) error {
	if err := executor.ExecuteCommand(ctx, step.Cmd, outputFolder, defaultCmdTimeout); err != nil {
		return fmt.Errorf("failed to execute external application: %w", err)
	}
	return nil
}
