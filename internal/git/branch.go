package git

import (
	"fmt"
	"strings"
	"time"
)

// Branch represents a git branch
type Branch struct {
	Name       string    // Branch name (without refs/heads/ or refs/remotes/)
	FullRef    string    // Full ref path (refs/heads/main, refs/remotes/origin/main)
	Commit     string    // HEAD commit SHA (short)
	CommitFull string    // HEAD commit SHA (full)
	IsRemote   bool      // True if remote branch
	Remote     string    // Remote name (for remote branches)
	Upstream   string    // Upstream branch (for local branches)
	IsHead     bool      // True if this is current HEAD
	LastCommit time.Time // Time of last commit
}

// ListLocalBranches returns all local branches
func ListLocalBranches(repoPath string) ([]Branch, error) {
	if err := ValidateRepository(repoPath); err != nil {
		return nil, err
	}

	// Use --format for reliable parsing
	// Format: refname objectname upstream HEAD (if current)
	result, err := RunInDir(repoPath, "branch", "--format=%(refname:short)|%(objectname)|%(objectname:short)|%(upstream:short)|%(HEAD)")
	if err != nil {
		return nil, &BranchError{
			Operation: "list",
			Err:       err,
		}
	}

	var branches []Branch
	for _, line := range result.Lines() {
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) < 5 {
			continue
		}

		branch := Branch{
			Name:       parts[0],
			FullRef:    "refs/heads/" + parts[0],
			CommitFull: parts[1],
			Commit:     parts[2],
			Upstream:   parts[3],
			IsHead:     parts[4] == "*",
			IsRemote:   false,
		}

		branches = append(branches, branch)
	}

	return branches, nil
}

// ListRemoteBranches returns all remote branches
func ListRemoteBranches(repoPath string) ([]Branch, error) {
	if err := ValidateRepository(repoPath); err != nil {
		return nil, err
	}

	// Use --format for reliable parsing
	result, err := RunInDir(repoPath, "branch", "-r", "--format=%(refname:short)|%(objectname)|%(objectname:short)")
	if err != nil {
		return nil, &BranchError{
			Operation: "list remote",
			Err:       err,
		}
	}

	var branches []Branch
	for _, line := range result.Lines() {
		if line == "" {
			continue
		}

		// Skip HEAD references (origin/HEAD -> origin/main)
		if strings.Contains(line, "->") {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) < 3 {
			continue
		}

		name := parts[0]
		commitFull := parts[1]
		commit := parts[2]

		// Extract remote name (origin/main -> remote=origin, name=main)
		var remote string
		slashIdx := strings.Index(name, "/")
		if slashIdx > 0 {
			remote = name[:slashIdx]
		}

		branch := Branch{
			Name:       name,
			FullRef:    "refs/remotes/" + name,
			CommitFull: commitFull,
			Commit:     commit,
			IsRemote:   true,
			Remote:     remote,
		}

		branches = append(branches, branch)
	}

	return branches, nil
}

// ListAllBranches returns all local and remote branches
func ListAllBranches(repoPath string) ([]Branch, error) {
	local, err := ListLocalBranches(repoPath)
	if err != nil {
		return nil, err
	}

	remote, err := ListRemoteBranches(repoPath)
	if err != nil {
		return nil, err
	}

	return append(local, remote...), nil
}

// GetBranch returns information about a specific branch
func GetBranch(repoPath, branchName string) (*Branch, error) {
	// Try local branch first
	result, err := RunWithOptions(RunOptions{
		Dir:          repoPath,
		Args:         []string{"show-ref", "--verify", "refs/heads/" + branchName},
		AllowFailure: true,
	})

	if err == nil && result.Success() {
		// Parse: <commit> refs/heads/<name>
		parts := strings.Fields(result.TrimOutput())
		if len(parts) >= 2 {
			commitFull := parts[0]
			commit := commitFull
			if len(commit) > 7 {
				commit = commit[:7]
			}

			// Get upstream info
			upstreamResult, _ := RunWithOptions(RunOptions{
				Dir:          repoPath,
				Args:         []string{"config", "branch." + branchName + ".merge"},
				AllowFailure: true,
			})

			upstream := ""
			if upstreamResult != nil && upstreamResult.Success() {
				upstream = strings.TrimPrefix(upstreamResult.TrimOutput(), "refs/heads/")
			}

			return &Branch{
				Name:       branchName,
				FullRef:    "refs/heads/" + branchName,
				CommitFull: commitFull,
				Commit:     commit,
				Upstream:   upstream,
				IsRemote:   false,
			}, nil
		}
	}

	// Try remote branch
	result, err = RunWithOptions(RunOptions{
		Dir:          repoPath,
		Args:         []string{"show-ref", "--verify", "refs/remotes/" + branchName},
		AllowFailure: true,
	})

	if err == nil && result.Success() {
		parts := strings.Fields(result.TrimOutput())
		if len(parts) >= 2 {
			commitFull := parts[0]
			commit := commitFull
			if len(commit) > 7 {
				commit = commit[:7]
			}

			// Extract remote name
			var remote string
			slashIdx := strings.Index(branchName, "/")
			if slashIdx > 0 {
				remote = branchName[:slashIdx]
			}

			return &Branch{
				Name:       branchName,
				FullRef:    "refs/remotes/" + branchName,
				CommitFull: commitFull,
				Commit:     commit,
				IsRemote:   true,
				Remote:     remote,
			}, nil
		}
	}

	return nil, &BranchError{
		Operation: "get",
		Branch:    branchName,
		Err:       fmt.Errorf("branch not found"),
	}
}

// BranchExists checks if a branch exists (local or remote)
func BranchExists(repoPath, branchName string) (bool, error) {
	_, err := GetBranch(repoPath, branchName)
	if err != nil {
		// If it's a "not found" error, return false without error
		if strings.Contains(err.Error(), "not found") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// LocalBranchExists checks if a local branch exists
func LocalBranchExists(repoPath, branchName string) (bool, error) {
	result, err := RunWithOptions(RunOptions{
		Dir:          repoPath,
		Args:         []string{"show-ref", "--verify", "--quiet", "refs/heads/" + branchName},
		AllowFailure: true,
	})

	if err != nil {
		return false, nil
	}

	return result.Success(), nil
}

// RemoteBranchExists checks if a remote branch exists
func RemoteBranchExists(repoPath, remoteBranch string) (bool, error) {
	result, err := RunWithOptions(RunOptions{
		Dir:          repoPath,
		Args:         []string{"show-ref", "--verify", "--quiet", "refs/remotes/" + remoteBranch},
		AllowFailure: true,
	})

	if err != nil {
		return false, nil
	}

	return result.Success(), nil
}

// CreateBranchOptions configures branch creation
type CreateBranchOptions struct {
	Name       string // Branch name
	StartPoint string // Starting point (commit, branch, tag)
	Track      string // Remote branch to track
	Force      bool   // Force creation (overwrite existing)
}

// CreateBranch creates a new branch
func CreateBranch(repoPath string, opts CreateBranchOptions) (*Branch, error) {
	if err := ValidateRepository(repoPath); err != nil {
		return nil, err
	}

	if opts.Name == "" {
		return nil, &BranchError{
			Operation: "create",
			Err:       fmt.Errorf("branch name is required"),
		}
	}

	// Validate branch name
	if err := validateBranchName(opts.Name); err != nil {
		return nil, &BranchError{
			Operation: "create",
			Branch:    opts.Name,
			Err:       err,
		}
	}

	// Build command
	args := []string{"branch"}

	if opts.Force {
		args = append(args, "--force")
	}

	if opts.Track != "" {
		args = append(args, "--track", opts.Name, opts.Track)
	} else {
		args = append(args, opts.Name)
		if opts.StartPoint != "" {
			args = append(args, opts.StartPoint)
		}
	}

	// Execute command
	_, err := RunInDir(repoPath, args...)
	if err != nil {
		return nil, &BranchError{
			Operation: "create",
			Branch:    opts.Name,
			Err:       err,
		}
	}

	// Get the newly created branch info
	return GetBranch(repoPath, opts.Name)
}

// DeleteBranchOptions configures branch deletion
type DeleteBranchOptions struct {
	Name       string // Branch name
	Force      bool   // Force delete unmerged branch
	Remote     bool   // Delete remote branch too
	RemoteName string // Remote name (default: origin)
}

// DeleteBranch deletes a branch
func DeleteBranch(repoPath string, opts DeleteBranchOptions) error {
	if err := ValidateRepository(repoPath); err != nil {
		return err
	}

	if opts.Name == "" {
		return &BranchError{
			Operation: "delete",
			Err:       fmt.Errorf("branch name is required"),
		}
	}

	// Check if branch is current branch
	currentBranch, err := GetCurrentBranch(repoPath)
	if err != nil {
		return err
	}

	if currentBranch == opts.Name {
		return &BranchError{
			Operation: "delete",
			Branch:    opts.Name,
			Err:       fmt.Errorf("cannot delete current branch"),
		}
	}

	// Delete local branch
	deleteFlag := "-d"
	if opts.Force {
		deleteFlag = "-D"
	}

	_, err = RunInDir(repoPath, "branch", deleteFlag, opts.Name)
	if err != nil {
		return &BranchError{
			Operation: "delete",
			Branch:    opts.Name,
			Err:       err,
		}
	}

	// Delete remote branch if requested
	if opts.Remote {
		remoteName := opts.RemoteName
		if remoteName == "" {
			remoteName = "origin"
		}

		_, err = RunInDir(repoPath, "push", remoteName, "--delete", opts.Name)
		if err != nil {
			// Don't fail the whole operation if remote delete fails
			// Just return a wrapped error
			return &BranchError{
				Operation: "delete remote",
				Branch:    opts.Name,
				Err:       err,
			}
		}
	}

	return nil
}

// RenameBranch renames a branch
func RenameBranch(repoPath, oldName, newName string, force bool) error {
	if err := ValidateRepository(repoPath); err != nil {
		return err
	}

	if oldName == "" || newName == "" {
		return &BranchError{
			Operation: "rename",
			Err:       fmt.Errorf("both old and new branch names are required"),
		}
	}

	// Validate new branch name
	if err := validateBranchName(newName); err != nil {
		return &BranchError{
			Operation: "rename",
			Branch:    newName,
			Err:       err,
		}
	}

	args := []string{"branch", "-m"}
	if force {
		args = append(args, "-f")
	}
	args = append(args, oldName, newName)

	_, err := RunInDir(repoPath, args...)
	if err != nil {
		return &BranchError{
			Operation: "rename",
			Branch:    oldName,
			Err:       err,
		}
	}

	return nil
}

// SetUpstreamBranch sets the upstream tracking branch
func SetUpstreamBranch(repoPath, localBranch, upstream string) error {
	if err := ValidateRepository(repoPath); err != nil {
		return err
	}

	if localBranch == "" || upstream == "" {
		return &BranchError{
			Operation: "set-upstream",
			Err:       fmt.Errorf("both local and upstream branch names are required"),
		}
	}

	_, err := RunInDir(repoPath, "branch", "--set-upstream-to="+upstream, localBranch)
	if err != nil {
		return &BranchError{
			Operation: "set-upstream",
			Branch:    localBranch,
			Err:       err,
		}
	}

	return nil
}

// validateBranchName checks if a branch name is valid
func validateBranchName(name string) error {
	if name == "" {
		return fmt.Errorf("branch name cannot be empty")
	}

	// Git branch name rules:
	// - Cannot contain spaces
	// - Cannot start with a dash
	// - Cannot contain ..
	// - Cannot contain control characters
	// - Cannot end with .lock

	if strings.ContainsAny(name, " \t\n\r") {
		return fmt.Errorf("branch name cannot contain spaces")
	}

	if strings.HasPrefix(name, "-") {
		return fmt.Errorf("branch name cannot start with a dash")
	}

	if strings.Contains(name, "..") {
		return fmt.Errorf("branch name cannot contain '..'")
	}

	if strings.HasSuffix(name, ".lock") {
		return fmt.Errorf("branch name cannot end with '.lock'")
	}

	return nil
}
