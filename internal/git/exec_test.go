package git

import (
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	// Test basic command execution
	result, err := Run("--version")
	if err != nil {
		t.Fatalf("git --version failed: %v", err)
	}

	if !result.Success() {
		t.Errorf("expected success, got exit code %d", result.ExitCode)
	}

	if !strings.Contains(result.Stdout, "git version") {
		t.Errorf("expected 'git version' in output, got: %s", result.Stdout)
	}
}

func TestRunInDir(t *testing.T) {
	// Test running command in specific directory
	// Using a command that works outside a git repo
	result, err := Run("--version")
	if err != nil {
		t.Fatalf("git --version failed: %v", err)
	}

	if !result.Success() {
		t.Errorf("expected success, got exit code %d", result.ExitCode)
	}
}

func TestRunWithOptions_Timeout(t *testing.T) {
	// Test that timeout works (though we won't actually trigger it)
	result, err := RunWithOptions(RunOptions{
		Args:    []string{"--version"},
		Timeout: 5 * 1000000000, // 5 seconds in nanoseconds
	})

	if err != nil {
		t.Fatalf("git --version failed: %v", err)
	}

	if !result.Success() {
		t.Errorf("expected success, got exit code %d", result.ExitCode)
	}
}

func TestRunWithOptions_AllowFailure(t *testing.T) {
	// Test that AllowFailure prevents errors on non-zero exit codes
	result, err := RunWithOptions(RunOptions{
		Args:         []string{"invalid-command"},
		AllowFailure: true,
	})

	// Should not return error even though command failed
	if err != nil {
		t.Errorf("expected no error with AllowFailure, got: %v", err)
	}

	if result.Success() {
		t.Errorf("expected command to fail, but it succeeded")
	}
}

func TestRunResult_TrimOutput(t *testing.T) {
	result := &RunResult{
		Stdout:   "  test output  \n",
		ExitCode: 0,
	}

	trimmed := result.TrimOutput()
	if trimmed != "test output" {
		t.Errorf("expected 'test output', got: %q", trimmed)
	}
}

func TestRunResult_Lines(t *testing.T) {
	result := &RunResult{
		Stdout:   "line1\nline2\nline3",
		ExitCode: 0,
	}

	lines := result.Lines()
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}

	if lines[0] != "line1" || lines[1] != "line2" || lines[2] != "line3" {
		t.Errorf("unexpected lines: %v", lines)
	}
}

func TestRunResult_Lines_Empty(t *testing.T) {
	result := &RunResult{
		Stdout:   "",
		ExitCode: 0,
	}

	lines := result.Lines()
	if len(lines) != 0 {
		t.Errorf("expected 0 lines, got %d", len(lines))
	}
}

func TestGitError_Error(t *testing.T) {
	err := &GitError{
		Command:  []string{"git", "invalid-command"},
		Stderr:   "error: unknown command",
		ExitCode: 1,
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "git invalid-command") {
		t.Errorf("error message should contain command: %s", errMsg)
	}

	if !strings.Contains(errMsg, "error: unknown command") {
		t.Errorf("error message should contain stderr: %s", errMsg)
	}

	if !strings.Contains(errMsg, "exit code: 1") {
		t.Errorf("error message should contain exit code: %s", errMsg)
	}
}
