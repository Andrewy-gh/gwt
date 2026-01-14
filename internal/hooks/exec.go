package hooks

import (
	"bytes"
	"context"
	"os/exec"
	"runtime"
	"time"
)

const DefaultTimeout = 5 * time.Minute

type ExecResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

type ExecOptions struct {
	Command string
	Dir     string
	Env     []string
	Timeout time.Duration
}

// ExecuteCommand runs a shell command with the given options.
// On Unix, uses /bin/sh -c; on Windows, uses cmd.exe /C.
func ExecuteCommand(opts ExecOptions) (*ExecResult, error) {
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd.exe", "/C", opts.Command)
	} else {
		cmd = exec.CommandContext(ctx, "/bin/sh", "-c", opts.Command)
	}

	cmd.Dir = opts.Dir
	if len(opts.Env) > 0 {
		cmd.Env = opts.Env
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := &ExecResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	if ctx.Err() == context.DeadlineExceeded {
		return result, ErrHookTimeout
	}

	if exitErr, ok := err.(*exec.ExitError); ok {
		result.ExitCode = exitErr.ExitCode()
		return result, nil // Non-zero exit is not a Go error
	}

	if err != nil {
		return result, err
	}

	return result, nil
}
