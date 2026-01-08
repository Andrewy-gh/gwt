package git

import (
	"fmt"
	"strings"
)

// Remote represents a git remote
type Remote struct {
	Name     string
	FetchURL string
	PushURL  string
}

// ListRemotes returns all configured remotes
func ListRemotes(repoPath string) ([]Remote, error) {
	if err := ValidateRepository(repoPath); err != nil {
		return nil, err
	}

	// Get remote list with URLs
	result, err := RunInDir(repoPath, "remote", "-v")
	if err != nil {
		return nil, &RemoteError{
			Operation: "list",
			Err:       err,
		}
	}

	// Parse output - format:
	// origin  https://github.com/user/repo.git (fetch)
	// origin  https://github.com/user/repo.git (push)

	remoteMap := make(map[string]*Remote)

	for _, line := range result.Lines() {
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}

		name := parts[0]
		url := parts[1]
		urlType := strings.Trim(parts[2], "()")

		// Get or create remote entry
		remote, exists := remoteMap[name]
		if !exists {
			remote = &Remote{Name: name}
			remoteMap[name] = remote
		}

		// Set fetch or push URL
		if urlType == "fetch" {
			remote.FetchURL = url
		} else if urlType == "push" {
			remote.PushURL = url
		}
	}

	// Convert map to slice
	var remotes []Remote
	for _, remote := range remoteMap {
		remotes = append(remotes, *remote)
	}

	return remotes, nil
}

// GetRemote returns information about a specific remote
func GetRemote(repoPath, name string) (*Remote, error) {
	if err := ValidateRepository(repoPath); err != nil {
		return nil, err
	}

	if name == "" {
		return nil, &RemoteError{
			Operation: "get",
			Err:       fmt.Errorf("remote name is required"),
		}
	}

	// Get fetch URL
	fetchResult, err := RunWithOptions(RunOptions{
		Dir:          repoPath,
		Args:         []string{"remote", "get-url", name},
		AllowFailure: true,
	})

	if err != nil || !fetchResult.Success() {
		return nil, &RemoteError{
			Operation: "get",
			Remote:    name,
			Err:       fmt.Errorf("remote not found"),
		}
	}

	fetchURL := strings.TrimSpace(fetchResult.Stdout)

	// Get push URL (might be different from fetch URL)
	pushResult, err := RunWithOptions(RunOptions{
		Dir:          repoPath,
		Args:         []string{"remote", "get-url", "--push", name},
		AllowFailure: true,
	})

	pushURL := fetchURL // Default to fetch URL
	if err == nil && pushResult.Success() {
		pushURL = strings.TrimSpace(pushResult.Stdout)
	}

	return &Remote{
		Name:     name,
		FetchURL: fetchURL,
		PushURL:  pushURL,
	}, nil
}

// Fetch fetches from a remote (or all remotes)
func Fetch(repoPath string, remote string, prune bool) error {
	if err := ValidateRepository(repoPath); err != nil {
		return err
	}

	args := []string{"fetch"}

	if remote != "" {
		args = append(args, remote)
	} else {
		args = append(args, "--all")
	}

	if prune {
		args = append(args, "--prune")
	}

	_, err := RunWithOptions(RunOptions{
		Dir:     repoPath,
		Args:    args,
		Timeout: 120000, // 2 minutes for network operations
	})

	if err != nil {
		return &RemoteError{
			Operation: "fetch",
			Remote:    remote,
			Err:       err,
		}
	}

	return nil
}

// FetchAll fetches from all remotes with pruning
func FetchAll(repoPath string) error {
	return Fetch(repoPath, "", true)
}

// GetDefaultRemote returns the default remote (usually "origin")
// If origin doesn't exist, returns the first remote found
func GetDefaultRemote(repoPath string) (string, error) {
	if err := ValidateRepository(repoPath); err != nil {
		return "", err
	}

	// Check if origin exists
	_, err := GetRemote(repoPath, "origin")
	if err == nil {
		return "origin", nil
	}

	// Get all remotes and return the first one
	remotes, err := ListRemotes(repoPath)
	if err != nil {
		return "", err
	}

	if len(remotes) == 0 {
		return "", &RemoteError{
			Operation: "get default",
			Err:       fmt.Errorf("no remotes configured"),
		}
	}

	return remotes[0].Name, nil
}

// GetUpstreamBranch returns the upstream branch for the current branch
// Returns empty string if no upstream is configured
func GetUpstreamBranch(repoPath string) (string, error) {
	if err := ValidateRepository(repoPath); err != nil {
		return "", err
	}

	result, err := RunWithOptions(RunOptions{
		Dir:          repoPath,
		Args:         []string{"rev-parse", "--abbrev-ref", "@{upstream}"},
		AllowFailure: true,
	})

	if err != nil || !result.Success() {
		// No upstream configured
		return "", nil
	}

	return strings.TrimSpace(result.Stdout), nil
}

// Push pushes to a remote
func Push(repoPath, remote, branch string, force bool) error {
	if err := ValidateRepository(repoPath); err != nil {
		return err
	}

	args := []string{"push"}

	if force {
		args = append(args, "--force")
	}

	if remote != "" {
		args = append(args, remote)
	}

	if branch != "" {
		args = append(args, branch)
	}

	_, err := RunWithOptions(RunOptions{
		Dir:     repoPath,
		Args:    args,
		Timeout: 120000, // 2 minutes for network operations
	})

	if err != nil {
		return &RemoteError{
			Operation: "push",
			Remote:    remote,
			Err:       err,
		}
	}

	return nil
}

// SetupRemoteTracking sets up a local branch to track a remote branch
func SetupRemoteTracking(repoPath, localBranch, remoteBranch string) error {
	if err := ValidateRepository(repoPath); err != nil {
		return err
	}

	if localBranch == "" || remoteBranch == "" {
		return &RemoteError{
			Operation: "setup tracking",
			Err:       fmt.Errorf("both local and remote branch names are required"),
		}
	}

	// Use git branch --set-upstream-to
	_, err := RunInDir(repoPath, "branch", "--set-upstream-to="+remoteBranch, localBranch)
	if err != nil {
		return &RemoteError{
			Operation: "setup tracking",
			Err:       err,
		}
	}

	return nil
}

// RemoteExists checks if a remote exists
func RemoteExists(repoPath, name string) (bool, error) {
	if err := ValidateRepository(repoPath); err != nil {
		return false, err
	}

	_, err := GetRemote(repoPath, name)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return false, nil
		}
		return false, err
	}

	return true, nil
}
