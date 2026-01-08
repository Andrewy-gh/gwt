package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// CreateTestRepo creates a temporary git repository for testing
// Returns the path to the repository
func CreateTestRepo(t *testing.T) string {
	t.Helper()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "gwt-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Cleanup on test end
	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	// Initialize git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Configure git user for commits
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to configure git user.name: %v", err)
	}

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to configure git user.email: %v", err)
	}

	// Create initial commit
	readmePath := filepath.Join(tmpDir, "README.md")
	if err := os.WriteFile(readmePath, []byte("# Test Repo\n"), 0644); err != nil {
		t.Fatalf("failed to create README: %v", err)
	}

	cmd = exec.Command("git", "add", "README.md")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to add README: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create initial commit: %v", err)
	}

	// Rename default branch to main (if not already)
	cmd = exec.Command("git", "branch", "-M", "main")
	cmd.Dir = tmpDir
	_ = cmd.Run() // Ignore error, might already be main

	return tmpDir
}

// CreateBareTestRepo creates a bare temporary git repository for testing
func CreateBareTestRepo(t *testing.T) string {
	t.Helper()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "gwt-bare-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Cleanup on test end
	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	// Initialize bare git repository
	cmd := exec.Command("git", "init", "--bare")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init bare git repo: %v", err)
	}

	return tmpDir
}

// CreateTestRepoWithWorktrees creates a repository with multiple worktrees
// Returns the main repo path
func CreateTestRepoWithWorktrees(t *testing.T) string {
	mainPath, _ := CreateTestRepoWithWorktreesAndPath(t)
	return mainPath
}

// CreateTestRepoWithWorktreesAndPath creates a repository with multiple worktrees
// Returns (main repo path, worktree path)
func CreateTestRepoWithWorktreesAndPath(t *testing.T) (string, string) {
	t.Helper()

	// Create main repo
	repoPath := CreateTestRepo(t)

	// Create a second branch
	cmd := exec.Command("git", "branch", "feature-1")
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create feature-1 branch: %v", err)
	}

	// Create worktree for feature-1
	// Create temp directory for the worktree
	worktreePath, err := os.MkdirTemp("", "gwt-worktree-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir for worktree: %v", err)
	}

	// Cleanup on test end
	t.Cleanup(func() {
		os.RemoveAll(worktreePath)
	})

	// Remove the directory so git worktree add can create it
	os.RemoveAll(worktreePath)

	cmd = exec.Command("git", "worktree", "add", worktreePath, "feature-1")
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to create worktree: %v\nOutput: %s", err, string(output))
	}

	return repoPath, worktreePath
}

// CreateTestRepoWithRemote creates two repositories with one as a remote of the other
// Returns (local repo path, remote repo path)
func CreateTestRepoWithRemote(t *testing.T) (string, string) {
	t.Helper()

	// Create bare remote repository
	remotePath := CreateBareTestRepo(t)

	// Create local repository
	localPath, err := os.MkdirTemp("", "gwt-local-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	t.Cleanup(func() {
		os.RemoveAll(localPath)
	})

	// Clone from the bare repo
	cmd := exec.Command("git", "clone", remotePath, localPath)
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to clone repo: %v", err)
	}

	// Configure git user for commits
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = localPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to configure git user.name: %v", err)
	}

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = localPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to configure git user.email: %v", err)
	}

	// Create initial commit
	readmePath := filepath.Join(localPath, "README.md")
	if err := os.WriteFile(readmePath, []byte("# Test Repo\n"), 0644); err != nil {
		t.Fatalf("failed to create README: %v", err)
	}

	cmd = exec.Command("git", "add", "README.md")
	cmd.Dir = localPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to add README: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = localPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create initial commit: %v", err)
	}

	// Push to remote
	cmd = exec.Command("git", "push", "-u", "origin", "main")
	cmd.Dir = localPath
	if err := cmd.Run(); err != nil {
		// Try master if main doesn't work
		cmd = exec.Command("git", "push", "-u", "origin", "master")
		cmd.Dir = localPath
		_ = cmd.Run()
	}

	return localPath, remotePath
}

// CommitFile creates a file and commits it in the given repository
func CommitFile(t *testing.T, repoPath, filename, content, message string) {
	t.Helper()

	// Write file
	filePath := filepath.Join(repoPath, filename)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Add file
	cmd := exec.Command("git", "add", filename)
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to add file: %v", err)
	}

	// Commit
	cmd = exec.Command("git", "commit", "-m", message)
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}
}
