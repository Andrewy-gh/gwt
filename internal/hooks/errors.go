package hooks

import (
	"errors"
	"fmt"
)

var (
	ErrHookFailed        = errors.New("hook execution failed")
	ErrNoHooksConfigured = errors.New("no hooks configured")
	ErrInvalidHookType   = errors.New("invalid hook type")
	ErrHookTimeout       = errors.New("hook execution timed out")
)

type HookError struct {
	Command  string
	ExitCode int
	Stderr   string
	Err      error
}

func (e *HookError) Error() string {
	if e.Stderr != "" {
		return fmt.Sprintf("hook '%s' failed (exit %d): %s", e.Command, e.ExitCode, e.Stderr)
	}
	return fmt.Sprintf("hook '%s' failed (exit %d)", e.Command, e.ExitCode)
}

func (e *HookError) Unwrap() error {
	return e.Err
}
