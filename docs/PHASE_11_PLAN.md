# Phase 11: TUI Framework Implementation Plan

## Status: ✅ COMPLETED

**Implementation Date:** 2026-01-13

All Phase 11 tasks have been successfully implemented and verified. The TUI framework is now functional with a working main menu.

## Overview

This document outlines the implementation plan for Phase 11 of the GWT (Git Worktree Manager) project - setting up the TUI (Text User Interface) framework using Bubble Tea and Lip Gloss.

---

## Current State

### What Exists
- **CLI Framework**: Cobra-based command structure in `internal/cli/`
- **Output Package**: Basic color support, progress bars, table rendering in `internal/output/`
- **Global Flag**: `--no-tui` flag already defined in `root.go` (line 17, 59)
- **Placeholder**: Interactive mode returns error "interactive mode not yet implemented" in `create.go` (line 89-90)

### What's Missing
- Bubble Tea and Lip Gloss dependencies
- TUI application structure
- Reusable UI components
- Styling/theming system
- Keyboard navigation handling

---

## Phase 11 Tasks

From `IMPLEMENTATION_PHASES.md`:
- [x] Set up Bubble Tea application structure
- [x] Create Lip Gloss styles and theme
- [x] Build reusable components (checkbox list, text input, table)
- [x] Implement main menu view
- [x] Add keyboard navigation and help footer

---

## Implementation Plan

### Step 1: Add Dependencies

Add Bubble Tea and Lip Gloss to `go.mod`:

```bash
go get github.com/charmbracelet/bubbletea
go get github.com/charmbracelet/lipgloss
go get github.com/charmbracelet/bubbles
```

**Files Modified:**
- `go.mod`
- `go.sum`

---

### Step 2: Create TUI Package Structure

Create new package: `internal/tui/`

```
internal/tui/
├── tui.go           # Main TUI entry point and app runner
├── model.go         # Root model that manages view switching
├── keys.go          # Key bindings and help definitions
├── styles/
│   └── styles.go    # Lip Gloss styles and theme
├── components/
│   ├── list.go      # Checkbox list component
│   ├── input.go     # Text input component
│   ├── table.go     # Table component
│   ├── spinner.go   # Loading spinner
│   └── help.go      # Help footer component
└── views/
    └── menu.go      # Main menu view (Phase 11)
```

---

### Step 3: Define Styles and Theme (`internal/tui/styles/styles.go`)

Create a cohesive theme using Lip Gloss:

```go
package styles

import "github.com/charmbracelet/lipgloss"

// Color palette
var (
    Primary   = lipgloss.Color("#7C3AED")  // Purple
    Secondary = lipgloss.Color("#06B6D4")  // Cyan
    Success   = lipgloss.Color("#22C55E")  // Green
    Warning   = lipgloss.Color("#F59E0B")  // Amber
    Error     = lipgloss.Color("#EF4444")  // Red
    Muted     = lipgloss.Color("#6B7280")  // Gray
    Text      = lipgloss.Color("#F9FAFB")  // Light
    Border    = lipgloss.Color("#374151")  // Dark gray
)

// Component styles
var (
    Title = lipgloss.NewStyle().
        Bold(true).
        Foreground(Primary).
        MarginBottom(1)

    Subtitle = lipgloss.NewStyle().
        Foreground(Muted).
        Italic(true)

    Selected = lipgloss.NewStyle().
        Foreground(Primary).
        Bold(true)

    Cursor = lipgloss.NewStyle().
        Foreground(Secondary)

    Help = lipgloss.NewStyle().
        Foreground(Muted).
        MarginTop(1)

    Box = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(Border).
        Padding(1, 2)

    StatusBar = lipgloss.NewStyle().
        Background(lipgloss.Color("#1F2937")).
        Foreground(Text).
        Padding(0, 1)

    ErrorText = lipgloss.NewStyle().
        Foreground(Error)

    SuccessText = lipgloss.NewStyle().
        Foreground(Success)

    WarningText = lipgloss.NewStyle().
        Foreground(Warning)
)

// Checkbox symbols
const (
    CheckedBox   = "[✓]"
    UncheckedBox = "[ ]"
    Cursor       = ">"
    NoCursor     = " "
)
```

---

### Step 4: Create Key Bindings (`internal/tui/keys.go`)

Define consistent keyboard navigation:

```go
package tui

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
    Up       key.Binding
    Down     key.Binding
    Select   key.Binding
    Confirm  key.Binding
    Back     key.Binding
    Quit     key.Binding
    Help     key.Binding
    Toggle   key.Binding
    SelectAll key.Binding
    DeselectAll key.Binding
}

var DefaultKeyMap = KeyMap{
    Up: key.NewBinding(
        key.WithKeys("up", "k"),
        key.WithHelp("↑/k", "up"),
    ),
    Down: key.NewBinding(
        key.WithKeys("down", "j"),
        key.WithHelp("↓/j", "down"),
    ),
    Select: key.NewBinding(
        key.WithKeys(" ", "x"),
        key.WithHelp("space/x", "toggle"),
    ),
    Confirm: key.NewBinding(
        key.WithKeys("enter"),
        key.WithHelp("enter", "confirm"),
    ),
    Back: key.NewBinding(
        key.WithKeys("esc", "backspace"),
        key.WithHelp("esc", "back"),
    ),
    Quit: key.NewBinding(
        key.WithKeys("q", "ctrl+c"),
        key.WithHelp("q", "quit"),
    ),
    Help: key.NewBinding(
        key.WithKeys("?"),
        key.WithHelp("?", "help"),
    ),
    Toggle: key.NewBinding(
        key.WithKeys(" "),
        key.WithHelp("space", "toggle"),
    ),
    SelectAll: key.NewBinding(
        key.WithKeys("a"),
        key.WithHelp("a", "select all"),
    ),
    DeselectAll: key.NewBinding(
        key.WithKeys("n"),
        key.WithHelp("n", "select none"),
    ),
}
```

---

### Step 5: Build Reusable Components

#### 5.1 Checkbox List Component (`internal/tui/components/list.go`)

A multi-select list with checkboxes:

```go
type CheckboxList struct {
    Items    []CheckboxItem
    Cursor   int
    Selected map[int]bool
    Title    string
    Height   int  // Viewport height for scrolling
}

type CheckboxItem struct {
    Label       string
    Description string
    Value       interface{}
    Disabled    bool
}

// Methods: Init, Update, View, Toggle, SelectAll, DeselectAll, GetSelected
```

**Features:**
- Vim-style navigation (j/k)
- Arrow key navigation
- Space to toggle selection
- `a` to select all, `n` to deselect all
- Scrolling for long lists
- Disabled item support

#### 5.2 Text Input Component (`internal/tui/components/input.go`)

Wrapper around bubbles/textinput with styling:

```go
type TextInput struct {
    Input       textinput.Model
    Label       string
    Placeholder string
    Validator   func(string) error
    ErrorMsg    string
}

// Methods: Init, Update, View, Value, SetValue, Focus, Blur, Validate
```

**Features:**
- Label display
- Placeholder text
- Real-time validation
- Error message display
- Focus management

#### 5.3 Table Component (`internal/tui/components/table.go`)

Styled table for displaying data:

```go
type Table struct {
    Headers   []string
    Rows      [][]string
    Widths    []int
    Cursor    int
    Selectable bool
}

// Methods: Init, Update, View, SelectedRow
```

**Features:**
- Auto-calculated column widths
- Optional row selection
- Styled headers and borders
- Scrolling support

#### 5.4 Help Footer Component (`internal/tui/components/help.go`)

Contextual help display:

```go
type Help struct {
    Keys     []key.Binding
    ShowFull bool
}

// Methods: View, ShortHelp, FullHelp
```

---

### Step 6: Create Root Model (`internal/tui/model.go`)

Central model managing view state:

```go
package tui

type View int

const (
    ViewMenu View = iota
    ViewCreateBranch
    ViewCreateSource
    ViewFileSelect
    ViewDockerMode
    ViewWorktreeList
    ViewDeleteConfirm
)

type Model struct {
    view       View
    width      int
    height     int
    keys       KeyMap
    err        error

    // Sub-models for each view
    menu       *MenuModel
    // ... other view models added in Phase 12
}

func New() Model {
    return Model{
        view: ViewMenu,
        keys: DefaultKeyMap,
        menu: NewMenuModel(),
    }
}

func (m Model) Init() tea.Cmd {
    return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
    case tea.KeyMsg:
        if key.Matches(msg, m.keys.Quit) {
            return m, tea.Quit
        }
    }

    // Delegate to current view
    switch m.view {
    case ViewMenu:
        return m.updateMenu(msg)
    // ... other views
    }

    return m, nil
}

func (m Model) View() string {
    switch m.view {
    case ViewMenu:
        return m.menu.View(m.width, m.height)
    // ... other views
    }
    return ""
}
```

---

### Step 7: Implement Main Menu View (`internal/tui/views/menu.go`)

```go
package views

type MenuModel struct {
    cursor  int
    items   []MenuItem
}

type MenuItem struct {
    Title       string
    Description string
    Action      func() tea.Cmd
}

func NewMenuModel() *MenuModel {
    return &MenuModel{
        items: []MenuItem{
            {
                Title:       "Create Worktree",
                Description: "Create a new worktree from a branch",
            },
            {
                Title:       "List Worktrees",
                Description: "View and manage existing worktrees",
            },
            {
                Title:       "Delete Worktree",
                Description: "Remove worktrees with safety checks",
            },
            {
                Title:       "Configuration",
                Description: "View and edit GWT settings",
            },
        },
    }
}

func (m *MenuModel) View(width, height int) string {
    var b strings.Builder

    // Title
    b.WriteString(styles.Title.Render("GWT - Git Worktree Manager"))
    b.WriteString("\n\n")

    // Menu items
    for i, item := range m.items {
        cursor := " "
        if i == m.cursor {
            cursor = styles.Cursor.Render(">")
        }

        title := item.Title
        if i == m.cursor {
            title = styles.Selected.Render(title)
        }

        b.WriteString(fmt.Sprintf("%s %s\n", cursor, title))
        b.WriteString(fmt.Sprintf("  %s\n\n", styles.Subtitle.Render(item.Description)))
    }

    // Help footer
    b.WriteString(styles.Help.Render("↑/↓: navigate • enter: select • q: quit"))

    return b.String()
}
```

---

### Step 8: Create TUI Entry Point (`internal/tui/tui.go`)

```go
package tui

import (
    "fmt"
    tea "github.com/charmbracelet/bubbletea"
)

// Run starts the TUI application
func Run() error {
    p := tea.NewProgram(
        New(),
        tea.WithAltScreen(),       // Use alternate screen buffer
        tea.WithMouseCellMotion(), // Enable mouse support
    )

    finalModel, err := p.Run()
    if err != nil {
        return fmt.Errorf("TUI error: %w", err)
    }

    // Handle any final state from the model
    if m, ok := finalModel.(Model); ok {
        if m.err != nil {
            return m.err
        }
    }

    return nil
}

// RunWithResult starts TUI and returns the result
func RunWithResult[T any]() (T, error) {
    // For views that return data (like branch selection)
    // Implementation depends on specific use case
}
```

---

### Step 9: Integration Point in CLI

Update `internal/cli/create.go` to launch TUI when no flags provided:

```go
// In runCreate(), around line 89
if createOpts.Branch == "" && createOpts.Checkout == "" && createOpts.Remote == "" {
    if GetNoTUI() {
        return fmt.Errorf("no branch specified; use --branch, --checkout, or --remote")
    }
    // Launch TUI (Phase 12 will implement the full flow)
    return tui.Run()
}
```

---

## File Summary

### New Files to Create

| File | Purpose |
|------|---------|
| `internal/tui/tui.go` | TUI entry point and runner |
| `internal/tui/model.go` | Root model with view switching |
| `internal/tui/keys.go` | Key bindings definitions |
| `internal/tui/styles/styles.go` | Lip Gloss theme and styles |
| `internal/tui/components/list.go` | Checkbox list component |
| `internal/tui/components/input.go` | Text input component |
| `internal/tui/components/table.go` | Table component |
| `internal/tui/components/help.go` | Help footer component |
| `internal/tui/views/menu.go` | Main menu view |

### Files to Modify

| File | Change |
|------|--------|
| `go.mod` | Add bubbletea, lipgloss, bubbles dependencies |
| `internal/cli/create.go` | Add TUI launch when no flags (minimal change) |

---

## Verification Plan

1. **Dependencies**: Run `go mod tidy` and verify no errors
2. **Build**: Run `go build ./...` to ensure compilation
3. **Unit Tests**: Create tests for:
   - Component rendering
   - Key binding handling
   - Style application
4. **Manual Testing**:
   - Run `gwt` with no arguments - should show main menu
   - Test keyboard navigation (j/k, arrows)
   - Test quit (q, Ctrl+C)
   - Test `gwt --no-tui` - should show error as before
5. **Cross-platform**: Test on Windows terminal (PowerShell, Windows Terminal)

---

## Dependencies on Other Phases

- **Phase 12** (TUI Views): Will add specific views for create/delete flows using this framework
- **Phase 13** (Integration): Will wire TUI views to core operations

---

## Notes

- Keep components generic and reusable for Phase 12
- Use the existing `internal/output/` package patterns for consistency
- Ensure graceful degradation when terminal doesn't support features
- Consider Windows Terminal compatibility throughout

---

## Implementation Summary

### Completed Tasks

All Phase 11 tasks have been successfully implemented:

1. **Dependencies Added**
   - `github.com/charmbracelet/bubbletea` v1.3.10
   - `github.com/charmbracelet/lipgloss` v1.1.0
   - `github.com/charmbracelet/bubbles` v0.21.0

2. **Package Structure Created**
   ```
   internal/tui/
   ├── tui.go              # Entry point and runner
   ├── model.go            # Root model with view switching
   ├── keys.go             # Key bindings and help
   ├── styles/
   │   └── styles.go       # Purple/cyan themed styles
   ├── components/
   │   ├── list.go         # Checkbox list with vim navigation
   │   ├── input.go        # Text input with validation
   │   ├── table.go        # Styled table with selection
   │   └── help.go         # Contextual help display
   └── views/
       └── menu.go         # Main menu implementation
   ```

3. **Features Implemented**
   - Purple (#7C3AED) and Cyan (#06B6D4) color theme
   - Vim-style navigation (j/k) and arrow keys
   - Reusable components ready for Phase 12
   - View switching architecture
   - Keyboard shortcuts (q to quit, enter to select)
   - Main menu with 4 options (Create, List, Delete, Configuration)

4. **CLI Integration**
   - Updated `internal/cli/create.go` to launch TUI when no flags provided
   - Respects `--no-tui` flag to fall back to error message
   - TUI launches with `gwt create` command

### Verification Results

- ✅ Build successful: `go build ./...` passes
- ✅ Binary created: `gwt.exe` builds correctly
- ✅ TUI launches: Main menu displays with proper styling
- ✅ Navigation works: Keyboard controls responsive
- ✅ Theme applied: Purple/cyan colors visible

### Files Created (9 new files)

| File | Lines | Purpose |
|------|-------|---------|
| `internal/tui/tui.go` | 75 | TUI entry point |
| `internal/tui/model.go` | 136 | Root model |
| `internal/tui/keys.go` | 76 | Key bindings |
| `internal/tui/styles/styles.go` | 68 | Theme & styles |
| `internal/tui/components/list.go` | 208 | Checkbox list |
| `internal/tui/components/input.go` | 125 | Text input |
| `internal/tui/components/table.go` | 273 | Table display |
| `internal/tui/components/help.go` | 152 | Help footer |
| `internal/tui/views/menu.go` | 171 | Main menu |

### Files Modified (2 files)

| File | Change |
|------|--------|
| `internal/cli/create.go` | Added TUI import and launch logic |
| `go.mod` | Added Bubble Tea dependencies |

### Known Issues / Future Work

- Menu selections don't navigate to views yet (Phase 12)
- No view implementations beyond menu (Phase 12)
- Components tested visually but need unit tests

### Ready for Phase 12

The TUI framework is fully functional and ready for Phase 12, which will:
- Implement create worktree flow views
- Add branch selection interface
- Implement file selection for copying
- Add Docker mode selection
- Wire views to actual operations

---

## Testing Commands

```bash
# Build and test
go build ./...
go build -o gwt.exe ./cmd/gwt

# Launch TUI
./gwt.exe create

# Test with flag
./gwt.exe create --no-tui
```
