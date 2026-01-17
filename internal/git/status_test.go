package git

import (
	"context"
	"testing"

	"github.com/Andrewy-gh/gwt/internal/testutil"
)

// TestGetWorktreeStatusBatch tests the batch status fetching operation
func TestGetWorktreeStatusBatch(t *testing.T) {
	repoPath := testutil.CreateTestRepoWithWorktrees(t)

	worktrees, err := ListWorktrees(repoPath)
	if err != nil {
		t.Fatalf("Failed to list worktrees: %v", err)
	}

	if len(worktrees) == 0 {
		t.Fatal("No worktrees found")
	}

	// Collect paths
	paths := make([]string, len(worktrees))
	for i, wt := range worktrees {
		paths[i] = wt.Path
	}

	// Test batch operation
	ctx := context.Background()
	statuses, errs := GetWorktreeStatusBatch(ctx, paths, 4)

	// Check that we got results for all paths
	if len(statuses) != len(paths) {
		t.Errorf("Expected %d statuses, got %d", len(paths), len(statuses))
	}

	// Check for errors
	if len(errs) > 0 {
		t.Errorf("Got %d errors during batch fetch: %v", len(errs), errs)
	}

	// Verify each status is not nil
	for path, status := range statuses {
		if status == nil {
			t.Errorf("Status for path %s is nil", path)
		}
	}
}

// TestGetWorktreeStatusBatch_Correctness verifies batch results match sequential
func TestGetWorktreeStatusBatch_Correctness(t *testing.T) {
	repoPath := testutil.CreateTestRepoWithWorktrees(t)

	worktrees, err := ListWorktrees(repoPath)
	if err != nil {
		t.Fatalf("Failed to list worktrees: %v", err)
	}

	if len(worktrees) < 2 {
		t.Skip("Need at least 2 worktrees for this test")
	}

	// Collect paths
	paths := make([]string, len(worktrees))
	for i, wt := range worktrees {
		paths[i] = wt.Path
	}

	// Get statuses sequentially
	sequentialStatuses := make(map[string]*WorktreeStatus)
	for _, path := range paths {
		status, err := GetWorktreeStatus(path)
		if err == nil {
			sequentialStatuses[path] = status
		}
	}

	// Get statuses in batch
	ctx := context.Background()
	batchStatuses, _ := GetWorktreeStatusBatch(ctx, paths, 4)

	// Compare results
	if len(batchStatuses) != len(sequentialStatuses) {
		t.Errorf("Batch returned %d results, sequential returned %d",
			len(batchStatuses), len(sequentialStatuses))
	}

	for path, batchStatus := range batchStatuses {
		seqStatus, ok := sequentialStatuses[path]
		if !ok {
			t.Errorf("Batch has status for %s but sequential doesn't", path)
			continue
		}

		// Compare key fields
		if batchStatus.Clean != seqStatus.Clean {
			t.Errorf("Path %s: batch.Clean=%v, sequential.Clean=%v",
				path, batchStatus.Clean, seqStatus.Clean)
		}
		if batchStatus.StagedCount != seqStatus.StagedCount {
			t.Errorf("Path %s: batch.StagedCount=%d, sequential.StagedCount=%d",
				path, batchStatus.StagedCount, seqStatus.StagedCount)
		}
		if batchStatus.UnstagedCount != seqStatus.UnstagedCount {
			t.Errorf("Path %s: batch.UnstagedCount=%d, sequential.UnstagedCount=%d",
				path, batchStatus.UnstagedCount, seqStatus.UnstagedCount)
		}
		if batchStatus.UntrackedCount != seqStatus.UntrackedCount {
			t.Errorf("Path %s: batch.UntrackedCount=%d, sequential.UntrackedCount=%d",
				path, batchStatus.UntrackedCount, seqStatus.UntrackedCount)
		}
	}
}

// TestGetWorktreeStatusBatch_WorkerCounts tests different worker pool sizes
func TestGetWorktreeStatusBatch_WorkerCounts(t *testing.T) {
	repoPath := testutil.CreateTestRepoWithWorktrees(t)

	worktrees, err := ListWorktrees(repoPath)
	if err != nil {
		t.Fatalf("Failed to list worktrees: %v", err)
	}

	if len(worktrees) < 2 {
		t.Skip("Need at least 2 worktrees for this test")
	}

	// Collect paths
	paths := make([]string, len(worktrees))
	for i, wt := range worktrees {
		paths[i] = wt.Path
	}

	ctx := context.Background()

	// Test with different worker counts
	workerCounts := []int{1, 4, 8}
	for _, workers := range workerCounts {
		t.Run("Workers="+string(rune('0'+workers)), func(t *testing.T) {
			statuses, errs := GetWorktreeStatusBatch(ctx, paths, workers)

			if len(statuses) != len(paths) {
				t.Errorf("Workers=%d: Expected %d statuses, got %d",
					workers, len(paths), len(statuses))
			}

			if len(errs) > 0 {
				t.Errorf("Workers=%d: Got %d errors: %v",
					workers, len(errs), errs)
			}

			// Verify all statuses are present
			for _, path := range paths {
				if _, ok := statuses[path]; !ok {
					t.Errorf("Workers=%d: Missing status for path %s",
						workers, path)
				}
			}
		})
	}
}

// TestGetWorktreeStatusBatch_EmptyPaths tests batch with no paths
func TestGetWorktreeStatusBatch_EmptyPaths(t *testing.T) {
	ctx := context.Background()
	statuses, errs := GetWorktreeStatusBatch(ctx, []string{}, 4)

	if len(statuses) != 0 {
		t.Errorf("Expected 0 statuses for empty paths, got %d", len(statuses))
	}

	if len(errs) != 0 {
		t.Errorf("Expected 0 errors for empty paths, got %d", len(errs))
	}
}

// TestGetWorktreeStatusBatch_InvalidPaths tests error handling
func TestGetWorktreeStatusBatch_InvalidPaths(t *testing.T) {
	ctx := context.Background()

	// Mix of valid and invalid paths
	paths := []string{
		"/invalid/path/1",
		"/invalid/path/2",
		"/nonexistent/worktree",
	}

	statuses, errs := GetWorktreeStatusBatch(ctx, paths, 4)

	// Should have some errors for invalid paths
	if len(errs) == 0 {
		t.Error("Expected errors for invalid paths, got none")
	}

	// Statuses map might be empty or have partial results
	if len(statuses) > len(paths) {
		t.Errorf("Got more statuses (%d) than paths (%d)", len(statuses), len(paths))
	}
}

// TestGetWorktreeStatusBatch_ContextCancellation tests context cancellation
func TestGetWorktreeStatusBatch_ContextCancellation(t *testing.T) {
	repoPath := testutil.CreateTestRepoWithWorktrees(t)

	worktrees, err := ListWorktrees(repoPath)
	if err != nil {
		t.Fatalf("Failed to list worktrees: %v", err)
	}

	if len(worktrees) < 2 {
		t.Skip("Need at least 2 worktrees for this test")
	}

	// Collect paths
	paths := make([]string, len(worktrees))
	for i, wt := range worktrees {
		paths[i] = wt.Path
	}

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	statuses, _ := GetWorktreeStatusBatch(ctx, paths, 4)

	// With a cancelled context, we might get partial or no results
	// This tests that the function handles cancellation gracefully
	if len(statuses) > len(paths) {
		t.Errorf("Got more statuses (%d) than paths (%d) with cancelled context",
			len(statuses), len(paths))
	}
}

// BenchmarkGetWorktreeStatusBatch benchmarks batch status fetching
func BenchmarkGetWorktreeStatusBatch(b *testing.B) {
	repoPath := testutil.CreateTestRepoWithWorktrees(&testing.T{})

	worktrees, err := ListWorktrees(repoPath)
	if err != nil {
		b.Fatalf("Failed to list worktrees: %v", err)
	}

	paths := make([]string, len(worktrees))
	for i, wt := range worktrees {
		paths[i] = wt.Path
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetWorktreeStatusBatch(ctx, paths, 8)
	}
}

// BenchmarkGetWorktreeStatusSequential benchmarks sequential status fetching for comparison
func BenchmarkGetWorktreeStatusSequential(b *testing.B) {
	repoPath := testutil.CreateTestRepoWithWorktrees(&testing.T{})

	worktrees, err := ListWorktrees(repoPath)
	if err != nil {
		b.Fatalf("Failed to list worktrees: %v", err)
	}

	paths := make([]string, len(worktrees))
	for i, wt := range worktrees {
		paths[i] = wt.Path
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, path := range paths {
			GetWorktreeStatus(path)
		}
	}
}
