package cli

import (
	"testing"

	"github.com/Andrewy-gh/gwt/internal/git"
	"github.com/Andrewy-gh/gwt/internal/testutil"
)

func TestPreDeleteChecks_MainWorktree(t *testing.T) {
	repoPath := testutil.CreateTestRepoWithWorktrees(t)

	// Get main worktree
	worktrees, err := git.ListWorktrees(repoPath)
	if err != nil {
		t.Fatalf("failed to list worktrees: %v", err)
	}

	var mainWt *git.Worktree
	for _, wt := range worktrees {
		if wt.IsMain {
			mainWt = &wt
			break
		}
	}

	if mainWt == nil {
		t.Fatal("expected to find main worktree")
	}

	// Run pre-delete checks
	checks := runPreDeleteChecks(repoPath, mainWt)

	// Should have blocking check for main worktree
	hasMainBlock := false
	for _, c := range checks {
		if c.Name == "IsMain" && c.Status == CheckBlock {
			hasMainBlock = true
			break
		}
	}

	if !hasMainBlock {
		t.Error("expected blocking check for main worktree")
	}

	// hasBlockingCheck should return true
	if !hasBlockingCheck(checks) {
		t.Error("hasBlockingCheck should return true for main worktree")
	}
}

func TestPreDeleteChecks_CleanWorktree(t *testing.T) {
	repoPath, worktreePath := testutil.CreateTestRepoWithWorktreesAndPath(t)

	// Get worktree
	wt, err := git.GetWorktree(worktreePath)
	if err != nil {
		t.Fatalf("failed to get worktree: %v", err)
	}

	// Run pre-delete checks
	checks := runPreDeleteChecks(repoPath, wt)

	// Should not have blocking checks (it's not the main worktree)
	if hasBlockingCheck(checks) {
		t.Error("expected no blocking checks for clean linked worktree")
	}

	// Should not have uncommitted changes warning
	hasUncommittedWarn := false
	for _, c := range checks {
		if c.Name == "UncommittedChanges" {
			hasUncommittedWarn = true
			break
		}
	}

	if hasUncommittedWarn {
		t.Error("expected no uncommitted changes warning for clean worktree")
	}
}

func TestPreDeleteChecks_LockedWorktree(t *testing.T) {
	repoPath, _ := testutil.CreateTestRepoWithWorktreesAndPath(t)

	// Create a mock locked worktree for testing the check logic
	wt := &git.Worktree{
		Path:   "/fake/path",
		Branch: "feature-test",
		Locked: true,
	}

	// Run pre-delete checks
	checks := runPreDeleteChecks(repoPath, wt)

	// Should have warning check for locked worktree
	hasLockedWarn := false
	for _, c := range checks {
		if c.Name == "Locked" && c.Status == CheckWarn {
			hasLockedWarn = true
			break
		}
	}

	if !hasLockedWarn {
		t.Error("expected warning check for locked worktree")
	}
}

func TestResolveDeleteTargets_ByBranch(t *testing.T) {
	repoPath := testutil.CreateTestRepoWithWorktrees(t)

	// Resolve by branch name
	targets := resolveDeleteTargets(repoPath, []string{"feature-1"})

	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(targets))
	}

	if targets[0].Error != nil {
		t.Errorf("unexpected error: %v", targets[0].Error)
	}

	if targets[0].Worktree == nil {
		t.Fatal("expected worktree to be resolved")
	}

	if targets[0].Worktree.Branch != "feature-1" {
		t.Errorf("expected feature-1 branch, got %s", targets[0].Worktree.Branch)
	}
}

func TestResolveDeleteTargets_NonExistent(t *testing.T) {
	repoPath := testutil.CreateTestRepoWithWorktrees(t)

	// Try to resolve non-existent branch
	targets := resolveDeleteTargets(repoPath, []string{"non-existent-branch"})

	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(targets))
	}

	if targets[0].Error == nil {
		t.Error("expected error for non-existent branch")
	}
}

func TestResolveDeleteTargets_Multiple(t *testing.T) {
	repoPath := testutil.CreateTestRepoWithWorktrees(t)

	// Resolve multiple targets (one exists, one doesn't)
	targets := resolveDeleteTargets(repoPath, []string{"feature-1", "non-existent"})

	if len(targets) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(targets))
	}

	// First should resolve
	if targets[0].Error != nil {
		t.Errorf("expected feature-1 to resolve: %v", targets[0].Error)
	}

	// Second should error
	if targets[1].Error == nil {
		t.Error("expected error for non-existent branch")
	}
}

func TestGetDefaultBranch(t *testing.T) {
	repoPath := testutil.CreateTestRepo(t)

	branch := getDefaultBranch(repoPath)

	// Should be main (as created by testutil)
	if branch != "main" && branch != "master" {
		t.Errorf("expected main or master, got %s", branch)
	}
}

func TestHasBlockingCheck(t *testing.T) {
	tests := []struct {
		name     string
		checks   []PreDeleteCheck
		expected bool
	}{
		{
			name:     "empty checks",
			checks:   []PreDeleteCheck{},
			expected: false,
		},
		{
			name: "only pass",
			checks: []PreDeleteCheck{
				{Name: "OK", Status: CheckPass},
			},
			expected: false,
		},
		{
			name: "only warn",
			checks: []PreDeleteCheck{
				{Name: "Dirty", Status: CheckWarn},
			},
			expected: false,
		},
		{
			name: "has block",
			checks: []PreDeleteCheck{
				{Name: "IsMain", Status: CheckBlock},
			},
			expected: true,
		},
		{
			name: "mixed with block",
			checks: []PreDeleteCheck{
				{Name: "Dirty", Status: CheckWarn},
				{Name: "IsMain", Status: CheckBlock},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasBlockingCheck(tt.checks)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestHasWarningCheck(t *testing.T) {
	tests := []struct {
		name     string
		checks   []PreDeleteCheck
		expected bool
	}{
		{
			name:     "empty checks",
			checks:   []PreDeleteCheck{},
			expected: false,
		},
		{
			name: "only pass",
			checks: []PreDeleteCheck{
				{Name: "OK", Status: CheckPass},
			},
			expected: false,
		},
		{
			name: "has warn",
			checks: []PreDeleteCheck{
				{Name: "Dirty", Status: CheckWarn},
			},
			expected: true,
		},
		{
			name: "only block",
			checks: []PreDeleteCheck{
				{Name: "IsMain", Status: CheckBlock},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasWarningCheck(tt.checks)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
