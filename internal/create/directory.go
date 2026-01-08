package create

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Andrewy-gh/gwt/internal/git"
)

// DirectoryExistsError indicates the target directory already exists
type DirectoryExistsError struct {
	Path         string // The path that exists
	IsWorktree   bool   // True if it's already a worktree
	IsEmpty      bool   // True if the directory is empty
	SuggestedAlt string // Alternative directory suggestion
}

// Error implements the error interface
func (e *DirectoryExistsError) Error() string {
	if e.IsWorktree {
		return fmt.Sprintf("directory '%s' is already a git worktree", e.Path)
	}
	if e.IsEmpty {
		return fmt.Sprintf("directory '%s' already exists but is empty", e.Path)
	}
	return fmt.Sprintf("directory '%s' already exists and is not empty", e.Path)
}

// CheckDirectory checks if the target directory is available
// Returns nil if available, DirectoryExistsError if not
func CheckDirectory(path string) error {
	// Check if directory exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Directory doesn't exist - good!
			return nil
		}
		// Some other error occurred
		return fmt.Errorf("failed to check directory: %w", err)
	}

	// Directory exists - check what it is
	if !info.IsDir() {
		return fmt.Errorf("path '%s' exists but is not a directory", path)
	}

	// Check if it's an existing worktree
	isWorktree, err := IsExistingWorktree(path)
	if err != nil {
		return fmt.Errorf("failed to check if path is a worktree: %w", err)
	}

	if isWorktree {
		return &DirectoryExistsError{
			Path:         path,
			IsWorktree:   true,
			SuggestedAlt: SuggestAlternativeDirectory(path),
		}
	}

	// Check if directory is empty
	isEmpty, err := IsEmptyDirectory(path)
	if err != nil {
		return fmt.Errorf("failed to check if directory is empty: %w", err)
	}

	return &DirectoryExistsError{
		Path:         path,
		IsEmpty:      isEmpty,
		IsWorktree:   false,
		SuggestedAlt: SuggestAlternativeDirectory(path),
	}
}

// IsEmptyDirectory checks if a directory exists and is empty
func IsEmptyDirectory(path string) (bool, error) {
	// Open directory
	dir, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	defer dir.Close()

	// Try to read one entry
	_, err = dir.Readdirnames(1)
	if err != nil {
		// If EOF, directory is empty
		if err.Error() == "EOF" || strings.Contains(err.Error(), "EOF") {
			return true, nil
		}
		return false, err
	}

	// Successfully read an entry, so directory is not empty
	return false, nil
}

// IsExistingWorktree checks if a path is an existing git worktree
func IsExistingWorktree(path string) (bool, error) {
	// Check if it's a git repository
	if !git.IsGitRepository(path) {
		return false, nil
	}

	// If it's a git repository, it could be:
	// 1. A main worktree (.git directory)
	// 2. A linked worktree (.git file pointing to main repo)
	// 3. A separate repository entirely

	// Check if .git exists
	gitPath := filepath.Join(path, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	// If .git is a directory, it's a main worktree or separate repo
	if info.IsDir() {
		// Check if it's registered as a worktree
		// Try to get worktree info - if it succeeds, it's a worktree
		_, err := git.GetWorktree(path)
		return err == nil, nil
	}

	// If .git is a file, it's a linked worktree
	return true, nil
}

// SuggestAlternativeDirectory generates an alternative directory name
// Appends -2, -3, etc. until finding an available name
func SuggestAlternativeDirectory(basePath string) string {
	// Start with -2
	for i := 2; i < 100; i++ {
		candidate := fmt.Sprintf("%s-%d", basePath, i)

		// Check if this path exists
		_, err := os.Stat(candidate)
		if os.IsNotExist(err) {
			// Found an available name
			return filepath.Base(candidate)
		}
	}

	// If we couldn't find one in 100 tries, just suggest -next
	return filepath.Base(basePath) + "-next"
}

// EnsureParentDirectory ensures the parent directory exists
func EnsureParentDirectory(path string) error {
	parent := filepath.Dir(path)

	// Check if parent exists
	info, err := os.Stat(parent)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("parent directory '%s' does not exist", parent)
		}
		return fmt.Errorf("failed to check parent directory: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("parent path '%s' exists but is not a directory", parent)
	}

	return nil
}

// GetDirectoryInfo returns information about a directory
type DirectoryInfo struct {
	Exists     bool   // True if directory exists
	IsDir      bool   // True if path is a directory
	IsEmpty    bool   // True if directory is empty
	IsWorktree bool   // True if directory is a git worktree
	Path       string // Absolute path
}

// GetDirectoryInfo returns detailed information about a directory
func GetDirectoryInfo(path string) (*DirectoryInfo, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	info := &DirectoryInfo{
		Path: absPath,
	}

	// Check if exists
	stat, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return info, nil
		}
		return nil, fmt.Errorf("failed to stat path: %w", err)
	}

	info.Exists = true
	info.IsDir = stat.IsDir()

	if info.IsDir {
		// Check if empty
		isEmpty, err := IsEmptyDirectory(absPath)
		if err != nil {
			return nil, fmt.Errorf("failed to check if empty: %w", err)
		}
		info.IsEmpty = isEmpty

		// Check if worktree
		isWorktree, err := IsExistingWorktree(absPath)
		if err != nil {
			return nil, fmt.Errorf("failed to check if worktree: %w", err)
		}
		info.IsWorktree = isWorktree
	}

	return info, nil
}
