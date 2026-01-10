# Phase 5: List & Delete Worktrees (CLI) - Implementation Plan

## Overview

Phase 5 implements the `gwt list`, `gwt status`, and `gwt delete` commands for viewing and managing worktrees. These commands follow the patterns established in Phase 4.

---

## Task Breakdown

### 1. Implement `gwt list` Command

**File:** `internal/cli/list.go`

**Purpose:** Display all worktrees with their status in a formatted table.

**Flags:**
- `--json` - Output as JSON array
- `--simple` - Output paths only (one per line, for scripting)
- `--all` / `-a` - Include bare/prunable worktrees (hidden by default)

**Implementation Steps:**

1. Create `ListOptions` struct with format flags
2. Define `listCmd` with Use, Short, Long, RunE
3. Register flags in `init()` and add to rootCmd
4. Implement `runList()`:
   - Validate repository
   - Call `git.ListWorktrees(repoPath)`
   - Optionally fetch status for each worktree
   - Format output based on flags
   - Use `output.Table()` for default format

**Table Columns (default format):**
| Column | Source |
|--------|--------|
| Path | `worktree.Path` (relative if possible) |
| Branch | `worktree.Branch` or "(detached)" |
| Commit | `worktree.Commit` (short SHA) |
| Status | Clean/Dirty indicator |
| Main | Star marker if `worktree.IsMain` |

**JSON Output Schema:**
```json
[
  {
    "path": "/path/to/worktree",
    "branch": "feature-foo",
    "commit": "abc1234",
    "commitFull": "abc1234567890...",
    "isMain": false,
    "isDetached": false,
    "locked": false,
    "status": {
      "clean": true,
      "stagedCount": 0,
      "unstagedCount": 0,
      "untrackedCount": 0
    }
  }
]
```

**Simple Output:**
```
/path/to/main
/path/to/feature-auth
/path/to/bugfix-login
```

---

### 2. Add JSON/Simple Output Utilities

**File:** `internal/output/output.go` (extend existing)

**Implementation Steps:**

1. Add `JSON(data interface{})` function for JSON output
2. Add `SimpleList(items []string)` function for line-per-item output
3. Ensure these functions respect `--quiet` flag appropriately

**Functions to Add:**
```go
// JSON outputs data as formatted JSON
func JSON(data interface{}) error

// SimpleList outputs items one per line
func SimpleList(items []string)
```

---

### 3. Implement `gwt status` Command

**File:** `internal/cli/status.go`

**Purpose:** Show detailed status of current or specified worktree.

**Usage:**
```bash
gwt status              # Status of current worktree
gwt status /path/to/wt  # Status of specific worktree
```

**Flags:**
- `--json` - Output as JSON

**Implementation Steps:**

1. Create `StatusOptions` struct
2. Define `statusCmd`
3. Implement `runStatus()`:
   - Determine target worktree (current or from args)
   - Call `git.GetWorktree()` to verify it's a valid worktree
   - Call `git.GetWorktreeStatus()` for detailed status
   - Format and display results

**Output Fields:**
- Path (absolute)
- Branch name
- HEAD commit (SHA + message)
- Last commit time
- Upstream tracking info (ahead/behind)
- Working tree status (staged/unstaged/untracked counts)
- Lock status

**Example Output:**
```
Worktree:  /path/to/feature-auth
Branch:    feature/auth-system
Commit:    abc1234 Add login validation
Modified:  2 hours ago

Status:
  Staged:    2 changes
  Unstaged:  3 changes
  Untracked: 1 file

Upstream:  origin/feature/auth-system
  Ahead:   2 commits
  Behind:  0 commits
```

---

### 4. Implement `gwt delete` Command

**File:** `internal/cli/delete.go`

**Purpose:** Safely delete one or more worktrees with confirmation and checks.

**Usage:**
```bash
gwt delete feature-auth           # Delete by branch name
gwt delete /path/to/worktree      # Delete by path
gwt delete feature-1 feature-2    # Batch delete
```

**Flags:**
- `--force` / `-f` - Skip confirmation and force delete dirty worktrees
- `--delete-branch` / `-b` - Also delete the branch after removing worktree
- `--dry-run` - Show what would be deleted without doing it

**Implementation Steps:**

1. Create `DeleteOptions` struct
2. Define `deleteCmd`
3. Register flags in `init()`
4. Implement `runDelete()`:
   - Validate repository
   - Resolve worktree targets (by path or branch name)
   - Run pre-deletion checks for each
   - Show confirmation with check results
   - Acquire lock
   - Delete worktrees
   - Optionally delete branches
   - Run post_delete hooks
   - Release lock

---

### 5. Pre-Deletion Checks

**Location:** Can be in `internal/cli/delete.go` or new `internal/delete/checks.go`

**Checks to Implement:**

| Check | Description | Blocking? |
|-------|-------------|-----------|
| IsMain | Worktree is the main worktree | Always blocks |
| HasUncommittedChanges | Dirty working tree | Blocks unless --force |
| IsBranchMerged | Branch merged to main/master | Warning only |
| HasRemoteTracking | Branch exists on remote | Warning only |
| IsLocked | Worktree is locked | Blocks unless --force |

**PreDeleteCheck Struct:**
```go
type PreDeleteCheck struct {
    Name     string
    Status   CheckStatus // Pass, Warn, Block
    Message  string
}

type CheckStatus int
const (
    CheckPass CheckStatus = iota
    CheckWarn
    CheckBlock
)
```

**Implementation Steps:**

1. Create `checkWorktreeForDeletion(wt *git.Worktree) []PreDeleteCheck`
2. Check if main worktree (block)
3. Check for uncommitted changes via `git.GetWorktreeStatus()`
4. Check if branch is merged via `git.IsBranchMerged()`
5. Check if remote exists via `git.RemoteBranchExists()`
6. Check if locked via `git.IsWorktreeLocked()`
7. Return array of check results

---

### 6. Batch Deletion with Confirmation

**Implementation Steps:**

1. Collect all worktrees to delete
2. Run pre-deletion checks on all
3. Display summary table:
   ```
   Worktrees to delete:

   PATH                  BRANCH          STATUS    CHECKS
   /path/to/feature-1    feature-1       Clean     OK
   /path/to/feature-2    feature-2       Dirty!    Uncommitted changes
   /path/to/main         main            Clean     BLOCKED: Main worktree

   1 worktree will be skipped (blocked).
   1 worktree has warnings.

   Delete 2 worktrees? [y/N]
   ```
4. If not `--force`, prompt for confirmation
5. Delete in sequence, reporting each result

**Confirmation Logic:**
- If any blocking checks, show which will be skipped
- If any warnings, show them
- Require explicit "yes" for deletion

---

### 7. Force and Delete-Branch Flags

**--force Flag Behavior:**
- Skip confirmation prompt
- Allow deletion of dirty worktrees
- Allow deletion of locked worktrees
- Still prevent main worktree deletion (unoverridable)

**--delete-branch Flag Behavior:**
- After worktree removal, delete the branch
- Use `git.DeleteBranch(branch, force=false)`
- If branch deletion fails, warn but continue
- Do NOT delete branch if it has unmerged changes (unless --force)

**Implementation:**
```go
if deleteOpts.DeleteBranch && wt.Branch != "" {
    if err := git.DeleteBranch(repoPath, wt.Branch, deleteOpts.Force); err != nil {
        output.Warning(fmt.Sprintf("Could not delete branch %s: %v", wt.Branch, err))
    } else {
        output.Success(fmt.Sprintf("Deleted branch %s", wt.Branch))
    }
}
```

---

### 8. Prevent Main Worktree Deletion

**Implementation Steps:**

1. In pre-deletion checks, always check `wt.IsMain`
2. If IsMain, return blocking check
3. In runDelete, filter out blocked worktrees before proceeding
4. Show clear error message: "Cannot delete main worktree"

**Safety:**
- This check cannot be overridden by --force
- Ensure check is performed early in the flow
- Use `git.Worktree.IsMain` field

---

## File Structure After Phase 5

```
internal/cli/
├── list.go       # NEW: gwt list command
├── status.go     # NEW: gwt status command
├── delete.go     # NEW: gwt delete command
├── create.go     # Existing
├── config.go     # Existing
├── doctor.go     # Existing
├── root.go       # Existing
└── errors.go     # Existing

internal/output/
└── output.go     # MODIFIED: Add JSON(), SimpleList()

internal/git/
├── worktree.go   # May need enhancements
├── branch.go     # Use existing IsBranchMerged()
└── status.go     # Use existing GetWorktreeStatus()
```

---

## Implementation Order

```
Step 1: Output utilities (JSON, SimpleList)
    ↓
Step 2: gwt list command (basic table output)
    ↓
Step 3: gwt list --json and --simple formats
    ↓
Step 4: gwt status command
    ↓
Step 5: gwt delete command (basic single delete)
    ↓
Step 6: Pre-deletion checks
    ↓
Step 7: Batch deletion with confirmation
    ↓
Step 8: --force and --delete-branch flags
    ↓
Step 9: Tests and documentation
```

---

## Testing Strategy

### Unit Tests

**list_test.go:**
- Test table formatting with various worktree configurations
- Test JSON output structure
- Test simple output format
- Test filtering of bare/prunable worktrees

**status_test.go:**
- Test status output for clean worktree
- Test status output for dirty worktree
- Test status with detached HEAD
- Test JSON output

**delete_test.go:**
- Test single worktree deletion
- Test main worktree protection
- Test force flag behavior
- Test delete-branch flag
- Test batch deletion confirmation
- Test pre-deletion checks

### Integration Tests

- Create worktrees, list them, verify output
- Create dirty worktree, attempt delete, verify blocked
- Create and merge branch, delete worktree and branch
- Test batch operations with mixed statuses

---

## Edge Cases to Handle

| Case | Expected Behavior |
|------|-------------------|
| No worktrees (bare repo only) | Show message "No worktrees found" |
| Worktree path doesn't exist | Show as prunable, allow deletion |
| Branch name matches multiple | Ask for clarification or use path |
| Detached HEAD worktree | Show "(detached)" in branch column |
| Locked worktree | Show lock icon/status, block delete |
| Remote-only branch | Can't create worktree, show error |
| Current directory is target | Warn user, don't delete if we're inside it |

---

## CLI Help Text Examples

### gwt list
```
Usage:
  gwt list [flags]

Aliases:
  list, ls

Flags:
  -a, --all      Include bare and prunable worktrees
      --json     Output as JSON
      --simple   Output paths only (one per line)
  -h, --help     help for list

Global Flags:
  -c, --config string   config file (default: .worktree.yaml)
      --no-tui          disable TUI, use simple prompts
  -q, --quiet           suppress non-essential output
  -v, --verbose         enable verbose output
```

### gwt status
```
Usage:
  gwt status [path] [flags]

Flags:
      --json     Output as JSON
  -h, --help     help for status
```

### gwt delete
```
Usage:
  gwt delete <branch-or-path>... [flags]

Aliases:
  delete, rm, remove

Flags:
  -b, --delete-branch   also delete the branch
      --dry-run         show what would be deleted
  -f, --force           skip confirmation, force delete dirty worktrees
  -h, --help            help for delete
```

---

## Dependencies

**Existing packages used:**
- `internal/git` - All worktree and branch operations
- `internal/output` - Output formatting
- `internal/config` - For post_delete hooks

**External packages:**
- `encoding/json` - For --json output
- `github.com/spf13/cobra` - CLI framework

**No new external dependencies required.**

---

## Acceptance Criteria

### gwt list
- [ ] Shows all worktrees in table format by default
- [ ] --json outputs valid JSON array
- [ ] --simple outputs one path per line
- [ ] Hides bare/prunable worktrees unless --all
- [ ] Shows branch, commit, and clean/dirty status
- [ ] Marks main worktree distinctly

### gwt status
- [ ] Shows detailed status of current worktree by default
- [ ] Accepts path argument for specific worktree
- [ ] Shows upstream tracking info
- [ ] Shows staged/unstaged/untracked counts
- [ ] --json outputs valid JSON

### gwt delete
- [ ] Deletes single worktree by branch name or path
- [ ] Prevents main worktree deletion (always)
- [ ] Shows pre-deletion checks before confirming
- [ ] Prompts for confirmation (unless --force)
- [ ] --force skips confirmation and deletes dirty worktrees
- [ ] --delete-branch also removes the branch
- [ ] Supports batch deletion of multiple worktrees
- [ ] --dry-run shows what would happen
- [ ] Runs post_delete hooks from config

---

## Notes

- Follow error handling patterns from Phase 4
- Use locking for delete operations (like create)
- Respect --verbose and --quiet flags
- Windows compatibility: test path handling
- Consider adding aliases: `ls` for `list`, `rm`/`remove` for `delete`
