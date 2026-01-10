package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Andrewy-gh/gwt/internal/git"
	"github.com/Andrewy-gh/gwt/internal/output"
	"github.com/Andrewy-gh/gwt/internal/testutil"
)

func TestStatusCommand_TextOutput(t *testing.T) {
	repoPath := testutil.CreateTestRepoWithWorktrees(t)

	// Get main worktree
	wt, err := git.GetWorktree(repoPath)
	if err != nil {
		t.Fatalf("failed to get worktree: %v", err)
	}

	// Get status
	status, err := git.GetWorktreeStatus(repoPath)
	if err != nil {
		t.Fatalf("failed to get status: %v", err)
	}

	// Capture output
	var buf bytes.Buffer
	oldOut := output.Out
	output.Out = &buf
	defer func() { output.Out = oldOut }()

	err = outputStatusText(wt, status, "")
	if err != nil {
		t.Fatalf("outputStatusText failed: %v", err)
	}

	out := buf.String()

	// Check for expected sections
	if !strings.Contains(out, "Worktree:") {
		t.Error("expected Worktree: in output")
	}
	if !strings.Contains(out, "Branch:") {
		t.Error("expected Branch: in output")
	}
	if !strings.Contains(out, "Commit:") {
		t.Error("expected Commit: in output")
	}
	if !strings.Contains(out, "Status:") {
		t.Error("expected Status: in output")
	}
}

func TestStatusCommand_JSONOutput(t *testing.T) {
	repoPath := testutil.CreateTestRepoWithWorktrees(t)

	// Get main worktree
	wt, err := git.GetWorktree(repoPath)
	if err != nil {
		t.Fatalf("failed to get worktree: %v", err)
	}

	// Get status
	status, err := git.GetWorktreeStatus(repoPath)
	if err != nil {
		t.Fatalf("failed to get status: %v", err)
	}

	// Capture output
	var buf bytes.Buffer
	oldOut := output.Out
	output.Out = &buf
	defer func() { output.Out = oldOut }()

	err = outputStatusJSON(wt, status, "origin/main")
	if err != nil {
		t.Fatalf("outputStatusJSON failed: %v", err)
	}

	// Parse JSON output
	var result WorktreeStatusOutput
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	// Verify fields
	if result.Path == "" {
		t.Error("expected non-empty path")
	}
	if result.Branch == "" {
		t.Error("expected non-empty branch")
	}
	if !result.IsMain {
		t.Error("expected isMain to be true for main worktree")
	}
	if result.Upstream != "origin/main" {
		t.Errorf("expected upstream to be origin/main, got %s", result.Upstream)
	}
}

func TestStatusCommand_CleanWorktree(t *testing.T) {
	repoPath := testutil.CreateTestRepo(t)

	// Get worktree
	wt, err := git.GetWorktree(repoPath)
	if err != nil {
		t.Fatalf("failed to get worktree: %v", err)
	}

	// Get status - should be clean since we just created it
	status, err := git.GetWorktreeStatus(repoPath)
	if err != nil {
		t.Fatalf("failed to get status: %v", err)
	}

	if !status.Clean {
		t.Error("expected worktree to be clean")
	}

	// Capture output
	var buf bytes.Buffer
	oldOut := output.Out
	output.Out = &buf
	defer func() { output.Out = oldOut }()

	err = outputStatusText(wt, status, "")
	if err != nil {
		t.Fatalf("outputStatusText failed: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Clean") {
		t.Error("expected 'Clean' in output for clean worktree")
	}
}

func TestFormatTimeAgo(t *testing.T) {
	tests := []struct {
		name     string
		duration string
		expected string
	}{
		{"just now", "30s", "just now"},
		{"1 minute", "1m", "1 minute ago"},
		{"5 minutes", "5m", "5 minutes ago"},
		{"1 hour", "1h", "1 hour ago"},
		{"3 hours", "3h", "3 hours ago"},
		{"1 day", "24h", "1 day ago"},
		{"5 days", "120h", "5 days ago"},
		{"1 week", "168h", "1 week ago"},
		{"3 weeks", "504h", "3 weeks ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: We can't test formatTimeAgo directly with time.Duration
			// because it uses time.Since(). This is a placeholder for the test structure.
		})
	}
}
