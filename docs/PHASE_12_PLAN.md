# Phase 12: TUI Views Implementation Plan

## Overview

Implement all interactive TUI views for the GWT application, building on the Phase 11 framework. This phase creates the complete user interface for worktree creation, listing, and deletion using Bubble Tea's Elm-style architecture.

## Implementation Progress

**Status**: 7/13 steps completed (54%)

### ✅ Completed (Steps 1-7)

1. **Foundation Files** - `internal/tui/messages.go`, `internal/tui/flow.go`
   - Custom Bubble Tea messages for view transitions, operations, and async events
   - CreateFlowState and DeleteFlowState for multi-step flows
   - Navigation helpers and state management

2. **Reusable Components** - `internal/tui/components/`
   - `radio.go` - Single-selection radio button list
   - `spinner.go` - Animated loading indicators with stage support
   - `progress.go` - Progress bars, file progress, and multi-stage progress indicators

3. **Create Branch View** - `internal/tui/views/create_branch.go`
   - Branch name input with inline validation
   - Radio selector for branch source type (new from HEAD, new from ref, existing, remote)
   - Tab navigation between input and selector
   - Integration with git.LocalBranchExists() and create.ValidateBranchName()

4. **Create Source View** - `internal/tui/views/create_source.go`
   - Ref/commit/tag input with validation
   - Suggestions table showing recent branches and common refs
   - Integration with git.ValidateRef()

5. **Remote Branch View** - `internal/tui/views/remote_branch.go`
   - Real-time filter input for branch search
   - Paginated table showing remote branches with commit and age
   - R key to refresh (fetch from remote) with spinner
   - Integration with git.ListRemoteBranches() and git.Fetch()

6. **File Selection View** - `internal/tui/views/file_select.go`
   - Async file discovery with spinner
   - Checkbox list for file selection with size display
   - Pre-selection based on config copy_defaults
   - Total/selected size summary
   - Integration with copy.DiscoverIgnored() and config.Load()

7. **Docker Mode View** - `internal/tui/views/docker_mode.go`
   - Radio list for mode selection (None, Shared, New)
   - Async Docker Compose detection with spinner
   - Info box explaining each mode with warnings
   - Integration with docker.DetectComposeFiles() and docker.ParseComposeFiles()

### 🚧 Remaining (Steps 8-13)

8. **Worktree List View** - `internal/tui/views/worktree_list.go`
   - Table with checkbox column for batch selection
   - Status indicators (clean/dirty/warnings)
   - D key to delete selected, R key to refresh

9. **Delete Confirmation View** - `internal/tui/views/delete_confirm.go`
   - Pre-flight checks table (BLOCK/WARN/OK)
   - Y/N confirmation with summary
   - Integration with internal/cli/delete.go logic

10. **Progress Views** - `internal/tui/views/progress.go`
    - FetchingView for remote operations
    - CopyingView for file operations
    - CreatingView for multi-stage worktree creation

11. **Root Model Integration** - `internal/tui/model.go`
    - Extend Update() with all view cases
    - Extend View() with all view renderers
    - Add view initialization methods
    - Implement operation execution methods

12. **Menu Integration** - `internal/tui/views/menu.go`
    - Update menu actions to return SwitchViewMsg
    - Pass repository path to views

13. **Async Operations** - `internal/tui/operations.go`
    - createWorktreeCmd() with progress reporting
    - fetchRemotesCmd() with error handling
    - deleteWorktreesCmd() with batch processing
    - discoverFilesCmd() for background file scanning

### Files Created

**Completed**:
- ✅ `internal/tui/messages.go` (183 lines)
- ✅ `internal/tui/flow.go` (266 lines)
- ✅ `internal/tui/components/radio.go` (229 lines)
- ✅ `internal/tui/components/spinner.go` (231 lines)
- ✅ `internal/tui/components/progress.go` (413 lines)
- ✅ `internal/tui/views/create_branch.go` (252 lines)
- ✅ `internal/tui/views/create_source.go` (238 lines)
- ✅ `internal/tui/views/remote_branch.go` (329 lines)
- ✅ `internal/tui/views/file_select.go` (283 lines)
- ✅ `internal/tui/views/docker_mode.go` (269 lines)

**Pending**:
- ⏳ `internal/tui/views/worktree_list.go`
- ⏳ `internal/tui/views/delete_confirm.go`
- ⏳ `internal/tui/views/progress.go`
- ⏳ `internal/tui/operations.go`

**To Modify**:
- ⏳ `internal/tui/model.go` (extend for all views)
- ⏳ `internal/tui/views/menu.go` (integrate actions)

### Notes for Next Agent

- All create flow views (steps 3-7) are complete and compile successfully
- Views follow consistent patterns: Init/Update/View methods, error handling, async operations
- Components use Bubble Tea's Elm architecture with message passing
- File discovery and Docker detection run asynchronously with spinners
- Tab navigation implemented for multi-component views
- All views integrate with existing core packages (git, create, copy, docker, config)
- go mod tidy needed to ensure github.com/charmbracelet/harmonica is in go.mod

## Architecture Summary

### Root Model Extension
**File**: `C:\E\2026\gwt\internal\tui\model.go`

Extend the root Model to hold sub-models for each view and shared state for multi-step flows:

```go
type Model struct {
    view   View
    width  int
    height int
    keys   KeyMap
    err    error
    repoPath string

    // Sub-models for each view
    menu              *views.MenuModel           // [EXISTS]
    createBranch      *views.CreateBranchModel   // [NEW]
    createSource      *views.CreateSourceModel   // [NEW]
    fileSelect        *views.FileSelectModel     // [NEW]
    dockerMode        *views.DockerModeModel     // [NEW]
    worktreeList      *views.WorktreeListModel   // [NEW]
    deleteConfirm     *views.DeleteConfirmModel  // [NEW]
    remoteBranches    *views.RemoteBranchModel   // [NEW]

    // Shared state for multi-step create flow
    createFlowState   *CreateFlowState
}
```

### Create Flow State
**File**: `C:\E\2026\gwt\internal\tui\flow.go` (new)

State accumulates across create flow steps:

```go
type CreateFlowState struct {
    // Step 1: Branch input
    BranchSpec        *create.BranchSpec
    BranchInput       string
    SourceType        create.BranchSource

    // Step 2: Source selection (conditional)
    StartPoint        string

    // Step 3: Remote branch selection (conditional)
    SelectedRemote    *git.Branch

    // Step 4: File selection
    TargetDir         string
    IgnoredFiles      []copy.IgnoredFile
    FileSelection     *copy.Selection

    // Step 5: Docker mode
    DockerMode        string
    ComposeDetected   bool
    ComposeConfig     *docker.ComposeConfig

    // Navigation
    CurrentStep       int
    TotalSteps        int
}
```

## View Flow

### Create Worktree Flow
```
Menu → ViewCreateBranch (branch input)
  ↓ [Branch type determined]
  ├─→ [New from HEAD] → ViewFileSelect
  ├─→ [New from ref] → ViewCreateSource → ViewFileSelect
  ├─→ [Existing local] → ViewFileSelect
  └─→ [Remote] → ViewRemoteBranch → ViewFileSelect
  ↓
ViewFileSelect → ViewDockerMode → [Execute with progress] → Menu
```

### List/Delete Flow
```
Menu → ViewWorktreeList (batch selection)
  ↓ [User selects worktrees]
  ↓
ViewDeleteConfirm (pre-flight checks) → [Execute deletions] → Menu
```

## New Files to Create

```
internal/tui/
├── messages.go                    [NEW] Custom Bubble Tea messages
├── flow.go                        [NEW] Create flow state management
├── operations.go                  [NEW] Async operation commands
├── views/
│   ├── create_branch.go           [NEW] Branch input view
│   ├── create_source.go           [NEW] Source selection view
│   ├── remote_branch.go           [NEW] Remote branch selection
│   ├── file_select.go             [NEW] File selection view
│   ├── docker_mode.go             [NEW] Docker mode selection
│   ├── worktree_list.go           [NEW] Worktree list view
│   ├── delete_confirm.go          [NEW] Delete confirmation view
│   └── progress.go                [NEW] Progress/spinner views
└── components/
    ├── radio.go                   [NEW] Radio button list
    ├── spinner.go                 [NEW] Loading spinner
    └── progress.go                [NEW] Progress bar
```

## Implementation Steps

### Step 1: Foundation (messages.go, flow.go)
**Files**: `internal/tui/messages.go`, `internal/tui/flow.go`

Define custom messages for view transitions and operations:
- `SwitchViewMsg`, `BackToPreviousViewMsg`
- `CreateFlowNextStepMsg`, `CreateFlowCompleteMsg`
- `FetchRemotesMsg`, `FetchRemotesCompleteMsg`
- `StartCreateOperationMsg`, `CreateProgressMsg`, `CreateCompleteMsg`
- `StartDeleteOperationMsg`, `DeleteProgressMsg`, `DeleteCompleteMsg`

Implement CreateFlowState with navigation helpers.

### Step 2: New Components
**Files**: `internal/tui/components/radio.go`, `spinner.go`, `progress.go`

**RadioList** - Single selection list for Docker mode:
- Cursor navigation
- Single selection (no checkboxes)
- Styled with Lip Gloss

**Spinner** - Loading indicator:
- Animated spinner frames
- Optional message
- Tick-based updates

**ProgressBar** - File copy progress:
- Current/total display
- Percentage and visual bar
- Byte formatting

### Step 3: Create Branch View
**File**: `internal/tui/views/create_branch.go`

```go
type CreateBranchModel struct {
    branchInput     *components.TextInput
    sourceSelector  int // 0=new from HEAD, 1=new from ref, 2=existing, 3=remote
    repoPath        string
    complete        bool
    branchSpec      *create.BranchSpec
}
```

**Features**:
- TextInput for branch name with inline validation
- Radio/selector for branch source type (4 options)
- Validates branch name format
- Checks branch existence appropriately
- Enter to proceed, Esc to return to menu

**Integration**:
- Calls `create.ValidateBranchName()` for validation
- Populates `BranchSpec` based on selections
- Determines next view based on source type

### Step 4: Create Source View
**File**: `internal/tui/views/create_source.go`

```go
type CreateSourceModel struct {
    refInput        *components.TextInput
    suggestions     *components.Table
    repoPath        string
    complete        bool
}
```

**Features**:
- TextInput for ref/commit/tag
- Table showing suggestions (recent branches, commits, tags)
- Arrow keys to navigate suggestions, Enter to select
- Validates ref exists with `git.ValidateRef()`

### Step 5: Remote Branch View
**File**: `internal/tui/views/remote_branch.go`

```go
type RemoteBranchModel struct {
    filterInput     *components.TextInput
    branchTable     *components.Table
    branches        []git.Branch
    filteredBranches []git.Branch
    loading         bool
    repoPath        string
    selected        *git.Branch
}
```

**Features**:
- Filter input updates table in real-time
- Shows remote branches with commit SHA and age
- R key to refresh (fetch from remote) - shows spinner
- Async fetch via `git.Fetch()` → `git.ListRemoteBranches()`

**Table Columns**: Remote Branch | Commit | Age

### Step 6: File Selection View
**File**: `internal/tui/views/file_select.go`

```go
type FileSelectModel struct {
    fileList        *components.CheckboxList
    selection       *copy.Selection
    loading         bool
    totalSize       string
    selectedSize    string
    repoPath        string
    complete        bool
}
```

**Features**:
- Async file discovery via `copy.DiscoverIgnored()` with spinner
- CheckboxList showing files with sizes
- Pre-selects based on config `copy_defaults`
- Status bar showing total/selected size
- Space to toggle, A to select all, N to select none

**Display**: `[✓] path/to/file.txt (1.2 MB)`

### Step 7: Docker Mode View
**File**: `internal/tui/views/docker_mode.go`

```go
type DockerModeModel struct {
    radioList       *components.RadioList
    composeDetected bool
    composeConfig   *docker.ComposeConfig
    selectedMode    string // "none", "shared", "new"
    complete        bool
}
```

**Features**:
- Radio list with 3 options: None, Shared, New
- Info box explaining each mode
- Warning if compose file not detected
- Auto-detect via `docker.ParseComposeFile()`

**Options**:
1. **None** - No Docker setup
2. **Shared** - Symlink data directories (good for read-only)
3. **New** - Isolated containers (copy data, rename volumes)

### Step 8: Worktree List View
**File**: `internal/tui/views/worktree_list.go`

```go
type WorktreeListModel struct {
    table           *components.Table
    worktrees       []git.Worktree
    selected        map[int]bool
    statuses        map[string]*git.WorktreeStatus
    repoPath        string
    actionRequested bool
}
```

**Features**:
- Table with checkbox column for selection
- Shows path, branch, status, last commit, age
- Space to toggle selection
- D to delete selected → transitions to DeleteConfirm view
- R to refresh list
- Loads data via `git.ListWorktrees()` and `git.GetWorktreeStatus()`

**Table Columns**: [ ] | Path | Branch | Status | Last Commit | Age

**Status Indicators**: ✓ (clean), ✘ (dirty), ⚠ (warnings)

### Step 9: Delete Confirmation View
**File**: `internal/tui/views/delete_confirm.go`

```go
type DeleteConfirmModel struct {
    targets         []*cli.DeleteTarget
    checkTable      *components.Table
    confirmed       bool
    cancelled       bool
}
```

**Features**:
- Table showing each target with pre-flight check results
- Color-coded indicators (OK/WARN/BLOCK)
- Summary line: "Delete X worktrees? Y blocked, Z warnings"
- Y/N confirmation prompt
- Integrates with existing `internal/cli/delete.go` logic

**Check Types**:
- **BLOCK**: Main worktree, current directory
- **WARN**: Uncommitted changes, unmerged branch, locked status

**Table Columns**: Path | Branch | Status | Issues

### Step 10: Progress Views
**File**: `internal/tui/views/progress.go`

Progress indicators for long operations:

**FetchingView** - Remote branch fetch:
- Spinner with "Fetching remote branches..."

**CopyingView** - File copy:
- Progress bar showing current/total bytes
- Current file name
- Files done / total files

**CreatingView** - Worktree creation:
- Stage-based progress (Creating worktree → Copying files → Setting up Docker → Running hooks)
- Each stage shows status: pending, running, complete, error

### Step 11: Root Model Integration
**File**: `C:\E\2026\gwt\internal\tui\model.go`

**Extend Update() method**:
- Add cases for all new views
- Handle view transitions via `SwitchViewMsg`
- Handle operation messages (start/progress/complete)
- Implement view initialization methods (`initCreateBranchView()`, etc.)

**Extend View() method**:
- Add cases for all new views
- Render current view based on `m.view`

**Add operation execution methods**:
```go
func (m *Model) executeCreate() tea.Cmd
func (m *Model) executeDelete() tea.Cmd
func (m *Model) executeFetch() tea.Cmd
```

These methods call core operations and return completion messages.

### Step 12: Menu Integration
**File**: `C:\E\2026\gwt\internal\tui\views\menu.go`

Update menu actions:
- "Create Worktree" → `SwitchViewMsg{View: ViewCreateBranch}`
- "List Worktrees" → `SwitchViewMsg{View: ViewWorktreeList}`
- "Delete Worktree" → `SwitchViewMsg{View: ViewWorktreeList}`

Pass repository path from menu to views.

### Step 13: Operations
**File**: `C:\E\2026\gwt\internal\tui\operations.go`

Async operation commands that call core functions:

```go
func createWorktreeCmd(state *CreateFlowState, repoPath string) tea.Cmd
func fetchRemotesCmd(repoPath string) tea.Cmd
func deleteWorktreesCmd(targets []*cli.DeleteTarget) tea.Cmd
func discoverFilesCmd(repoPath string) tea.Cmd
```

Each command:
1. Executes the operation
2. Returns a completion message with result/error
3. Handles progress updates where applicable

## Integration Points

### Core Operations Called

**Worktree Operations**:
- `git.ListWorktrees(repoPath)` → Worktree list view
- `create.CreateWorktree(repoPath, spec, targetDir)` → Create operation
- `git.RemoveWorktree(repoPath, opts)` → Delete operation
- `git.GetWorktreeStatus(worktreePath)` → Status for list view

**Branch Operations**:
- `git.ListLocalBranches(repoPath)` → Branch selection
- `git.ListRemoteBranches(repoPath)` → Remote branch view
- `git.Fetch(repoPath, remote, prune)` → Refresh remotes
- `create.ValidateBranchSpec(repoPath, spec)` → Validation

**File Operations**:
- `copy.DiscoverIgnored(repoPath)` → File selection
- `copy.NewSelection(files, matcher)` → Selection management
- `copy.Copy(opts)` → Copy with progress callback

**Docker Operations**:
- `docker.ParseComposeFile(path)` → Docker detection
- `docker.SetupNewMode(opts)` → Docker setup

### Error Handling Pattern

Views never call operations directly. Instead:
1. View sets `complete = true` or returns `SwitchViewMsg`
2. Root Model initiates operation via command
3. Operation runs asynchronously
4. Completion message updates state or sets error
5. If error, display error view with recovery options

## Key Design Decisions

### 1. Centralized State
- **Choice**: CreateFlowState in root Model
- **Rationale**: Simpler state management, easier back navigation, preserves state across views

### 2. Async Operations in Root Model
- **Choice**: Views return commands, root executes operations
- **Rationale**: Views remain pure/testable, consistent error handling, root has all context

### 3. File Selection Before Docker
- **Choice**: File select → Docker mode (not reversed)
- **Rationale**: File selection takes longer, Docker is final step, better UX flow

### 4. Inline Validation
- **Choice**: Validate as user types where possible
- **Rationale**: Better UX, TextInput component supports it, reduces error views

## Verification

After implementation, test these scenarios:

### Create Flow Tests
1. Create new branch from HEAD → select files → choose Docker mode → verify success
2. Create new branch from specific ref → verify correct start point
3. Checkout existing local branch → verify skips source selection
4. Checkout remote branch → use filter → verify tracking setup
5. Cancel at various steps → verify returns to menu
6. Enter invalid branch name → verify inline validation

### List/Delete Tests
7. List worktrees → verify status indicators (clean/dirty)
8. Select multiple worktrees → delete → verify batch deletion
9. Attempt to delete with BLOCK checks → verify prevented
10. Delete with WARN checks → verify user sees warnings
11. Refresh list → verify updates

### Error Handling Tests
12. Fetch remotes with no network → verify error display
13. Create duplicate branch → verify error message
14. Copy files with permission errors → verify partial success handling
15. Docker setup without compose file → verify warning

### Progress Tests
16. Large file copy → verify progress bar updates
17. Slow fetch → verify spinner appears
18. Multi-stage create → verify stage progression

## Critical Files Summary

**Must Modify**:
- `C:\E\2026\gwt\internal\tui\model.go` - Root model, view switching, operation execution
- `C:\E\2026\gwt\internal\tui\views\menu.go` - Menu actions to trigger views

**Must Create (Priority Order)**:
1. `C:\E\2026\gwt\internal\tui\messages.go` - All custom messages
2. `C:\E\2026\gwt\internal\tui\flow.go` - Create flow state
3. `C:\E\2026\gwt\internal\tui\components\radio.go` - Radio list component
4. `C:\E\2026\gwt\internal\tui\components\spinner.go` - Spinner component
5. `C:\E\2026\gwt\internal\tui\components\progress.go` - Progress bar component
6. `C:\E\2026\gwt\internal\tui\views\create_branch.go` - First view in create flow
7. `C:\E\2026\gwt\internal\tui\views\file_select.go` - Core copy feature
8. `C:\E\2026\gwt\internal\tui\views\docker_mode.go` - Docker selection
9. `C:\E\2026\gwt\internal\tui\views\worktree_list.go` - List and batch selection
10. `C:\E\2026\gwt\internal\tui\views\delete_confirm.go` - Delete with checks
11. `C:\E\2026\gwt\internal\tui\views\create_source.go` - Source selection
12. `C:\E\2026\gwt\internal\tui\views\remote_branch.go` - Remote branch selection
13. `C:\E\2026\gwt\internal\tui\views\progress.go` - Progress views
14. `C:\E\2026\gwt\internal\tui\operations.go` - Async operation commands

## Success Criteria

Phase 12 is complete when:
- All 6 TUI views are implemented and functional
- Create worktree flow works end-to-end with all branch types
- File selection shows sizes and respects config defaults
- Docker mode selection and setup works for all modes
- Worktree list shows status and supports batch deletion
- Delete confirmation shows pre-flight checks correctly
- All views have proper error handling and user feedback
- Progress indicators work for long operations
- Navigation between views is smooth with Esc/back support
- The TUI can handle the full lifecycle: create → list → delete
