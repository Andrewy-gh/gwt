# Phase 4: Create Worktree (CLI) - Implementation Plan

## Overview

Phase 4 implements the `gwt create` command, which creates new git worktrees with proper branch handling, validation, and rollback on failure. This phase focuses on the CLI interface only; TUI integration is handled in later phases.

**Deliverables:**
- `gwt create` command structure with flags
- Branch name validation and directory name conversion
- New branch creation from HEAD or specific ref
- Existing local branch checkout support
- Remote branch checkout with local tracking branch
- Directory collision detection and handling
- Rollback on failure (cleanup partial worktree)
- Concurrent operation locking

**Prerequisites:**
- Phase 1 completed (CLI framework, output utilities, error handling)
- Phase 2 completed (Git operations core, worktree/branch operations)
- Phase 3 completed (Configuration system)

---

## Tasks

### 1. Define Create Command Structure

**Objective:** Implement the `gwt create` command with all required flags and argument handling.

**Files to Create:**
- `internal/cli/create.go` - Create command implementation

**Command Structure:**

```
gwt create                           # Interactive mode (future TUI)
gwt create -b <branch>               # New branch from HEAD
gwt create -b <branch> --from <ref>  # New branch from specific ref
gwt create --checkout <branch>       # Existing local branch
gwt create --remote <remote/branch>  # Remote branch checkout
```

**Flags:**

```go
type CreateOptions struct {
    Branch        string // -b, --branch: New branch name
    From          string // --from: Starting point for new branch (default: HEAD)
    Checkout      string // --checkout: Existing local branch to checkout
    Remote        string // --remote: Remote branch to checkout (creates tracking branch)
    Directory     string // -d, --directory: Override target directory name
    Force         bool   // -f, --force: Force creation even with warnings
    SkipInstall   bool   // --skip-install: Skip dependency installation
    SkipMigrations bool  // --skip-migrations: Skip running migrations
    CopyConfig    bool   // --copy-config: Copy .worktree.yaml to new worktree
    NoTUI         bool   // --no-tui: Disable TUI, use simple prompts
}
```

**Command Implementation:**

```go
var createCmd = &cobra.Command{
    Use:   "create",
    Short: "Create a new worktree",
    Long: `Create a new worktree from a new or existing branch.

Examples:
  gwt create -b feature-auth          Create worktree with new branch from HEAD
  gwt create -b feature-auth --from main  Create from specific branch
  gwt create --checkout existing-branch   Use existing local branch
  gwt create --remote origin/feature      Checkout remote branch`,
    RunE: runCreate,
}

func init() {
    createCmd.Flags().StringVarP(&createOpts.Branch, "branch", "b", "", "new branch name")
    createCmd.Flags().StringVar(&createOpts.From, "from", "", "starting point for new branch")
    createCmd.Flags().StringVar(&createOpts.Checkout, "checkout", "", "existing local branch")
    createCmd.Flags().StringVar(&createOpts.Remote, "remote", "", "remote branch to checkout")
    createCmd.Flags().StringVarP(&createOpts.Directory, "directory", "d", "", "override directory name")
    createCmd.Flags().BoolVarP(&createOpts.Force, "force", "f", false, "force creation")
    createCmd.Flags().BoolVar(&createOpts.SkipInstall, "skip-install", false, "skip dependency installation")
    createCmd.Flags().BoolVar(&createOpts.SkipMigrations, "skip-migrations", false, "skip migrations")
    createCmd.Flags().BoolVar(&createOpts.CopyConfig, "copy-config", false, "copy .worktree.yaml to new worktree")

    rootCmd.AddCommand(createCmd)
}
```

**Implementation Notes:**
- Flags are mutually exclusive: `--branch`, `--checkout`, and `--remote` cannot be combined
- `--from` only valid with `--branch`
- If no flags provided, should launch interactive mode (placeholder for now, TUI in Phase 12)
- Validate at least one branch source is specified for non-interactive use

**Acceptance Criteria:**
- [ ] Command registered with proper help text
- [ ] All flags properly defined and parsed
- [ ] Mutual exclusivity enforced between branch source flags
- [ ] `--from` only allowed with `--branch`
- [ ] Helpful error messages for invalid flag combinations

---

### 2. Implement Branch Name Validation

**Objective:** Validate branch names and convert them to valid directory names.

**Files to Create:**
- `internal/create/validate.go` - Validation logic

**Functions to Implement:**

```go
// ValidateBranchName checks if a branch name is valid for git
// Returns nil if valid, error with message if invalid
func ValidateBranchName(name string) error

// SanitizeDirectoryName converts a branch name to a valid directory name
// Example: "feature/auth/login" -> "feature-auth-login"
func SanitizeDirectoryName(branchName string) string

// GenerateWorktreePath generates the target directory path for a worktree
// Places worktrees as siblings to the main worktree: ../project-branch-name
func GenerateWorktreePath(mainWorktreePath, branchName string) string

// ValidateDirectoryName checks if a directory name is valid for the OS
func ValidateDirectoryName(name string) error
```

**Branch Name Rules (git rules):**
- Cannot contain spaces or control characters
- Cannot start with a dash `-`
- Cannot contain `..`
- Cannot end with `.lock`
- Cannot contain `~`, `^`, `:`, `?`, `*`, `[`, `\`
- Cannot contain `@{`
- Cannot be a single `@`

**Directory Name Conversion Rules:**
- Replace `/` with `-`
- Replace `\` with `-`
- Remove leading/trailing dashes
- Collapse multiple consecutive dashes

**Path Generation:**

```go
func GenerateWorktreePath(mainWorktreePath, branchName string) string {
    // Get parent directory of main worktree
    parentDir := filepath.Dir(mainWorktreePath)

    // Get project name from main worktree
    projectName := filepath.Base(mainWorktreePath)

    // Sanitize branch name for directory
    dirName := SanitizeDirectoryName(branchName)

    // Combine: parent/project-branch
    return filepath.Join(parentDir, projectName+"-"+dirName)
}
```

**Examples:**

| Branch Name | Directory Name |
|-------------|----------------|
| `feature-auth` | `project-feature-auth` |
| `feature/auth/login` | `project-feature-auth-login` |
| `bugfix/header` | `project-bugfix-header` |
| `release/v1.0.0` | `project-release-v1.0.0` |

**Acceptance Criteria:**
- [ ] Validates git branch name rules
- [ ] Sanitizes branch names for directory use
- [ ] Generates sibling directory paths correctly
- [ ] Handles edge cases (slashes, special chars)
- [ ] Works correctly on Windows and Unix paths

---

### 3. Implement Branch Source Detection and Handling

**Objective:** Detect the type of branch operation and execute the appropriate git commands.

**Files to Create:**
- `internal/create/branch.go` - Branch handling logic

**Branch Source Types:**

```go
type BranchSource int

const (
    BranchSourceNewFromHEAD BranchSource = iota  // New branch from current HEAD
    BranchSourceNewFromRef                        // New branch from specific ref
    BranchSourceLocalExisting                     // Existing local branch
    BranchSourceRemote                            // Remote branch (create tracking)
)

type BranchSpec struct {
    Source      BranchSource
    BranchName  string  // Target branch name
    StartPoint  string  // Starting point for new branch (commit, branch, tag)
    RemoteName  string  // Remote name (for remote branches)
}
```

**Functions to Implement:**

```go
// ParseBranchSpec parses create options into a BranchSpec
func ParseBranchSpec(opts CreateOptions) (*BranchSpec, error)

// ValidateBranchSpec validates the branch spec before creation
// Checks if branch exists, if remote branch exists, etc.
func ValidateBranchSpec(repoPath string, spec *BranchSpec) error

// ResolveBranchName resolves the actual branch name for worktree creation
// For remote branches: origin/feature -> feature (local tracking branch)
func ResolveBranchName(spec *BranchSpec) string
```

**Validation Logic:**

```go
func ValidateBranchSpec(repoPath string, spec *BranchSpec) error {
    switch spec.Source {
    case BranchSourceNewFromHEAD, BranchSourceNewFromRef:
        // Check if branch already exists
        exists, err := git.LocalBranchExists(repoPath, spec.BranchName)
        if err != nil {
            return err
        }
        if exists {
            return fmt.Errorf("branch '%s' already exists", spec.BranchName)
        }

        // If from ref, validate ref exists
        if spec.StartPoint != "" {
            if err := git.ValidateRef(repoPath, spec.StartPoint); err != nil {
                return fmt.Errorf("invalid starting point '%s': %w", spec.StartPoint, err)
            }
        }

    case BranchSourceLocalExisting:
        // Check branch exists
        exists, err := git.LocalBranchExists(repoPath, spec.BranchName)
        if err != nil {
            return err
        }
        if !exists {
            return fmt.Errorf("branch '%s' does not exist", spec.BranchName)
        }

        // Check branch not already checked out in a worktree
        wt, err := git.FindWorktreeByBranch(repoPath, spec.BranchName)
        if err != nil {
            return err
        }
        if wt != nil {
            return fmt.Errorf("branch '%s' is already checked out in %s", spec.BranchName, wt.Path)
        }

    case BranchSourceRemote:
        // Check remote branch exists
        exists, err := git.RemoteBranchExists(repoPath, spec.BranchName)
        if err != nil {
            return err
        }
        if !exists {
            return fmt.Errorf("remote branch '%s' does not exist", spec.BranchName)
        }

        // Check if local branch with same name exists with different ref
        localName := ResolveBranchName(spec)
        localExists, err := git.LocalBranchExists(repoPath, localName)
        if err != nil {
            return err
        }
        if localExists {
            // Check if it tracks the same remote
            localBranch, err := git.GetBranch(repoPath, localName)
            if err != nil {
                return err
            }
            if localBranch.Upstream != spec.BranchName {
                return fmt.Errorf("local branch '%s' exists but tracks '%s' instead of '%s'",
                    localName, localBranch.Upstream, spec.BranchName)
            }
        }
    }

    return nil
}
```

**Acceptance Criteria:**
- [ ] Correctly parses create options into BranchSpec
- [ ] Validates new branch doesn't already exist
- [ ] Validates existing branch exists and isn't checked out elsewhere
- [ ] Validates remote branch exists
- [ ] Detects local/remote ref conflicts
- [ ] Clear error messages for each validation failure

---

### 4. Implement Directory Collision Detection

**Objective:** Detect and handle when the target directory already exists.

**Files to Create:**
- `internal/create/directory.go` - Directory handling logic

**Functions to Implement:**

```go
// CheckDirectory checks if the target directory is available
// Returns nil if available, DirectoryExistsError if not
func CheckDirectory(path string) error

// DirectoryExistsError indicates the target directory already exists
type DirectoryExistsError struct {
    Path        string
    IsWorktree  bool    // True if it's already a worktree
    SuggestedAlt string // Alternative directory suggestion
}

// SuggestAlternativeDirectory generates an alternative directory name
// Appends -2, -3, etc. until finding an available name
func SuggestAlternativeDirectory(basePath string) string

// IsEmptyDirectory checks if a directory exists and is empty
func IsEmptyDirectory(path string) (bool, error)

// IsExistingWorktree checks if a path is an existing git worktree
func IsExistingWorktree(path string) (bool, error)
```

**Collision Handling Flow:**

```
1. Check if directory exists
   ├─ No: Proceed with creation
   └─ Yes:
       ├─ Is it an existing worktree?
       │   └─ Error: "Directory is already a worktree for branch X"
       ├─ Is it empty?
       │   └─ Prompt: "Directory exists but is empty. Use anyway?"
       └─ Has content?
           └─ Error: "Directory exists and is not empty"
               Suggest: "Use --directory to specify an alternative name"
               Suggest: project-branch-2, project-branch-3, etc.
```

**Implementation Notes:**
- Use `--directory` flag to override calculated directory name
- Suggest numbered alternatives (branch-2, branch-3)
- Special handling for empty directories (allow with confirmation)
- Check if directory is already registered as a worktree

**Acceptance Criteria:**
- [ ] Detects existing directories
- [ ] Distinguishes between worktrees and regular directories
- [ ] Suggests alternative directory names
- [ ] Handles empty directories specially
- [ ] `--directory` flag overrides calculated name
- [ ] Clear error messages with suggestions

---

### 5. Implement Worktree Creation Logic

**Objective:** Create the worktree using the appropriate git commands based on branch source.

**Files to Create:**
- `internal/create/worktree.go` - Worktree creation orchestration

**Functions to Implement:**

```go
// CreateWorktreeResult contains information about the created worktree
type CreateWorktreeResult struct {
    Path       string
    Branch     string
    Commit     string
    IsNew      bool   // True if a new branch was created
    FromRef    string // Source ref (for new branches)
}

// CreateWorktree creates a new worktree based on the provided spec
func CreateWorktree(repoPath string, spec *BranchSpec, targetDir string) (*CreateWorktreeResult, error)

// createNewBranchWorktree creates worktree with a new branch
func createNewBranchWorktree(repoPath string, spec *BranchSpec, targetDir string) (*CreateWorktreeResult, error)

// createExistingBranchWorktree creates worktree for existing local branch
func createExistingBranchWorktree(repoPath string, spec *BranchSpec, targetDir string) (*CreateWorktreeResult, error)

// createRemoteBranchWorktree creates worktree tracking a remote branch
func createRemoteBranchWorktree(repoPath string, spec *BranchSpec, targetDir string) (*CreateWorktreeResult, error)
```

**Creation Flow:**

```go
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

func createNewBranchWorktree(repoPath string, spec *BranchSpec, targetDir string) (*CreateWorktreeResult, error) {
    startPoint := spec.StartPoint
    if startPoint == "" {
        startPoint = "HEAD"
    }

    wt, err := git.AddWorktreeForNewBranch(repoPath, targetDir, spec.BranchName, startPoint)
    if err != nil {
        return nil, err
    }

    return &CreateWorktreeResult{
        Path:    wt.Path,
        Branch:  wt.Branch,
        Commit:  wt.Commit,
        IsNew:   true,
        FromRef: startPoint,
    }, nil
}
```

**Acceptance Criteria:**
- [ ] Creates worktree from new branch at HEAD
- [ ] Creates worktree from new branch at specific ref
- [ ] Creates worktree for existing local branch
- [ ] Creates worktree tracking remote branch
- [ ] Returns accurate result information
- [ ] Proper error propagation

---

### 6. Implement Rollback on Failure

**Objective:** Clean up partial state if worktree creation fails at any step.

**Files to Create:**
- `internal/create/rollback.go` - Rollback and cleanup logic

**Functions to Implement:**

```go
// Rollback cleans up a failed worktree creation
type Rollback struct {
    worktreePath   string
    branchCreated  string
    directoryCreated bool
    repoPath       string
}

// NewRollback creates a new rollback tracker
func NewRollback(repoPath string) *Rollback

// TrackWorktree records that a worktree was created
func (r *Rollback) TrackWorktree(path string)

// TrackBranch records that a branch was created
func (r *Rollback) TrackBranch(name string)

// TrackDirectory records that a directory was created
func (r *Rollback) TrackDirectory(path string)

// Execute performs the rollback, cleaning up all tracked resources
func (r *Rollback) Execute() error

// Clear clears the rollback tracker (call on success)
func (r *Rollback) Clear()
```

**Rollback Order (reverse of creation):**

```
1. Remove worktree (git worktree remove --force)
2. Delete branch if we created it (git branch -D)
3. Remove directory if we created it (rm -rf)
4. Clean up any lock files
```

**Usage Pattern:**

```go
func runCreate(opts CreateOptions) error {
    rollback := NewRollback(repoPath)
    defer func() {
        if rollback != nil {
            if err := rollback.Execute(); err != nil {
                output.Warn("Rollback failed: %v", err)
            }
        }
    }()

    // Create worktree
    result, err := CreateWorktree(repoPath, spec, targetDir)
    if err != nil {
        return err // rollback will execute
    }
    rollback.TrackWorktree(result.Path)

    if result.IsNew {
        rollback.TrackBranch(result.Branch)
    }

    // ... more steps that could fail ...

    // Success - prevent rollback
    rollback.Clear()
    rollback = nil

    return nil
}
```

**Acceptance Criteria:**
- [ ] Tracks created resources during operation
- [ ] Removes worktree on failure
- [ ] Deletes created branch on failure
- [ ] Removes created directory on failure
- [ ] Rollback is idempotent (safe to call multiple times)
- [ ] Clear() prevents rollback on success
- [ ] Errors during rollback don't mask original error

---

### 7. Implement Concurrent Operation Locking

**Objective:** Prevent multiple simultaneous `gwt create` operations that could conflict.

**Files to Create:**
- `internal/create/lock.go` - Operation locking logic

**Functions to Implement:**

```go
// OperationLock represents a lock for gwt operations
type OperationLock struct {
    lockFile string
    file     *os.File
}

// AcquireLock attempts to acquire the operation lock
// Returns error if lock is held by another process
func AcquireLock(repoPath string) (*OperationLock, error)

// Release releases the operation lock
func (l *OperationLock) Release() error

// IsLocked checks if operations are locked
func IsLocked(repoPath string) (bool, error)

// GetLockInfo returns information about the current lock holder
func GetLockInfo(repoPath string) (*LockInfo, error)

// LockInfo contains information about a lock
type LockInfo struct {
    PID       int
    Command   string
    StartTime time.Time
}
```

**Lock File Location:**
- Store in `.git/gwt.lock` (or `.git/worktrees/../gwt.lock` for linked worktrees)
- Use file locking for cross-process safety

**Lock File Format:**

```json
{
    "pid": 12345,
    "command": "gwt create -b feature-auth",
    "started": "2024-01-15T10:30:00Z"
}
```

**Implementation Notes:**

```go
func AcquireLock(repoPath string) (*OperationLock, error) {
    // Get git directory
    gitDir, err := git.GetGitDir(repoPath)
    if err != nil {
        return nil, err
    }

    lockPath := filepath.Join(gitDir, "gwt.lock")

    // Try to create lock file with exclusive access
    file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
    if err != nil {
        if os.IsExist(err) {
            // Lock exists, check if process is still running
            info, err := GetLockInfo(repoPath)
            if err != nil {
                return nil, fmt.Errorf("lock exists but cannot read info: %w", err)
            }

            // Check if process is still running
            if isProcessRunning(info.PID) {
                return nil, fmt.Errorf("another gwt operation is in progress (PID %d, started %s)",
                    info.PID, info.StartTime.Format(time.RFC3339))
            }

            // Stale lock, remove it
            os.Remove(lockPath)
            return AcquireLock(repoPath) // Retry
        }
        return nil, err
    }

    // Write lock info
    info := LockInfo{
        PID:       os.Getpid(),
        Command:   strings.Join(os.Args, " "),
        StartTime: time.Now(),
    }

    encoder := json.NewEncoder(file)
    if err := encoder.Encode(info); err != nil {
        file.Close()
        os.Remove(lockPath)
        return nil, err
    }

    return &OperationLock{
        lockFile: lockPath,
        file:     file,
    }, nil
}
```

**Error Message:**

```
Error: Another gwt operation is in progress

Process ID: 12345
Command:    gwt create -b feature-auth
Started:    2024-01-15 10:30:00

Wait for the other operation to complete, or if it's stuck, run:
  gwt unlock
```

**Acceptance Criteria:**
- [ ] Acquires lock before starting operation
- [ ] Releases lock on completion (success or failure)
- [ ] Detects and reports existing locks
- [ ] Handles stale locks (process no longer running)
- [ ] Cross-platform file locking
- [ ] Clear error messages with lock holder info

---

### 8. Wire Up Create Command

**Objective:** Integrate all components into the main `gwt create` command flow.

**Files to Modify:**
- `internal/cli/create.go` - Main command orchestration

**Main Flow:**

```go
func runCreate(cmd *cobra.Command, args []string) error {
    // 1. Validate we're in a git repository
    repoPath, err := git.GetRepoRoot(".")
    if err != nil {
        return err
    }

    // 2. Validate not in bare repository
    if err := git.ValidateRepository(repoPath); err != nil {
        return err
    }

    // 3. Acquire operation lock
    lock, err := create.AcquireLock(repoPath)
    if err != nil {
        return err
    }
    defer lock.Release()

    // 4. Parse and validate options
    if !hasAnyBranchFlag(createOpts) {
        // No flags provided - would launch TUI
        // For now, return error until Phase 12
        return fmt.Errorf("interactive mode not yet implemented; use --branch, --checkout, or --remote")
    }

    // 5. Parse branch specification
    spec, err := create.ParseBranchSpec(createOpts)
    if err != nil {
        return err
    }

    // 6. Validate branch specification
    if err := create.ValidateBranchSpec(repoPath, spec); err != nil {
        return err
    }

    // 7. Calculate target directory
    mainWorktree, err := git.GetMainWorktree(repoPath)
    if err != nil {
        return err
    }

    targetDir := createOpts.Directory
    if targetDir == "" {
        targetDir = create.GenerateWorktreePath(mainWorktree.Path, spec.BranchName)
    }

    // 8. Check directory availability
    if err := create.CheckDirectory(targetDir); err != nil {
        if dirErr, ok := err.(*create.DirectoryExistsError); ok {
            output.Error("Directory already exists: %s", dirErr.Path)
            if dirErr.SuggestedAlt != "" {
                output.Info("Suggestion: use --directory %s", dirErr.SuggestedAlt)
            }
        }
        return err
    }

    // 9. Set up rollback
    rollback := create.NewRollback(repoPath)
    defer func() {
        if rollback != nil {
            if err := rollback.Execute(); err != nil {
                output.Warn("Rollback failed: %v", err)
            }
        }
    }()

    // 10. Create the worktree
    output.Info("Creating worktree at %s...", targetDir)
    result, err := create.CreateWorktree(repoPath, spec, targetDir)
    if err != nil {
        return err
    }
    rollback.TrackWorktree(result.Path)
    if result.IsNew {
        rollback.TrackBranch(result.Branch)
    }

    // 11. Copy config if requested
    if createOpts.CopyConfig {
        // TODO: Implement in Phase 6 (file copying)
    }

    // 12. Success - prevent rollback
    rollback.Clear()
    rollback = nil

    // 13. Print success message
    output.Success("Created worktree:")
    output.Info("  Path:   %s", result.Path)
    output.Info("  Branch: %s", result.Branch)
    output.Info("  Commit: %s", result.Commit)
    if result.IsNew {
        output.Info("  From:   %s", result.FromRef)
    }

    output.Info("")
    output.Info("To start working:")
    output.Info("  cd %s", result.Path)

    return nil
}
```

**Acceptance Criteria:**
- [ ] Full create flow works end-to-end
- [ ] Lock acquired before any operations
- [ ] All validations run before creation
- [ ] Rollback works on any failure
- [ ] Success message with useful information
- [ ] Placeholder for TUI mode returns helpful error

---

### 9. Add Helper Functions to Git Package

**Objective:** Add any missing helper functions needed by the create command.

**Files to Modify:**
- `internal/git/repo.go` - Repository helpers
- `internal/git/worktree.go` - Worktree helpers

**Functions to Add:**

```go
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

// GetGitDir returns the path to the .git directory
func GetGitDir(repoPath string) (string, error) {
    result, err := RunInDir(repoPath, "rev-parse", "--git-dir")
    if err != nil {
        return "", err
    }

    gitDir := result.TrimOutput()
    if !filepath.IsAbs(gitDir) {
        gitDir = filepath.Join(repoPath, gitDir)
    }

    return filepath.Clean(gitDir), nil
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

// GetCurrentBranch returns the current branch name (or error if detached)
func GetCurrentBranch(repoPath string) (string, error) {
    result, err := RunInDir(repoPath, "symbolic-ref", "--short", "HEAD")
    if err != nil {
        return "", fmt.Errorf("not on a branch (detached HEAD)")
    }

    return result.TrimOutput(), nil
}
```

**Acceptance Criteria:**
- [ ] GetMainWorktree returns the main worktree
- [ ] GetGitDir returns correct path for both main and linked worktrees
- [ ] ValidateRef works for branches, tags, and commits
- [ ] GetCurrentBranch handles detached HEAD gracefully

---

## Implementation Order

```
1. Create command structure        ─┐
2. Branch name validation          ─┴─▶ Foundation

3. Branch source detection         ─┐
4. Directory collision detection   ─┴─▶ Validation logic

5. Worktree creation logic         ────▶ Core functionality

6. Rollback on failure             ─┐
7. Concurrent operation locking    ─┴─▶ Safety features

8. Wire up create command          ────▶ Integration
9. Git package helpers             ────▶ Support (as needed)
```

---

## Testing Strategy

### Unit Tests

- `internal/create/validate_test.go` - Branch name and directory validation
- `internal/create/branch_test.go` - Branch spec parsing and validation
- `internal/create/directory_test.go` - Directory collision detection
- `internal/create/worktree_test.go` - Worktree creation
- `internal/create/rollback_test.go` - Rollback functionality
- `internal/create/lock_test.go` - Operation locking

### Test Fixtures

Create test scenarios in `internal/create/testdata/`:

```
testdata/
├── repos/
│   ├── simple/              # Simple repo with main branch
│   ├── with-branches/       # Repo with existing local branches
│   ├── with-remotes/        # Repo with remote tracking
│   └── with-worktrees/      # Repo with existing worktrees
└── names/
    ├── valid.txt            # Valid branch names
    └── invalid.txt          # Invalid branch names
```

### Integration Tests

**Test Scenarios:**

- [ ] Create worktree with new branch from HEAD
- [ ] Create worktree with new branch from specific ref
- [ ] Create worktree for existing local branch
- [ ] Create worktree tracking remote branch
- [ ] Error: branch already exists
- [ ] Error: branch checked out in another worktree
- [ ] Error: directory already exists
- [ ] Rollback: cleanup on git worktree add failure
- [ ] Rollback: cleanup on branch creation failure
- [ ] Lock: prevent concurrent operations
- [ ] Lock: handle stale locks

### Manual Testing Checklist

- [ ] `gwt create -b feature-test` creates worktree with new branch
- [ ] `gwt create -b feature-test --from main` creates from specific branch
- [ ] `gwt create --checkout existing-branch` uses existing branch
- [ ] `gwt create --remote origin/feature` creates tracking branch
- [ ] Worktree placed in sibling directory (`../project-branch`)
- [ ] Branch name with slashes converted to dashes
- [ ] Existing directory detected and helpful error shown
- [ ] Failed creation cleans up properly
- [ ] Second `gwt create` while first is running shows lock error
- [ ] Works correctly on Windows
- [ ] Works correctly from linked worktree

---

## Definition of Done

Phase 4 is complete when:

1. **Create Command**
   - [ ] All flags implemented and documented
   - [ ] Mutual exclusivity enforced
   - [ ] Help text is clear and helpful

2. **Branch Validation**
   - [ ] Git branch name rules validated
   - [ ] Branch names sanitized for directories
   - [ ] Path generation works on all platforms

3. **Branch Sources**
   - [ ] New branch from HEAD works
   - [ ] New branch from ref works
   - [ ] Existing local branch works
   - [ ] Remote branch with tracking works

4. **Directory Handling**
   - [ ] Collision detection works
   - [ ] `--directory` override works
   - [ ] Helpful suggestions provided

5. **Safety Features**
   - [ ] Rollback cleans up on failure
   - [ ] Concurrent operation locking works
   - [ ] Stale locks handled

6. **Error Handling**
   - [ ] All error cases have clear messages
   - [ ] Suggestions provided where applicable
   - [ ] Exit codes are correct

7. **Code Quality**
   - [ ] Code passes `go vet`
   - [ ] Unit tests for all functions
   - [ ] Integration tests pass
   - [ ] Documentation comments on exports

---

## File Summary

| File | Purpose |
|------|---------|
| `internal/cli/create.go` | Create command and flag definitions |
| `internal/create/validate.go` | Branch name and directory validation |
| `internal/create/branch.go` | Branch source detection and handling |
| `internal/create/directory.go` | Directory collision detection |
| `internal/create/worktree.go` | Worktree creation orchestration |
| `internal/create/rollback.go` | Rollback and cleanup logic |
| `internal/create/lock.go` | Concurrent operation locking |
| `internal/git/repo.go` | Additional git helpers (if needed) |

---

## Notes

- This phase implements CLI-only; TUI integration is Phase 12
- File copying, Docker scaffolding, and dependencies are separate phases (5-10)
- The `--skip-install` and `--skip-migrations` flags are parsed but features implemented later
- Interactive mode placeholder returns error until TUI is implemented
- Consider adding `--dry-run` flag in future to preview what would be created
- Lock file cleanup is critical - ensure it happens even on panic/signal
