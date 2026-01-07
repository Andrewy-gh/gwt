package cli

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/Andrewy-gh/gwt/internal/output"
)

// Common exit codes
const (
	ExitSuccess      = 0
	ExitGeneralError = 1
	ExitGitNotFound  = 2
	ExitNotGitRepo   = 3
	ExitConfigError  = 4
)

// ExitError wraps an error with an exit code
type ExitError struct {
	Err      error
	ExitCode int
	Message  string
}

// Error implements the error interface
func (e *ExitError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return "unknown error"
}

// Unwrap returns the underlying error
func (e *ExitError) Unwrap() error {
	return e.Err
}

// NewExitError creates a new ExitError
func NewExitError(err error, code int, message string) *ExitError {
	return &ExitError{
		Err:      err,
		ExitCode: code,
		Message:  message,
	}
}

// HandleError handles an error and exits with the appropriate code
func HandleError(err error) {
	if err == nil {
		return
	}

	// Check if it's an ExitError
	if exitErr, ok := err.(*ExitError); ok {
		output.Error(exitErr.Error())

		// Show stack trace in verbose mode
		if GetVerbose() && exitErr.Err != nil {
			output.Verbose(fmt.Sprintf("Caused by: %v", exitErr.Err))
			output.Verbose(fmt.Sprintf("Stack trace:\n%s", debug.Stack()))
		}

		os.Exit(exitErr.ExitCode)
	}

	// Generic error
	output.Error(err.Error())

	if GetVerbose() {
		output.Verbose(fmt.Sprintf("Stack trace:\n%s", debug.Stack()))
	}

	os.Exit(ExitGeneralError)
}

// GitNotFoundError creates a helpful error for when git is not found
func GitNotFoundError() *ExitError {
	msg := `Git is not installed or not in PATH.

gwt requires Git to be installed. Please install Git:
  • Windows: https://git-scm.com/download/win
  • macOS:   brew install git
  • Linux:   apt install git / dnf install git

After installing, restart your terminal and try again.`

	return &ExitError{
		ExitCode: ExitGitNotFound,
		Message:  msg,
	}
}

// NotGitRepoError creates a helpful error for when not in a git repository
func NotGitRepoError() *ExitError {
	msg := `Not a git repository.

gwt must be run from within a Git repository. Navigate to your project
directory and initialize a repository with:

  git init

Or clone an existing repository:

  git clone <repository-url>`

	return &ExitError{
		ExitCode: ExitNotGitRepo,
		Message:  msg,
	}
}

// ConfigError creates an error for configuration issues
func ConfigError(err error, details string) *ExitError {
	msg := fmt.Sprintf("Configuration error: %s", details)
	if err != nil {
		msg = fmt.Sprintf("%s\n\nDetails: %v", msg, err)
	}

	return &ExitError{
		Err:      err,
		ExitCode: ExitConfigError,
		Message:  msg,
	}
}
