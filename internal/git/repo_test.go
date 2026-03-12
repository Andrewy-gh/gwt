package git

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Andrewy-gh/gwt/internal/testutil"
)

func TestIsGitRepository(t *testing.T) {
	// Test valid repository
	repoPath := testutil.CreateTestRepo(t)
	if !IsGitRepository(repoPath) {
		t.Errorf("expected %s to be a git repository", repoPath)
	}

	// Test non-repository
	tmpDir, _ := os.MkdirTemp("", "not-a-repo-*")
	defer os.RemoveAll(tmpDir)

	if IsGitRepository(tmpDir) {
		t.Errorf("expected %s to not be a git repository", tmpDir)
	}
}

func TestIsWorktree(t *testing.T) {
	// Create repo with worktrees
	repoPath, worktreePath := testutil.CreateTestRepoWithWorktreesAndPath(t)

	// Main worktree should return false
	isWorktree, err := IsWorktree(repoPath)
	if err != nil {
		t.Fatalf("IsWorktree failed: %v", err)
	}
	if isWorktree {
		t.Errorf("main worktree should return false")
	}

	// Linked worktree should return true
	isWorktree, err = IsWorktree(worktreePath)
	if err != nil {
		t.Fatalf("IsWorktree failed: %v", err)
	}
	if !isWorktree {
		t.Errorf("linked worktree should return true")
	}
}

func TestGetRepoRoot(t *testing.T) {
	repoPath := testutil.CreateTestRepo(t)

	// Test from repo root
	root, err := GetRepoRoot(repoPath)
	if err != nil {
		t.Fatalf("GetRepoRoot failed: %v", err)
	}

	expectedRoot := mustNormalizeTestPath(t, repoPath)
	actualRoot := mustNormalizeTestPath(t, root)

	if actualRoot != expectedRoot {
		t.Errorf("expected root %s, got %s", expectedRoot, actualRoot)
	}

	// Test from subdirectory
	subDir := filepath.Join(repoPath, "subdir")
	os.Mkdir(subDir, 0755)

	root, err = GetRepoRoot(subDir)
	if err != nil {
		t.Fatalf("GetRepoRoot from subdir failed: %v", err)
	}

	actualRoot = mustNormalizeTestPath(t, root)
	if actualRoot != expectedRoot {
		t.Errorf("expected root %s from subdir, got %s", expectedRoot, actualRoot)
	}
}

func TestGetGitDir(t *testing.T) {
	repoPath := testutil.CreateTestRepo(t)

	gitDir, err := GetGitDir(repoPath)
	if err != nil {
		t.Fatalf("GetGitDir failed: %v", err)
	}

	// Should end with .git
	if filepath.Base(gitDir) != ".git" {
		t.Errorf("expected git dir to end with .git, got: %s", gitDir)
	}

	// Should exist
	if _, err := os.Stat(gitDir); err != nil {
		t.Errorf("git dir should exist: %v", err)
	}
}

func TestGetCurrentBranch(t *testing.T) {
	repoPath := testutil.CreateTestRepo(t)

	branch, err := GetCurrentBranch(repoPath)
	if err != nil {
		t.Fatalf("GetCurrentBranch failed: %v", err)
	}

	// Should be on main or master branch
	if branch != "main" && branch != "master" {
		t.Errorf("expected main or master branch, got: %s", branch)
	}
}

func TestGetMainWorktreePath(t *testing.T) {
	repoPath := testutil.CreateTestRepo(t)

	mainPath, err := GetMainWorktreePath(repoPath)
	if err != nil {
		t.Fatalf("GetMainWorktreePath failed: %v", err)
	}

	expectedPath := mustNormalizeTestPath(t, repoPath)
	actualPath := mustNormalizeTestPath(t, mainPath)

	if actualPath != expectedPath {
		t.Errorf("expected main worktree %s, got %s", expectedPath, actualPath)
	}
}

func mustNormalizeTestPath(t *testing.T, path string) string {
	t.Helper()

	normalized, err := normalizeWorktreePath(path)
	if err != nil {
		t.Fatalf("failed to normalize path %q: %v", path, err)
	}

	return filepath.Clean(normalized)
}

func TestIsInsideWorktree(t *testing.T) {
	repoPath := testutil.CreateTestRepo(t)

	inside, err := IsInsideWorktree(repoPath)
	if err != nil {
		t.Fatalf("IsInsideWorktree failed: %v", err)
	}

	if !inside {
		t.Errorf("expected to be inside worktree")
	}

	// Test non-worktree directory
	tmpDir, _ := os.MkdirTemp("", "not-a-worktree-*")
	defer os.RemoveAll(tmpDir)

	inside, err = IsInsideWorktree(tmpDir)
	if err != nil {
		// Error is okay for non-repo
		return
	}

	if inside {
		t.Errorf("expected to not be inside worktree")
	}
}

func TestValidateRepository(t *testing.T) {
	repoPath := testutil.CreateTestRepo(t)

	// Valid repository should return no error
	if err := ValidateRepository(repoPath); err != nil {
		t.Errorf("ValidateRepository failed for valid repo: %v", err)
	}

	// Non-repository should return NotARepoError
	tmpDir, _ := os.MkdirTemp("", "not-a-repo-*")
	defer os.RemoveAll(tmpDir)

	err := ValidateRepository(tmpDir)
	if err == nil {
		t.Errorf("expected error for non-repository")
	}

	if _, ok := err.(*NotARepoError); !ok {
		t.Errorf("expected NotARepoError, got: %T", err)
	}
}
