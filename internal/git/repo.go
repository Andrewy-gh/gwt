package git

import (
	"fmt"
	"path/filepath"
	"strings"
)

// IsGitRepository checks if the given path is inside a git repository
func IsGitRepository(path string) bool {
	result, err := RunInDir(path, "rev-parse", "--git-dir")
	return err == nil && result.Success()
}

// IsWorktree checks if the given path is a git worktree (not main working tree)
// This works by comparing --git-common-dir and --git-dir
// In a linked worktree, these will be different
func IsWorktree(path string) (bool, error) {
	// Get the git-common-dir (points to main .git)
	commonDirResult, err := RunInDir(path, "rev-parse", "--git-common-dir")
	if err != nil {
		return false, err
	}

	// Get the git-dir (points to worktree's .git)
	gitDirResult, err := RunInDir(path, "rev-parse", "--git-dir")
	if err != nil {
		return false, err
	}

	commonDir := filepath.Clean(strings.TrimSpace(commonDirResult.Stdout))
	gitDir := filepath.Clean(strings.TrimSpace(gitDirResult.Stdout))

	// If they're different, we're in a linked worktree
	return commonDir != gitDir, nil
}

// GetRepoRoot returns the root of the git repository
func GetRepoRoot(path string) (string, error) {
	result, err := RunInDir(path, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", &NotARepoError{Path: path}
	}

	root := strings.TrimSpace(result.Stdout)
	// Convert to native path separators
	return filepath.FromSlash(root), nil
}

// GetGitDir returns the path to the .git directory
func GetGitDir(path string) (string, error) {
	result, err := RunInDir(path, "rev-parse", "--git-dir")
	if err != nil {
		return "", &NotARepoError{Path: path}
	}

	gitDir := strings.TrimSpace(result.Stdout)

	// If it's a relative path, make it absolute
	if !filepath.IsAbs(gitDir) {
		root, err := GetRepoRoot(path)
		if err != nil {
			return "", err
		}
		gitDir = filepath.Join(root, gitDir)
	}

	return filepath.FromSlash(gitDir), nil
}

// GetMainWorktreePath returns the path to the main worktree
func GetMainWorktreePath(path string) (string, error) {
	// Use git worktree list --porcelain and parse the first entry
	result, err := RunInDir(path, "worktree", "list", "--porcelain")
	if err != nil {
		return "", err
	}

	// Parse porcelain output - first "worktree" line is the main worktree
	lines := result.Lines()
	for _, line := range lines {
		if strings.HasPrefix(line, "worktree ") {
			mainPath := strings.TrimPrefix(line, "worktree ")
			return filepath.FromSlash(mainPath), nil
		}
	}

	return "", fmt.Errorf("failed to determine main worktree path")
}

// GetCurrentBranch returns the current branch name (or empty string if detached)
func GetCurrentBranch(path string) (string, error) {
	// Try to get the symbolic ref for HEAD
	result, err := RunWithOptions(RunOptions{
		Dir:          path,
		Args:         []string{"symbolic-ref", "--short", "HEAD"},
		AllowFailure: true,
	})

	// If successful, return the branch name
	if err == nil && result.Success() {
		return strings.TrimSpace(result.Stdout), nil
	}

	// If we're in detached HEAD state, symbolic-ref will fail
	// Return empty string to indicate detached HEAD
	return "", nil
}

// IsInsideWorktree checks if currently inside a worktree
func IsInsideWorktree(path string) (bool, error) {
	result, err := RunWithOptions(RunOptions{
		Dir:          path,
		Args:         []string{"rev-parse", "--is-inside-work-tree"},
		AllowFailure: true,
	})

	if err != nil {
		return false, nil
	}

	return strings.TrimSpace(result.Stdout) == "true", nil
}

// GetCommonDir returns the path to the common git directory
// For linked worktrees, this points to the main repository's .git
// For the main worktree, this is the same as GetGitDir
func GetCommonDir(path string) (string, error) {
	result, err := RunInDir(path, "rev-parse", "--git-common-dir")
	if err != nil {
		return "", &NotARepoError{Path: path}
	}

	commonDir := strings.TrimSpace(result.Stdout)

	// If it's a relative path, make it absolute
	if !filepath.IsAbs(commonDir) {
		root, err := GetRepoRoot(path)
		if err != nil {
			return "", err
		}
		commonDir = filepath.Join(root, commonDir)
	}

	return filepath.FromSlash(commonDir), nil
}

// ValidateRepository validates that the given path is a git repository
// and returns an error with helpful context if not
func ValidateRepository(path string) error {
	if !IsGitRepository(path) {
		return &NotARepoError{Path: path}
	}
	return nil
}

// ValidateNotBare validates that the repository is not bare
func ValidateNotBare(path string) error {
	// First validate it's a repository
	if err := ValidateRepository(path); err != nil {
		return err
	}

	// Check if bare using the existing function
	result, err := RunInDir(path, "rev-parse", "--is-bare-repository")
	if err != nil {
		return err
	}

	if strings.TrimSpace(result.Stdout) == "true" {
		return fmt.Errorf("operation not supported in bare repository")
	}
	return nil
}

// ValidateRef checks if a ref (branch, tag, commit) exists
func ValidateRef(repoPath, ref string) error {
	result, err := RunWithOptions(RunOptions{
		Dir:          repoPath,
		Args:         []string{"rev-parse", "--verify", ref},
		AllowFailure: true,
	})

	if err != nil || !result.Success() {
		return fmt.Errorf("ref '%s' not found", ref)
	}

	return nil
}

// GetMainWorktree returns the main (non-linked) worktree
func GetMainWorktree(repoPath string) (*Worktree, error) {
	worktrees, err := ListWorktrees(repoPath)
	if err != nil {
		return nil, err
	}

	for _, wt := range worktrees {
		if wt.IsMain {
			return &wt, nil
		}
	}

	return nil, fmt.Errorf("main worktree not found")
}
