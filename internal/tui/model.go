package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/key"
	"github.com/Andrewy-gh/gwt/internal/tui/views"
)

// View represents different views in the TUI
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

// Model is the root model that manages view switching
type Model struct {
	view   View
	width  int
	height int
	keys   KeyMap
	err    error

	// Sub-models for each view
	menu *views.MenuModel
	// Additional view models will be added in Phase 12
}

// New creates a new root model
func New() Model {
	return Model{
		view: ViewMenu,
		keys: DefaultKeyMap,
		menu: views.NewMenuModel(),
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		// Global quit handler
		if key.Matches(msg, m.keys.Quit) {
			return m, tea.Quit
		}
	}

	// Delegate to current view
	switch m.view {
	case ViewMenu:
		return m.updateMenu(msg)
	// Additional view handlers will be added in Phase 12
	default:
		return m, nil
	}
}

// View renders the current view
func (m Model) View() string {
	if m.err != nil {
		return m.renderError()
	}

	switch m.view {
	case ViewMenu:
		return m.menu.View(m.width, m.height)
	// Additional view renderers will be added in Phase 12
	default:
		return "Unknown view"
	}
}

// updateMenu handles updates for the menu view
func (m Model) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.menu, cmd = m.menu.Update(msg)

	// Handle menu actions
	if m.menu.ShouldQuit() {
		return m, tea.Quit
	}

	// Handle menu selection (Phase 12 will implement view transitions)
	if m.menu.HasSelection() {
		// Future: transition to selected view
		// For now, just stay on menu
	}

	return m, cmd
}

// renderError renders an error message
func (m Model) renderError() string {
	return "Error: " + m.err.Error() + "\n\nPress q to quit."
}

// setView switches to a different view
func (m *Model) setView(view View) {
	m.view = view
}

// getView returns the current view
func (m Model) getView() View {
	return m.view
}

// setError sets an error on the model
func (m *Model) setError(err error) {
	m.err = err
}

// clearError clears the error
func (m *Model) clearError() {
	m.err = nil
}

// Width returns the current terminal width
func (m Model) Width() int {
	return m.width
}

// Height returns the current terminal height
func (m Model) Height() int {
	return m.height
}
