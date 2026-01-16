# Phase 13: Polish & Advanced Features - Implementation Plan

A detailed implementation plan for completing the remaining Phase 13 tasks of the Git Worktree Manager (gwt).

---

## Overview

### Implementation Status: 80% Complete (4 of 5 tasks)

**Completed:**
- ✅ Task 1: Branch cleanup utilities (CLI + TUI + tests)
- ✅ Task 2: Advanced filtering and search (filter system + CLI integration + tests)
- ✅ Task 3: Configuration editor UI (TUI editor + CLI commands + config save)
- ✅ Task 4: Windows testing & documentation (comprehensive guide + test suite + CI)

**Remaining:**
- 🔴 Task 5: Performance optimization for large repositories (HIGH PRIORITY)

### Task List

| # | Task | Priority | Status | Estimated Effort |
|---|------|----------|--------|------------------|
| 1 | Add branch cleanup utilities | High | ✅ Complete | ~8 hours |
| 2 | Implement advanced filtering and search | Medium | ✅ Complete | ~6 hours |
| 3 | Add configuration editor UI | Medium | ✅ Complete | ~10 hours |
| 4 | Test and document all Windows-specific behavior | High | ✅ Complete | ~12 hours |
| 5 | Performance optimization for large repositories | High | 🔴 Pending | ~8 hours |

### Recommended Implementation Order

1. **Branch cleanup utilities** - Core functionality, heavily used
2. **Windows testing & documentation** - Critical for reliability
3. **Performance optimization** - Foundational improvements
4. **Advanced filtering** - Quality-of-life enhancement
5. **Configuration editor UI** - User convenience feature

---

## Task 1: Branch Cleanup Utilities

### Objective

Implement commands and TUI views for managing stale and merged branches.

### New CLI Command: `gwt cleanup`

```bash
# List merged branches
gwt cleanup --list

# Delete merged branches (with confirmation)
gwt cleanup --merged

# Delete branches older than 30 days (with confirmation)
gwt cleanup --stale 30d

# Preview what would be deleted
gwt cleanup --merged --dry-run

# Force delete without confirmation
gwt cleanup --merged --force

# Exclude specific branches
gwt cleanup --merged --exclude main,develop
```

### Implementation Details

#### 1. New Git Functions (`internal/git/branch.go`)

```go
// GetMergedBranches returns local branches merged into the specified base branch
func GetMergedBranches(repoPath, baseBranch string) ([]Branch, error)

// GetStaleBranches returns branches with no commits in the specified duration
func GetStaleBranches(repoPath string, age time.Duration) ([]Branch, error)

// GetBranchLastCommitDate returns the timestamp of the last commit on a branch
func GetBranchLastCommitDate(repoPath, branch string) (time.Time, error)

// DeleteBranches batch deletes the specified branches
func DeleteBranches(repoPath string, branches []string, force bool) error
```

**Git Commands Used:**
- `git branch --merged <base>` - List merged branches
- `git log -1 --format=%ct <branch>` - Get last commit timestamp
- `git branch -d <branch>` - Delete merged branch
- `git branch -D <branch>` - Force delete branch

#### 2. New CLI Command (`internal/cli/cleanup.go`)

```go
type CleanupOptions struct {
    ListOnly   bool
    Merged     bool
    Stale      string        // Duration string like "30d", "2w"
    DryRun     bool
    Force      bool
    Exclude    []string
    BaseBranch string        // Default: main or master
}

var cleanupCmd = &cobra.Command{
    Use:   "cleanup",
    Short: "Clean up merged or stale branches",
    Long:  `Remove branches that have been merged or haven't been updated recently.`,
    RunE:  runCleanup,
}
```

**Flags:**
- `--list, -l` - Only list branches, don't delete
- `--merged, -m` - Target merged branches
- `--stale <duration>` - Target branches older than duration (e.g., "30d", "2w")
- `--dry-run, -n` - Show what would be deleted
- `--force, -f` - Delete without confirmation
- `--exclude, -e` - Branches to exclude (comma-separated)
- `--base, -b` - Base branch for merge detection (default: main)

#### 3. New TUI View (`internal/tui/views/cleanup_branches.go`)

```go
type CleanupBranchesModel struct {
    branches    []BranchInfo
    selected    map[string]bool
    cursor      int
    filter      string          // "merged", "stale", or "all"
    staleDays   int
    baseBranch  string
    loading     bool
    err         error
    repoPath    string
    width       int
    height      int
}

type BranchInfo struct {
    Name           string
    IsMerged       bool
    LastCommitDate time.Time
    Age            string       // Human-readable age
    Selected       bool
}
```

**View Features:**
- Filter toggle: All / Merged / Stale
- Stale threshold input (days)
- Checkbox list with branch info
- Shows last commit date and age
- Select all / deselect all
- Delete confirmation dialog

#### 4. Menu Integration (`internal/tui/model.go`)

Add "Cleanup Branches" option to main menu:

```go
const (
    ViewMenu = iota
    ViewCreateBranch
    ViewRemoteBranch
    // ... existing views ...
    ViewCleanupBranches    // NEW
)
```

### Files to Create/Modify

| File | Action | Lines |
|------|--------|-------|
| `internal/git/branch.go` | Modify | +~150 |
| `internal/cli/cleanup.go` | Create | ~200 |
| `internal/tui/views/cleanup_branches.go` | Create | ~400 |
| `internal/tui/model.go` | Modify | +~30 |
| `internal/tui/views/menu.go` | Modify | +~10 |
| `internal/git/branch_test.go` | Modify | +~100 |
| `internal/cli/cleanup_test.go` | Create | ~150 |

### Test Cases

1. List merged branches correctly identifies merged branches
2. List stale branches filters by age correctly
3. Exclude pattern works for multiple patterns
4. Dry run doesn't delete anything
5. Force flag skips confirmation
6. Cannot delete current branch
7. Cannot delete branch with worktree
8. Handles branches with special characters in name
9. Handles empty branch list gracefully
10. TUI navigation and selection works correctly

---

## Task 2: Advanced Filtering and Search

### Objective

Add powerful filtering capabilities to worktree and branch lists.

### Filter Syntax

```bash
# Simple filters
gwt list --filter "branch:feature"      # Branch contains "feature"
gwt list --filter "status:dirty"        # Dirty worktrees only
gwt list --filter "status:clean"        # Clean worktrees only

# Comparison operators
gwt list --filter "age:>7d"             # Older than 7 days
gwt list --filter "age:<30d"            # Newer than 30 days
gwt list --filter "commits:>10"         # More than 10 unpushed commits

# Regex patterns
gwt list --filter "branch:^feature/.*"  # Regex match

# Multiple filters (AND logic)
gwt list --filter "status:clean" --filter "age:<30d"

# Negation
gwt list --filter "branch:!main"        # Not main branch
gwt list --filter "status:!dirty"       # Not dirty
```

### Implementation Details

#### 1. Filter Package (`internal/filter/`)

```go
// filter.go
package filter

type FilterExpr struct {
    Field    string    // "branch", "status", "age", "path", "commits"
    Operator string    // "=", "!=", ">", "<", ">=", "<=", "~" (regex)
    Value    string
    Negate   bool
}

type Filter struct {
    Expressions []FilterExpr
}

func Parse(expr string) (*Filter, error)
func (f *Filter) Match(wt *git.Worktree, status *git.WorktreeStatus) bool
```

```go
// parser.go
package filter

func parseExpression(s string) (*FilterExpr, error)
func parseOperator(s string) (field, op, value string, err error)
func parseDuration(s string) (time.Duration, error)
```

#### 2. Supported Filter Fields

| Field | Operators | Description | Examples |
|-------|-----------|-------------|----------|
| `branch` | `=`, `!=`, `~` | Branch name | `branch:feature`, `branch:^fix/.*` |
| `status` | `=`, `!=` | Clean/dirty status | `status:dirty`, `status:clean` |
| `age` | `>`, `<`, `>=`, `<=` | Worktree age | `age:>7d`, `age:<30d` |
| `path` | `=`, `!=`, `~` | Worktree path | `path:~/projects` |
| `commits` | `>`, `<`, `>=`, `<=`, `=` | Unpushed commits | `commits:>0`, `commits:=0` |
| `main` | `=` | Is main worktree | `main:true`, `main:false` |

#### 3. CLI Integration (`internal/cli/list.go`)

Add `--filter` flag to list command:

```go
var listFlags struct {
    // Existing flags...
    Filter []string  // NEW: repeatable filter expressions
}

func init() {
    listCmd.Flags().StringArrayVarP(&listFlags.Filter, "filter", "f", nil,
        "Filter worktrees (can be specified multiple times)")
}
```

#### 4. TUI Integration (`internal/tui/views/worktree_list.go`)

Add search box to worktree list:

```go
type WorktreeListModel struct {
    // Existing fields...
    filterInput    textinput.Model  // NEW
    filterActive   bool             // NEW
    filteredItems  []WorktreeItem   // NEW: filtered list
}
```

**TUI Features:**
- Press `/` to activate search
- Real-time filtering as user types
- Press `Esc` to clear filter
- Show filter help with `?`
- Display match count

### Files to Create/Modify

| File | Action | Lines |
|------|--------|-------|
| `internal/filter/filter.go` | Create | ~200 |
| `internal/filter/parser.go` | Create | ~150 |
| `internal/filter/filter_test.go` | Create | ~200 |
| `internal/cli/list.go` | Modify | +~50 |
| `internal/tui/views/worktree_list.go` | Modify | +~100 |

### Test Cases

1. Parse simple filter expressions
2. Parse comparison operators correctly
3. Parse regex patterns
4. Handle invalid filter syntax gracefully
5. Multiple filters use AND logic
6. Negation works correctly
7. Duration parsing handles various formats (d, w, h)
8. Case-insensitive matching for status
9. TUI filter updates list in real-time
10. Empty filter shows all items

---

## Task 3: Configuration Editor UI

### Objective

Build a TUI-based configuration editor for `.worktree.yaml`.

### Design

```
┌─────────────────────────────────────────────────────────────┐
│  Configuration Editor                                    [?]│
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Copy Settings                                              │
│  ├─ copy_defaults     [.env, .env.local]           [Edit]   │
│  └─ copy_exclude      [node_modules, .git]         [Edit]   │
│                                                             │
│  Docker Settings                                            │
│  ├─ compose_files     [auto-detect]                [Edit]   │
│  ├─ data_directories  [auto-detect]                [Edit]   │
│  ├─ default_mode      shared                       [Edit]   │
│  └─ port_offset       0                            [Edit]   │
│                                                             │
│  Dependencies                                               │
│  ├─ auto_install      true                         [Edit]   │
│  └─ paths             [., ./packages/*]            [Edit]   │
│                                                             │
│  Migrations                                                 │
│  ├─ auto_detect       true                         [Edit]   │
│  └─ command           (not set)                    [Edit]   │
│                                                             │
│  Hooks                                                      │
│  ├─ post_create       []                           [Edit]   │
│  └─ post_delete       []                           [Edit]   │
│                                                             │
├─────────────────────────────────────────────────────────────┤
│  ↑↓ Navigate  Enter Edit  S Save  R Reset  Esc Back         │
└─────────────────────────────────────────────────────────────┘
```

### Implementation Details

#### 1. Config Editor View (`internal/tui/views/config_editor.go`)

```go
type ConfigEditorModel struct {
    config       *config.Config
    originalYAML string           // For detecting changes
    sections     []ConfigSection
    cursor       int
    editing      bool
    editModel    tea.Model        // Current field editor
    dirty        bool             // Has unsaved changes
    configPath   string
    err          error
}

type ConfigSection struct {
    Name   string
    Fields []ConfigField
}

type ConfigField struct {
    Key         string
    Value       interface{}
    Type        FieldType        // String, StringArray, Bool, Int
    Description string
    Validator   func(v interface{}) error
}

type FieldType int

const (
    FieldString FieldType = iota
    FieldStringArray
    FieldBool
    FieldInt
)
```

#### 2. Field Editors (`internal/tui/components/`)

**String Editor:**
```go
type StringEditorModel struct {
    input textinput.Model
    label string
    help  string
}
```

**String Array Editor:**
```go
type StringArrayEditorModel struct {
    items  []string
    input  textinput.Model
    cursor int
    mode   string  // "view", "add", "edit"
}
```

**Boolean Editor:**
```go
type BoolEditorModel struct {
    value bool
    label string
}
```

**Integer Editor:**
```go
type IntEditorModel struct {
    input textinput.Model
    min   int
    max   int
    label string
}
```

#### 3. CLI Integration (`internal/cli/config.go`)

Add `edit` subcommand:

```bash
gwt config edit           # Edit config in TUI
gwt config edit --path    # Show config file path being edited
```

#### 4. Validation (`internal/config/validate.go`)

```go
func ValidateField(field string, value interface{}) error

func ValidateDockerMode(mode string) error {
    if mode != "shared" && mode != "new" && mode != "" {
        return fmt.Errorf("docker mode must be 'shared' or 'new'")
    }
    return nil
}

func ValidatePortOffset(offset int) error {
    if offset < 0 || offset > 65535 {
        return fmt.Errorf("port offset must be between 0 and 65535")
    }
    return nil
}
```

### Files to Create/Modify

| File | Action | Lines |
|------|--------|-------|
| `internal/tui/views/config_editor.go` | Create | ~500 |
| `internal/tui/components/string_editor.go` | Create | ~100 |
| `internal/tui/components/array_editor.go` | Create | ~200 |
| `internal/tui/components/bool_editor.go` | Create | ~80 |
| `internal/tui/components/int_editor.go` | Create | ~100 |
| `internal/tui/model.go` | Modify | +~40 |
| `internal/cli/config.go` | Modify | +~30 |
| `internal/config/validate.go` | Modify | +~100 |

### Test Cases

1. Load existing config into editor
2. Edit string field and save
3. Add/remove items from array field
4. Toggle boolean field
5. Validate integer bounds
6. Detect unsaved changes
7. Reset to defaults
8. Cancel editing without saving
9. Handle missing config file (create new)
10. Validate docker mode values

---

## Task 4: Windows Testing & Documentation

### Objective

Ensure all Windows-specific code paths are tested and documented.

### Windows-Specific Code Locations

| File | Windows Behavior |
|------|------------------|
| `internal/cli/doctor.go` | Symlink permission check |
| `internal/docker/symlink_windows.go` | Junction creation (mklink /J) |
| `internal/docker/symlink.go` | Fallback chain |
| `internal/create/lock.go` | Process locking |
| `internal/create/validate.go` | Reserved name validation |
| `internal/hooks/exec.go` | cmd.exe execution |

### Test Suite

#### 1. Windows-Only Tests (`internal/test/windows_test.go`)

```go
//go:build windows

package test

import (
    "testing"
)

// Symlink and Junction Tests
func TestSymlinkCreation(t *testing.T)
func TestSymlinkFallbackToJunction(t *testing.T)
func TestJunctionCreation(t *testing.T)
func TestSymlinkPermissionDetection(t *testing.T)

// Path Handling Tests
func TestWindowsAbsolutePaths(t *testing.T)
func TestDriveLetterPaths(t *testing.T)
func TestUNCPaths(t *testing.T)
func TestLongPaths(t *testing.T)  // >260 characters
func TestBackslashNormalization(t *testing.T)

// Reserved Names Tests
func TestWindowsReservedNames(t *testing.T)  // CON, PRN, AUX, NUL, COM1-9, LPT1-9
func TestReservedNameVariants(t *testing.T)  // con.txt, COM1.log, etc.

// Hook Execution Tests
func TestHookExecutionWithCmdExe(t *testing.T)
func TestHookWithBatchFile(t *testing.T)
func TestHookWithPowerShell(t *testing.T)
func TestHookEnvironmentVariables(t *testing.T)

// Process Locking Tests
func TestProcessLockCreation(t *testing.T)
func TestProcessLockConflict(t *testing.T)
func TestProcessLockCleanup(t *testing.T)

// File Operations Tests
func TestFileCopyWithLongPaths(t *testing.T)
func TestFileCopyWithSpecialChars(t *testing.T)
func TestCaseSensitivityHandling(t *testing.T)
```

#### 2. Cross-Platform Tests with Windows Variants

```go
// internal/create/validate_test.go
func TestBranchNameValidation(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        // Common tests...

        // Windows-specific
        {"windows reserved CON", "feature/CON", runtime.GOOS == "windows"},
        {"windows reserved com1", "fix/com1", runtime.GOOS == "windows"},
        {"windows reserved aux.txt", "aux.txt", runtime.GOOS == "windows"},
    }
    // ...
}
```

### Documentation (`docs/WINDOWS_GUIDE.md`)

```markdown
# GWT Windows Guide

## Prerequisites

### Required
- Git for Windows (2.25 or later)
- Windows 10/11 or Windows Server 2016+

### Recommended
- Windows Terminal (for better TUI experience)
- Developer Mode enabled (for symlink support)

## Symlink Support

GWT uses symlinks for Docker Compose shared mode. On Windows, symlinks require
either:

### Option 1: Enable Developer Mode (Recommended)
1. Open Settings > Update & Security > For Developers
2. Enable "Developer Mode"
3. Restart your terminal

### Option 2: Run as Administrator
- Right-click your terminal and select "Run as Administrator"
- This grants symlink creation privileges

### Fallback Behavior
If symlinks cannot be created, GWT automatically falls back to:
1. **Junctions** (directory junctions via `mklink /J`)
2. **Copy** (full directory copy as last resort)

## Path Handling

### Supported Path Formats
- Drive letters: `C:\Users\name\projects`
- Forward slashes: `C:/Users/name/projects`
- UNC paths: `\\server\share\projects` (limited support)

### Long Path Support
Windows traditionally limits paths to 260 characters. To enable long paths:
1. Open Group Policy Editor (gpedit.msc)
2. Navigate to: Local Computer Policy > Computer Configuration >
   Administrative Templates > System > Filesystem
3. Enable "Enable Win32 long paths"

Alternatively, set the registry key:
```
HKLM\SYSTEM\CurrentControlSet\Control\FileSystem\LongPathsEnabled = 1
```

## Reserved Names

Windows reserves certain filenames. GWT prevents creating worktrees with these names:
- CON, PRN, AUX, NUL
- COM1-COM9
- LPT1-LPT9

These cannot be used as directory names, even with extensions (e.g., `CON.txt`).

## Hook Execution

Hooks are executed using `cmd.exe` on Windows:
```yaml
hooks:
  post_create:
    - "echo Setting up environment"
    - "npm install"
```

For PowerShell scripts:
```yaml
hooks:
  post_create:
    - "powershell -File setup.ps1"
```

For batch files:
```yaml
hooks:
  post_create:
    - "setup.bat"
```

## Docker Compose

### Docker Desktop
Ensure Docker Desktop for Windows is installed and running.

### Volume Paths
GWT automatically converts paths for Docker:
- Host path: `C:\projects\app\data`
- Docker path: `/c/projects/app/data` (in WSL2 backend)

### Port Conflicts
Use `port_offset` in config to avoid conflicts:
```yaml
docker:
  port_offset: 100  # Adds 100 to all ports
```

## Troubleshooting

### "Symlink privilege not held"
- Enable Developer Mode, or
- Run terminal as Administrator

### "The filename, directory name, or volume label syntax is incorrect"
- Check for reserved names in your branch name
- Check for invalid characters: `< > : " / \ | ? *`

### "The process cannot access the file because it is being used"
- Close any programs using files in the worktree
- Wait for git operations to complete
- Check for antivirus software scanning

### TUI display issues
- Use Windows Terminal for best compatibility
- Set console font to a monospace font
- Enable "Use legacy console" if needed

## Known Limitations

1. **Case Sensitivity**: Windows filesystems are case-insensitive by default
2. **Junctions**: Cannot span drives (use symlinks or copy)
3. **File Locking**: Some files may be locked by Windows
4. **Path Length**: Enable long path support for deep directory structures
```

### GitHub Actions Windows CI

```yaml
# .github/workflows/test.yml
name: Tests

on: [push, pull_request]

jobs:
  test-windows:
    runs-on: windows-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Run Windows tests
        run: go test -v -tags=windows ./...

      - name: Run integration tests
        run: |
          go build -o gwt.exe ./cmd/gwt
          ./gwt.exe doctor
```

### Files to Create/Modify

| File | Action | Lines |
|------|--------|-------|
| `docs/WINDOWS_GUIDE.md` | Create | ~400 |
| `internal/test/windows_test.go` | Create | ~500 |
| `internal/docker/symlink_windows_test.go` | Create | ~200 |
| `internal/create/validate_test.go` | Modify | +~100 |
| `internal/hooks/exec_test.go` | Modify | +~100 |
| `.github/workflows/test.yml` | Modify | +~30 |

### Test Matrix

| Area | Test Type | Windows-Specific |
|------|-----------|------------------|
| Symlinks | Unit + Integration | Junction fallback |
| Paths | Unit | Drive letters, UNC, long paths |
| Reserved names | Unit | CON, PRN, AUX, NUL, COM, LPT |
| Hooks | Integration | cmd.exe, PowerShell |
| Locks | Unit + Integration | File locking behavior |
| Docker | Integration | Path conversion |

---

## Task 5: Performance Optimization

### Objective

Ensure gwt performs well with large repositories (50+ worktrees, 1000+ branches).

### Performance Targets

| Operation | Current | Target |
|-----------|---------|--------|
| `gwt list` (50 worktrees) | ~2s | <500ms |
| `gwt list` (with status) | ~5s | <1s |
| Branch list (1000 branches) | ~1s | <300ms |
| TUI startup | ~500ms | <200ms |

### Implementation Details

#### 1. Caching Layer (`internal/cache/`)

```go
// cache.go
package cache

import (
    "sync"
    "time"
)

type Cache struct {
    mu         sync.RWMutex
    branches   *CacheEntry[[]git.Branch]
    worktrees  *CacheEntry[[]git.Worktree]
    statuses   map[string]*CacheEntry[*git.WorktreeStatus]
    defaultTTL time.Duration
}

type CacheEntry[T any] struct {
    Data      T
    FetchedAt time.Time
    TTL       time.Duration
}

func New(ttl time.Duration) *Cache

func (c *Cache) GetBranches(repoPath string, fetch func() ([]git.Branch, error)) ([]git.Branch, error)
func (c *Cache) GetWorktrees(repoPath string, fetch func() ([]git.Worktree, error)) ([]git.Worktree, error)
func (c *Cache) GetStatus(wtPath string, fetch func() (*git.WorktreeStatus, error)) (*git.WorktreeStatus, error)

func (c *Cache) Invalidate(category string)
func (c *Cache) InvalidateAll()
```

#### 2. Parallel Status Fetching (`internal/git/worktree.go`)

```go
// GetWorktreeStatusBatch fetches status for multiple worktrees in parallel
func GetWorktreeStatusBatch(worktrees []Worktree, workers int) map[string]*WorktreeStatus {
    results := make(map[string]*WorktreeStatus)
    var mu sync.Mutex
    var wg sync.WaitGroup

    sem := make(chan struct{}, workers) // Limit concurrent operations

    for _, wt := range worktrees {
        wg.Add(1)
        go func(wt Worktree) {
            defer wg.Done()
            sem <- struct{}{}
            defer func() { <-sem }()

            status, _ := GetWorktreeStatus(wt.Path)
            mu.Lock()
            results[wt.Path] = status
            mu.Unlock()
        }(wt)
    }

    wg.Wait()
    return results
}
```

#### 3. Lazy Loading in TUI

```go
// internal/tui/views/worktree_list.go

type WorktreeListModel struct {
    // Existing fields...
    statusCache   map[string]*git.WorktreeStatus
    loadingStatus map[string]bool
    visibleRange  struct{ start, end int }
}

// Only load status for visible items
func (m *WorktreeListModel) loadVisibleStatuses() tea.Cmd {
    return func() tea.Msg {
        toLoad := m.getVisibleWorktrees()
        statuses := git.GetWorktreeStatusBatch(toLoad, 4)
        return statusLoadedMsg{statuses: statuses}
    }
}
```

#### 4. Git Command Optimization

```go
// Use --porcelain and -z for faster parsing
func ListWorktreesFast(repoPath string) ([]Worktree, error) {
    // git worktree list --porcelain is already efficient
}

// Use --format for selective output
func ListBranchesFast(repoPath string) ([]Branch, error) {
    // git branch --format='%(refname:short)' -l
}

// Batch ref resolution
func ResolveBranches(repoPath string, branches []string) (map[string]string, error) {
    // git rev-parse --stdin < branches
}
```

#### 5. TUI Virtualization

```go
// internal/tui/components/virtual_list.go

type VirtualList struct {
    items       []interface{}
    renderItem  func(item interface{}, index int, selected bool) string
    height      int
    scrollPos   int
    cursor      int
}

func (v *VirtualList) View() string {
    // Only render visible items
    start := v.scrollPos
    end := min(start+v.height, len(v.items))

    var b strings.Builder
    for i := start; i < end; i++ {
        b.WriteString(v.renderItem(v.items[i], i, i == v.cursor))
        b.WriteString("\n")
    }
    return b.String()
}
```

### Benchmarks (`internal/git/bench_test.go`)

```go
func BenchmarkListWorktrees(b *testing.B) {
    // Setup repo with many worktrees
    for i := 0; i < b.N; i++ {
        ListWorktrees(repoPath)
    }
}

func BenchmarkListBranches1000(b *testing.B) {
    // Setup repo with 1000 branches
    for i := 0; i < b.N; i++ {
        ListLocalBranches(repoPath)
    }
}

func BenchmarkGetWorktreeStatusBatch(b *testing.B) {
    for i := 0; i < b.N; i++ {
        GetWorktreeStatusBatch(worktrees, 4)
    }
}

func BenchmarkCacheHit(b *testing.B) {
    cache := cache.New(time.Minute)
    // Pre-populate cache
    for i := 0; i < b.N; i++ {
        cache.GetBranches(repoPath, nil)
    }
}
```

### Files to Create/Modify

| File | Action | Lines |
|------|--------|-------|
| `internal/cache/cache.go` | Create | ~200 |
| `internal/cache/cache_test.go` | Create | ~150 |
| `internal/git/worktree.go` | Modify | +~80 |
| `internal/git/branch.go` | Modify | +~50 |
| `internal/git/bench_test.go` | Create | ~200 |
| `internal/tui/components/virtual_list.go` | Create | ~150 |
| `internal/tui/views/worktree_list.go` | Modify | +~100 |

### Performance Test Cases

1. List 50 worktrees under 500ms
2. List 1000 branches under 300ms
3. Cache hit returns immediately
4. Cache invalidation works correctly
5. Parallel status fetch completes faster than sequential
6. TUI remains responsive during background loading
7. Memory usage stays bounded with large lists
8. No goroutine leaks in parallel operations

---

## Summary

### Total Estimated Effort

| Task | Files | New Lines | Test Cases |
|------|-------|-----------|------------|
| Branch cleanup | 7 | ~1,010 | 10 |
| Advanced filtering | 5 | ~700 | 10 |
| Config editor | 8 | ~1,150 | 10 |
| Windows testing | 6 | ~1,330 | 20+ |
| Performance | 7 | ~930 | 10 |
| **Total** | **33** | **~5,120** | **60+** |

### Dependencies Between Tasks

```
Branch Cleanup ─────────────────────────┐
                                        │
Advanced Filtering ─────────────────────┼──→ All complete
                                        │
Config Editor ──────────────────────────┤
                                        │
Windows Testing ─── (can run parallel) ─┤
                                        │
Performance ────────────────────────────┘
```

### Risk Areas

1. **Windows symlink testing** - Requires actual Windows environment
2. **Performance benchmarks** - Need large test repositories
3. **TUI complexity** - Config editor has many field types
4. **Filter parser** - Edge cases in regex handling

### Success Criteria

- [x] All 60+ test cases pass (120+ tests passing)
- [x] Windows CI pipeline green (GitHub Actions workflow created)
- [ ] Performance targets met (Task 5 pending)
- [x] Documentation complete (Windows guide + filter help + config editor)
- [x] No regressions in existing functionality
- [x] Branch cleanup fully functional (CLI + TUI + tests)
- [x] Advanced filtering system working (CLI + TUI + comprehensive tests)
- [x] Configuration editor operational (TUI + CLI commands + save/load)

### Progress Summary

**Completed (4/5 tasks, 80%):**
1. ✅ Branch Cleanup - Full CLI + TUI implementation with tests
2. ✅ Advanced Filtering - Complete filter system with comprehensive tests
3. ✅ Configuration Editor UI - TUI editor with array support + CLI commands + config save
4. ✅ Windows Testing - 40+ Windows-specific tests + comprehensive guide + CI

**Remaining (1/5 tasks, 20%):**
1. 🔴 Performance Optimization - Caching, parallel ops, benchmarks (HIGH PRIORITY)
