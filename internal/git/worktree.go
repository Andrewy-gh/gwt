package git

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// Worktree represents a git worktree
type Worktree struct {
	Path       string // Absolute path to worktree
	Branch     string // Branch name (empty if detached)
	Commit     string // HEAD commit SHA (short)
	CommitFull string // HEAD commit SHA (full)
	IsMain     bool   // True if this is the main worktree
	IsDetached bool   // True if HEAD is detached
	IsBare     bool   // True if this is a bare worktree entry
	Locked     bool   // True if worktree is locked
	Prunable   bool   // True if worktree can be pruned
}

// WorktreeStatus contains status information for a worktree
type WorktreeStatus struct {
	Clean          bool      // True if no uncommitted changes
	StagedCount    int       // Number of staged changes
	UnstagedCount  int       // Number of unstaged changes
	UntrackedCount int       // Number of untracked files
	LastCommitTime time.Time // Time of last commit
	LastCommitMsg  string    // Last commit message (first line)
	AheadCount     int       // Commits ahead of upstream
	BehindCount    int       // Commits behind upstream
}

// ListWorktrees returns all worktrees for the repository
func ListWorktrees(repoPath string) ([]Worktree, error) {
	// Validate repository
	if err := ValidateRepository(repoPath); err != nil {
		return nil, err
	}

	// Run git worktree list --porcelain
	result, err := RunInDir(repoPath, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, &WorktreeError{
			Operation: "list",
			Err:       err,
		}
	}

	// Parse porcelain output
	worktrees, err := parseWorktreeList(result.Stdout)
	if err != nil {
		return nil, &WorktreeError{
			Operation: "list",
			Err:       err,
		}
	}

	return worktrees, nil
}

// parseWorktreeList parses the output of git worktree list --porcelain
// Format:
// worktree /path/to/worktree
// HEAD abc123...
// branch refs/heads/main
// [bare]
// [detached]
// [locked]
// [prunable]
//
// <blank line between entries>
func parseWorktreeList(output string) ([]Worktree, error) {
	var worktrees []Worktree
	var current *Worktree

	lines := strings.Split(strings.TrimSpace(output), "\n")
	isFirstWorktree := true

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Empty line indicates end of worktree entry
		if line == "" {
			if current != nil {
				worktrees = append(worktrees, *current)
				current = nil
				isFirstWorktree = false
			}
			continue
		}

		// Start new worktree entry
		if strings.HasPrefix(line, "worktree ") {
			if current != nil {
				worktrees = append(worktrees, *current)
				isFirstWorktree = false
			}
			current = &Worktree{
				Path:   filepath.FromSlash(strings.TrimPrefix(line, "worktree ")),
				IsMain: isFirstWorktree,
			}
			continue
		}

		// Parse fields for current worktree
		if current == nil {
			continue
		}

		switch {
		case strings.HasPrefix(line, "HEAD "):
			current.CommitFull = strings.TrimPrefix(line, "HEAD ")
			if len(current.CommitFull) >= 7 {
				current.Commit = current.CommitFull[:7]
			}

		case strings.HasPrefix(line, "branch "):
			branchRef := strings.TrimPrefix(line, "branch ")
			// Extract branch name from refs/heads/branch-name
			if strings.HasPrefix(branchRef, "refs/heads/") {
				current.Branch = strings.TrimPrefix(branchRef, "refs/heads/")
			}

		case line == "bare":
			current.IsBare = true

		case line == "detached":
			current.IsDetached = true

		case line == "locked":
			current.Locked = true

		case line == "prunable":
			current.Prunable = true
		}
	}

	// Don't forget the last worktree
	if current != nil {
		worktrees = append(worktrees, *current)
	}

	return worktrees, nil
}

// GetWorktree returns information about a specific worktree
func GetWorktree(worktreePath string) (*Worktree, error) {
	// Validate that this is a git repository
	if err := ValidateRepository(worktreePath); err != nil {
		return nil, err
	}

	// Get the absolute path
	absPath, err := filepath.Abs(worktreePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}
	absPath = filepath.Clean(absPath)

	// List all worktrees
	worktrees, err := ListWorktrees(worktreePath)
	if err != nil {
		return nil, err
	}

	// Find the matching worktree
	for _, wt := range worktrees {
		wtPath := filepath.Clean(wt.Path)
		if wtPath == absPath {
			return &wt, nil
		}
	}

	return nil, fmt.Errorf("worktree not found: %s", worktreePath)
}

// FindWorktreeByBranch finds a worktree by branch name
func FindWorktreeByBranch(repoPath, branch string) (*Worktree, error) {
	worktrees, err := ListWorktrees(repoPath)
	if err != nil {
		return nil, err
	}

	for _, wt := range worktrees {
		if wt.Branch == branch {
			return &wt, nil
		}
	}

	return nil, nil // Not found, but not an error
}

// AddWorktreeOptions configures worktree creation
type AddWorktreeOptions struct {
	Path       string // Directory path for new worktree
	Branch     string // Branch name to check out
	NewBranch  bool   // Create new branch
	StartPoint string // Starting point for new branch (commit, branch, tag)
	Track      bool   // Set up tracking for remote branch
	Force      bool   // Force creation even if branch exists elsewhere
	Detach     bool   // Create in detached HEAD state
	Lock       bool   // Lock worktree after creation
}

// AddWorktree creates a new worktree
func AddWorktree(repoPath string, opts AddWorktreeOptions) (*Worktree, error) {
	if err := ValidateRepository(repoPath); err != nil {
		return nil, err
	}

	// Validate options
	if opts.Path == "" {
		return nil, &WorktreeError{
			Operation: "add",
			Err:       fmt.Errorf("worktree path is required"),
		}
	}

	// Build command arguments
	args := []string{"worktree", "add"}

	if opts.Force {
		args = append(args, "--force")
	}

	if opts.Detach {
		args = append(args, "--detach")
	}

	if opts.Lock {
		args = append(args, "--lock")
	}

	if opts.NewBranch {
		if opts.Branch == "" {
			return nil, &WorktreeError{
				Operation: "add",
				Err:       fmt.Errorf("branch name is required when creating new branch"),
			}
		}
		args = append(args, "-b", opts.Branch)
	}

	if opts.Track {
		args = append(args, "--track")
	}

	// Add path
	args = append(args, opts.Path)

	// Add branch or start point
	if !opts.NewBranch && opts.Branch != "" {
		args = append(args, opts.Branch)
	} else if opts.StartPoint != "" {
		args = append(args, opts.StartPoint)
	}

	// Execute command
	_, err := RunInDir(repoPath, args...)
	if err != nil {
		return nil, &WorktreeError{
			Operation: "add",
			Path:      opts.Path,
			Err:       err,
		}
	}

	// Invalidate cache after adding worktree
	InvalidateWorktreeCache(repoPath)

	// Get the newly created worktree info
	return GetWorktree(opts.Path)
}

// AddWorktreeForNewBranch creates worktree with a new branch
func AddWorktreeForNewBranch(repoPath, path, branch, startPoint string) (*Worktree, error) {
	opts := AddWorktreeOptions{
		Path:       path,
		Branch:     branch,
		NewBranch:  true,
		StartPoint: startPoint,
	}
	return AddWorktree(repoPath, opts)
}

// AddWorktreeForExistingBranch creates worktree for existing local branch
func AddWorktreeForExistingBranch(repoPath, path, branch string) (*Worktree, error) {
	opts := AddWorktreeOptions{
		Path:   path,
		Branch: branch,
	}
	return AddWorktree(repoPath, opts)
}

// AddWorktreeForRemoteBranch creates worktree tracking a remote branch
func AddWorktreeForRemoteBranch(repoPath, path, remoteBranch string) (*Worktree, error) {
	// Extract local branch name from remote branch (origin/feature -> feature)
	parts := strings.SplitN(remoteBranch, "/", 2)
	var localBranch string
	if len(parts) == 2 {
		localBranch = parts[1]
	} else {
		localBranch = remoteBranch
	}

	opts := AddWorktreeOptions{
		Path:       path,
		Branch:     localBranch,
		NewBranch:  true,
		StartPoint: remoteBranch,
		Track:      true,
	}
	return AddWorktree(repoPath, opts)
}

// RemoveWorktreeOptions configures worktree removal
type RemoveWorktreeOptions struct {
	Path  string // Worktree path to remove
	Force bool   // Force removal even with changes
}

// RemoveWorktree removes a worktree
func RemoveWorktree(repoPath string, opts RemoveWorktreeOptions) error {
	if err := ValidateRepository(repoPath); err != nil {
		return err
	}

	if opts.Path == "" {
		return &WorktreeError{
			Operation: "remove",
			Err:       fmt.Errorf("worktree path is required"),
		}
	}

	// Check if this is the main worktree
	wt, err := GetWorktree(opts.Path)
	if err != nil {
		return &WorktreeError{
			Operation: "remove",
			Path:      opts.Path,
			Err:       err,
		}
	}

	if wt.IsMain {
		return &WorktreeError{
			Operation: "remove",
			Path:      opts.Path,
			Err:       fmt.Errorf("cannot remove main worktree"),
		}
	}

	// Build command
	args := []string{"worktree", "remove"}
	if opts.Force {
		args = append(args, "--force")
	}
	args = append(args, opts.Path)

	// Execute command
	_, err = RunInDir(repoPath, args...)
	if err != nil {
		return &WorktreeError{
			Operation: "remove",
			Path:      opts.Path,
			Err:       err,
		}
	}

	// Invalidate cache after removing worktree
	InvalidateWorktreeCache(repoPath)

	return nil
}

// PruneWorktrees removes stale worktree entries
// Returns a list of pruned worktree paths
func PruneWorktrees(repoPath string, dryRun bool) ([]string, error) {
	if err := ValidateRepository(repoPath); err != nil {
		return nil, err
	}

	args := []string{"worktree", "prune"}
	if dryRun {
		args = append(args, "--dry-run", "--verbose")
	}

	result, err := RunInDir(repoPath, args...)
	if err != nil {
		return nil, &WorktreeError{
			Operation: "prune",
			Err:       err,
		}
	}

	// Parse output to get list of pruned worktrees
	var pruned []string
	if dryRun && result.Stdout != "" {
		lines := result.Lines()
		for _, line := range lines {
			// Output format is typically "Removing worktrees/<name>: ..."
			if strings.Contains(line, "Removing") {
				pruned = append(pruned, line)
			}
		}
	}

	return pruned, nil
}

// LockWorktree locks a worktree to prevent pruning
func LockWorktree(worktreePath, reason string) error {
	if err := ValidateRepository(worktreePath); err != nil {
		return err
	}

	args := []string{"worktree", "lock"}
	if reason != "" {
		args = append(args, "--reason", reason)
	}
	args = append(args, worktreePath)

	_, err := Run(args...)
	if err != nil {
		return &WorktreeError{
			Operation: "lock",
			Path:      worktreePath,
			Err:       err,
		}
	}

	return nil
}

// UnlockWorktree unlocks a worktree
func UnlockWorktree(worktreePath string) error {
	if err := ValidateRepository(worktreePath); err != nil {
		return err
	}

	_, err := Run("worktree", "unlock", worktreePath)
	if err != nil {
		return &WorktreeError{
			Operation: "unlock",
			Path:      worktreePath,
			Err:       err,
		}
	}

	return nil
}

// IsWorktreeLocked checks if a worktree is locked
// Returns (locked, reason, error)
func IsWorktreeLocked(worktreePath string) (bool, string, error) {
	wt, err := GetWorktree(worktreePath)
	if err != nil {
		return false, "", err
	}

	// The locked field from porcelain output tells us if it's locked
	// To get the reason, we'd need to read the lock file, but that's internal
	return wt.Locked, "", nil
}
