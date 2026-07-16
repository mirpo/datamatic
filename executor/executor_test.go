package executor_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

// W1 regression: Go escapes '"' per MSVCRT rules when building the command
// line, but cmd.exe does not parse backslash escapes — double quotes in run:
// commands must reach the real shell unmangled. These tests execute the real
// platform shell (sh -c / cmd /C) and inspect what actually happened.
func TestExecuteCommand_QuotingReachesShellIntact(t *testing.T) {
	tests := []struct {
		name     string
		unix     string
		windows  string
		wantFile string
		want     string
	}{
		{
			name:     "double-quoted JSON survives to file",
			unix:     `echo '{"a":1}' > out.json`,
			windows:  `echo {"a":1}> out.json`,
			wantFile: "out.json",
			want:     `{"a":1}`,
		},
		{
			name:     "double quotes wrapping single quotes (duckdb-style SQL)",
			unix:     `echo "COPY x TO 'out.csv' (FORMAT JSON)" > q.txt`,
			windows:  `echo "COPY x TO 'out.csv' (FORMAT JSON)"> q.txt`,
			wantFile: "q.txt",
			// cmd's echo keeps the surrounding quotes literally; sh strips them —
			// either way there must be no backslash-escaped quotes inside
			want: `COPY x TO 'out.csv' (FORMAT JSON)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workingDir := t.TempDir()

			stepCmd := tt.unix
			if runtime.GOOS == "windows" {
				stepCmd = tt.windows
			}

			err := executor.ExecuteCommand(context.Background(), stepCmd, workingDir, 30*time.Second)
			assert.NoError(t, err)

			data, err := os.ReadFile(filepath.Join(workingDir, tt.wantFile))
			assert.NoError(t, err)

			content := strings.TrimSpace(string(data))
			assert.NotContains(t, content, `\"`, "double quotes must not arrive backslash-escaped (W1)")
			assert.Equal(t, tt.want, strings.Trim(content, `"`))
		})
	}
}
