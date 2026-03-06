package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Andrewy-gh/gwt/internal/testutil"
)

// createBenchmarkRepo creates a test repository with specified number of worktrees and branches
// Returns the main repository path
func createBenchmarkRepo(b *testing.B, numWorktrees, numBranches int) string {
	b.Helper()

	// Create main repo
	repoPath := testutil.CreateTestRepo(&testing.T{})

	// Create branches
	for i := 0; i < numBranches; i++ {
		branchName := fmt.Sprintf("branch-%d", i)
		cmd := exec.Command("git", "branch", branchName)
		cmd.Dir = repoPath
		if err := cmd.Run(); err != nil {
			b.Fatalf("failed to create branch %s: %v", branchName, err)
		}

		// Add a commit to each branch to give it a commit date
		cmd = exec.Command("git", "checkout", branchName)
		cmd.Dir = repoPath
		if err := cmd.Run(); err != nil {
			b.Fatalf("failed to checkout branch %s: %v", branchName, err)
		}

		// Create a unique file for this branch
		filename := fmt.Sprintf("file-%d.txt", i)
		filePath := filepath.Join(repoPath, filename)
		if err := os.WriteFile(filePath, []byte(fmt.Sprintf("content %d\n", i)), 0644); err != nil {
			b.Fatalf("failed to write file: %v", err)
		}

		cmd = exec.Command("git", "add", filename)
		cmd.Dir = repoPath
		if err := cmd.Run(); err != nil {
			b.Fatalf("failed to add file: %v", err)
		}

		cmd = exec.Command("git", "commit", "-m", fmt.Sprintf("Commit for %s", branchName))
		cmd.Dir = repoPath
		if err := cmd.Run(); err != nil {
			b.Fatalf("failed to commit: %v", err)
		}
	}

	// Return to main branch
	cmd := exec.Command("git", "checkout", "main")
	cmd.Dir = repoPath
	_ = cmd.Run() // Ignore error if main doesn't exist

	// Create worktrees (up to the number of branches available)
	actualWorktrees := numWorktrees
	if actualWorktrees > numBranches {
		actualWorktrees = numBranches
	}

	for i := 0; i < actualWorktrees; i++ {
		branchName := fmt.Sprintf("branch-%d", i)
		worktreePath := filepath.Join(filepath.Dir(repoPath), fmt.Sprintf("worktree-%d", i))

		cmd := exec.Command("git", "worktree", "add", worktreePath, branchName)
		cmd.Dir = repoPath
		if err := cmd.Run(); err != nil {
			b.Fatalf("failed to create worktree %d: %v", i, err)
		}
	}

	// Register cleanup
	b.Cleanup(func() {
		// Remove worktrees
		for i := 0; i < actualWorktrees; i++ {
			worktreePath := filepath.Join(filepath.Dir(repoPath), fmt.Sprintf("worktree-%d", i))
			os.RemoveAll(worktreePath)
		}
		// Remove main repo
		os.RemoveAll(repoPath)
	})

	return repoPath
}

// BenchmarkListWorktrees benchmarks the worktree listing operation
func BenchmarkListWorktrees(b *testing.B) {
	testCases := []struct {
		name      string
		worktrees int
		branches  int
	}{
		{"Small_5wt_10br", 5, 10},
		{"Medium_20wt_50br", 20, 50},
		{"Large_50wt_100br", 50, 100},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			repoPath := createBenchmarkRepo(b, tc.worktrees, tc.branches)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := ListWorktrees(repoPath)
				if err != nil {
					b.Fatalf("ListWorktrees failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkGetWorktreeStatus benchmarks the status fetching operation
func BenchmarkGetWorktreeStatus(b *testing.B) {
	repoPath := testutil.CreateTestRepoWithWorktrees(&testing.T{})

	worktrees, err := ListWorktrees(repoPath)
	if err != nil {
		b.Fatalf("Failed to list worktrees: %v", err)
	}

	if len(worktrees) == 0 {
		b.Fatal("No worktrees found")
	}

	// Get the first non-main worktree
	var testWorktree *Worktree
	for i := range worktrees {
		if !worktrees[i].IsMain {
			testWorktree = &worktrees[i]
			break
		}
	}

	if testWorktree == nil {
		b.Fatal("No non-main worktree found")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := GetWorktreeStatus(testWorktree.Path)
		if err != nil {
			b.Fatalf("GetWorktreeStatus failed: %v", err)
		}
	}
}

// BenchmarkListBranches benchmarks the branch listing operation
func BenchmarkListBranches(b *testing.B) {
	testCases := []struct {
		name     string
		branches int
	}{
		{"Small_10br", 10},
		{"Medium_100br", 100},
		{"Large_1000br", 1000},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			// Create a repo with 0 worktrees but many branches
			repoPath := createBenchmarkRepo(b, 0, tc.branches)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := ListLocalBranches(repoPath)
				if err != nil {
					b.Fatalf("ListLocalBranches failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkGetBranchLastCommitDate benchmarks getting commit date for a single branch
func BenchmarkGetBranchLastCommitDate(b *testing.B) {
	repoPath := createBenchmarkRepo(b, 0, 10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := GetBranchLastCommitDate(repoPath, "branch-0")
		if err != nil {
			b.Fatalf("GetBranchLastCommitDate failed: %v", err)
		}
	}
}

// BenchmarkGetStaleBranches benchmarks finding stale branches
func BenchmarkGetStaleBranches(b *testing.B) {
	testCases := []struct {
		name     string
		branches int
	}{
		{"Small_10br", 10},
		{"Medium_100br", 100},
		{"Large_500br", 500},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			repoPath := createBenchmarkRepo(b, 0, tc.branches)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := GetStaleBranches(repoPath, 365) // 1 year
				if err != nil {
					b.Fatalf("GetStaleBranches failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkWorktreeOperations benchmarks a full cycle of worktree operations
func BenchmarkWorktreeOperations(b *testing.B) {
	b.Run("CreateListDelete", func(b *testing.B) {
		repoPath := testutil.CreateTestRepo(&testing.T{})
		defer os.RemoveAll(repoPath)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			branchName := fmt.Sprintf("bench-branch-%d", i)
			worktreePath := filepath.Join(filepath.Dir(repoPath), fmt.Sprintf("bench-wt-%d", i))

			// Create branch
			cmd := exec.Command("git", "branch", branchName)
			cmd.Dir = repoPath
			_ = cmd.Run()

			b.StartTimer()

			// Add worktree
			_, err := AddWorktree(repoPath, AddWorktreeOptions{
				Path:   worktreePath,
				Branch: branchName,
			})
			if err != nil {
				b.Fatalf("AddWorktree failed: %v", err)
			}

			// List worktrees
			_, err = ListWorktrees(repoPath)
			if err != nil {
				b.Fatalf("ListWorktrees failed: %v", err)
			}

			// Remove worktree
			err = RemoveWorktree(repoPath, RemoveWorktreeOptions{
				Path:  worktreePath,
				Force: false,
			})
			if err != nil {
				b.Fatalf("RemoveWorktree failed: %v", err)
			}

			b.StopTimer()
			os.RemoveAll(worktreePath)
		}
	})
}

// BenchmarkConcurrentStatusFetch benchmarks concurrent status fetching
// This will be useful for comparing with batch operations
func BenchmarkConcurrentStatusFetch(b *testing.B) {
	testCases := []struct {
		name      string
		worktrees int
	}{
		{"5wt", 5},
		{"20wt", 20},
		{"50wt", 50},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			repoPath := createBenchmarkRepo(b, tc.worktrees, tc.worktrees+10)

			worktrees, err := ListWorktrees(repoPath)
			if err != nil {
				b.Fatalf("Failed to list worktrees: %v", err)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Sequential status fetching
				for j := range worktrees {
					_, err := GetWorktreeStatus(worktrees[j].Path)
					if err != nil {
						b.Fatalf("GetWorktreeStatus failed: %v", err)
					}
				}
			}
		})
	}
}
