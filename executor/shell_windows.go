//go:build windows

package executor

import (
	"context"
	"os/exec"
	"syscall"
)

func newShellCommand(ctx context.Context, command string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "cmd.exe")
	// Go escapes double quotes per MSVCRT rules when joining Args, but cmd.exe
	// does not parse backslash escapes — pass the raw command line instead so
	// quotes in run: commands reach the shell exactly as written (W1).
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: "cmd.exe /C " + command}
	return cmd
}
