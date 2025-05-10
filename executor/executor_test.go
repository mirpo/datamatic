package executor_test

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/mirpo/datamatic/executor"
	"github.com/stretchr/testify/assert"
)

func TestExecuteCommand_Success(t *testing.T) {
	workingDir := t.TempDir()
	stepCmd := "echo hello"

	err := executor.ExecuteCommand(context.Background(), stepCmd, workingDir, 5*time.Second)

	assert.NoError(t, err)
}

func TestExecuteCommand_Failure(t *testing.T) {
	workingDir := t.TempDir()
	stepCmd := "false"
	if runtime.GOOS == "windows" {
		stepCmd = "exit 1"
	}

	err := executor.ExecuteCommand(context.Background(), stepCmd, workingDir, 5*time.Second)

	assert.Error(t, err)
}

func TestExecuteCommand_InvalidCommand(t *testing.T) {
	workingDir := t.TempDir()
	stepCmd := "nonexistentcommand12345"

	err := executor.ExecuteCommand(context.Background(), stepCmd, workingDir, 5*time.Second)

	assert.Error(t, err)
}

func TestExecuteCommand_Timeout(t *testing.T) {
	workingDir := t.TempDir()
	stepCmd := "sleep 5"
	if runtime.GOOS == "windows" {
		stepCmd = "ping -n 6 127.0.0.1 > nul"
	}

	timeoutDuration := 2 * time.Second

	err := executor.ExecuteCommand(context.Background(), stepCmd, workingDir, timeoutDuration)

	assert.Error(t, err)
}

func TestExecuteCommand_NoTimeout(t *testing.T) {
	workingDir := t.TempDir()
	stepCmd := "sleep 1"
	if runtime.GOOS == "windows" {
		// Use ping for delay on Windows, more reliable for exiting with status 0
		stepCmd = "ping -n 2 127.0.0.1 > nul" // 2 pings with 1s delay = ~1 second
	}

	err := executor.ExecuteCommand(context.Background(), stepCmd, workingDir, 0)

	assert.NoError(t, err)
}

func TestExecuteCommand_InvalidWorkingDir(t *testing.T) {
	workingDir := "/nonexistent-dir-12345"
	stepCmd := "echo hello"

	err := executor.ExecuteCommand(context.Background(), stepCmd, workingDir, 5*time.Second)

	assert.Error(t, err)
}
