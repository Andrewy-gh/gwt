package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// GetWorktreeStatus returns detailed status for a worktree
func GetWorktreeStatus(worktreePath string) (*WorktreeStatus, error) {
	if err := ValidateRepository(worktreePath); err != nil {
		return nil, err
	}

	status := &WorktreeStatus{}

	// Get clean/dirty status
	result, err := RunInDir(worktreePath, "status", "--porcelain")
	if err != nil {
		return nil, err
	}

	// Parse status output
	lines := result.Lines()
	status.Clean = len(lines) == 0

	for _, line := range lines {
		if len(line) < 2 {
			continue
		}

		// Format: XY filename
		// X = staged status, Y = unstaged status
		staged := line[0]
		unstaged := line[1]

		// Count staged changes (first character)
		if staged != ' ' && staged != '?' {
			status.StagedCount++
		}

		// Count unstaged changes (second character)
		if unstaged != ' ' && unstaged != '?' {
			status.UnstagedCount++
		}

		// Count untracked files
		if staged == '?' && unstaged == '?' {
			status.UntrackedCount++
		}
	}

	// Get last commit info
	_, msg, commitTime, err := GetLastCommit(worktreePath)
	if err == nil {
		status.LastCommitMsg = msg
		status.LastCommitTime = commitTime
	}

	// Get ahead/behind counts
	aheadCount, behindCount, err := GetAheadBehindCounts(worktreePath)
	if err == nil {
		status.AheadCount = aheadCount
		status.BehindCount = behindCount
	}

	return status, nil
}

// IsWorktreeClean checks if worktree has no uncommitted changes
func IsWorktreeClean(worktreePath string) (bool, error) {
	if err := ValidateRepository(worktreePath); err != nil {
		return false, err
	}

	result, err := RunInDir(worktreePath, "status", "--porcelain")
	if err != nil {
		return false, err
	}

	return strings.TrimSpace(result.Stdout) == "", nil
}

// GetLastCommit returns information about the last commit
// Returns: (sha, message, time, error)
func GetLastCommit(worktreePath string) (string, string, time.Time, error) {
	if err := ValidateRepository(worktreePath); err != nil {
		return "", "", time.Time{}, err
	}

	// Get commit SHA, subject, and timestamp
	// Format: %H = full hash, %s = subject, %ct = committer timestamp (Unix)
	result, err := RunInDir(worktreePath, "log", "-1", "--format=%H%n%s%n%ct")
	if err != nil {
		return "", "", time.Time{}, err
	}

	lines := result.Lines()
	if len(lines) < 3 {
		return "", "", time.Time{}, fmt.Errorf("unexpected git log output")
	}

	sha := lines[0]
	message := lines[1]
	timestampStr := lines[2]

	// Parse Unix timestamp
	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return sha, message, time.Time{}, fmt.Errorf("failed to parse timestamp: %w", err)
	}

	commitTime := time.Unix(timestamp, 0)

	return sha, message, commitTime, nil
}

// GetWorktreeAge returns time since worktree was created
func GetWorktreeAge(worktreePath string) (time.Duration, error) {
	if err := ValidateRepository(worktreePath); err != nil {
		return 0, err
	}

	// Get the .git directory for this worktree
	gitDir, err := GetGitDir(worktreePath)
	if err != nil {
		return 0, err
	}

	// Check the modification time of the git directory
	info, err := os.Stat(gitDir)
	if err != nil {
		return 0, fmt.Errorf("failed to stat git directory: %w", err)
	}

	age := time.Since(info.ModTime())
	return age, nil
}

// HasUnpushedCommits checks if worktree has commits not pushed to upstream
// Returns: (hasUnpushed, count, error)
func HasUnpushedCommits(worktreePath string) (bool, int, error) {
	if err := ValidateRepository(worktreePath); err != nil {
		return false, 0, err
	}

	// Get the count of commits ahead of upstream
	result, err := RunWithOptions(RunOptions{
		Dir:          worktreePath,
		Args:         []string{"rev-list", "@{upstream}..HEAD", "--count"},
		AllowFailure: true,
	})

	// If there's no upstream, this will fail
	if err != nil || !result.Success() {
		return false, 0, nil
	}

	count, err := strconv.Atoi(strings.TrimSpace(result.Stdout))
	if err != nil {
		return false, 0, fmt.Errorf("failed to parse commit count: %w", err)
	}

	return count > 0, count, nil
}

// GetAheadBehindCounts returns how many commits the current branch is ahead/behind upstream
// Returns: (ahead, behind, error)
func GetAheadBehindCounts(worktreePath string) (int, int, error) {
	if err := ValidateRepository(worktreePath); err != nil {
		return 0, 0, err
	}

	// Get ahead count
	aheadResult, err := RunWithOptions(RunOptions{
		Dir:          worktreePath,
		Args:         []string{"rev-list", "@{upstream}..HEAD", "--count"},
		AllowFailure: true,
	})

	ahead := 0
	if err == nil && aheadResult.Success() {
		ahead, _ = strconv.Atoi(strings.TrimSpace(aheadResult.Stdout))
	}

	// Get behind count
	behindResult, err := RunWithOptions(RunOptions{
		Dir:          worktreePath,
		Args:         []string{"rev-list", "HEAD..@{upstream}", "--count"},
		AllowFailure: true,
	})

	behind := 0
	if err == nil && behindResult.Success() {
		behind, _ = strconv.Atoi(strings.TrimSpace(behindResult.Stdout))
	}

	return ahead, behind, nil
}

// IsBranchMerged checks if a branch is merged into another branch
func IsBranchMerged(repoPath, branch, into string) (bool, error) {
	if err := ValidateRepository(repoPath); err != nil {
		return false, err
	}

	// Get list of merged branches
	result, err := RunInDir(repoPath, "branch", "--merged", into)
	if err != nil {
		return false, err
	}

	// Parse output - branches are listed one per line, current branch has *
	for _, line := range result.Lines() {
		// Remove leading spaces and * marker
		branchName := strings.TrimSpace(line)
		branchName = strings.TrimPrefix(branchName, "* ")

		if branchName == branch {
			return true, nil
		}
	}

	return false, nil
}

// GetFileChanges returns the list of changed files in a worktree
// Returns maps of: staged files, unstaged files, untracked files
func GetFileChanges(worktreePath string) (staged, unstaged, untracked []string, err error) {
	if err := ValidateRepository(worktreePath); err != nil {
		return nil, nil, nil, err
	}

	result, err := RunInDir(worktreePath, "status", "--porcelain")
	if err != nil {
		return nil, nil, nil, err
	}

	staged = []string{}
	unstaged = []string{}
	untracked = []string{}

	for _, line := range result.Lines() {
		if len(line) < 3 {
			continue
		}

		// Format: XY filename
		stagedFlag := line[0]
		unstagedFlag := line[1]
		filename := strings.TrimSpace(line[2:])

		// Handle renamed files (format: "R  old -> new")
		if strings.Contains(filename, " -> ") {
			parts := strings.Split(filename, " -> ")
			if len(parts) == 2 {
				filename = parts[1]
			}
		}

		// Staged changes
		if stagedFlag != ' ' && stagedFlag != '?' {
			staged = append(staged, filename)
		}

		// Unstaged changes
		if unstagedFlag != ' ' && unstagedFlag != '?' {
			unstaged = append(unstaged, filename)
		}

		// Untracked files
		if stagedFlag == '?' && unstagedFlag == '?' {
			untracked = append(untracked, filename)
		}
	}

	return staged, unstaged, untracked, nil
}

// HasConflicts checks if there are merge conflicts in the worktree
func HasConflicts(worktreePath string) (bool, []string, error) {
	if err := ValidateRepository(worktreePath); err != nil {
		return false, nil, err
	}

	result, err := RunInDir(worktreePath, "status", "--porcelain")
	if err != nil {
		return false, nil, err
	}

	conflicts := []string{}

	for _, line := range result.Lines() {
		if len(line) < 3 {
			continue
		}

		// Conflict markers: UU, AA, DD, AU, UA, DU, UD
		stagedFlag := line[0]
		unstagedFlag := line[1]

		isConflict := false
		if stagedFlag == 'U' || unstagedFlag == 'U' {
			isConflict = true
		}
		if stagedFlag == 'A' && unstagedFlag == 'A' {
			isConflict = true
		}
		if stagedFlag == 'D' && unstagedFlag == 'D' {
			isConflict = true
		}

		if isConflict {
			filename := strings.TrimSpace(line[2:])
			conflicts = append(conflicts, filename)
		}
	}

	return len(conflicts) > 0, conflicts, nil
}

// GetWorktreeCreationTime attempts to determine when a worktree was created
func GetWorktreeCreationTime(worktreePath string) (time.Time, error) {
	if err := ValidateRepository(worktreePath); err != nil {
		return time.Time{}, err
	}

	// Get the .git directory
	gitDir, err := GetGitDir(worktreePath)
	if err != nil {
		return time.Time{}, err
	}

	// For linked worktrees, the git directory is a file or a directory
	// Check the modification time
	info, err := os.Stat(gitDir)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to stat git directory: %w", err)
	}

	// For more accurate creation time, try to check the gitdir file
	// in the main worktree's .git/worktrees/<name>/gitdir
	if isWorktree, _ := IsWorktree(worktreePath); isWorktree {
		commonDir, err := GetCommonDir(worktreePath)
		if err == nil {
			// Get worktree name from path
			wtName := filepath.Base(worktreePath)
			gitdirPath := filepath.Join(commonDir, "worktrees", wtName, "gitdir")

			if gitdirInfo, err := os.Stat(gitdirPath); err == nil {
				return gitdirInfo.ModTime(), nil
			}
		}
	}

	return info.ModTime(), nil
}
