package git

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Andrewy-gh/gwt/internal/testutil"
)

func TestListWorktrees(t *testing.T) {
	repoPath := testutil.CreateTestRepoWithWorktrees(t)

	worktrees, err := ListWorktrees(repoPath)
	if err != nil {
		t.Fatalf("ListWorktrees failed: %v", err)
	}

	// Should have 2 worktrees (main + feature-1)
	if len(worktrees) != 2 {
		t.Errorf("expected 2 worktrees, got %d", len(worktrees))
	}

	// First worktree should be main
	if !worktrees[0].IsMain {
		t.Errorf("first worktree should be main")
	}

	// Second worktree should not be main
	if worktrees[1].IsMain {
		t.Errorf("second worktree should not be main")
	}

	// Check branch names
	if worktrees[0].Branch != "main" && worktrees[0].Branch != "master" {
		t.Errorf("expected main/master branch, got: %s", worktrees[0].Branch)
	}

	if worktrees[1].Branch != "feature-1" {
		t.Errorf("expected feature-1 branch, got: %s", worktrees[1].Branch)
	}
}

func TestGetWorktree(t *testing.T) {
	repoPath, worktreePath := testutil.CreateTestRepoWithWorktreesAndPath(t)

	wt, err := GetWorktree(repoPath)
	if err != nil {
		t.Fatalf("GetWorktree failed: %v", err)
	}

	if !wt.IsMain {
		t.Errorf("expected main worktree")
	}

	// Test linked worktree
	wt, err = GetWorktree(worktreePath)
	if err != nil {
		t.Fatalf("GetWorktree failed for linked worktree: %v", err)
	}

	if wt.IsMain {
		t.Errorf("expected linked worktree, not main")
	}

	if wt.Branch != "feature-1" {
		t.Errorf("expected feature-1 branch, got: %s", wt.Branch)
	}
}

func TestFindWorktreeByBranch(t *testing.T) {
	repoPath := testutil.CreateTestRepoWithWorktrees(t)

	// Find feature-1 worktree
	wt, err := FindWorktreeByBranch(repoPath, "feature-1")
	if err != nil {
		t.Fatalf("FindWorktreeByBranch failed: %v", err)
	}

	if wt == nil {
		t.Fatalf("expected to find feature-1 worktree")
	}

	if wt.Branch != "feature-1" {
		t.Errorf("expected feature-1 branch, got: %s", wt.Branch)
	}

	// Try non-existent branch
	wt, err = FindWorktreeByBranch(repoPath, "non-existent")
	if err != nil {
		t.Fatalf("FindWorktreeByBranch failed: %v", err)
	}

	if wt != nil {
		t.Errorf("expected nil for non-existent branch")
	}
}

func TestAddWorktree(t *testing.T) {
	repoPath := testutil.CreateTestRepo(t)

	// Create a new worktree with new branch using unique temp directory
	worktreePath, err := os.MkdirTemp("", "gwt-add-worktree-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(worktreePath)

	// Remove the directory so git worktree add can create it
	os.RemoveAll(worktreePath)

	wt, err := AddWorktree(repoPath, AddWorktreeOptions{
		Path:      worktreePath,
		Branch:    "test-branch",
		NewBranch: true,
	})

	if err != nil {
		t.Fatalf("AddWorktree failed: %v", err)
	}

	if wt.Branch != "test-branch" {
		t.Errorf("expected test-branch, got: %s", wt.Branch)
	}

	// Verify worktree exists
	if _, err := os.Stat(worktreePath); err != nil {
		t.Errorf("worktree directory should exist: %v", err)
	}

	// Verify it appears in list
	worktrees, err := ListWorktrees(repoPath)
	if err != nil {
		t.Fatalf("ListWorktrees failed: %v", err)
	}

	found := false
	for _, wt := range worktrees {
		if wt.Branch == "test-branch" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("test-branch worktree not found in list")
	}
}

func TestAddWorktreeForNewBranch(t *testing.T) {
	repoPath := testutil.CreateTestRepo(t)

	worktreePath, err := os.MkdirTemp("", "gwt-new-branch-worktree-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(worktreePath)

	// Remove the directory so git worktree add can create it
	os.RemoveAll(worktreePath)

	wt, err := AddWorktreeForNewBranch(repoPath, worktreePath, "new-branch", "")
	if err != nil {
		t.Fatalf("AddWorktreeForNewBranch failed: %v", err)
	}

	if wt.Branch != "new-branch" {
		t.Errorf("expected new-branch, got: %s", wt.Branch)
	}
}

func TestRemoveWorktree(t *testing.T) {
	repoPath, worktreePath := testutil.CreateTestRepoWithWorktreesAndPath(t)

	// Remove the worktree
	err := RemoveWorktree(repoPath, RemoveWorktreeOptions{
		Path: worktreePath,
	})

	if err != nil {
		t.Fatalf("RemoveWorktree failed: %v", err)
	}

	// Verify it's gone from list
	worktrees, err := ListWorktrees(repoPath)
	if err != nil {
		t.Fatalf("ListWorktrees failed: %v", err)
	}

	for _, wt := range worktrees {
		if wt.Branch == "feature-1" {
			t.Errorf("feature-1 worktree should be removed")
		}
	}
}

func TestRemoveWorktree_CannotRemoveMain(t *testing.T) {
	repoPath := testutil.CreateTestRepo(t)

	// Try to remove main worktree
	err := RemoveWorktree(repoPath, RemoveWorktreeOptions{
		Path: repoPath,
	})

	if err == nil {
		t.Errorf("expected error when removing main worktree")
	}

	if _, ok := err.(*WorktreeError); !ok {
		t.Errorf("expected WorktreeError, got: %T", err)
	}
}

func TestParseWorktreeList(t *testing.T) {
	input := `worktree /path/to/main
HEAD abc123def456
branch refs/heads/main

worktree /path/to/feature
HEAD def456abc789
branch refs/heads/feature-branch
locked

worktree /path/to/detached
HEAD 123456789abc
detached`

	worktrees, err := parseWorktreeList(input)
	if err != nil {
		t.Fatalf("parseWorktreeList failed: %v", err)
	}

	if len(worktrees) != 3 {
		t.Fatalf("expected 3 worktrees, got %d", len(worktrees))
	}

	// Check first worktree
	if !worktrees[0].IsMain {
		t.Errorf("first worktree should be main")
	}
	if worktrees[0].Branch != "main" {
		t.Errorf("expected main branch, got: %s", worktrees[0].Branch)
	}

	// Check second worktree
	if worktrees[1].Branch != "feature-branch" {
		t.Errorf("expected feature-branch, got: %s", worktrees[1].Branch)
	}
	if !worktrees[1].Locked {
		t.Errorf("second worktree should be locked")
	}

	// Check third worktree
	if !worktrees[2].IsDetached {
		t.Errorf("third worktree should be detached")
	}
}

func TestNormalizeWorktreePath_ResolvesSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink normalization is covered by Windows-specific path handling")
	}

	rootDir := t.TempDir()
	targetDir := filepath.Join(rootDir, "target")
	linkDir := filepath.Join(rootDir, "link")

	if err := os.Mkdir(targetDir, 0755); err != nil {
		t.Fatalf("failed to create target dir: %v", err)
	}

	if err := os.Symlink(targetDir, linkDir); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}

	targetPath, err := normalizeWorktreePath(targetDir)
	if err != nil {
		t.Fatalf("normalizeWorktreePath failed for target dir: %v", err)
	}

	linkPath, err := normalizeWorktreePath(linkDir)
	if err != nil {
		t.Fatalf("normalizeWorktreePath failed for symlink dir: %v", err)
	}

	if targetPath != linkPath {
		t.Fatalf("expected symlink and target to normalize to the same path, got %q and %q", targetPath, linkPath)
	}
}
