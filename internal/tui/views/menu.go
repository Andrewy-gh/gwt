package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/key"
	"github.com/Andrewy-gh/gwt/internal/tui/styles"
)

// MenuItem represents a menu item
type MenuItem struct {
	Title       string
	Description string
	Action      func() tea.Cmd
}

// MenuModel is the main menu view
type MenuModel struct {
	cursor      int
	items       []MenuItem
	selected    bool
	shouldQuit  bool
}

// NewMenuModel creates a new menu model
func NewMenuModel() *MenuModel {
	return &MenuModel{
		cursor: 0,
		items: []MenuItem{
			{
				Title:       "Create Worktree",
				Description: "Create a new worktree from a branch",
				Action:      nil, // Will be implemented in Phase 12
			},
			{
				Title:       "List Worktrees",
				Description: "View and manage existing worktrees",
				Action:      nil, // Will be implemented in Phase 12
			},
			{
				Title:       "Delete Worktree",
				Description: "Remove worktrees with safety checks",
				Action:      nil, // Will be implemented in Phase 12
			},
			{
				Title:       "Configuration",
				Description: "View and edit GWT settings",
				Action:      nil, // Will be implemented in Phase 12
			},
		},
		selected:   false,
		shouldQuit: false,
	}
}

// Init initializes the menu model
func (m *MenuModel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m *MenuModel) Update(msg tea.Msg) (*MenuModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
			m.CursorUp()
		case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
			m.CursorDown()
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter", " "))):
			m.selected = true
			if m.items[m.cursor].Action != nil {
				return m, m.items[m.cursor].Action()
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("q", "ctrl+c"))):
			m.shouldQuit = true
		}
	}
	return m, nil
}

// View renders the menu
func (m *MenuModel) View(width, height int) string {
	var b strings.Builder

	// Title
	title := "GWT - Git Worktree Manager"
	b.WriteString(styles.Title.Render(title))
	b.WriteString("\n")
	b.WriteString(styles.Subtitle.Render("Interactive worktree management"))
	b.WriteString("\n\n")

	// Menu items
	for i, item := range m.items {
		cursor := styles.NoCursor
		if i == m.cursor {
			cursor = styles.CursorSymbol
		}

		title := item.Title
		description := item.Description

		// Apply styles
		if i == m.cursor {
			cursor = styles.Cursor.Render(cursor)
			title = styles.Selected.Render(title)
		}

		b.WriteString(fmt.Sprintf("%s %s\n", cursor, title))
		b.WriteString(fmt.Sprintf("  %s\n", styles.Subtitle.Render(description)))
		b.WriteString("\n")
	}

	// Help footer
	helpText := "↑/k: up • ↓/j: down • enter: select • q: quit"
	b.WriteString(styles.Help.Render(helpText))

	return b.String()
}

// CursorUp moves the cursor up
func (m *MenuModel) CursorUp() {
	if m.cursor > 0 {
		m.cursor--
	}
}

// CursorDown moves the cursor down
func (m *MenuModel) CursorDown() {
	if m.cursor < len(m.items)-1 {
		m.cursor++
	}
}

// SelectedItem returns the currently selected menu item
func (m *MenuModel) SelectedItem() MenuItem {
	if m.cursor >= 0 && m.cursor < len(m.items) {
		return m.items[m.cursor]
	}
	return MenuItem{}
}

// HasSelection returns true if a menu item has been selected
func (m *MenuModel) HasSelection() bool {
	return m.selected
}

// ClearSelection clears the selection state
func (m *MenuModel) ClearSelection() {
	m.selected = false
}

// GetSelection returns a string identifier for the selected menu item
func (m *MenuModel) GetSelection() string {
	switch m.cursor {
	case 0:
		return "create"
	case 1:
		return "list"
	case 2:
		return "delete"
	case 3:
		return "config"
	default:
		return ""
	}
}

// ShouldQuit returns true if the user wants to quit
func (m *MenuModel) ShouldQuit() bool {
	return m.shouldQuit
}

// GetCursor returns the current cursor position
func (m *MenuModel) GetCursor() int {
	return m.cursor
}

// SetCursor sets the cursor position
func (m *MenuModel) SetCursor(cursor int) {
	if cursor >= 0 && cursor < len(m.items) {
		m.cursor = cursor
	}
}

// AddItem adds a new menu item
func (m *MenuModel) AddItem(item MenuItem) {
	m.items = append(m.items, item)
}

// RemoveItem removes a menu item by index
func (m *MenuModel) RemoveItem(index int) {
	if index >= 0 && index < len(m.items) {
		m.items = append(m.items[:index], m.items[index+1:]...)
		if m.cursor >= len(m.items) {
			m.cursor = len(m.items) - 1
		}
	}
}

// ItemCount returns the number of menu items
func (m *MenuModel) ItemCount() int {
	return len(m.items)
}
