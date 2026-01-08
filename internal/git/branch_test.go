package git

import (
	"testing"

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
