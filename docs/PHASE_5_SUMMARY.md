# Phase 5: List & Delete Worktrees (CLI) - Summary

**Status:** ✓ Complete
**Date Completed:** 2026-01-09

## Overview

Phase 5 implements the `gwt list`, `gwt status`, and `gwt delete` commands for viewing and managing worktrees. These commands provide comprehensive CLI interfaces for worktree inspection and safe deletion with pre-flight checks.

## Deliverables

### 1. List Command (`internal/cli/list.go`)

**Command:** `gwt list` (alias: `ls`)

**Features:**
- Table output with PATH, BRANCH, COMMIT, STATUS columns
- Main worktree marked with `*`
- Locked worktrees marked with lock icon
- Clean/Dirty status indicator

**Flags:**
| Flag | Description |
|------|-------------|
| `--json` | Output as JSON array |
| `--simple` | Output paths only (one per line) |
| `-a, --all` | Include bare and prunable worktrees |

**Usage Examples:**
```bash
gwt list                  # Table output
gwt list --json           # JSON for scripting
gwt list --simple         # Paths only
gwt list -a               # Include all worktrees
```

**Table Output:**
```
PATH        BRANCH     COMMIT   STATUS
--------------------------------------------
gwt         main       2d716c8  Clean   *
gwt-auth    feature-1  abc1234  Dirty
```

**JSON Output:**
```json
[
  {
    "path": "/path/to/worktree",
    "branch": "main",
    "commit": "abc1234",
    "commitFull": "abc1234567890...",
    "isMain": true,
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

### 2. Status Command (`internal/cli/status.go`)

**Command:** `gwt status [path]`

**Features:**
- Detailed worktree information
- Working tree status (staged/unstaged/untracked)
- Upstream tracking info (ahead/behind)
- Human-readable time formatting

**Flags:**
| Flag | Description |
|------|-------------|
| `--json` | Output as JSON |

**Usage Examples:**
```bash
gwt status              # Current worktree
gwt status /path/to/wt  # Specific worktree
gwt status --json       # JSON output
```

**Text Output:**
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

### 3. Delete Command (`internal/cli/delete.go`)

**Command:** `gwt delete <target>...` (aliases: `rm`, `remove`)

**Features:**
- Delete by branch name or path
- Pre-deletion safety checks
- Batch deletion support
- Confirmation prompt
- Main worktree protection (cannot be overridden)

**Flags:**
| Flag | Description |
|------|-------------|
| `-f, --force` | Skip confirmation, force delete dirty worktrees |
| `-b, --delete-branch` | Also delete the branch after removing worktree |
| `--dry-run` | Show what would be deleted without doing it |

**Usage Examples:**
```bash
gwt delete feature-auth           # Delete by branch name
gwt delete /path/to/worktree      # Delete by path
gwt delete feature-1 feature-2    # Batch delete
gwt delete -f feature-auth        # Force delete
gwt delete -b feature-auth        # Delete worktree and branch
gwt delete --dry-run feature-1    # Preview deletion
```

**Pre-Deletion Checks:**
| Check | Status | Behavior |
|-------|--------|----------|
| IsMain | Block | Always blocked, cannot override |
| UncommittedChanges | Warn | Blocks unless `--force` |
| NotMerged | Warn | Warning only |
| Locked | Warn | Blocks unless `--force` |
| CurrentDirectory | Block | Cannot delete if inside worktree |

**Confirmation Output:**
```
Worktrees to delete:

PATH                  BRANCH        STATUS   CHECKS
------------------------------------------------------
/path/to/feature-1    feature-1     Clean    OK
/path/to/feature-2    feature-2     Dirty    Uncommitted changes

1 worktree has warnings.

Delete 2 worktrees? [y/N]
```

### 4. Output Utilities (`internal/output/output.go`)

**New Functions:**
```go
// JSON outputs data as formatted JSON
func JSON(data interface{}) error

// SimpleList outputs items one per line
func SimpleList(items []string)
```

## Testing

### Test Files Created
- `internal/cli/list_test.go` - 3 tests
- `internal/cli/status_test.go` - 4 tests
- `internal/cli/delete_test.go` - 10 tests

### Test Coverage
- Table, JSON, and simple output formatting
- Status text and JSON output
- Pre-deletion check logic
- Target resolution (by branch, by path)
- Blocking and warning check detection
- Main worktree protection

**Total New Tests:** 17

## Files Created/Modified

| File | Lines | Purpose |
|------|-------|---------|
| `internal/cli/list.go` | ~180 | List command |
| `internal/cli/status.go` | ~200 | Status command |
| `internal/cli/delete.go` | ~400 | Delete command with checks |
| `internal/cli/list_test.go` | ~130 | List tests |
| `internal/cli/status_test.go` | ~140 | Status tests |
| `internal/cli/delete_test.go` | ~200 | Delete tests |
| `internal/output/output.go` | +15 | JSON and SimpleList functions |
| **Total** | **~1,265** | **New/modified code** |

## Code Quality

- All code passes `go build`
- All code passes `go vet`
- 17 new tests passing
- Full project test suite passing
- Cross-platform compatible

## Integration with Previous Phases

**Phase 1 (Foundation):**
- Uses output utilities with color support
- Inherits global flags (`--verbose`, `--quiet`)

**Phase 2 (Git Operations):**
- `ListWorktrees()` - Core listing functionality
- `GetWorktree()` - Single worktree lookup
- `FindWorktreeByBranch()` - Branch-based lookup
- `GetWorktreeStatus()` - Status information
- `RemoveWorktree()` - Worktree deletion
- `DeleteBranch()` - Branch deletion

**Phase 4 (Create):**
- Uses same locking mechanism for delete operations
- Consistent error handling patterns

## Design Decisions

1. **Multiple output formats** - Table for humans, JSON/simple for scripts
2. **Pre-deletion checks** - Validate before destructive operations
3. **Main worktree protection** - Unoverridable safety measure
4. **Batch operations** - Efficient multi-worktree management
5. **Confirmation by default** - Prevent accidental deletions
6. **Dry-run support** - Preview changes before committing

## Next Steps

With Phase 5 complete, worktree lifecycle management is fully functional:

**Phase 6: File Copying**
- Copy gitignored files to new worktrees
- Apply patterns from config

**Phase 7-10: Post-Creation Features**
- Docker Compose scaffolding
- Dependency installation
- Database migrations
- Lifecycle hooks

**Phase 11-12: TUI**
- Interactive worktree selection
- Visual deletion confirmation
