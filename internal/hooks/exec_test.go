package hooks

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestExecuteCommandSuccess(t *testing.T) {
	var cmd string
	if runtime.GOOS == "windows" {
		cmd = "echo test output"
	} else {
		cmd = "echo 'test output'"
	}

	result, err := ExecuteCommand(ExecOptions{
		Command: cmd,
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	if !strings.Contains(result.Stdout, "test output") {
		t.Errorf("Expected stdout to contain 'test output', got: %s", result.Stdout)
	}
}

func TestExecuteCommandFailure(t *testing.T) {
	var cmd string
	if runtime.GOOS == "windows" {
		cmd = "exit 42"
	} else {
		cmd = "exit 42"
	}

	result, err := ExecuteCommand(ExecOptions{
		Command: cmd,
	})

	if err != nil {
		t.Fatalf("Expected no error for non-zero exit (error is in result), got %v", err)
	}

	if result.ExitCode != 42 {
		t.Errorf("Expected exit code 42, got %d", result.ExitCode)
	}
}

func TestExecuteCommandTimeout(t *testing.T) {
	var cmd string
	if runtime.GOOS == "windows" {
		// Use ping to localhost with long delay as a reliable way to sleep on Windows
		cmd = "ping -n 11 127.0.0.1 > nul"
	} else {
		cmd = "sleep 10"
	}

	_, err := ExecuteCommand(ExecOptions{
		Command: cmd,
		Timeout: 100 * time.Millisecond,
	})

	if err != ErrHookTimeout {
		t.Errorf("Expected ErrHookTimeout, got %v", err)
	}
}

func TestExecuteCommandWorkingDirectory(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	var cmd string
	if runtime.GOOS == "windows" {
		cmd = "cd"
	} else {
		cmd = "pwd"
	}

	result, err := ExecuteCommand(ExecOptions{
		Command: cmd,
		Dir:     tmpDir,
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// The output should contain the temp directory path
	if !strings.Contains(result.Stdout, filepath.Base(tmpDir)) {
		t.Errorf("Expected working directory to be %s, got: %s", tmpDir, result.Stdout)
	}
}

func TestExecuteCommandEnvironmentVariables(t *testing.T) {
	testKey := "GWT_TEST_VAR"
	testValue := "test_value_123"

	env := os.Environ()
	env = append(env, testKey+"="+testValue)

	var cmd string
	if runtime.GOOS == "windows" {
		cmd = "echo %" + testKey + "%"
	} else {
		cmd = "echo $" + testKey
	}

	result, err := ExecuteCommand(ExecOptions{
		Command: cmd,
		Env:     env,
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !strings.Contains(result.Stdout, testValue) {
		t.Errorf("Expected stdout to contain %s, got: %s", testValue, result.Stdout)
	}
}

func TestExecuteCommandStderr(t *testing.T) {
	var cmd string
	if runtime.GOOS == "windows" {
		cmd = "echo error message 1>&2 && exit 1"
	} else {
		cmd = "echo 'error message' >&2 && exit 1"
	}

	result, err := ExecuteCommand(ExecOptions{
		Command: cmd,
	})

	if err != nil {
		t.Fatalf("Expected no error for non-zero exit, got %v", err)
	}

	if result.ExitCode != 1 {
		t.Errorf("Expected exit code 1, got %d", result.ExitCode)
	}

	if !strings.Contains(result.Stderr, "error message") {
		t.Errorf("Expected stderr to contain 'error message', got: %s", result.Stderr)
	}
}

func TestExecuteCommandDefaultTimeout(t *testing.T) {
	var cmd string
	if runtime.GOOS == "windows" {
		cmd = "echo test"
	} else {
		cmd = "echo test"
	}

	result, err := ExecuteCommand(ExecOptions{
		Command: cmd,
		// No timeout specified, should use default
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}
}
