//go:build windows

package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Andrewy-gh/gwt/internal/create"
	"github.com/Andrewy-gh/gwt/internal/docker"
	"github.com/Andrewy-gh/gwt/internal/git"
	"github.com/Andrewy-gh/gwt/internal/hooks"
	"github.com/Andrewy-gh/gwt/internal/testutil"
)

// TestSymlinkCreation tests basic symlink creation on Windows
func TestSymlinkCreation(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test")
	}

	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "target")
	link := filepath.Join(tmpDir, "link")

	// Create target directory
	if err := os.Mkdir(target, 0755); err != nil {
		t.Fatalf("failed to create target: %v", err)
	}

	// Try to create symlink
	err := docker.CreateSymlink(target, link)

	// Either it succeeds or falls back to junction/copy
	if err != nil {
		// If it failed, check that fallback was attempted
		if !os.IsPermission(err) && !strings.Contains(err.Error(), "privilege") {
			t.Fatalf("unexpected error: %v", err)
		}
		t.Logf("Symlink creation failed (expected without Developer Mode): %v", err)
	} else {
		// Verify link exists
		if _, err := os.Lstat(link); err != nil {
			t.Fatalf("symlink not created: %v", err)
		}
		t.Logf("Symlink created successfully")
	}
}

// TestJunctionCreation tests directory junction creation (Windows mklink /J)
func TestJunctionCreation(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test")
	}

	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "target")
	junction := filepath.Join(tmpDir, "junction")

	// Create target directory
	if err := os.Mkdir(target, 0755); err != nil {
		t.Fatalf("failed to create target: %v", err)
	}

	// Create junction using mklink /J
	cmd := exec.Command("cmd", "/C", "mklink", "/J", junction, target)
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create junction: %v", err)
	}

	// Verify junction exists and is readable
	if _, err := os.Stat(junction); err != nil {
		t.Fatalf("junction not accessible: %v", err)
	}

	// Verify it points to target
	if link, err := os.Readlink(junction); err == nil {
		t.Logf("Junction points to: %s", link)
	}

	// Test writing through junction
	testFile := filepath.Join(junction, "test.txt")
	if err := os.WriteFile(testFile, []byte("hello"), 0644); err != nil {
		t.Fatalf("failed to write through junction: %v", err)
	}

	// Verify file exists in target
	targetFile := filepath.Join(target, "test.txt")
	if data, err := os.ReadFile(targetFile); err != nil {
		t.Fatalf("file not in target: %v", err)
	} else if string(data) != "hello" {
		t.Fatalf("incorrect data: got %q, want %q", string(data), "hello")
	}
}

// TestSymlinkFallbackToJunction tests the fallback chain
func TestSymlinkFallbackToJunction(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test")
	}

	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "target")
	link := filepath.Join(tmpDir, "link")

	// Create target directory
	if err := os.Mkdir(target, 0755); err != nil {
		t.Fatalf("failed to create target: %v", err)
	}

	// CreateSymlink should try symlink, then junction, then copy
	err := docker.CreateSymlink(target, link)
	if err != nil {
		t.Fatalf("all fallback methods failed: %v", err)
	}

	// Verify link exists (by any method)
	if _, err := os.Stat(link); err != nil {
		t.Fatalf("link not created: %v", err)
	}

	t.Logf("Symlink/junction/copy created successfully")
}

// TestSymlinkPermissionDetection tests detection of symlink privileges
func TestSymlinkPermissionDetection(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test")
	}

	// This is tested by gwt doctor command
	// We just verify the check doesn't panic
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "target")
	link := filepath.Join(tmpDir, "link")

	if err := os.Mkdir(target, 0755); err != nil {
		t.Fatalf("failed to create target: %v", err)
	}

	// Try creating a symlink to test permissions
	err := os.Symlink(target, link)
	if err != nil {
		if os.IsPermission(err) || strings.Contains(err.Error(), "privilege") {
			t.Logf("No symlink privilege (expected): %v", err)
		} else {
			t.Errorf("unexpected error: %v", err)
		}
	} else {
		t.Logf("Symlink privilege available")
		os.Remove(link)
	}
}

// TestWindowsAbsolutePaths tests Windows absolute path handling
func TestWindowsAbsolutePaths(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test")
	}

	tests := []struct {
		name     string
		path     string
		wantAbs  bool
	}{
		{"drive letter", `C:\projects\app`, true},
		{"forward slashes", `C:/projects/app`, true},
		{"UNC path", `\\server\share\app`, true},
		{"relative dot", `.\app`, false},
		{"relative dotdot", `..\app`, false},
		{"relative plain", `app`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filepath.IsAbs(tt.path)
			if got != tt.wantAbs {
				t.Errorf("IsAbs(%q) = %v, want %v", tt.path, got, tt.wantAbs)
			}
		})
	}
}

// TestDriveLetterPaths tests drive letter path handling
func TestDriveLetterPaths(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test")
	}

	// Test various drive letter formats
	paths := []string{
		`C:\projects`,
		`C:/projects`,
		`c:\projects`,
		`c:/projects`,
	}

	for _, p := range paths {
		abs, err := filepath.Abs(p)
		if err != nil {
			t.Errorf("Abs(%q) failed: %v", p, err)
			continue
		}
		if !filepath.IsAbs(abs) {
			t.Errorf("Abs(%q) = %q is not absolute", p, abs)
		}
		t.Logf("Abs(%q) = %q", p, abs)
	}
}

// TestUNCPaths tests UNC path handling
func TestUNCPaths(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test")
	}

	tests := []struct {
		path  string
		isUNC bool
	}{
		{`\\server\share\dir`, true},
		{`//server/share/dir`, true},
		{`C:\local\dir`, false},
		{`relative\dir`, false},
	}

	for _, tt := range tests {
		isUNC := strings.HasPrefix(filepath.ToSlash(tt.path), "//")
		if isUNC != tt.isUNC {
			t.Errorf("path %q: got UNC=%v, want %v", tt.path, isUNC, tt.isUNC)
		}
	}
}

// TestLongPaths tests >260 character path support
func TestLongPaths(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test")
	}

	tmpDir := t.TempDir()

	// Create a very long path (>260 chars)
	// Windows has a 260 char limit unless long paths are enabled
	longName := strings.Repeat("a", 100)
	longPath := filepath.Join(tmpDir, longName, longName, longName)

	// Try to create the directory structure
	err := os.MkdirAll(longPath, 0755)
	if err != nil {
		if strings.Contains(err.Error(), "file name too long") ||
		   strings.Contains(err.Error(), "cannot find the path") {
			t.Logf("Long paths not enabled (expected): %v", err)
			t.Skip("Long paths not enabled on this system")
		}
		t.Fatalf("unexpected error creating long path: %v", err)
	}

	// If we got here, long paths are enabled
	t.Logf("Long path support enabled, path length: %d", len(longPath))

	// Test file operations
	testFile := filepath.Join(longPath, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to write to long path: %v", err)
	}

	if _, err := os.ReadFile(testFile); err != nil {
		t.Fatalf("failed to read from long path: %v", err)
	}
}

// TestBackslashNormalization tests path separator handling
func TestBackslashNormalization(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test")
	}

	tests := []struct {
		input    string
		expected string
	}{
		{`C:\projects\app`, `C:/projects/app`},
		{`C:\projects/mixed\slashes`, `C:/projects/mixed/slashes`},
		{`relative\path`, `relative/path`},
	}

	for _, tt := range tests {
		got := filepath.ToSlash(tt.input)
		if got != tt.expected {
			t.Errorf("ToSlash(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

// TestWindowsReservedNames tests detection of Windows reserved device names
func TestWindowsReservedNames(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test")
	}

	// Reserved device names that Windows prohibits
	reserved := []string{
		"CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9",
	}

	for _, name := range reserved {
		t.Run(name, func(t *testing.T) {
			// create.ValidateBranchName should reject these
			err := create.ValidateBranchName(name)
			if err == nil {
				t.Errorf("ValidateBranchName(%q) should fail on Windows", name)
			} else {
				t.Logf("ValidateBranchName(%q) correctly rejected: %v", name, err)
			}
		})
	}
}

// TestReservedNameVariants tests reserved names with extensions
func TestReservedNameVariants(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test")
	}

	// These are also invalid on Windows
	variants := []string{
		"CON.txt",
		"con.log",
		"COM1.dat",
		"feature/AUX",
		"fix/prn",
	}

	for _, name := range variants {
		t.Run(name, func(t *testing.T) {
			err := create.ValidateBranchName(name)
			if err == nil {
				t.Errorf("ValidateBranchName(%q) should fail on Windows", name)
			} else {
				t.Logf("ValidateBranchName(%q) correctly rejected: %v", name, err)
			}
		})
	}
}

// TestHookExecutionWithCmdExe tests hook execution using cmd.exe
func TestHookExecutionWithCmdExe(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test")
	}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Create a simple hook that writes to a file
	hook := `echo hello > "` + testFile + `"`

	// Execute the hook
	executor := hooks.NewExecutor(tmpDir)
	err := executor.Execute([]string{hook}, map[string]string{
		"TEST_VAR": "test_value",
	})

	if err != nil {
		t.Fatalf("hook execution failed: %v", err)
	}

	// Verify output file was created
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Fatalf("hook did not create expected file")
	}

	// Read content
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	// cmd.exe echo adds a space and newline
	expected := "hello"
	if !strings.Contains(string(data), expected) {
		t.Errorf("got %q, want to contain %q", string(data), expected)
	}
}

// TestHookWithBatchFile tests executing batch files
func TestHookWithBatchFile(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test")
	}

	tmpDir := t.TempDir()
	batchFile := filepath.Join(tmpDir, "test.bat")
	outputFile := filepath.Join(tmpDir, "output.txt")

	// Create a batch file
	batchContent := `@echo off
echo Batch file executed > "` + outputFile + `"`
	if err := os.WriteFile(batchFile, []byte(batchContent), 0755); err != nil {
		t.Fatalf("failed to create batch file: %v", err)
	}

	// Execute the batch file as a hook
	executor := hooks.NewExecutor(tmpDir)
	err := executor.Execute([]string{batchFile}, nil)
	if err != nil {
		t.Fatalf("batch file execution failed: %v", err)
	}

	// Verify output
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Fatalf("batch file did not create output")
	}
}

// TestHookWithPowerShell tests executing PowerShell commands
func TestHookWithPowerShell(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test")
	}

	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.txt")

	// PowerShell command to write to file
	hook := `powershell -Command "Set-Content -Path '` + outputFile + `' -Value 'PowerShell executed'"`

	executor := hooks.NewExecutor(tmpDir)
	err := executor.Execute([]string{hook}, nil)
	if err != nil {
		t.Fatalf("PowerShell hook execution failed: %v", err)
	}

	// Verify output
	if data, err := os.ReadFile(outputFile); err != nil {
		t.Fatalf("PowerShell did not create output: %v", err)
	} else if !strings.Contains(string(data), "PowerShell executed") {
		t.Errorf("unexpected output: %s", string(data))
	}
}

// TestHookEnvironmentVariables tests environment variable passing
func TestHookEnvironmentVariables(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test")
	}

	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "env.txt")

	// Hook that writes an environment variable to file
	hook := `echo %TEST_VAR% > "` + outputFile + `"`

	env := map[string]string{
		"TEST_VAR": "test_value_123",
	}

	executor := hooks.NewExecutor(tmpDir)
	err := executor.Execute([]string{hook}, env)
	if err != nil {
		t.Fatalf("hook execution failed: %v", err)
	}

	// Verify environment variable was passed
	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}

	if !strings.Contains(string(data), "test_value_123") {
		t.Errorf("environment variable not passed: got %q", string(data))
	}
}

// TestProcessLockCreation tests process lock file creation
func TestProcessLockCreation(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test")
	}

	tmpDir := t.TempDir()
	lockFile := filepath.Join(tmpDir, ".gwt.lock")

	// Create a lock
	lock, err := create.AcquireLock(tmpDir)
	if err != nil {
		t.Fatalf("failed to acquire lock: %v", err)
	}
	defer lock.Release()

	// Verify lock file exists
	if _, err := os.Stat(lockFile); os.IsNotExist(err) {
		t.Fatalf("lock file not created")
	}

	t.Logf("Lock acquired successfully")
}

// TestProcessLockConflict tests lock conflict detection
func TestProcessLockConflict(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test")
	}

	tmpDir := t.TempDir()

	// Acquire first lock
	lock1, err := create.AcquireLock(tmpDir)
	if err != nil {
		t.Fatalf("failed to acquire first lock: %v", err)
	}
	defer lock1.Release()

	// Try to acquire second lock (should fail)
	lock2, err := create.AcquireLock(tmpDir)
	if err == nil {
		lock2.Release()
		t.Fatalf("second lock should have failed")
	}

	t.Logf("Lock conflict correctly detected: %v", err)
}

// TestProcessLockCleanup tests lock cleanup
func TestProcessLockCleanup(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test")
	}

	tmpDir := t.TempDir()
	lockFile := filepath.Join(tmpDir, ".gwt.lock")

	// Acquire and release lock
	lock, err := create.AcquireLock(tmpDir)
	if err != nil {
		t.Fatalf("failed to acquire lock: %v", err)
	}

	// Release should remove the file
	if err := lock.Release(); err != nil {
		t.Fatalf("failed to release lock: %v", err)
	}

	// Verify lock file is removed
	if _, err := os.Stat(lockFile); !os.IsNotExist(err) {
		t.Errorf("lock file not cleaned up")
	}
}

// TestFileCopyWithLongPaths tests copying files with long paths
func TestFileCopyWithLongPaths(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test")
	}

	// This test requires long path support
	tmpDir := t.TempDir()

	// Create a moderately long path
	longName := strings.Repeat("subdir", 20) // 120 chars
	srcDir := filepath.Join(tmpDir, "src", longName)
	dstDir := filepath.Join(tmpDir, "dst", longName)

	// Try to create directories
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Skipf("Cannot create long path (long paths not enabled): %v", err)
	}

	// Create a test file
	srcFile := filepath.Join(srcDir, "test.txt")
	if err := os.WriteFile(srcFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Copy using testutil (if available) or manual copy
	if err := testutil.CopyDir(filepath.Dir(srcDir), filepath.Dir(dstDir)); err != nil {
		t.Fatalf("failed to copy directory: %v", err)
	}

	// Verify copy
	dstFile := filepath.Join(dstDir, "test.txt")
	if data, err := os.ReadFile(dstFile); err != nil {
		t.Fatalf("failed to read copied file: %v", err)
	} else if string(data) != "test content" {
		t.Errorf("copied file content incorrect")
	}
}

// TestCaseSensitivityHandling tests Windows case-insensitive behavior
func TestCaseSensitivityHandling(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test")
	}

	tmpDir := t.TempDir()

	// Create file with lowercase name
	file1 := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(file1, []byte("content1"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Access with different case
	file2 := filepath.Join(tmpDir, "TEST.TXT")
	data, err := os.ReadFile(file2)
	if err != nil {
		t.Fatalf("failed to read file with different case: %v", err)
	}

	if string(data) != "content1" {
		t.Errorf("case-insensitive access failed: got %q", string(data))
	}

	// Overwrite with different case (should be same file)
	if err := os.WriteFile(file2, []byte("content2"), 0644); err != nil {
		t.Fatalf("failed to overwrite: %v", err)
	}

	// Read with original case
	data, err = os.ReadFile(file1)
	if err != nil {
		t.Fatalf("failed to read original file: %v", err)
	}

	if string(data) != "content2" {
		t.Errorf("case-insensitive overwrite failed")
	}
}

// TestGitWorktreeOnWindows tests basic git worktree operations
func TestGitWorktreeOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test")
	}

	// Setup test repository
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "repo")

	// Initialize repository
	if err := testutil.InitRepo(repoPath); err != nil {
		t.Fatalf("failed to init repo: %v", err)
	}

	// Create initial commit
	testFile := filepath.Join(repoPath, "README.md")
	if err := os.WriteFile(testFile, []byte("# Test"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	if err := testutil.GitCommit(repoPath, "Initial commit"); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	// Add a worktree
	wtPath := filepath.Join(tmpDir, "worktree", "feature-x")
	if err := git.AddWorktree(repoPath, wtPath, git.AddWorktreeOptions{
		Branch:       "feature-x",
		CreateBranch: true,
	}); err != nil {
		t.Fatalf("failed to add worktree: %v", err)
	}

	// Verify worktree exists
	if _, err := os.Stat(wtPath); os.IsNotExist(err) {
		t.Fatalf("worktree directory not created")
	}

	// List worktrees
	worktrees, err := git.ListWorktrees(repoPath)
	if err != nil {
		t.Fatalf("failed to list worktrees: %v", err)
	}

	if len(worktrees) != 2 { // main + feature-x
		t.Errorf("expected 2 worktrees, got %d", len(worktrees))
	}

	// Remove worktree
	if err := git.RemoveWorktree(repoPath, wtPath, git.RemoveWorktreeOptions{}); err != nil {
		t.Fatalf("failed to remove worktree: %v", err)
	}

	// Verify removed
	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Errorf("worktree directory not removed")
	}
}
