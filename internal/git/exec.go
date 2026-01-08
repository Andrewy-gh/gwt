package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/Andrewy-gh/gwt/internal/output"
)

const (
	// DefaultTimeout is the default timeout for git commands
	DefaultTimeout = 30 * time.Second
)

// RunResult contains the result of a git command execution
type RunResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// Run executes a git command and returns the result
// Example: Run("status", "--porcelain")
func Run(args ...string) (*RunResult, error) {
	return RunWithOptions(RunOptions{
		Args: args,
	})
}

// RunInDir executes a git command in a specific directory
func RunInDir(dir string, args ...string) (*RunResult, error) {
	return RunWithOptions(RunOptions{
		Dir:  dir,
		Args: args,
	})
}

// RunWithStdin executes a git command with stdin input
func RunWithStdin(stdin string, args ...string) (*RunResult, error) {
	return RunWithOptions(RunOptions{
		Args:  args,
		Stdin: stdin,
	})
}

// MustRun executes a git command and panics on error (for internal use only)
// This should only be used in situations where failure is truly unexpected
func MustRun(args ...string) *RunResult {
	result, err := Run(args...)
	if err != nil {
		panic(fmt.Sprintf("git command failed: %v", err))
	}
	return result
}

// RunOptions configures how a git command is executed
type RunOptions struct {
	// Args are the git command arguments (without "git" prefix)
	Args []string

	// Dir is the working directory for the command
	// If empty, uses the current working directory
	Dir string

	// Stdin is the input to pipe to the command
	Stdin string

	// Timeout is the maximum time to wait for the command
	// If zero, uses DefaultTimeout
	Timeout time.Duration

	// AllowFailure if true, does not return an error on non-zero exit code
	AllowFailure bool
}

// RunWithOptions executes a git command with the specified options
func RunWithOptions(opts RunOptions) (*RunResult, error) {
	// Set default timeout
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Prepare command
	cmd := exec.CommandContext(ctx, "git", opts.Args...)

	// Set working directory if specified
	if opts.Dir != "" {
		cmd.Dir = opts.Dir
	}

	// Set stdin if provided
	if opts.Stdin != "" {
		cmd.Stdin = strings.NewReader(opts.Stdin)
	}

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Log command in verbose mode
	cmdStr := formatCommand(opts.Dir, opts.Args)
	output.Verbose(fmt.Sprintf("$ %s", cmdStr))

	// Execute command
	err := cmd.Run()

	// Get exit code
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			// Command failed to start or context timeout
			return nil, fmt.Errorf("failed to execute git command: %w", err)
		}
	}

	// Clean output (strip ANSI codes if not in terminal)
	stdoutStr := cleanOutput(stdout.String())
	stderrStr := cleanOutput(stderr.String())

	result := &RunResult{
		Stdout:   stdoutStr,
		Stderr:   stderrStr,
		ExitCode: exitCode,
	}

	// Return error if command failed and failure is not allowed
	if exitCode != 0 && !opts.AllowFailure {
		return result, &GitError{
			Command:  append([]string{"git"}, opts.Args...),
			Stderr:   stderrStr,
			ExitCode: exitCode,
		}
	}

	return result, nil
}

// formatCommand formats a command for display
func formatCommand(dir string, args []string) string {
	cmd := "git " + strings.Join(args, " ")
	if dir != "" {
		cmd = fmt.Sprintf("(cd %s && %s)", dir, cmd)
	}
	return cmd
}

// cleanOutput removes ANSI escape codes from output
// This is useful when output is captured or logged
func cleanOutput(s string) string {
	// Strip ANSI escape codes
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return ansiRegex.ReplaceAllString(s, "")
}

// TrimOutput returns the stdout with leading/trailing whitespace removed
func (r *RunResult) TrimOutput() string {
	return strings.TrimSpace(r.Stdout)
}

// Lines returns the stdout split into lines
func (r *RunResult) Lines() []string {
	output := r.TrimOutput()
	if output == "" {
		return []string{}
	}
	return strings.Split(output, "\n")
}

// Success returns true if the command succeeded (exit code 0)
func (r *RunResult) Success() bool {
	return r.ExitCode == 0
}

// Failed returns true if the command failed (exit code non-zero)
func (r *RunResult) Failed() bool {
	return r.ExitCode != 0
}
