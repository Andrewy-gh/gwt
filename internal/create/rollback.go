package create

import (
	"fmt"
	"os"

	"github.com/Andrewy-gh/gwt/internal/git"
	"github.com/Andrewy-gh/gwt/internal/output"
)

// Rollback tracks resources created during worktree creation
// and can clean them up if the operation fails
type Rollback struct {
	repoPath         string
	worktreePath     string
	branchCreated    string
	directoryCreated bool
	enabled          bool
}

// NewRollback creates a new rollback tracker
func NewRollback(repoPath string) *Rollback {
	return &Rollback{
		repoPath: repoPath,
		enabled:  true,
	}
}

// TrackWorktree records that a worktree was created
func (r *Rollback) TrackWorktree(path string) {
	if r.enabled {
		r.worktreePath = path
	}
}

// TrackBranch records that a branch was created
func (r *Rollback) TrackBranch(name string) {
	if r.enabled {
		r.branchCreated = name
	}
}

// TrackDirectory records that a directory was created
func (r *Rollback) TrackDirectory(path string) {
	if r.enabled {
		r.directoryCreated = true
		// The path should match worktreePath, but we don't need to store it separately
	}
}

// Execute performs the rollback, cleaning up all tracked resources
// Order: worktree -> branch -> directory
func (r *Rollback) Execute() error {
	if !r.enabled {
		return nil
	}

	var errors []error

	// 1. Remove worktree (if created)
	if r.worktreePath != "" {
		output.Verbose(fmt.Sprintf("Rolling back: removing worktree %s", r.worktreePath))
		if err := r.removeWorktree(); err != nil {
			errors = append(errors, fmt.Errorf("failed to remove worktree: %w", err))
		}
	}

	// 2. Delete branch (if we created it)
	if r.branchCreated != "" {
		output.Verbose(fmt.Sprintf("Rolling back: deleting branch %s", r.branchCreated))
		if err := r.deleteBranch(); err != nil {
			errors = append(errors, fmt.Errorf("failed to delete branch: %w", err))
		}
	}

	// 3. Remove directory (if we created it)
	if r.directoryCreated && r.worktreePath != "" {
		output.Verbose(fmt.Sprintf("Rolling back: removing directory %s", r.worktreePath))
		if err := r.removeDirectory(); err != nil {
			errors = append(errors, fmt.Errorf("failed to remove directory: %w", err))
		}
	}

	// Return combined errors if any
	if len(errors) > 0 {
		return fmt.Errorf("rollback completed with errors: %v", errors)
	}

	return nil
}

// Clear clears the rollback tracker (call on success to prevent rollback)
func (r *Rollback) Clear() {
	r.enabled = false
	r.worktreePath = ""
	r.branchCreated = ""
	r.directoryCreated = false
}

// removeWorktree removes the worktree using git worktree remove
func (r *Rollback) removeWorktree() error {
	// Use force to ensure removal even if there are changes
	err := git.RemoveWorktree(r.repoPath, git.RemoveWorktreeOptions{
		Path:  r.worktreePath,
		Force: true,
	})

	if err != nil {
		// If git worktree remove fails, the directory might still exist
		// We'll try to remove it manually in removeDirectory
		output.Verbose(fmt.Sprintf("git worktree remove failed: %v", err))
		return err
	}

	return nil
}

// deleteBranch deletes the branch we created
func (r *Rollback) deleteBranch() error {
	// Use force delete since the branch might not be merged
	err := git.DeleteBranch(r.repoPath, git.DeleteBranchOptions{
		Name:  r.branchCreated,
		Force: true,
	})

	if err != nil {
		output.Verbose(fmt.Sprintf("git branch delete failed: %v", err))
		return err
	}

	return nil
}

// removeDirectory removes the directory we created
func (r *Rollback) removeDirectory() error {
	// Check if directory still exists
	if _, err := os.Stat(r.worktreePath); os.IsNotExist(err) {
		// Already removed (probably by git worktree remove)
		return nil
	}

	// Try to remove the directory
	err := os.RemoveAll(r.worktreePath)
	if err != nil {
		output.Verbose(fmt.Sprintf("directory removal failed: %v", err))
		return err
	}

	return nil
}

// IsEnabled returns whether rollback is enabled
func (r *Rollback) IsEnabled() bool {
	return r.enabled
}

// HasChanges returns whether any resources were tracked
func (r *Rollback) HasChanges() bool {
	return r.worktreePath != "" || r.branchCreated != "" || r.directoryCreated
}

// String returns a string representation of what will be rolled back
func (r *Rollback) String() string {
	if !r.enabled {
		return "rollback disabled"
	}

	var items []string
	if r.worktreePath != "" {
		items = append(items, fmt.Sprintf("worktree:%s", r.worktreePath))
	}
	if r.branchCreated != "" {
		items = append(items, fmt.Sprintf("branch:%s", r.branchCreated))
	}
	if r.directoryCreated {
		items = append(items, "directory")
	}

	if len(items) == 0 {
		return "rollback (no changes)"
	}

	return fmt.Sprintf("rollback (%v)", items)
}
