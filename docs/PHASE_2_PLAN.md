# Phase 2: Git Operations Core - Implementation Plan

## Overview

Phase 2 builds the core Git operations layer that all other phases depend on. This phase focuses on creating a robust wrapper for git commands and implementing the fundamental worktree and branch operations.

**Deliverables:**
- Git command execution wrapper with error handling
- Worktree operations (list, add, remove)
- Branch operations (create, delete, list local/remote)
- Repository state validation utilities
- Worktree status detection (clean/dirty, last commit, age)

**Prerequisites:**
- Phase 1 completed (CLI framework, output utilities, error handling)

---

## Tasks

### 1. Implement Git Command Execution Wrapper

**Objective:** Create a robust wrapper for executing git commands with consistent error handling, output capture, and timeout support.

**Files to Create/Modify:**
- `internal/git/exec.go` - Git command execution wrapper
- `internal/git/errors.go` - Git-specific error types

**Core Types and Functions:**

```go
// RunResult contains the result of a git command execution
type RunResult struct {
    Stdout   string
    Stderr   string
    ExitCode int
}

// Run executes a git command and returns the result
func Run(args ...string) (*RunResult, error)

// RunInDir executes a git command in a specific directory
func RunInDir(dir string, args ...string) (*RunResult, error)

// RunWithStdin executes a git command with stdin input
func RunWithStdin(stdin string, args ...string) (*RunResult, error)

// MustRun executes a git command and panics on error (for internal use)
func MustRun(args ...string) *RunResult
```

**Error Types:**

```go
// GitError represents a git command failure
type GitError struct {
    Command  []string
    Stderr   string
    ExitCode int
}

// NotARepoError indicates the current directory is not a git repository
type NotARepoError struct {
    Path string
}

// WorktreeError represents a worktree-specific error
type WorktreeError struct {
    Operation string
    Path      string
    Err       error
}
```

**Implementation Notes:**
- Use `exec.Command` with proper PATH handling
- Capture both stdout and stderr separately
- Set reasonable timeouts (30s default, configurable)
- Handle Windows-specific path separators
- Support verbose mode for command logging
- Strip ANSI codes from output if not in terminal

**Acceptance Criteria:**
- [ ] Commands execute and return output correctly
- [ ] Errors include helpful context (command, stderr, exit code)
- [ ] Verbose mode logs all git commands
- [ ] Works on Windows, macOS, and Linux
- [ ] Handles git commands that prompt for input (fail gracefully)

---

### 2. Implement Repository State Validation

**Objective:** Create utilities to validate repository state before performing operations.

**Files to Create:**
- `internal/git/repo.go` - Repository validation functions

**Functions to Implement:**

```go
// IsGitRepository checks if the given path is inside a git repository
func IsGitRepository(path string) bool

// IsBareRepository checks if the repository is bare
func IsBareRepository(path string) bool

// IsWorktree checks if the given path is a git worktree (not main working tree)
func IsWorktree(path string) bool

// GetRepoRoot returns the root of the git repository
func GetRepoRoot(path string) (string, error)

// GetGitDir returns the path to the .git directory
func GetGitDir(path string) (string, error)

// GetMainWorktreePath returns the path to the main worktree
func GetMainWorktreePath(path string) (string, error)

// GetCurrentBranch returns the current branch name (or HEAD if detached)
func GetCurrentBranch(path string) (string, error)

// IsInsideWorktree checks if currently inside a worktree
func IsInsideWorktree(path string) bool
```

**Git Commands Used:**
| Function | Git Command |
|----------|-------------|
| IsGitRepository | `git rev-parse --git-dir` |
| IsBareRepository | `git rev-parse --is-bare-repository` |
| IsWorktree | `git rev-parse --git-common-dir` vs `--git-dir` |
| GetRepoRoot | `git rev-parse --show-toplevel` |
| GetGitDir | `git rev-parse --git-dir` |
| GetMainWorktreePath | `git worktree list --porcelain` (first entry) |
| GetCurrentBranch | `git symbolic-ref --short HEAD` |
| IsInsideWorktree | `git rev-parse --is-inside-work-tree` |

**Acceptance Criteria:**
- [ ] All functions return correct results in main worktree
- [ ] All functions return correct results in linked worktree
- [ ] Bare repository detected correctly
- [ ] Non-git directory handled gracefully
- [ ] Works with nested git repositories (returns innermost)

---

### 3. Build Worktree Operations - List

**Objective:** Implement worktree listing with detailed information parsing.

**Files to Create:**
- `internal/git/worktree.go` - Worktree operations

**Types:**

```go
// Worktree represents a git worktree
type Worktree struct {
    Path       string    // Absolute path to worktree
    Branch     string    // Branch name (empty if detached)
    Commit     string    // HEAD commit SHA (short)
    CommitFull string    // HEAD commit SHA (full)
    IsMain     bool      // True if this is the main worktree
    IsDetached bool      // True if HEAD is detached
    IsBare     bool      // True if this is a bare worktree entry
    Locked     bool      // True if worktree is locked
    Prunable   bool      // True if worktree can be pruned
}

// WorktreeStatus contains status information for a worktree
type WorktreeStatus struct {
    Clean           bool      // True if no uncommitted changes
    StagedCount     int       // Number of staged changes
    UnstagedCount   int       // Number of unstaged changes
    UntrackedCount  int       // Number of untracked files
    LastCommitTime  time.Time // Time of last commit
    LastCommitMsg   string    // Last commit message (first line)
    AheadCount      int       // Commits ahead of upstream
    BehindCount     int       // Commits behind upstream
}
```

**Functions to Implement:**

```go
// ListWorktrees returns all worktrees for the repository
func ListWorktrees(repoPath string) ([]Worktree, error)

// GetWorktree returns information about a specific worktree
func GetWorktree(worktreePath string) (*Worktree, error)

// FindWorktreeByBranch finds a worktree by branch name
func FindWorktreeByBranch(repoPath, branch string) (*Worktree, error)
```

**Parsing `git worktree list --porcelain`:**

```
worktree /path/to/main
HEAD abc123def456...
branch refs/heads/main

worktree /path/to/feature
HEAD def456abc789...
branch refs/heads/feature-branch
locked
```

**Implementation Notes:**
- Parse porcelain output for reliable parsing
- Handle detached HEAD state
- Handle locked worktrees
- Handle bare repository entries
- Normalize paths for Windows compatibility

**Acceptance Criteria:**
- [ ] Lists all worktrees correctly
- [ ] Parses branch names correctly
- [ ] Detects main vs linked worktrees
- [ ] Handles detached HEAD
- [ ] Handles locked worktrees
- [ ] Handles bare worktree entries

---

### 4. Build Worktree Operations - Add

**Objective:** Implement worktree creation with various branch scenarios.

**Functions to Implement:**

```go
// AddWorktreeOptions configures worktree creation
type AddWorktreeOptions struct {
    Path         string // Directory path for new worktree
    Branch       string // Branch name to check out
    NewBranch    bool   // Create new branch
    StartPoint   string // Starting point for new branch (commit, branch, tag)
    Track        bool   // Set up tracking for remote branch
    Force        bool   // Force creation even if branch exists elsewhere
    Detach       bool   // Create in detached HEAD state
    Lock         bool   // Lock worktree after creation
}

// AddWorktree creates a new worktree
func AddWorktree(repoPath string, opts AddWorktreeOptions) (*Worktree, error)

// AddWorktreeForNewBranch creates worktree with a new branch
func AddWorktreeForNewBranch(repoPath, path, branch, startPoint string) (*Worktree, error)

// AddWorktreeForExistingBranch creates worktree for existing local branch
func AddWorktreeForExistingBranch(repoPath, path, branch string) (*Worktree, error)

// AddWorktreeForRemoteBranch creates worktree tracking a remote branch
func AddWorktreeForRemoteBranch(repoPath, path, remoteBranch string) (*Worktree, error)
```

**Git Commands Used:**
| Scenario | Git Command |
|----------|-------------|
| New branch from HEAD | `git worktree add -b <branch> <path>` |
| New branch from ref | `git worktree add -b <branch> <path> <start>` |
| Existing local branch | `git worktree add <path> <branch>` |
| Remote branch | `git worktree add --track -b <local> <path> <remote>` |
| Detached HEAD | `git worktree add --detach <path> <commit>` |

**Implementation Notes:**
- Validate branch name before creation
- Check if branch already has a worktree
- Handle path collision (directory already exists)
- Support both absolute and relative paths
- Return newly created worktree info

**Acceptance Criteria:**
- [ ] Creates worktree with new branch from HEAD
- [ ] Creates worktree with new branch from specific ref
- [ ] Creates worktree for existing local branch
- [ ] Creates worktree tracking remote branch
- [ ] Rejects invalid branch names
- [ ] Detects and reports when branch already in use
- [ ] Handles directory collision gracefully

---

### 5. Build Worktree Operations - Remove

**Objective:** Implement worktree removal with safety checks.

**Functions to Implement:**

```go
// RemoveWorktreeOptions configures worktree removal
type RemoveWorktreeOptions struct {
    Path  string // Worktree path to remove
    Force bool   // Force removal even with changes
}

// RemoveWorktree removes a worktree
func RemoveWorktree(repoPath string, opts RemoveWorktreeOptions) error

// PruneWorktrees removes stale worktree entries
func PruneWorktrees(repoPath string, dryRun bool) ([]string, error)

// LockWorktree locks a worktree to prevent pruning
func LockWorktree(worktreePath, reason string) error

// UnlockWorktree unlocks a worktree
func UnlockWorktree(worktreePath string) error

// IsWorktreeLocked checks if a worktree is locked
func IsWorktreeLocked(worktreePath string) (bool, string, error)
```

**Git Commands Used:**
| Function | Git Command |
|----------|-------------|
| RemoveWorktree | `git worktree remove <path>` |
| RemoveWorktree (force) | `git worktree remove --force <path>` |
| PruneWorktrees | `git worktree prune` |
| PruneWorktrees (dry-run) | `git worktree prune --dry-run` |
| LockWorktree | `git worktree lock <path> --reason <reason>` |
| UnlockWorktree | `git worktree unlock <path>` |

**Implementation Notes:**
- Never remove main worktree
- Check for uncommitted changes before removal
- Support force removal flag
- Clean up orphaned worktree entries with prune
- Handle locked worktrees appropriately

**Acceptance Criteria:**
- [ ] Removes worktree successfully
- [ ] Refuses to remove main worktree
- [ ] Warns about uncommitted changes
- [ ] Force flag bypasses change check
- [ ] Prune removes stale entries
- [ ] Lock/unlock works correctly

---

### 6. Build Branch Operations - List

**Objective:** Implement branch listing for local and remote branches.

**Files to Create:**
- `internal/git/branch.go` - Branch operations

**Types:**

```go
// Branch represents a git branch
type Branch struct {
    Name        string    // Branch name (without refs/heads/ or refs/remotes/)
    FullRef     string    // Full ref path (refs/heads/main, refs/remotes/origin/main)
    Commit      string    // HEAD commit SHA (short)
    CommitFull  string    // HEAD commit SHA (full)
    IsRemote    bool      // True if remote branch
    Remote      string    // Remote name (for remote branches)
    Upstream    string    // Upstream branch (for local branches)
    IsHead      bool      // True if this is current HEAD
    LastCommit  time.Time // Time of last commit
}
```

**Functions to Implement:**

```go
// ListLocalBranches returns all local branches
func ListLocalBranches(repoPath string) ([]Branch, error)

// ListRemoteBranches returns all remote branches
func ListRemoteBranches(repoPath string) ([]Branch, error)

// ListAllBranches returns all local and remote branches
func ListAllBranches(repoPath string) ([]Branch, error)

// GetBranch returns information about a specific branch
func GetBranch(repoPath, branchName string) (*Branch, error)

// BranchExists checks if a branch exists (local or remote)
func BranchExists(repoPath, branchName string) (bool, error)

// LocalBranchExists checks if a local branch exists
func LocalBranchExists(repoPath, branchName string) (bool, error)

// RemoteBranchExists checks if a remote branch exists
func RemoteBranchExists(repoPath, remoteBranch string) (bool, error)
```

**Git Commands Used:**
| Function | Git Command |
|----------|-------------|
| ListLocalBranches | `git branch --format='%(refname:short) %(objectname:short) %(upstream:short)'` |
| ListRemoteBranches | `git branch -r --format='%(refname:short) %(objectname:short)'` |
| GetBranch | `git show-ref --verify refs/heads/<name>` or `refs/remotes/<name>` |
| Fetch for refresh | `git fetch --all --prune` |

**Implementation Notes:**
- Use `--format` for reliable parsing
- Handle branches with unusual characters
- Cache remote branch list if called frequently
- Support fetching fresh remote data

**Acceptance Criteria:**
- [ ] Lists all local branches
- [ ] Lists all remote branches
- [ ] Identifies current HEAD branch
- [ ] Shows upstream tracking info
- [ ] Handles branches with slashes in names

---

### 7. Build Branch Operations - Create/Delete

**Objective:** Implement branch creation and deletion.

**Functions to Implement:**

```go
// CreateBranchOptions configures branch creation
type CreateBranchOptions struct {
    Name       string // Branch name
    StartPoint string // Starting point (commit, branch, tag)
    Track      string // Remote branch to track
    Force      bool   // Force creation (overwrite existing)
}

// CreateBranch creates a new branch
func CreateBranch(repoPath string, opts CreateBranchOptions) (*Branch, error)

// DeleteBranchOptions configures branch deletion
type DeleteBranchOptions struct {
    Name       string // Branch name
    Force      bool   // Force delete unmerged branch
    Remote     bool   // Delete remote branch too
    RemoteName string // Remote name (default: origin)
}

// DeleteBranch deletes a branch
func DeleteBranch(repoPath string, opts DeleteBranchOptions) error

// RenameBranch renames a branch
func RenameBranch(repoPath, oldName, newName string, force bool) error

// SetUpstreamBranch sets the upstream tracking branch
func SetUpstreamBranch(repoPath, localBranch, upstream string) error
```

**Git Commands Used:**
| Function | Git Command |
|----------|-------------|
| CreateBranch | `git branch <name> [start-point]` |
| CreateBranch (track) | `git branch --track <name> <remote>` |
| DeleteBranch | `git branch -d <name>` |
| DeleteBranch (force) | `git branch -D <name>` |
| DeleteBranch (remote) | `git push <remote> --delete <name>` |
| RenameBranch | `git branch -m <old> <new>` |
| SetUpstreamBranch | `git branch --set-upstream-to=<upstream> <branch>` |

**Implementation Notes:**
- Validate branch names (no spaces, special chars)
- Check if branch is checked out before deletion
- Warn if deleting unmerged branch
- Handle remote deletion separately (requires push access)

**Acceptance Criteria:**
- [ ] Creates branch from HEAD
- [ ] Creates branch from specific ref
- [ ] Creates tracking branch
- [ ] Deletes local branch
- [ ] Refuses to delete current branch
- [ ] Warns about unmerged branches
- [ ] Force delete works for unmerged
- [ ] Renames branch correctly

---

### 8. Add Worktree Status Detection

**Objective:** Detect worktree status including dirty state, last commit info, and age.

**Functions to Implement:**

```go
// GetWorktreeStatus returns detailed status for a worktree
func GetWorktreeStatus(worktreePath string) (*WorktreeStatus, error)

// IsWorktreeClean checks if worktree has no uncommitted changes
func IsWorktreeClean(worktreePath string) (bool, error)

// GetLastCommit returns information about the last commit
func GetLastCommit(worktreePath string) (sha, message string, time time.Time, err error)

// GetWorktreeAge returns time since worktree was created
func GetWorktreeAge(worktreePath string) (time.Duration, error)

// HasUnpushedCommits checks if worktree has commits not pushed to upstream
func HasUnpushedCommits(worktreePath string) (bool, int, error)

// IsBranchMerged checks if a branch is merged into another branch
func IsBranchMerged(repoPath, branch, into string) (bool, error)
```

**Git Commands Used:**
| Function | Git Command |
|----------|-------------|
| IsWorktreeClean | `git status --porcelain` |
| GetLastCommit | `git log -1 --format='%H%n%s%n%ct'` |
| GetWorktreeAge | Check worktree creation time (git-dir mtime or reflog) |
| HasUnpushedCommits | `git rev-list @{upstream}..HEAD --count` |
| IsBranchMerged | `git branch --merged <into>` |

**Status Detection Details:**

```go
// Parse git status --porcelain output
// M  = staged modification
//  M = unstaged modification
// A  = staged addition
// ?? = untracked
// etc.
```

**Implementation Notes:**
- Use porcelain format for reliable parsing
- Handle case where upstream doesn't exist
- Cache status if called multiple times
- Consider performance for large repos

**Acceptance Criteria:**
- [ ] Correctly detects clean worktree
- [ ] Correctly detects staged changes
- [ ] Correctly detects unstaged changes
- [ ] Correctly detects untracked files
- [ ] Gets last commit info
- [ ] Detects ahead/behind upstream
- [ ] Handles missing upstream gracefully

---

### 9. Implement Remote Operations

**Objective:** Support remote repository operations for fetching and remote branch management.

**Functions to Implement:**

```go
// Remote represents a git remote
type Remote struct {
    Name     string
    FetchURL string
    PushURL  string
}

// ListRemotes returns all configured remotes
func ListRemotes(repoPath string) ([]Remote, error)

// GetRemote returns information about a specific remote
func GetRemote(repoPath, name string) (*Remote, error)

// Fetch fetches from a remote (or all remotes)
func Fetch(repoPath string, remote string, prune bool) error

// FetchAll fetches from all remotes with pruning
func FetchAll(repoPath string) error

// GetDefaultRemote returns the default remote (usually "origin")
func GetDefaultRemote(repoPath string) (string, error)
```

**Git Commands Used:**
| Function | Git Command |
|----------|-------------|
| ListRemotes | `git remote -v` |
| GetRemote | `git remote get-url <name>` |
| Fetch | `git fetch <remote> [--prune]` |
| FetchAll | `git fetch --all --prune` |

**Implementation Notes:**
- Parse remote -v output for fetch/push URLs
- Handle multiple remotes
- Support prune option to clean stale refs
- Timeout handling for network operations

**Acceptance Criteria:**
- [ ] Lists all remotes
- [ ] Fetches from specific remote
- [ ] Fetches from all remotes
- [ ] Prune removes stale remote branches
- [ ] Handles network errors gracefully

---

## Implementation Order

```
1. Git exec wrapper         ─┐
2. Git error types          ─┴─▶ Foundation for all git operations

3. Repository validation    ────▶ Required by all operations

4. Worktree list parsing    ─┐
5. Worktree add             ─┼─▶ Core worktree operations
6. Worktree remove          ─┘

7. Branch list              ─┐
8. Branch create/delete     ─┴─▶ Branch management

9. Worktree status          ────▶ Status detection

10. Remote operations       ────▶ Remote support
```

---

## Dependencies

```
github.com/spf13/cobra v1.8.0      # CLI framework (from Phase 1)
github.com/spf13/pflag v1.0.5      # Flag parsing (from Phase 1)
```

**No new dependencies required for Phase 2.** All git operations use the standard library's `os/exec` package.

---

## Testing Strategy

### Unit Tests
- `internal/git/exec_test.go` - Mock command execution tests
- `internal/git/repo_test.go` - Repository validation tests
- `internal/git/worktree_test.go` - Worktree operation tests
- `internal/git/branch_test.go` - Branch operation tests

### Integration Tests

Create a test helper that sets up temporary git repositories:

```go
// testutil/git.go
func CreateTestRepo(t *testing.T) string           // Creates temp repo
func CreateBareTestRepo(t *testing.T) string       // Creates bare temp repo
func CreateTestRepoWithWorktrees(t *testing.T) string // Repo with worktrees
func CreateTestRepoWithRemote(t *testing.T) (local, remote string) // With remote
```

**Test Scenarios:**
- [ ] List worktrees in repo with multiple worktrees
- [ ] Add worktree with new branch
- [ ] Add worktree with existing branch
- [ ] Remove worktree (with and without force)
- [ ] List branches (local and remote)
- [ ] Create and delete branches
- [ ] Detect clean vs dirty worktree
- [ ] Handle repo with no remote
- [ ] Handle bare repository

### Manual Testing Checklist
- [ ] `gwt doctor` still works after changes
- [ ] Git operations work in main worktree
- [ ] Git operations work from linked worktree
- [ ] Operations work on Windows with spaces in path
- [ ] Operations handle unicode branch names
- [ ] Network operations timeout gracefully

---

## Definition of Done

Phase 2 is complete when:

1. **Git Execution**
   - [ ] Command wrapper handles all edge cases
   - [ ] Errors provide helpful context
   - [ ] Verbose mode logs commands

2. **Repository Validation**
   - [ ] All validation functions work correctly
   - [ ] Edge cases handled (bare, non-repo, nested)

3. **Worktree Operations**
   - [ ] List parses all worktree states
   - [ ] Add supports all branch scenarios
   - [ ] Remove has proper safety checks

4. **Branch Operations**
   - [ ] List returns accurate branch info
   - [ ] Create handles all scenarios
   - [ ] Delete has safety checks

5. **Status Detection**
   - [ ] Clean/dirty detection accurate
   - [ ] Last commit info retrieved
   - [ ] Ahead/behind counts correct

6. **Code Quality**
   - [ ] Code passes `go vet`
   - [ ] Unit tests for all functions
   - [ ] Integration tests pass
   - [ ] No platform-specific assumptions without handling

---

## File Summary

| File | Purpose |
|------|---------|
| `internal/git/exec.go` | Git command execution wrapper |
| `internal/git/errors.go` | Git-specific error types |
| `internal/git/repo.go` | Repository state validation |
| `internal/git/worktree.go` | Worktree operations (list, add, remove) |
| `internal/git/branch.go` | Branch operations (list, create, delete) |
| `internal/git/status.go` | Worktree status detection |
| `internal/git/remote.go` | Remote operations |

---

## Notes

- This phase focuses on the git abstraction layer only - no CLI commands
- The functions here will be used by Phase 4 (`gwt create`) and Phase 5 (`gwt list`, `gwt delete`)
- Consider caching for expensive operations (branch list, remote fetch)
- All functions should accept a repo/worktree path rather than assuming cwd
- Phase 1's `internal/git/git.go` can be refactored into these new files
