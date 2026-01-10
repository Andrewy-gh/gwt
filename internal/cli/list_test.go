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

func TestListCommand_TableOutput(t *testing.T) {
	repoPath := testutil.CreateTestRepoWithWorktrees(t)

	// Capture output
	var buf bytes.Buffer
	oldOut := output.Out
	output.Out = &buf
	defer func() { output.Out = oldOut }()

	// Change to repo directory and run list
	err := runListInDir(repoPath)
	if err != nil {
		t.Fatalf("runList failed: %v", err)
	}

	out := buf.String()

	// Check for expected columns
	if !strings.Contains(out, "PATH") {
		t.Error("expected PATH column header")
	}
	if !strings.Contains(out, "BRANCH") {
		t.Error("expected BRANCH column header")
	}
	if !strings.Contains(out, "COMMIT") {
		t.Error("expected COMMIT column header")
	}

	// Check for worktree entries
	if !strings.Contains(out, "main") && !strings.Contains(out, "master") {
		t.Error("expected main/master branch in output")
	}
	if !strings.Contains(out, "feature-1") {
		t.Error("expected feature-1 branch in output")
	}
}

func TestListCommand_JSONOutput(t *testing.T) {
	repoPath := testutil.CreateTestRepoWithWorktrees(t)

	// Get worktrees for JSON output test
	worktrees, err := git.ListWorktrees(repoPath)
	if err != nil {
		t.Fatalf("failed to list worktrees: %v", err)
	}

	// Filter non-bare/prunable
	var filtered []git.Worktree
	for _, wt := range worktrees {
		if !wt.IsBare && !wt.Prunable {
			filtered = append(filtered, wt)
		}
	}

	// Capture output
	var buf bytes.Buffer
	oldOut := output.Out
	output.Out = &buf
	defer func() { output.Out = oldOut }()

	err = outputJSON(filtered)
	if err != nil {
		t.Fatalf("outputJSON failed: %v", err)
	}

	// Parse JSON output
	var items []WorktreeListItem
	if err := json.Unmarshal(buf.Bytes(), &items); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if len(items) != 2 {
		t.Errorf("expected 2 items in JSON, got %d", len(items))
	}

	// Check first item is main
	if !items[0].IsMain {
		t.Error("expected first item to be main worktree")
	}

	// Check second item is feature-1
	if items[1].Branch != "feature-1" {
		t.Errorf("expected second item branch to be feature-1, got %s", items[1].Branch)
	}
}

func TestListCommand_SimpleOutput(t *testing.T) {
	repoPath := testutil.CreateTestRepoWithWorktrees(t)

	// Get worktrees
	worktrees, err := git.ListWorktrees(repoPath)
	if err != nil {
		t.Fatalf("failed to list worktrees: %v", err)
	}

	// Filter non-bare/prunable
	var filtered []git.Worktree
	for _, wt := range worktrees {
		if !wt.IsBare && !wt.Prunable {
			filtered = append(filtered, wt)
		}
	}

	// Capture output
	var buf bytes.Buffer
	oldOut := output.Out
	output.Out = &buf
	defer func() { output.Out = oldOut }()

	err = outputSimple(filtered)
	if err != nil {
		t.Fatalf("outputSimple failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines in simple output, got %d", len(lines))
	}
}

// runListInDir is a helper to run list command in a specific directory
func runListInDir(dir string) error {
	worktrees, err := git.ListWorktrees(dir)
	if err != nil {
		return err
	}

	// Filter
	var filtered []git.Worktree
	for _, wt := range worktrees {
		if !wt.IsBare && !wt.Prunable {
			filtered = append(filtered, wt)
		}
	}

	return outputTable(filtered, dir)
}
