package create

import (
	"fmt"
	"strings"

	"github.com/Andrewy-gh/gwt/internal/git"
)

// CreateOptions holds options for worktree creation
// This is duplicated from cli package to avoid import cycle
type CreateOptions struct {
	Branch   string
	From     string
	Checkout string
	Remote   string
}

// BranchSource represents the type of branch operation
type BranchSource int

const (
	BranchSourceNewFromHEAD   BranchSource = iota // New branch from current HEAD
	BranchSourceNewFromRef                        // New branch from specific ref
	BranchSourceLocalExisting                     // Existing local branch
	BranchSourceRemote                            // Remote branch (create tracking)
)

// BranchSpec contains the specification for worktree creation
type BranchSpec struct {
	Source       BranchSource // Type of branch operation
	BranchName   string       // Target branch name
	StartPoint   string       // Starting point for new branch (commit, branch, tag)
	RemoteName   string       // Remote name (for remote branches)
	RemoteBranch string       // Full remote branch name (origin/feature)
}

// ParseBranchSpec parses create options into a BranchSpec
func ParseBranchSpec(opts CreateOptions) (*BranchSpec, error) {
	spec := &BranchSpec{}

	// Determine branch source type
	switch {
	case opts.Branch != "":
		// New branch creation
		spec.BranchName = opts.Branch
		spec.StartPoint = opts.From

		if opts.From != "" {
			spec.Source = BranchSourceNewFromRef
		} else {
			spec.Source = BranchSourceNewFromHEAD
		}

		// Validate branch name
		if err := ValidateBranchName(spec.BranchName); err != nil {
			return nil, fmt.Errorf("invalid branch name: %w", err)
		}

	case opts.Checkout != "":
		// Existing local branch
		spec.Source = BranchSourceLocalExisting
		spec.BranchName = opts.Checkout

	case opts.Remote != "":
		// Remote branch
		spec.Source = BranchSourceRemote
		spec.RemoteBranch = opts.Remote

		// Parse remote name and branch name
		// Format: origin/feature or refs/remotes/origin/feature
		remoteBranch := strings.TrimPrefix(opts.Remote, "refs/remotes/")

		parts := strings.SplitN(remoteBranch, "/", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid remote branch format '%s': expected 'remote/branch'", opts.Remote)
		}

		spec.RemoteName = parts[0]
		spec.BranchName = parts[1] // Local tracking branch name

	default:
		return nil, fmt.Errorf("no branch source specified")
	}

	return spec, nil
}

// ValidateBranchSpec validates the branch spec before creation
// Checks if branch exists, if remote branch exists, etc.
func ValidateBranchSpec(repoPath string, spec *BranchSpec) error {
	switch spec.Source {
	case BranchSourceNewFromHEAD, BranchSourceNewFromRef:
		// Check if branch already exists locally
		exists, err := git.LocalBranchExists(repoPath, spec.BranchName)
		if err != nil {
			return fmt.Errorf("failed to check if branch exists: %w", err)
		}
		if exists {
			return fmt.Errorf("branch '%s' already exists", spec.BranchName)
		}

		// If creating from specific ref, validate it exists
		if spec.Source == BranchSourceNewFromRef && spec.StartPoint != "" {
			if err := git.ValidateRef(repoPath, spec.StartPoint); err != nil {
				return fmt.Errorf("invalid starting point '%s': %w", spec.StartPoint, err)
			}
		}

	case BranchSourceLocalExisting:
		// Check branch exists
		exists, err := git.LocalBranchExists(repoPath, spec.BranchName)
		if err != nil {
			return fmt.Errorf("failed to check if branch exists: %w", err)
		}
		if !exists {
			return fmt.Errorf("branch '%s' does not exist", spec.BranchName)
		}

		// Check branch not already checked out in a worktree
		wt, err := git.FindWorktreeByBranch(repoPath, spec.BranchName)
		if err != nil {
			return fmt.Errorf("failed to check worktrees: %w", err)
		}
		if wt != nil {
			return fmt.Errorf("branch '%s' is already checked out in %s", spec.BranchName, wt.Path)
		}

	case BranchSourceRemote:
		// Check remote branch exists
		exists, err := git.RemoteBranchExists(repoPath, spec.RemoteBranch)
		if err != nil {
			return fmt.Errorf("failed to check if remote branch exists: %w", err)
		}
		if !exists {
			return fmt.Errorf("remote branch '%s' does not exist", spec.RemoteBranch)
		}

		// Check if local branch with same name exists
		localExists, err := git.LocalBranchExists(repoPath, spec.BranchName)
		if err != nil {
			return fmt.Errorf("failed to check local branches: %w", err)
		}

		if localExists {
			// Check if it tracks the same remote
			localBranch, err := git.GetBranch(repoPath, spec.BranchName)
			if err != nil {
				return fmt.Errorf("failed to get local branch info: %w", err)
			}

			// If the local branch has a different upstream, warn the user
			if localBranch.Upstream != "" && localBranch.Upstream != spec.RemoteBranch {
				return fmt.Errorf("local branch '%s' exists but tracks '%s' instead of '%s'",
					spec.BranchName, localBranch.Upstream, spec.RemoteBranch)
			}

			// If local branch exists and tracks the correct remote, we can use it
			// Check it's not already checked out in a worktree
			wt, err := git.FindWorktreeByBranch(repoPath, spec.BranchName)
			if err != nil {
				return fmt.Errorf("failed to check worktrees: %w", err)
			}
			if wt != nil {
				return fmt.Errorf("branch '%s' is already checked out in %s", spec.BranchName, wt.Path)
			}
		}
	}

	return nil
}

// ResolveBranchName resolves the actual branch name for worktree creation
// For remote branches: origin/feature -> feature (local tracking branch)
func ResolveBranchName(spec *BranchSpec) string {
	return spec.BranchName
}

// GetSourceDescription returns a human-readable description of the branch source
func GetSourceDescription(spec *BranchSpec) string {
	switch spec.Source {
	case BranchSourceNewFromHEAD:
		return fmt.Sprintf("new branch '%s' from HEAD", spec.BranchName)
	case BranchSourceNewFromRef:
		return fmt.Sprintf("new branch '%s' from %s", spec.BranchName, spec.StartPoint)
	case BranchSourceLocalExisting:
		return fmt.Sprintf("existing branch '%s'", spec.BranchName)
	case BranchSourceRemote:
		return fmt.Sprintf("remote branch '%s' (tracking as '%s')", spec.RemoteBranch, spec.BranchName)
	default:
		return "unknown"
	}
}
