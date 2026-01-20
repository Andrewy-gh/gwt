package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/key"
	"github.com/Andrewy-gh/gwt/internal/create"
	"github.com/Andrewy-gh/gwt/internal/git"
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
	ViewCleanupBranches
	ViewConfigEditor
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
	menu            *views.MenuModel
	createBranch    *views.CreateBranchModel
	createSource    *views.CreateSourceModel
	remoteBranch    *views.RemoteBranchModel
	fileSelect      *views.FileSelectModel
	dockerMode      *views.DockerModeModel
	worktreeList    *views.WorktreeListModel
	deleteConfirm   *views.DeleteConfirmModel
	fetchingView    *views.FetchingModel
	copyingView     *views.CopyingModel
	creatingView    *views.CreatingModel
	deletingView    *views.DeletingModel
	cleanupBranches *views.CleanupBranchesModel
	configEditor    *views.ConfigEditorModel

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
	case ViewCleanupBranches:
		return m.updateCleanupBranches(msg)
	case ViewConfigEditor:
		return m.updateConfigEditor(msg)
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
	case ViewCleanupBranches:
		if m.cleanupBranches != nil {
			return m.cleanupBranches.View(m.width, m.height)
		}
	case ViewConfigEditor:
		if m.configEditor != nil {
			return m.configEditor.View(m.width, m.height)
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

	// Handle menu selection
	if m.menu.HasSelection() {
		selection := m.menu.GetSelection()
		m.menu.ClearSelection() // Clear selection for next time

		switch selection {
		case "create":
			// Start create worktree flow
			m.createFlowState.Reset()
			m.view = ViewCreateBranch
			m.createBranch = views.NewCreateBranchModel(m.repoPath)
			return m, m.createBranch.Init()

		case "list", "delete":
			// Go to worktree list view (delete is handled within the list)
			m.view = ViewWorktreeList
			m.worktreeList = views.NewWorktreeListModel(m.repoPath)
			return m, m.worktreeList.Init()

		case "cleanup":
			// Go to cleanup branches view
			m.view = ViewCleanupBranches
			m.cleanupBranches = views.NewCleanupBranchesModel(m.repoPath)
			return m, m.cleanupBranches.Init()

		case "config":
			// Go to config editor view
			m.view = ViewConfigEditor
			m.configEditor = views.NewConfigEditorModel(m.repoPath)
			return m, m.configEditor.Init()
		}
	}

	return m, cmd
}

// updateCreateBranch handles updates for the create branch view
func (m Model) updateCreateBranch(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Check for Esc key to return to menu
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if key.Matches(keyMsg, key.NewBinding(key.WithKeys("esc"))) {
			m.view = ViewMenu
			return m, nil
		}
	}

	// Update the view
	var cmd tea.Cmd
	m.createBranch, cmd = m.createBranch.Update(msg)

	// Check if view is complete
	if m.createBranch.IsComplete() {
		// Save to flow state
		m.createFlowState.BranchSpec = m.createBranch.GetBranchSpec()
		m.createFlowState.SourceType = m.createFlowState.BranchSpec.Source
		m.createFlowState.BranchInput = m.createFlowState.BranchSpec.BranchName
		m.createFlowState.CalculateTotalSteps()
		m.createFlowState.PushView(ViewCreateBranch)

		// Determine next view based on source type
		nextView := m.createFlowState.NextView()
		m.view = nextView

		switch nextView {
		case ViewCreateSource:
			m.createSource = views.NewCreateSourceModel(m.repoPath)
			return m, m.createSource.Init()
		case ViewRemoteBranch:
			m.remoteBranch = views.NewRemoteBranchModel(m.repoPath)
			return m, m.remoteBranch.Init()
		case ViewFileSelect:
			m.fileSelect = views.NewFileSelectModel(m.repoPath)
			return m, m.fileSelect.Init()
		}
	}

	return m, cmd
}

// updateCreateSource handles updates for the create source view
func (m Model) updateCreateSource(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Check for Esc key to go back or return to menu
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if key.Matches(keyMsg, key.NewBinding(key.WithKeys("esc"))) {
			// Go back to previous view or menu
			if prevView, ok := m.createFlowState.PopView(); ok {
				m.view = prevView
				return m, nil
			}
			m.view = ViewMenu
			return m, nil
		}
	}

	// Update the view
	var cmd tea.Cmd
	m.createSource, cmd = m.createSource.Update(msg)

	// Check if view is complete
	if m.createSource.IsComplete() {
		// Save to flow state
		m.createFlowState.StartPoint = m.createSource.GetStartPoint()
		m.createFlowState.PushView(ViewCreateSource)

		// Move to file select
		m.view = ViewFileSelect
		m.fileSelect = views.NewFileSelectModel(m.repoPath)
		return m, m.fileSelect.Init()
	}

	return m, cmd
}

// updateRemoteBranch handles updates for the remote branch view
func (m Model) updateRemoteBranch(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Check for Esc key to go back or return to menu
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if key.Matches(keyMsg, key.NewBinding(key.WithKeys("esc"))) {
			// Go back to previous view or menu
			if prevView, ok := m.createFlowState.PopView(); ok {
				m.view = prevView
				return m, nil
			}
			m.view = ViewMenu
			return m, nil
		}
	}

	// Update the view
	var cmd tea.Cmd
	m.remoteBranch, cmd = m.remoteBranch.Update(msg)

	// Check if view is complete
	if m.remoteBranch.IsComplete() {
		// Save to flow state
		m.createFlowState.SelectedRemote = m.remoteBranch.GetSelected()
		m.createFlowState.PushView(ViewRemoteBranch)

		// Move to file select
		m.view = ViewFileSelect
		m.fileSelect = views.NewFileSelectModel(m.repoPath)
		return m, m.fileSelect.Init()
	}

	return m, cmd
}

// updateFileSelect handles updates for the file select view
func (m Model) updateFileSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Check for Esc key to go back or return to menu
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if key.Matches(keyMsg, key.NewBinding(key.WithKeys("esc"))) {
			// Go back to previous view or menu
			if prevView, ok := m.createFlowState.PopView(); ok {
				m.view = prevView
				return m, nil
			}
			m.view = ViewMenu
			return m, nil
		}
	}

	// Update the view
	var cmd tea.Cmd
	m.fileSelect, cmd = m.fileSelect.Update(msg)

	// Check if view is complete
	if m.fileSelect.IsComplete() {
		// Save to flow state
		m.createFlowState.FileSelection = m.fileSelect.GetSelection()

		// Calculate target directory if not already set
		if m.createFlowState.TargetDir == "" && m.createFlowState.BranchSpec != nil {
			// Get main worktree to calculate target path
			mainWorktree, err := git.GetMainWorktree(m.repoPath)
			if err != nil {
				m.err = fmt.Errorf("failed to get main worktree: %w", err)
				m.view = ViewMenu
				return m, nil
			}

			// Generate target directory path
			m.createFlowState.TargetDir = create.GenerateWorktreePath(
				mainWorktree.Path,
				m.createFlowState.BranchSpec.BranchName,
			)
		}

		m.createFlowState.PushView(ViewFileSelect)

		// Move to docker mode
		m.view = ViewDockerMode
		m.dockerMode = views.NewDockerModeModel(m.repoPath)
		return m, m.dockerMode.Init()
	}

	return m, cmd
}

// updateDockerMode handles updates for the docker mode view
func (m Model) updateDockerMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Check for Esc key to go back or return to menu
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if key.Matches(keyMsg, key.NewBinding(key.WithKeys("esc"))) {
			// Go back to previous view or menu
			if prevView, ok := m.createFlowState.PopView(); ok {
				m.view = prevView
				return m, nil
			}
			m.view = ViewMenu
			return m, nil
		}
	}

	// Update the view
	var cmd tea.Cmd
	m.dockerMode, cmd = m.dockerMode.Update(msg)

	// Check if view is complete
	if m.dockerMode.IsComplete() {
		// Save to flow state
		m.createFlowState.DockerMode = m.dockerMode.GetSelectedMode()
		m.createFlowState.ComposeDetected = m.dockerMode.IsComposeDetected()
		m.createFlowState.ComposeConfig = m.dockerMode.GetComposeConfig()
		m.createFlowState.ComposeFiles = m.dockerMode.GetComposeFiles()

		// Start the worktree creation operation
		m.view = ViewProgress
		m.creatingView = views.NewCreatingModel()
		return m, tea.Batch(
			m.creatingView.Init(),
			createWorktreeCmd(m.createFlowState, m.repoPath),
		)
	}

	return m, cmd
}

// updateWorktreeList handles updates for the worktree list view
func (m Model) updateWorktreeList(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Check for Esc key or cancel to return to menu
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if key.Matches(keyMsg, key.NewBinding(key.WithKeys("esc"))) {
			m.view = ViewMenu
			return m, nil
		}
	}

	// Update the view
	var cmd tea.Cmd
	m.worktreeList, cmd = m.worktreeList.Update(msg)

	// Check if user cancelled
	if m.worktreeList.ShouldCancel() {
		m.view = ViewMenu
		return m, nil
	}

	// Check if user requested deletion
	if m.worktreeList.ShouldDelete() {
		// Get selected paths and create delete confirmation view
		selectedPaths := m.worktreeList.GetSelectedPaths()
		if len(selectedPaths) > 0 {
			m.view = ViewDeleteConfirm
			m.deleteConfirm = views.NewDeleteConfirmModel(m.repoPath, selectedPaths)
			return m, m.deleteConfirm.Init()
		}
	}

	return m, cmd
}

// updateDeleteConfirm handles updates for the delete confirmation view
func (m Model) updateDeleteConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Update the view
	var cmd tea.Cmd
	m.deleteConfirm, cmd = m.deleteConfirm.Update(msg)

	// Check if user cancelled
	if m.deleteConfirm.IsCancelled() {
		// Go back to worktree list
		m.view = ViewWorktreeList
		return m, nil
	}

	// Check if user confirmed
	if m.deleteConfirm.IsConfirmed() {
		// Get non-blocked targets and start deletion
		targets := m.deleteConfirm.GetNonBlockedTargets()
		if len(targets) > 0 {
			// Extract paths from targets
			paths := make([]string, 0, len(targets))
			for _, target := range targets {
				paths = append(paths, target.Worktree.Path)
			}

			m.view = ViewProgress
			m.deletingView = views.NewDeletingModel(paths)
			return m, tea.Batch(
				m.deletingView.Init(),
				deleteWorktreesCmd(m.repoPath, paths, false),
			)
		} else {
			// No targets to delete, return to worktree list
			m.view = ViewWorktreeList
			return m, nil
		}
	}

	return m, cmd
}

// updateProgress handles updates for progress views
func (m Model) updateProgress(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case CreateCompleteMsg:
		// Worktree creation completed
		if msg.Error != nil {
			m.err = msg.Error
			m.view = ViewMenu
			return m, nil
		}

		// Mark all stages as complete for visual feedback before returning
		if m.creatingView != nil {
			for i := 0; i < 4; i++ {
				m.creatingView.CompleteStage(i)
			}
		}

		// Success - return to menu after a brief moment
		m.view = ViewMenu
		// Reset create flow state for next time
		m.createFlowState.Reset()
		return m, nil

	case DeleteCompleteMsg:
		// Deletion completed
		if msg.Error != nil {
			m.err = msg.Error
		}

		// Return to worktree list to show updated list
		m.view = ViewWorktreeList
		m.worktreeList = views.NewWorktreeListModel(m.repoPath)
		return m, m.worktreeList.Init()

	case CreateProgressMsg:
		// Update creating view with progress
		if m.creatingView != nil {
			// Update the stage based on the message
			if msg.StageIndex >= 0 && msg.StageIndex < 4 {
				// Mark previous stages as complete
				for i := 0; i < msg.StageIndex; i++ {
					m.creatingView.CompleteStage(i)
				}
				// Set current stage to running
				m.creatingView.SetStage(msg.StageIndex, views.StageRunning, msg.Message)
			}
		}
		return m, nil

	case DeleteProgressMsg:
		// Update deleting view with progress
		if m.deletingView != nil {
			var cmd tea.Cmd
			m.deletingView, cmd = m.deletingView.Update(msg)
			return m, cmd
		}
	}

	// Delegate to active progress view
	var cmd tea.Cmd
	if m.fetchingView != nil {
		m.fetchingView, cmd = m.fetchingView.Update(msg)
	} else if m.copyingView != nil {
		m.copyingView, cmd = m.copyingView.Update(msg)
	} else if m.creatingView != nil {
		m.creatingView, cmd = m.creatingView.Update(msg)
	} else if m.deletingView != nil {
		m.deletingView, cmd = m.deletingView.Update(msg)
	}

	return m, cmd
}

// updateCleanupBranches handles updates for the cleanup branches view
func (m Model) updateCleanupBranches(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Check for Esc key to return to menu
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if key.Matches(keyMsg, key.NewBinding(key.WithKeys("esc", "q"))) {
			m.view = ViewMenu
			return m, nil
		}
	}

	// Update the view
	var cmd tea.Cmd
	m.cleanupBranches, cmd = m.cleanupBranches.Update(msg)

	// Check if user cancelled
	if m.cleanupBranches.IsCancelled() {
		m.view = ViewMenu
		return m, nil
	}

	return m, cmd
}

// updateConfigEditor handles updates for the config editor view
func (m Model) updateConfigEditor(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Update the view
	var cmd tea.Cmd
	m.configEditor, cmd = m.configEditor.Update(msg)

	// Check if user cancelled
	if m.configEditor.IsCancelled() {
		m.view = ViewMenu
		return m, nil
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
