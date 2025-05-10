package executor

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/rs/zerolog/log"
)

func ExecuteCommand(ctx context.Context, command string, workingDir string, timeout time.Duration) error {
	log.Info().Msgf("Preparing to run command: %s in directory: %s with timeout: %s", command, workingDir, timeout)

	cmdCtx := ctx
	var cancelFunc context.CancelFunc
	if timeout > 0 {
		cmdCtx, cancelFunc = context.WithTimeout(ctx, timeout)
		defer cancelFunc()
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(cmdCtx, "cmd", "/C", command)
	} else {
		cmd = exec.CommandContext(cmdCtx, "sh", "-c", "set -e; "+command)
	}

	cmd.Dir = workingDir
	cmd.Env = os.Environ()

	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start command '%s': %w", command, err)
	}

	err = cmd.Wait()
	if cmdCtx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("command '%s' timed out after %s. Output: %s, err: %w", command, timeout, output.String(), cmdCtx.Err())
	}

	if err != nil {
		return fmt.Errorf("command '%s' failed with error: %w. Output: %s", command, err, output.String())
	}

	log.Info().Msgf("Command executed successfully: %s", command)
	return nil
}
