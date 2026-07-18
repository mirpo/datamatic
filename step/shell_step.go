package step

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/mirpo/datamatic/config"
	"github.com/mirpo/datamatic/executor"
)

type ShellStep struct{}

const defaultCmdTimeout = 1 * time.Hour

func (p *ShellStep) Run(ctx context.Context, cfg *config.Config, step config.Step, outputFolder string) error {
	if err := os.MkdirAll(step.WorkDir, 0o755); err != nil {
		return fmt.Errorf("failed to create workDir %s: %w", step.WorkDir, err)
	}

	if err := executor.ExecuteCommand(ctx, step.Run, step.WorkDir, defaultCmdTimeout); err != nil {
		return fmt.Errorf("failed to execute external application: %w", err)
	}

	// the run command is opaque, so verify it actually produced the declared
	// output — otherwise a typo silently breaks later steps that read this file
	if _, err := os.Stat(step.OutputFilename); err != nil {
		return fmt.Errorf("step '%s': command finished but its declared outputFilename '%s' was not created — make sure the run command writes exactly this file: %w",
			step.Name, step.OutputFilename, err)
	}
	return nil
}
