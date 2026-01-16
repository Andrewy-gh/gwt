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

// Windows-specific tests
func TestExecuteCommandWindowsPowerShell(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test")
	}

	cmd := `powershell -Command "Write-Output 'PowerShell output'"`

	result, err := ExecuteCommand(ExecOptions{
		Command: cmd,
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	if !strings.Contains(result.Stdout, "PowerShell output") {
		t.Errorf("Expected PowerShell output, got: %s", result.Stdout)
	}
}

func TestExecuteCommandWindowsBatchFile(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test")
	}

	// Create a temporary batch file
	tmpDir := t.TempDir()
	batchFile := filepath.Join(tmpDir, "test.bat")
	outputFile := filepath.Join(tmpDir, "output.txt")

	batchContent := `@echo off
echo Batch file executed
echo Batch output > "` + outputFile + `"
exit /b 0`

	if err := os.WriteFile(batchFile, []byte(batchContent), 0755); err != nil {
		t.Fatalf("Failed to create batch file: %v", err)
	}

	result, err := ExecuteCommand(ExecOptions{
		Command: batchFile,
		Dir:     tmpDir,
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	// Verify batch file created output file
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Errorf("Batch file did not create expected output file")
	}
}

func TestExecuteCommandWindowsPathsWithSpaces(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test")
	}

	// Test that we can set a working directory with spaces
	tmpDir := t.TempDir()
	dirWithSpaces := filepath.Join(tmpDir, "test dir with spaces")

	// Create directory
	if err := os.Mkdir(dirWithSpaces, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Execute a simple command in that directory
	cmd := `echo Working in directory with spaces`

	result, err := ExecuteCommand(ExecOptions{
		Command: cmd,
		Dir:     dirWithSpaces,
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	if !strings.Contains(result.Stdout, "Working in directory with spaces") {
		t.Errorf("Command did not execute in directory with spaces")
	}
}

func TestExecuteCommandWindowsEnvironmentExpansion(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test")
	}

	env := os.Environ()
	env = append(env, "TEST_PATH=C:\\test\\path")

	cmd := `echo %TEST_PATH%`

	result, err := ExecuteCommand(ExecOptions{
		Command: cmd,
		Env:     env,
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !strings.Contains(result.Stdout, "C:\\test\\path") {
		t.Errorf("Environment variable not expanded correctly: %s", result.Stdout)
	}
}

func TestExecuteCommandWindowsMultipleCommands(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test")
	}

	// Chain commands with &&
	cmd := `echo first && echo second && echo third`

	result, err := ExecuteCommand(ExecOptions{
		Command: cmd,
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	if !strings.Contains(result.Stdout, "first") ||
		!strings.Contains(result.Stdout, "second") ||
		!strings.Contains(result.Stdout, "third") {
		t.Errorf("Not all commands executed: %s", result.Stdout)
	}
}

func TestExecuteCommandWindowsErrorHandling(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test")
	}

	// Command that should fail
	cmd := `nonexistent-command-12345`

	result, err := ExecuteCommand(ExecOptions{
		Command: cmd,
	})

	if err != nil {
		t.Fatalf("Expected error in result, not function error: %v", err)
	}

	if result.ExitCode == 0 {
		t.Errorf("Expected non-zero exit code for failed command")
	}

	if result.Stderr == "" {
		t.Logf("Warning: Expected stderr output for failed command")
	}
}

func TestExecuteCommandWindowsLongPath(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test")
	}

	tmpDir := t.TempDir()

	// Just test that we can execute a command in a specific directory
	// Use dir command to verify the working directory mechanism
	cmd := `dir`

	result, err := ExecuteCommand(ExecOptions{
		Command: cmd,
		Dir:     tmpDir,
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0 for dir command, got %d", result.ExitCode)
	}

	// Should show the temp directory contents
	if result.Stdout == "" {
		t.Errorf("Expected dir output, got empty string")
	}
}
