package cli

import (
	"testing"
	"time"

	"github.com/Andrewy-gh/gwt/internal/git"
	"github.com/Andrewy-gh/gwt/internal/testutil"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Duration
		wantErr bool
	}{
		{"days", "30d", 30 * 24 * time.Hour, false},
		{"weeks", "2w", 14 * 24 * time.Hour, false},
		{"months", "6m", 180 * 24 * time.Hour, false},
		{"years", "1y", 365 * 24 * time.Hour, false},
		{"single day", "1d", 24 * time.Hour, false},
		{"with spaces", " 30d ", 30 * 24 * time.Hour, false},
		{"uppercase", "30D", 30 * 24 * time.Hour, false},
		{"go duration hours", "24h", 24 * time.Hour, false},
		{"empty", "", 0, true},
		{"invalid unit", "30x", 0, true},
		{"no number", "d", 0, true},
		{"invalid format", "abc", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDuration(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDuration(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseDuration(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestShowBranchList(t *testing.T) {
	// Create test branch info
	branches := []git.BranchCleanupInfo{
		{
			Branch: git.Branch{
				Name:   "feature-1",
				Commit: "abc1234",
			},
			IsMerged:    true,
			IsStale:     false,
			AgeString:   "3 days",
			HasWorktree: false,
		},
		{
			Branch: git.Branch{
				Name:   "feature-2",
				Commit: "def5678",
			},
			IsMerged:    false,
			IsStale:     true,
			AgeString:   "2 weeks",
			HasWorktree: true,
		},
	}

	// This should not panic
	err := showBranchList(branches, "main")
	if err != nil {
		t.Errorf("showBranchList returned error: %v", err)
	}
}

func TestShowDryRunCleanup(t *testing.T) {
	// Create test branch info
	branches := []git.BranchCleanupInfo{
		{
			Branch: git.Branch{
				Name:   "merged-branch",
				Commit: "abc1234",
			},
			IsMerged:  true,
			AgeString: "3 days",
		},
		{
			Branch: git.Branch{
				Name:   "stale-branch",
				Commit: "def5678",
			},
			IsStale:   true,
			AgeString: "60 days",
		},
	}

	// This should not panic
	err := showDryRunCleanup(branches)
	if err != nil {
		t.Errorf("showDryRunCleanup returned error: %v", err)
	}
}

func TestDeleteBranchesInCleanup(t *testing.T) {
	repoPath := testutil.CreateTestRepo(t)

	// Create some branches
	for _, name := range []string{"cleanup-1", "cleanup-2"} {
		_, err := git.CreateBranch(repoPath, git.CreateBranchOptions{
			Name: name,
		})
		if err != nil {
			t.Fatalf("CreateBranch failed: %v", err)
		}
	}

	// Create cleanup info
	branches := []git.BranchCleanupInfo{
		{
			Branch: git.Branch{
				Name: "cleanup-1",
			},
			IsMerged: true,
		},
		{
			Branch: git.Branch{
				Name: "cleanup-2",
			},
			IsMerged: true,
		},
	}

	// Reset the global opts for the test
	cleanupOpts = CleanupOptions{}

	// Delete the branches
	err := deleteBranches(repoPath, branches)
	if err != nil {
		t.Errorf("deleteBranches returned error: %v", err)
	}

	// Verify they're gone
	for _, name := range []string{"cleanup-1", "cleanup-2"} {
		exists, err := git.LocalBranchExists(repoPath, name)
		if err != nil {
			t.Fatalf("LocalBranchExists failed: %v", err)
		}
		if exists {
			t.Errorf("%s should have been deleted", name)
		}
	}
}

func TestCleanupExcludePattern(t *testing.T) {
	// Test that exclude map works correctly
	excludeMap := make(map[string]bool)
	for _, e := range []string{"main", "develop", "release"} {
		excludeMap[e] = true
	}

	tests := []struct {
		name     string
		branch   string
		excluded bool
	}{
		{"main excluded", "main", true},
		{"develop excluded", "develop", true},
		{"feature not excluded", "feature-1", false},
		{"release excluded", "release", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if excludeMap[tt.branch] != tt.excluded {
				t.Errorf("branch %s: expected excluded=%v, got %v", tt.branch, tt.excluded, excludeMap[tt.branch])
			}
		})
	}
}

func TestCleanupFilterCriteria(t *testing.T) {
	// Create test branch info representing different scenarios
	branches := []git.BranchCleanupInfo{
		{
			Branch:      git.Branch{Name: "merged-only"},
			IsMerged:    true,
			IsStale:     false,
			HasWorktree: false,
		},
		{
			Branch:      git.Branch{Name: "stale-only"},
			IsMerged:    false,
			IsStale:     true,
			HasWorktree: false,
		},
		{
			Branch:      git.Branch{Name: "merged-and-stale"},
			IsMerged:    true,
			IsStale:     true,
			HasWorktree: false,
		},
		{
			Branch:      git.Branch{Name: "has-worktree"},
			IsMerged:    true,
			IsStale:     true,
			HasWorktree: true, // Should be skipped
		},
		{
			Branch:      git.Branch{Name: "fresh"},
			IsMerged:    false,
			IsStale:     false,
			HasWorktree: false,
		},
	}

	t.Run("filter merged only", func(t *testing.T) {
		var candidates []git.BranchCleanupInfo
		for _, b := range branches {
			if b.HasWorktree {
				continue
			}
			if b.IsMerged {
				candidates = append(candidates, b)
			}
		}

		// Should have merged-only and merged-and-stale
		if len(candidates) != 2 {
			t.Errorf("expected 2 merged candidates, got %d", len(candidates))
		}
	})

	t.Run("filter stale only", func(t *testing.T) {
		var candidates []git.BranchCleanupInfo
		for _, b := range branches {
			if b.HasWorktree {
				continue
			}
			if b.IsStale {
				candidates = append(candidates, b)
			}
		}

		// Should have stale-only and merged-and-stale
		if len(candidates) != 2 {
			t.Errorf("expected 2 stale candidates, got %d", len(candidates))
		}
	})

	t.Run("filter with worktree exclusion", func(t *testing.T) {
		var candidates []git.BranchCleanupInfo
		for _, b := range branches {
			if !b.HasWorktree {
				candidates = append(candidates, b)
			}
		}

		// Should have 4 branches (all except has-worktree)
		if len(candidates) != 4 {
			t.Errorf("expected 4 candidates without worktree, got %d", len(candidates))
		}
	})
}
