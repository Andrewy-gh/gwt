package git

import (
	"fmt"
	"strconv"
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

	// Invalidate branch cache after creating branch
	InvalidateBranchCache(repoPath)

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

	// Invalidate branch cache after deleting branch
	InvalidateBranchCache(repoPath)

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

// GetMergedBranches returns local branches that have been merged into the specified base branch
func GetMergedBranches(repoPath, baseBranch string) ([]Branch, error) {
	if err := ValidateRepository(repoPath); err != nil {
		return nil, err
	}

	if baseBranch == "" {
		// Try to find default branch
		baseBranch = GetDefaultBranch(repoPath)
		if baseBranch == "" {
			return nil, &BranchError{
				Operation: "list merged",
				Err:       fmt.Errorf("base branch not specified and no default branch found"),
			}
		}
	}

	// Get list of merged branches
	result, err := RunInDir(repoPath, "branch", "--merged", baseBranch, "--format=%(refname:short)|%(objectname:short)")
	if err != nil {
		return nil, &BranchError{
			Operation: "list merged",
			Branch:    baseBranch,
			Err:       err,
		}
	}

	var branches []Branch
	for _, line := range result.Lines() {
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) < 2 {
			continue
		}

		name := strings.TrimSpace(parts[0])
		commit := strings.TrimSpace(parts[1])

		// Skip the base branch itself
		if name == baseBranch {
			continue
		}

		branch := Branch{
			Name:    name,
			FullRef: "refs/heads/" + name,
			Commit:  commit,
		}

		// Get last commit date for this branch
		lastCommit, err := GetBranchLastCommitDate(repoPath, name)
		if err == nil {
			branch.LastCommit = lastCommit
		}

		branches = append(branches, branch)
	}

	return branches, nil
}

// GetStaleBranches returns local branches with no commits in the specified duration
func GetStaleBranches(repoPath string, age time.Duration) ([]Branch, error) {
	if err := ValidateRepository(repoPath); err != nil {
		return nil, err
	}

	// Get all local branches
	allBranches, err := ListLocalBranches(repoPath)
	if err != nil {
		return nil, err
	}

	// Get branch names for batch lookup
	branchNames := make([]string, 0, len(allBranches))
	for _, branch := range allBranches {
		if !branch.IsHead { // Skip current branch
			branchNames = append(branchNames, branch.Name)
		}
	}

	// Get all commit dates in one batch operation
	commitDates, err := getBranchLastCommitDatesBatch(repoPath, branchNames)
	if err != nil {
		// Fall back to individual lookups if batch fails
		return getStaleBranchesSequential(repoPath, allBranches, age)
	}

	cutoff := time.Now().Add(-age)
	var staleBranches []Branch

	for _, branch := range allBranches {
		// Skip current branch
		if branch.IsHead {
			continue
		}

		// Get commit date from batch results
		lastCommit, found := commitDates[branch.Name]
		if !found {
			continue
		}

		branch.LastCommit = lastCommit

		// Check if the branch is stale
		if lastCommit.Before(cutoff) {
			staleBranches = append(staleBranches, branch)
		}
	}

	return staleBranches, nil
}

// getStaleBranchesSequential is a fallback for GetStaleBranches that uses sequential lookups
// This is used when batch operation fails
func getStaleBranchesSequential(repoPath string, allBranches []Branch, age time.Duration) ([]Branch, error) {
	cutoff := time.Now().Add(-age)
	var staleBranches []Branch

	for _, branch := range allBranches {
		// Skip current branch
		if branch.IsHead {
			continue
		}

		// Get last commit date
		lastCommit, err := GetBranchLastCommitDate(repoPath, branch.Name)
		if err != nil {
			continue
		}

		branch.LastCommit = lastCommit

		// Check if the branch is stale
		if lastCommit.Before(cutoff) {
			staleBranches = append(staleBranches, branch)
		}
	}

	return staleBranches, nil
}

// GetBranchLastCommitDate returns the timestamp of the last commit on a branch
func GetBranchLastCommitDate(repoPath, branch string) (time.Time, error) {
	if err := ValidateRepository(repoPath); err != nil {
		return time.Time{}, err
	}

	// Get the committer timestamp of the last commit on the branch
	result, err := RunInDir(repoPath, "log", "-1", "--format=%ct", branch)
	if err != nil {
		return time.Time{}, &BranchError{
			Operation: "get last commit date",
			Branch:    branch,
			Err:       err,
		}
	}

	timestampStr := strings.TrimSpace(result.Stdout)
	if timestampStr == "" {
		return time.Time{}, &BranchError{
			Operation: "get last commit date",
			Branch:    branch,
			Err:       fmt.Errorf("no commits found"),
		}
	}

	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return time.Time{}, &BranchError{
			Operation: "get last commit date",
			Branch:    branch,
			Err:       fmt.Errorf("failed to parse timestamp: %w", err),
		}
	}

	return time.Unix(timestamp, 0), nil
}

// getBranchLastCommitDatesBatch returns last commit dates for multiple branches in a single command
// This is significantly faster than calling GetBranchLastCommitDate in a loop
// Returns a map of branch name to commit date
func getBranchLastCommitDatesBatch(repoPath string, branches []string) (map[string]time.Time, error) {
	if err := ValidateRepository(repoPath); err != nil {
		return nil, err
	}

	if len(branches) == 0 {
		return make(map[string]time.Time), nil
	}

	// Build format string: branch name | commit hash | timestamp
	// Using --simplify-by-decoration to get only branch tips
	// Format: %(refname:short)|%(objectname)|%(committerdate:unix)
	result, err := RunInDir(repoPath, "for-each-ref",
		"--format=%(refname:short)|%(committerdate:unix)",
		"refs/heads/")
	if err != nil {
		return nil, err
	}

	// Parse output into map
	dates := make(map[string]time.Time)
	for _, line := range result.Lines() {
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) != 2 {
			continue
		}

		branchName := parts[0]
		timestampStr := parts[1]

		timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
		if err != nil {
			continue
		}

		dates[branchName] = time.Unix(timestamp, 0)
	}

	return dates, nil
}

// DeleteBranches batch deletes the specified branches
func DeleteBranches(repoPath string, branches []string, force bool) error {
	if err := ValidateRepository(repoPath); err != nil {
		return err
	}

	if len(branches) == 0 {
		return nil
	}

	// Get current branch to ensure we don't delete it
	currentBranch, err := GetCurrentBranch(repoPath)
	if err != nil {
		return err
	}

	// Check for branches with worktrees
	worktrees, err := ListWorktrees(repoPath)
	if err != nil {
		return err
	}

	worktreeBranches := make(map[string]bool)
	for _, wt := range worktrees {
		if wt.Branch != "" {
			worktreeBranches[wt.Branch] = true
		}
	}

	var errors []string
	deletedCount := 0

	for _, branch := range branches {
		// Skip current branch
		if branch == currentBranch {
			errors = append(errors, fmt.Sprintf("%s: cannot delete current branch", branch))
			continue
		}

		// Skip branches with worktrees
		if worktreeBranches[branch] {
			errors = append(errors, fmt.Sprintf("%s: branch has an active worktree", branch))
			continue
		}

		// Delete the branch
		deleteFlag := "-d"
		if force {
			deleteFlag = "-D"
		}

		_, err := RunInDir(repoPath, "branch", deleteFlag, branch)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", branch, err))
			continue
		}

		deletedCount++
	}

	if len(errors) > 0 {
		return &BranchError{
			Operation: "delete branches",
			Err:       fmt.Errorf("failed to delete %d branch(es): %s", len(errors), strings.Join(errors, "; ")),
		}
	}

	return nil
}

// GetDefaultBranch returns the default branch name (main or master)
func GetDefaultBranch(repoPath string) string {
	// Check for common default branch names
	for _, name := range []string{"main", "master"} {
		exists, _ := LocalBranchExists(repoPath, name)
		if exists {
			return name
		}
	}
	return ""
}

// BranchCleanupInfo contains information about a branch for cleanup purposes
type BranchCleanupInfo struct {
	Branch     Branch
	IsMerged   bool
	IsStale    bool
	Age        time.Duration
	AgeString  string // Human-readable age (e.g., "3 days", "2 weeks")
	HasWorktree bool
}

// GetBranchCleanupInfo returns detailed information about branches for cleanup
func GetBranchCleanupInfo(repoPath, baseBranch string, staleAge time.Duration) ([]BranchCleanupInfo, error) {
	if err := ValidateRepository(repoPath); err != nil {
		return nil, err
	}

	// Get default branch if not specified
	if baseBranch == "" {
		baseBranch = GetDefaultBranch(repoPath)
	}

	// Get all local branches
	allBranches, err := ListLocalBranches(repoPath)
	if err != nil {
		return nil, err
	}

	// Get merged branches
	mergedBranches := make(map[string]bool)
	if baseBranch != "" {
		merged, err := GetMergedBranches(repoPath, baseBranch)
		if err == nil {
			for _, b := range merged {
				mergedBranches[b.Name] = true
			}
		}
	}

	// Get worktree branches
	worktrees, _ := ListWorktrees(repoPath)
	worktreeBranches := make(map[string]bool)
	for _, wt := range worktrees {
		if wt.Branch != "" {
			worktreeBranches[wt.Branch] = true
		}
	}

	now := time.Now()
	staleCutoff := now.Add(-staleAge)

	var result []BranchCleanupInfo

	for _, branch := range allBranches {
		// Skip current/HEAD branch
		if branch.IsHead {
			continue
		}

		// Skip the base branch
		if branch.Name == baseBranch {
			continue
		}

		info := BranchCleanupInfo{
			Branch:      branch,
			IsMerged:    mergedBranches[branch.Name],
			HasWorktree: worktreeBranches[branch.Name],
		}

		// Get last commit date and calculate age
		lastCommit, err := GetBranchLastCommitDate(repoPath, branch.Name)
		if err == nil {
			info.Branch.LastCommit = lastCommit
			info.Age = now.Sub(lastCommit)
			info.AgeString = formatDuration(info.Age)
			info.IsStale = lastCommit.Before(staleCutoff)
		}

		result = append(result, info)
	}

	return result, nil
}

// formatDuration formats a duration into a human-readable string
func formatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	if days == 0 {
		hours := int(d.Hours())
		if hours == 0 {
			return "less than an hour"
		}
		if hours == 1 {
			return "1 hour"
		}
		return fmt.Sprintf("%d hours", hours)
	}
	if days == 1 {
		return "1 day"
	}
	if days < 7 {
		return fmt.Sprintf("%d days", days)
	}
	weeks := days / 7
	if weeks == 1 {
		return "1 week"
	}
	if weeks < 4 {
		return fmt.Sprintf("%d weeks", weeks)
	}
	months := days / 30
	if months == 1 {
		return "1 month"
	}
	if months < 12 {
		return fmt.Sprintf("%d months", months)
	}
	years := days / 365
	if years == 1 {
		return "1 year"
	}
	return fmt.Sprintf("%d years", years)
}
