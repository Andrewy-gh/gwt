package git

import (
	"fmt"
	"strings"
)

// GitError represents a git command failure
type GitError struct {
	Command  []string
	Stderr   string
	ExitCode int
}

func (e *GitError) Error() string {
	cmd := strings.Join(e.Command, " ")
	if e.Stderr != "" {
		return fmt.Sprintf("git command failed: %s\nstderr: %s\nexit code: %d", cmd, e.Stderr, e.ExitCode)
	}
	return fmt.Sprintf("git command failed: %s\nexit code: %d", cmd, e.ExitCode)
}

// NotARepoError indicates the current directory is not a git repository
type NotARepoError struct {
	Path string
}

func (e *NotARepoError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("not a git repository: %s", e.Path)
	}
	return "not a git repository"
}

// WorktreeError represents a worktree-specific error
type WorktreeError struct {
	Operation string
	Path      string
	Err       error
}

func (e *WorktreeError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("worktree %s failed for %s: %v", e.Operation, e.Path, e.Err)
	}
	return fmt.Sprintf("worktree %s failed: %v", e.Operation, e.Err)
}

func (e *WorktreeError) Unwrap() error {
	return e.Err
}

// BranchError represents a branch-specific error
type BranchError struct {
	Operation string
	Branch    string
	Err       error
}

func (e *BranchError) Error() string {
	if e.Branch != "" {
		return fmt.Sprintf("branch %s failed for %s: %v", e.Operation, e.Branch, e.Err)
	}
	return fmt.Sprintf("branch %s failed: %v", e.Operation, e.Err)
}

func (e *BranchError) Unwrap() error {
	return e.Err
}

// RemoteError represents a remote-specific error
type RemoteError struct {
	Operation string
	Remote    string
	Err       error
}

func (e *RemoteError) Error() string {
	if e.Remote != "" {
		return fmt.Sprintf("remote %s failed for %s: %v", e.Operation, e.Remote, e.Err)
	}
	return fmt.Sprintf("remote %s failed: %v", e.Operation, e.Err)
}

func (e *RemoteError) Unwrap() error {
	return e.Err
}
