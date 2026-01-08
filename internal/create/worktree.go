package create

import (
	"fmt"

	"github.com/Andrewy-gh/gwt/internal/git"
)

// CreateWorktreeResult contains information about the created worktree
type CreateWorktreeResult struct {
	Path    string // Absolute path to the worktree
	Branch  string // Branch name
	Commit  string // HEAD commit SHA (short)
	IsNew   bool   // True if a new branch was created
	FromRef string // Source ref (for new branches)
}

// CreateWorktree creates a new worktree based on the provided spec
func CreateWorktree(repoPath string, spec *BranchSpec, targetDir string) (*CreateWorktreeResult, error) {
	switch spec.Source {
	case BranchSourceNewFromHEAD:
		return createNewBranchWorktree(repoPath, spec, targetDir)

	case BranchSourceNewFromRef:
		return createNewBranchWorktree(repoPath, spec, targetDir)

	case BranchSourceLocalExisting:
		return createExistingBranchWorktree(repoPath, spec, targetDir)

	case BranchSourceRemote:
		return createRemoteBranchWorktree(repoPath, spec, targetDir)

	default:
		return nil, fmt.Errorf("unknown branch source type")
	}
}

// createNewBranchWorktree creates worktree with a new branch
func createNewBranchWorktree(repoPath string, spec *BranchSpec, targetDir string) (*CreateWorktreeResult, error) {
	// Determine start point
	startPoint := spec.StartPoint
	if startPoint == "" {
		startPoint = "HEAD"
	}

	// Create the worktree with new branch
	wt, err := git.AddWorktreeForNewBranch(repoPath, targetDir, spec.BranchName, startPoint)
	if err != nil {
		return nil, fmt.Errorf("failed to create worktree: %w", err)
	}

	return &CreateWorktreeResult{
		Path:    wt.Path,
		Branch:  wt.Branch,
		Commit:  wt.Commit,
		IsNew:   true,
		FromRef: startPoint,
	}, nil
}

// createExistingBranchWorktree creates worktree for existing local branch
func createExistingBranchWorktree(repoPath string, spec *BranchSpec, targetDir string) (*CreateWorktreeResult, error) {
	// Create the worktree with existing branch
	wt, err := git.AddWorktreeForExistingBranch(repoPath, targetDir, spec.BranchName)
	if err != nil {
		return nil, fmt.Errorf("failed to create worktree: %w", err)
	}

	return &CreateWorktreeResult{
		Path:   wt.Path,
		Branch: wt.Branch,
		Commit: wt.Commit,
		IsNew:  false,
	}, nil
}

// createRemoteBranchWorktree creates worktree tracking a remote branch
func createRemoteBranchWorktree(repoPath string, spec *BranchSpec, targetDir string) (*CreateWorktreeResult, error) {
	// Create the worktree with remote tracking branch
	wt, err := git.AddWorktreeForRemoteBranch(repoPath, targetDir, spec.RemoteBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to create worktree: %w", err)
	}

	return &CreateWorktreeResult{
		Path:    wt.Path,
		Branch:  wt.Branch,
		Commit:  wt.Commit,
		IsNew:   true,
		FromRef: spec.RemoteBranch,
	}, nil
}

// GetWorktreeInfo returns information about a worktree for display
func GetWorktreeInfo(result *CreateWorktreeResult) map[string]string {
	info := map[string]string{
		"Path":   result.Path,
		"Branch": result.Branch,
		"Commit": result.Commit,
	}

	if result.IsNew {
		info["Source"] = result.FromRef
		info["Status"] = "New branch created"
	} else {
		info["Status"] = "Existing branch"
	}

	return info
}
