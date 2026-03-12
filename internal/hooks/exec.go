package hooks

import (
	"bytes"
	"context"
	"os"
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
		wrapper, err := os.CreateTemp("", "gwt-hook-*.cmd")
		if err != nil {
			return nil, err
		}
		defer os.Remove(wrapper.Name())

		script := "@echo off\r\n" + opts.Command + "\r\n"
		if _, err := wrapper.WriteString(script); err != nil {
			wrapper.Close()
			return nil, err
		}
		if err := wrapper.Close(); err != nil {
			return nil, err
		}

		// Use a temporary .cmd wrapper to avoid Go's argv quoting mismatch with
		// cmd.exe and batch files on Windows.
		cmd = exec.CommandContext(ctx, "cmd.exe", "/D", "/C", wrapper.Name())
	} else {
		cmd = exec.CommandContext(ctx, "/bin/sh", "-c", opts.Command)
	}

	cmd.Dir = opts.Dir
	if len(opts.Env) > 0 {
		cmd.Env = append(os.Environ(), opts.Env...)
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
