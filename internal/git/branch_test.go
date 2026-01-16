package git

import (
	"testing"
	"time"

	"github.com/Andrewy-gh/gwt/internal/testutil"
)

func TestListLocalBranches(t *testing.T) {
	repoPath := testutil.CreateTestRepo(t)

	branches, err := ListLocalBranches(repoPath)
	if err != nil {
		t.Fatalf("ListLocalBranches failed: %v", err)
	}

	// Should have at least main/master branch
	if len(branches) == 0 {
		t.Errorf("expected at least 1 branch")
	}

	// Check that we have the main branch
	found := false
	for _, b := range branches {
		if b.Name == "main" || b.Name == "master" {
			found = true
			if !b.IsHead {
				t.Errorf("main/master branch should be HEAD")
			}
		}
	}

	if !found {
		t.Errorf("expected to find main or master branch")
	}
}

func TestCreateBranch(t *testing.T) {
	repoPath := testutil.CreateTestRepo(t)

	// Create a new branch
	branch, err := CreateBranch(repoPath, CreateBranchOptions{
		Name: "test-branch",
	})

	if err != nil {
		t.Fatalf("CreateBranch failed: %v", err)
	}

	if branch.Name != "test-branch" {
		t.Errorf("expected test-branch, got: %s", branch.Name)
	}

	// Verify branch exists
	exists, err := LocalBranchExists(repoPath, "test-branch")
	if err != nil {
		t.Fatalf("LocalBranchExists failed: %v", err)
	}

	if !exists {
		t.Errorf("test-branch should exist")
	}

	// Verify it appears in list
	branches, err := ListLocalBranches(repoPath)
	if err != nil {
		t.Fatalf("ListLocalBranches failed: %v", err)
	}

	found := false
	for _, b := range branches {
		if b.Name == "test-branch" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("test-branch not found in branch list")
	}
}

func TestDeleteBranch(t *testing.T) {
	repoPath := testutil.CreateTestRepo(t)

	// Create a branch
	_, err := CreateBranch(repoPath, CreateBranchOptions{
		Name: "to-delete",
	})
	if err != nil {
		t.Fatalf("CreateBranch failed: %v", err)
	}

	// Delete the branch
	err = DeleteBranch(repoPath, DeleteBranchOptions{
		Name: "to-delete",
	})

	if err != nil {
		t.Fatalf("DeleteBranch failed: %v", err)
	}

	// Verify it's gone
	exists, err := LocalBranchExists(repoPath, "to-delete")
	if err != nil {
		t.Fatalf("LocalBranchExists failed: %v", err)
	}

	if exists {
		t.Errorf("to-delete branch should not exist after deletion")
	}
}

func TestDeleteBranch_CannotDeleteCurrentBranch(t *testing.T) {
	repoPath := testutil.CreateTestRepo(t)

	// Get current branch
	currentBranch, err := GetCurrentBranch(repoPath)
	if err != nil {
		t.Fatalf("GetCurrentBranch failed: %v", err)
	}

	// Try to delete current branch
	err = DeleteBranch(repoPath, DeleteBranchOptions{
		Name: currentBranch,
	})

	if err == nil {
		t.Errorf("expected error when deleting current branch")
	}
}

func TestRenameBranch(t *testing.T) {
	repoPath := testutil.CreateTestRepo(t)

	// Create a branch
	_, err := CreateBranch(repoPath, CreateBranchOptions{
		Name: "old-name",
	})
	if err != nil {
		t.Fatalf("CreateBranch failed: %v", err)
	}

	// Rename it
	err = RenameBranch(repoPath, "old-name", "new-name", false)
	if err != nil {
		t.Fatalf("RenameBranch failed: %v", err)
	}

	// Verify old name doesn't exist
	exists, err := LocalBranchExists(repoPath, "old-name")
	if err != nil {
		t.Fatalf("LocalBranchExists failed: %v", err)
	}

	if exists {
		t.Errorf("old-name should not exist after rename")
	}

	// Verify new name exists
	exists, err = LocalBranchExists(repoPath, "new-name")
	if err != nil {
		t.Fatalf("LocalBranchExists failed: %v", err)
	}

	if !exists {
		t.Errorf("new-name should exist after rename")
	}
}

func TestValidateBranchName(t *testing.T) {
	tests := []struct {
		name      string
		branchName string
		wantError bool
	}{
		{"valid name", "feature-branch", false},
		{"valid with slashes", "feature/branch", false},
		{"empty name", "", true},
		{"with spaces", "feature branch", true},
		{"starts with dash", "-feature", true},
		{"contains double dot", "feature..branch", true},
		{"ends with .lock", "feature.lock", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBranchName(tt.branchName)
			if tt.wantError && err == nil {
				t.Errorf("expected error for branch name: %s", tt.branchName)
			}
			if !tt.wantError && err != nil {
				t.Errorf("unexpected error for branch name %s: %v", tt.branchName, err)
			}
		})
	}
}

func TestLocalBranchExists(t *testing.T) {
	repoPath := testutil.CreateTestRepo(t)

	// Test existing branch
	exists, err := LocalBranchExists(repoPath, "main")
	if err != nil {
		// Try master if main doesn't exist
		exists, err = LocalBranchExists(repoPath, "master")
		if err != nil {
			t.Fatalf("LocalBranchExists failed: %v", err)
		}
	}

	if !exists {
		t.Errorf("main/master branch should exist")
	}

	// Test non-existent branch
	exists, err = LocalBranchExists(repoPath, "non-existent")
	if err != nil {
		t.Fatalf("LocalBranchExists failed: %v", err)
	}

	if exists {
		t.Errorf("non-existent branch should not exist")
	}
}

func TestGetBranch(t *testing.T) {
	repoPath := testutil.CreateTestRepo(t)

	// Create a test branch
	_, err := CreateBranch(repoPath, CreateBranchOptions{
		Name: "test-get-branch",
	})
	if err != nil {
		t.Fatalf("CreateBranch failed: %v", err)
	}

	// Get branch info
	branch, err := GetBranch(repoPath, "test-get-branch")
	if err != nil {
		t.Fatalf("GetBranch failed: %v", err)
	}

	if branch.Name != "test-get-branch" {
		t.Errorf("expected test-get-branch, got: %s", branch.Name)
	}

	if branch.IsRemote {
		t.Errorf("branch should not be remote")
	}

	if branch.Commit == "" {
		t.Errorf("branch should have a commit")
	}
}

func TestGetMergedBranches(t *testing.T) {
	repoPath := testutil.CreateTestRepo(t)

	// Get the default branch
	defaultBranch := GetDefaultBranch(repoPath)
	if defaultBranch == "" {
		t.Skip("no default branch found")
	}

	// Create a branch
	_, err := CreateBranch(repoPath, CreateBranchOptions{
		Name: "merged-branch",
	})
	if err != nil {
		t.Fatalf("CreateBranch failed: %v", err)
	}

	// The branch is created from the same point as the default branch,
	// so it should appear as merged
	merged, err := GetMergedBranches(repoPath, defaultBranch)
	if err != nil {
		t.Fatalf("GetMergedBranches failed: %v", err)
	}

	found := false
	for _, b := range merged {
		if b.Name == "merged-branch" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("merged-branch should be in merged branches list")
	}
}

func TestGetBranchLastCommitDate(t *testing.T) {
	repoPath := testutil.CreateTestRepo(t)

	// Get the default branch
	defaultBranch := GetDefaultBranch(repoPath)
	if defaultBranch == "" {
		t.Skip("no default branch found")
	}

	// Get last commit date
	lastCommit, err := GetBranchLastCommitDate(repoPath, defaultBranch)
	if err != nil {
		t.Fatalf("GetBranchLastCommitDate failed: %v", err)
	}

	// Commit should be recent (within the last minute)
	if time.Since(lastCommit) > time.Minute {
		t.Errorf("last commit should be recent, got: %v", lastCommit)
	}
}

func TestGetStaleBranches(t *testing.T) {
	repoPath := testutil.CreateTestRepo(t)

	// Create a branch (it will have a recent commit)
	_, err := CreateBranch(repoPath, CreateBranchOptions{
		Name: "new-branch",
	})
	if err != nil {
		t.Fatalf("CreateBranch failed: %v", err)
	}

	// Get stale branches with a very short duration
	// All branches should be "stale" with 0 duration
	stale, err := GetStaleBranches(repoPath, 0)
	if err != nil {
		t.Fatalf("GetStaleBranches failed: %v", err)
	}

	// new-branch should be in the list (it's not current)
	found := false
	for _, b := range stale {
		if b.Name == "new-branch" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("new-branch should be in stale branches list with 0 duration")
	}

	// Get stale branches with a long duration
	// No branches should be stale
	stale, err = GetStaleBranches(repoPath, 365*24*time.Hour)
	if err != nil {
		t.Fatalf("GetStaleBranches failed: %v", err)
	}

	if len(stale) > 0 {
		t.Errorf("no branches should be stale with 365 day duration")
	}
}

func TestDeleteBranches(t *testing.T) {
	repoPath := testutil.CreateTestRepo(t)

	// Create multiple branches
	for _, name := range []string{"branch-1", "branch-2", "branch-3"} {
		_, err := CreateBranch(repoPath, CreateBranchOptions{
			Name: name,
		})
		if err != nil {
			t.Fatalf("CreateBranch failed: %v", err)
		}
	}

	// Delete multiple branches
	err := DeleteBranches(repoPath, []string{"branch-1", "branch-2"}, false)
	if err != nil {
		t.Fatalf("DeleteBranches failed: %v", err)
	}

	// Verify they're gone
	for _, name := range []string{"branch-1", "branch-2"} {
		exists, err := LocalBranchExists(repoPath, name)
		if err != nil {
			t.Fatalf("LocalBranchExists failed: %v", err)
		}
		if exists {
			t.Errorf("%s should not exist after deletion", name)
		}
	}

	// branch-3 should still exist
	exists, err := LocalBranchExists(repoPath, "branch-3")
	if err != nil {
		t.Fatalf("LocalBranchExists failed: %v", err)
	}
	if !exists {
		t.Errorf("branch-3 should still exist")
	}
}

func TestDeleteBranches_CannotDeleteCurrentBranch(t *testing.T) {
	repoPath := testutil.CreateTestRepo(t)

	// Get current branch
	currentBranch, err := GetCurrentBranch(repoPath)
	if err != nil {
		t.Fatalf("GetCurrentBranch failed: %v", err)
	}

	// Create a branch to also delete
	_, err = CreateBranch(repoPath, CreateBranchOptions{
		Name: "deletable",
	})
	if err != nil {
		t.Fatalf("CreateBranch failed: %v", err)
	}

	// Try to delete current branch along with another
	err = DeleteBranches(repoPath, []string{currentBranch, "deletable"}, false)

	// Should return an error because current branch can't be deleted
	if err == nil {
		t.Errorf("expected error when deleting current branch")
	}

	// But deletable should still be deleted
	exists, err := LocalBranchExists(repoPath, "deletable")
	if err != nil {
		t.Fatalf("LocalBranchExists failed: %v", err)
	}
	if exists {
		t.Errorf("deletable should have been deleted")
	}
}

func TestGetDefaultBranch(t *testing.T) {
	repoPath := testutil.CreateTestRepo(t)

	defaultBranch := GetDefaultBranch(repoPath)

	// Should be either main or master
	if defaultBranch != "main" && defaultBranch != "master" {
		t.Errorf("expected main or master, got: %s", defaultBranch)
	}
}

func TestGetBranchCleanupInfo(t *testing.T) {
	repoPath := testutil.CreateTestRepo(t)

	// Create some branches
	for _, name := range []string{"feature-1", "feature-2"} {
		_, err := CreateBranch(repoPath, CreateBranchOptions{
			Name: name,
		})
		if err != nil {
			t.Fatalf("CreateBranch failed: %v", err)
		}
	}

	// Get cleanup info
	info, err := GetBranchCleanupInfo(repoPath, "", 30*24*time.Hour)
	if err != nil {
		t.Fatalf("GetBranchCleanupInfo failed: %v", err)
	}

	// Should have at least 2 branches (excluding current and base)
	if len(info) < 2 {
		t.Errorf("expected at least 2 branches in cleanup info, got: %d", len(info))
	}

	// All should have IsMerged=true (they're at same commit as base)
	for _, b := range info {
		if !b.IsMerged {
			t.Errorf("branch %s should be marked as merged", b.Branch.Name)
		}
	}

	// All should have age information
	for _, b := range info {
		if b.AgeString == "" {
			t.Errorf("branch %s should have age string", b.Branch.Name)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{"less than hour", 30 * time.Minute, "less than an hour"},
		{"1 hour", time.Hour, "1 hour"},
		{"5 hours", 5 * time.Hour, "5 hours"},
		{"1 day", 24 * time.Hour, "1 day"},
		{"3 days", 3 * 24 * time.Hour, "3 days"},
		{"1 week", 7 * 24 * time.Hour, "1 week"},
		{"2 weeks", 14 * 24 * time.Hour, "2 weeks"},
		{"1 month", 30 * 24 * time.Hour, "1 month"},
		{"3 months", 90 * 24 * time.Hour, "3 months"},
		{"1 year", 365 * 24 * time.Hour, "1 year"},
		{"2 years", 730 * 24 * time.Hour, "2 years"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDuration(tt.duration)
			if got != tt.want {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.duration, got, tt.want)
			}
		})
	}
}
