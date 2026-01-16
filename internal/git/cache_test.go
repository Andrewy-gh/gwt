package git

import (
	"context"
	"testing"
	"time"

	"github.com/Andrewy-gh/gwt/internal/testutil"
)

func TestCacheEnabledDisabled(t *testing.T) {
	// Save original state
	originalState := IsCacheEnabled()
	defer SetCacheEnabled(originalState)

	// Test enabling cache
	SetCacheEnabled(true)
	if !IsCacheEnabled() {
		t.Error("Expected cache to be enabled")
	}

	// Test disabling cache
	SetCacheEnabled(false)
	if IsCacheEnabled() {
		t.Error("Expected cache to be disabled")
	}
}

func TestListWorktreesCached(t *testing.T) {
	// Enable cache for test
	SetCacheEnabled(true)
	defer SetCacheEnabled(false)

	repoPath := testutil.CreateTestRepoWithWorktrees(t)

	// First call should hit git
	worktrees1, err := ListWorktreesCached(repoPath, false)
	if err != nil {
		t.Fatalf("ListWorktreesCached failed: %v", err)
	}

	if len(worktrees1) != 2 {
		t.Errorf("Expected 2 worktrees, got %d", len(worktrees1))
	}

	// Second call should hit cache (should be instant)
	start := time.Now()
	worktrees2, err := ListWorktreesCached(repoPath, false)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("ListWorktreesCached failed: %v", err)
	}

	// Cache hit should be very fast (< 10ms)
	if duration > 10*time.Millisecond {
		t.Logf("Warning: Cache hit took %v, expected < 10ms", duration)
	}

	// Results should be the same
	if len(worktrees2) != len(worktrees1) {
		t.Errorf("Cached result has different length: %d vs %d", len(worktrees2), len(worktrees1))
	}
}

func TestListWorktreesCachedBypass(t *testing.T) {
	SetCacheEnabled(true)
	defer SetCacheEnabled(false)

	repoPath := testutil.CreateTestRepoWithWorktrees(t)

	// First call
	worktrees1, err := ListWorktreesCached(repoPath, false)
	if err != nil {
		t.Fatalf("ListWorktreesCached failed: %v", err)
	}

	// Second call with bypass should not use cache
	worktrees2, err := ListWorktreesCached(repoPath, true)
	if err != nil {
		t.Fatalf("ListWorktreesCached with bypass failed: %v", err)
	}

	// Results should still be the same
	if len(worktrees2) != len(worktrees1) {
		t.Errorf("Bypassed result has different length: %d vs %d", len(worktrees2), len(worktrees1))
	}
}

func TestListBranchesCached(t *testing.T) {
	SetCacheEnabled(true)
	defer SetCacheEnabled(false)

	repoPath := testutil.CreateTestRepo(t)

	// First call
	branches1, err := ListLocalBranchesCached(repoPath, false)
	if err != nil {
		t.Fatalf("ListLocalBranchesCached failed: %v", err)
	}

	// Should have at least main branch
	if len(branches1) == 0 {
		t.Error("Expected at least one branch")
	}

	// Second call should hit cache
	start := time.Now()
	branches2, err := ListLocalBranchesCached(repoPath, false)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("ListLocalBranchesCached failed: %v", err)
	}

	// Cache hit should be fast
	if duration > 10*time.Millisecond {
		t.Logf("Warning: Cache hit took %v, expected < 10ms", duration)
	}

	// Results should match
	if len(branches2) != len(branches1) {
		t.Errorf("Cached result has different length: %d vs %d", len(branches2), len(branches1))
	}
}

func TestGetWorktreeStatusCached(t *testing.T) {
	SetCacheEnabled(true)
	defer SetCacheEnabled(false)

	_, worktreePath := testutil.CreateTestRepoWithWorktreesAndPath(t)

	// First call
	status1, err := GetWorktreeStatusCached(worktreePath, false)
	if err != nil {
		t.Fatalf("GetWorktreeStatusCached failed: %v", err)
	}

	if status1 == nil {
		t.Fatal("Expected status to be non-nil")
	}

	// Second call should hit cache
	start := time.Now()
	status2, err := GetWorktreeStatusCached(worktreePath, false)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("GetWorktreeStatusCached failed: %v", err)
	}

	// Cache hit should be very fast
	if duration > 10*time.Millisecond {
		t.Logf("Warning: Cache hit took %v, expected < 10ms", duration)
	}

	// Results should match
	if status2.Clean != status1.Clean {
		t.Error("Cached status.Clean doesn't match")
	}
}

func TestGetWorktreeStatusBatchCached(t *testing.T) {
	SetCacheEnabled(true)
	defer SetCacheEnabled(false)

	repoPath := testutil.CreateTestRepoWithWorktrees(t)

	worktrees, err := ListWorktrees(repoPath)
	if err != nil {
		t.Fatalf("ListWorktrees failed: %v", err)
	}

	paths := make([]string, len(worktrees))
	for i, wt := range worktrees {
		paths[i] = wt.Path
	}

	// First call - all cache misses
	statuses1, errors1 := GetWorktreeStatusBatchCached(context.Background(), paths, 4, false)
	if len(errors1) > 0 {
		t.Fatalf("GetWorktreeStatusBatchCached failed: %v", errors1)
	}

	if len(statuses1) != len(paths) {
		t.Errorf("Expected %d statuses, got %d", len(paths), len(statuses1))
	}

	// Second call - all cache hits (should be much faster)
	start := time.Now()
	statuses2, errors2 := GetWorktreeStatusBatchCached(context.Background(), paths, 4, false)
	duration := time.Since(start)

	if len(errors2) > 0 {
		t.Fatalf("GetWorktreeStatusBatchCached failed: %v", errors2)
	}

	// All cache hits should be very fast
	if duration > 20*time.Millisecond {
		t.Logf("Warning: All cache hits took %v, expected < 20ms", duration)
	}

	// Results should match
	if len(statuses2) != len(statuses1) {
		t.Errorf("Cached result has different length: %d vs %d", len(statuses2), len(statuses1))
	}
}

func TestInvalidateWorktreeCache(t *testing.T) {
	SetCacheEnabled(true)
	defer SetCacheEnabled(false)

	repoPath := testutil.CreateTestRepoWithWorktrees(t)

	// Populate cache
	worktrees1, err := ListWorktreesCached(repoPath, false)
	if err != nil {
		t.Fatalf("ListWorktreesCached failed: %v", err)
	}

	// Invalidate cache
	InvalidateWorktreeCache(repoPath)

	// Next call should hit git again (not cache)
	// We can't directly test if it hits git, but we can verify it still works
	worktrees2, err := ListWorktreesCached(repoPath, false)
	if err != nil {
		t.Fatalf("ListWorktreesCached after invalidation failed: %v", err)
	}

	if len(worktrees2) != len(worktrees1) {
		t.Errorf("Result after invalidation has different length: %d vs %d", len(worktrees2), len(worktrees1))
	}
}

func TestInvalidateBranchCache(t *testing.T) {
	SetCacheEnabled(true)
	defer SetCacheEnabled(false)

	repoPath := testutil.CreateTestRepo(t)

	// Populate cache
	branches1, err := ListLocalBranchesCached(repoPath, false)
	if err != nil {
		t.Fatalf("ListLocalBranchesCached failed: %v", err)
	}

	// Invalidate cache
	InvalidateBranchCache(repoPath)

	// Next call should work correctly
	branches2, err := ListLocalBranchesCached(repoPath, false)
	if err != nil {
		t.Fatalf("ListLocalBranchesCached after invalidation failed: %v", err)
	}

	if len(branches2) != len(branches1) {
		t.Errorf("Result after invalidation has different length: %d vs %d", len(branches2), len(branches1))
	}
}

func TestInvalidateStatusCache(t *testing.T) {
	SetCacheEnabled(true)
	defer SetCacheEnabled(false)

	_, worktreePath := testutil.CreateTestRepoWithWorktreesAndPath(t)

	// Populate cache
	status1, err := GetWorktreeStatusCached(worktreePath, false)
	if err != nil {
		t.Fatalf("GetWorktreeStatusCached failed: %v", err)
	}

	// Invalidate cache
	InvalidateStatusCache(worktreePath)

	// Next call should work correctly
	status2, err := GetWorktreeStatusCached(worktreePath, false)
	if err != nil {
		t.Fatalf("GetWorktreeStatusCached after invalidation failed: %v", err)
	}

	if status2.Clean != status1.Clean {
		t.Error("Status changed after invalidation (unexpected)")
	}
}

func TestClearAllCaches(t *testing.T) {
	SetCacheEnabled(true)
	defer SetCacheEnabled(false)

	repoPath := testutil.CreateTestRepoWithWorktrees(t)

	// Populate all caches
	_, _ = ListWorktreesCached(repoPath, false)
	_, _ = ListLocalBranchesCached(repoPath, false)

	worktrees, _ := ListWorktrees(repoPath)
	if len(worktrees) > 0 {
		_, _ = GetWorktreeStatusCached(worktrees[0].Path, false)
	}

	// Clear all caches
	ClearAllCaches()

	// Get stats - all should be empty
	stats := GetCacheStats()
	if stats.WorktreeList.TotalItems != 0 {
		t.Errorf("Expected worktree cache to be empty, got %d items", stats.WorktreeList.TotalItems)
	}
	if stats.BranchList.TotalItems != 0 {
		t.Errorf("Expected branch cache to be empty, got %d items", stats.BranchList.TotalItems)
	}
	if stats.Status.TotalItems != 0 {
		t.Errorf("Expected status cache to be empty, got %d items", stats.Status.TotalItems)
	}
}

func TestCacheStats(t *testing.T) {
	SetCacheEnabled(true)
	defer SetCacheEnabled(false)

	repoPath := testutil.CreateTestRepoWithWorktrees(t)

	// Clear to start fresh
	ClearAllCaches()

	// Populate some caches
	_, _ = ListWorktreesCached(repoPath, false)
	_, _ = ListLocalBranchesCached(repoPath, false)

	// Get stats
	stats := GetCacheStats()

	// Should have at least one item in each cache
	if stats.WorktreeList.TotalItems == 0 {
		t.Error("Expected worktree cache to have items")
	}
	if stats.BranchList.TotalItems == 0 {
		t.Error("Expected branch cache to have items")
	}
}

func TestCacheTTLExpiration(t *testing.T) {
	SetCacheEnabled(true)
	defer SetCacheEnabled(false)

	// Set very short TTLs for testing
	SetCacheTTLs(50*time.Millisecond, 50*time.Millisecond, 50*time.Millisecond)
	defer SetCacheTTLs(time.Minute, 5*time.Minute, 30*time.Second)

	repoPath := testutil.CreateTestRepoWithWorktrees(t)

	// Populate cache
	worktrees1, err := ListWorktreesCached(repoPath, false)
	if err != nil {
		t.Fatalf("ListWorktreesCached failed: %v", err)
	}

	// Wait for cache to expire
	time.Sleep(100 * time.Millisecond)

	// Next call should re-fetch (cache expired)
	worktrees2, err := ListWorktreesCached(repoPath, false)
	if err != nil {
		t.Fatalf("ListWorktreesCached after expiration failed: %v", err)
	}

	// Results should still be the same
	if len(worktrees2) != len(worktrees1) {
		t.Errorf("Result after expiration has different length: %d vs %d", len(worktrees2), len(worktrees1))
	}
}

func TestCacheWithAddWorktree(t *testing.T) {
	SetCacheEnabled(true)
	defer SetCacheEnabled(false)

	repoPath := testutil.CreateTestRepo(t)

	// Populate cache
	worktrees1, err := ListWorktreesCached(repoPath, false)
	if err != nil {
		t.Fatalf("ListWorktreesCached failed: %v", err)
	}

	initialCount := len(worktrees1)

	// Add a new worktree (should invalidate cache)
	_, err = AddWorktree(repoPath, AddWorktreeOptions{
		Path:      repoPath + "-new-wt",
		Branch:    "test-branch",
		NewBranch: true,
	})
	if err != nil {
		t.Fatalf("AddWorktree failed: %v", err)
	}

	// Next call should see the new worktree
	worktrees2, err := ListWorktreesCached(repoPath, false)
	if err != nil {
		t.Fatalf("ListWorktreesCached after adding worktree failed: %v", err)
	}

	// Should have one more worktree
	if len(worktrees2) != initialCount+1 {
		t.Errorf("Expected %d worktrees after adding, got %d", initialCount+1, len(worktrees2))
	}
}

func TestCacheWithRemoveWorktree(t *testing.T) {
	SetCacheEnabled(true)
	defer SetCacheEnabled(false)

	repoPath, worktreePath := testutil.CreateTestRepoWithWorktreesAndPath(t)

	// Populate cache
	worktrees1, err := ListWorktreesCached(repoPath, false)
	if err != nil {
		t.Fatalf("ListWorktreesCached failed: %v", err)
	}

	initialCount := len(worktrees1)

	// Remove a worktree (should invalidate cache)
	err = RemoveWorktree(repoPath, RemoveWorktreeOptions{
		Path:  worktreePath,
		Force: true,
	})
	if err != nil {
		t.Fatalf("RemoveWorktree failed: %v", err)
	}

	// Next call should reflect the removal
	worktrees2, err := ListWorktreesCached(repoPath, false)
	if err != nil {
		t.Fatalf("ListWorktreesCached after removing worktree failed: %v", err)
	}

	// Should have one fewer worktree
	if len(worktrees2) != initialCount-1 {
		t.Errorf("Expected %d worktrees after removing, got %d", initialCount-1, len(worktrees2))
	}
}
