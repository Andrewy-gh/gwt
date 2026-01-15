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
	ViewRemoteBranch
	ViewFileSelect
	ViewDockerMode
	ViewWorktreeList
	ViewDeleteConfirm
	ViewProgress
)

// Model is the root model that manages view switching
type Model struct {
	view     View
	width    int
	height   int
	keys     KeyMap
	err      error
	repoPath string

	// Sub-models for each view
	menu           *views.MenuModel
	createBranch   *views.CreateBranchModel
	createSource   *views.CreateSourceModel
	remoteBranch   *views.RemoteBranchModel
	fileSelect     *views.FileSelectModel
	dockerMode     *views.DockerModeModel
	worktreeList   *views.WorktreeListModel
	deleteConfirm  *views.DeleteConfirmModel
	fetchingView   *views.FetchingModel
	copyingView    *views.CopyingModel
	creatingView   *views.CreatingModel
	deletingView   *views.DeletingModel

	// Shared state for multi-step create flow
	createFlowState *CreateFlowState
}

// New creates a new root model
func New(repoPath string) Model {
	return Model{
		view:            ViewMenu,
		keys:            DefaultKeyMap,
		repoPath:        repoPath,
		menu:            views.NewMenuModel(),
		createFlowState: NewCreateFlowState(),
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
	case ViewCreateBranch:
		return m.updateCreateBranch(msg)
	case ViewCreateSource:
		return m.updateCreateSource(msg)
	case ViewRemoteBranch:
		return m.updateRemoteBranch(msg)
	case ViewFileSelect:
		return m.updateFileSelect(msg)
	case ViewDockerMode:
		return m.updateDockerMode(msg)
	case ViewWorktreeList:
		return m.updateWorktreeList(msg)
	case ViewDeleteConfirm:
		return m.updateDeleteConfirm(msg)
	case ViewProgress:
		return m.updateProgress(msg)
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
	case ViewCreateBranch:
		if m.createBranch != nil {
			return m.createBranch.View(m.width, m.height)
		}
	case ViewCreateSource:
		if m.createSource != nil {
			return m.createSource.View(m.width, m.height)
		}
	case ViewRemoteBranch:
		if m.remoteBranch != nil {
			return m.remoteBranch.View(m.width, m.height)
		}
	case ViewFileSelect:
		if m.fileSelect != nil {
			return m.fileSelect.View(m.width, m.height)
		}
	case ViewDockerMode:
		if m.dockerMode != nil {
			return m.dockerMode.View(m.width, m.height)
		}
	case ViewWorktreeList:
		if m.worktreeList != nil {
			return m.worktreeList.View(m.width, m.height)
		}
	case ViewDeleteConfirm:
		if m.deleteConfirm != nil {
			return m.deleteConfirm.View(m.width, m.height)
		}
	case ViewProgress:
		// Progress view can be fetching, copying, creating, or deleting
		if m.fetchingView != nil {
			return m.fetchingView.View(m.width, m.height)
		} else if m.copyingView != nil {
			return m.copyingView.View(m.width, m.height)
		} else if m.creatingView != nil {
			return m.creatingView.View(m.width, m.height)
		} else if m.deletingView != nil {
			return m.deletingView.View(m.width, m.height)
		}
	}

	return "Unknown view"
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
