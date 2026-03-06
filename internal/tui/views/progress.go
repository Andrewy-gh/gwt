package views

import (
	"fmt"
	"strings"

	"github.com/Andrewy-gh/gwt/internal/copy"
	"github.com/Andrewy-gh/gwt/internal/tui/components"
	"github.com/Andrewy-gh/gwt/internal/tui/styles"
	tea "github.com/charmbracelet/bubbletea"
)

// ProgressStage represents a stage in a multi-stage operation
type ProgressStage struct {
	Name    string
	Status  StageStatus
	Message string
}

// StageStatus represents the status of a stage
type StageStatus int

const (
	StagePending StageStatus = iota
	StageRunning
	StageComplete
	StageError
)

// FetchingModel is the view for remote fetch operations
type FetchingModel struct {
	spinner *components.Spinner
	message string
	width   int
	height  int
}

// NewFetchingModel creates a new fetching view
func NewFetchingModel(message string) *FetchingModel {
	if message == "" {
		message = "Fetching remote branches..."
	}

	return &FetchingModel{
		spinner: components.NewSpinner(message),
		message: message,
	}
}

// Init initializes the model
func (m *FetchingModel) Init() tea.Cmd {
	return m.spinner.Init()
}

// Update handles messages
func (m *FetchingModel) Update(msg tea.Msg) (*FetchingModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case components.SpinnerTickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

// View renders the view
func (m *FetchingModel) View(width, height int) string {
	m.width = width
	m.height = height

	var b strings.Builder
	b.WriteString(styles.Title.Render("Fetching Remote Branches"))
	b.WriteString("\n\n")
	b.WriteString(m.spinner.View())
	b.WriteString("\n\n")
	b.WriteString(styles.Subtitle.Render("Please wait..."))
	b.WriteString("\n")

	return b.String()
}

// CopyingModel is the view for file copy operations
type CopyingModel struct {
	progressBar   *components.ProgressBar
	currentFile   string
	filesComplete int
	filesTotal    int
	bytesComplete int64
	bytesTotal    int64
	width         int
	height        int
}

// NewCopyingModel creates a new copying view
func NewCopyingModel(filesTotal int, bytesTotal int64) *CopyingModel {
	progressBar := components.NewProgressBar(50)
	progressBar.Label = "Copying files..."
	progressBar.ShowCount = true
	progressBar.ShowPercent = true

	return &CopyingModel{
		progressBar: progressBar,
		filesTotal:  filesTotal,
		bytesTotal:  bytesTotal,
	}
}

// Init initializes the model
func (m *CopyingModel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m *CopyingModel) Update(msg tea.Msg) (*CopyingModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

// UpdateProgress updates the progress with current file and byte counts
func (m *CopyingModel) UpdateProgress(file string, filesComplete int, bytesComplete int64) {
	m.currentFile = file
	m.filesComplete = filesComplete
	m.bytesComplete = bytesComplete

	// Update progress bar
	m.progressBar.Current = m.bytesComplete
	m.progressBar.Total = m.bytesTotal
}

// View renders the view
func (m *CopyingModel) View(width, height int) string {
	m.width = width
	m.height = height

	var b strings.Builder

	// Title
	b.WriteString(styles.Title.Render("Copying Files"))
	b.WriteString("\n\n")

	// Progress bar
	b.WriteString(m.progressBar.View())
	b.WriteString("\n\n")

	// File progress
	b.WriteString(styles.Subtitle.Render(fmt.Sprintf("Files: %d / %d", m.filesComplete, m.filesTotal)))
	b.WriteString("\n")

	// Byte progress
	b.WriteString(styles.Subtitle.Render(fmt.Sprintf("Progress: %s / %s",
		copy.FormatSize(m.bytesComplete),
		copy.FormatSize(m.bytesTotal))))
	b.WriteString("\n\n")

	// Current file
	if m.currentFile != "" {
		currentFileDisplay := m.currentFile
		if len(currentFileDisplay) > 60 {
			currentFileDisplay = "..." + currentFileDisplay[len(currentFileDisplay)-57:]
		}
		b.WriteString(styles.MutedText.Render("Copying: " + currentFileDisplay))
		b.WriteString("\n")
	}

	return b.String()
}

// CreatingModel is the view for multi-stage worktree creation
type CreatingModel struct {
	stages       []ProgressStage
	currentStage int
	width        int
	height       int
	spinner      *components.Spinner
}

// NewCreatingModel creates a new creating view
func NewCreatingModel() *CreatingModel {
	stages := []ProgressStage{
		{Name: "Creating worktree", Status: StagePending, Message: ""},
		{Name: "Copying files", Status: StagePending, Message: ""},
		{Name: "Setting up Docker", Status: StagePending, Message: ""},
		{Name: "Running hooks", Status: StagePending, Message: ""},
	}

	return &CreatingModel{
		stages:       stages,
		currentStage: 0,
		spinner:      components.NewSpinner("Creating worktree..."),
	}
}

// Init initializes the model
func (m *CreatingModel) Init() tea.Cmd {
	return m.spinner.Init()
}

// Update handles messages
func (m *CreatingModel) Update(msg tea.Msg) (*CreatingModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case components.SpinnerTickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

// SetStage sets the current stage and its status
func (m *CreatingModel) SetStage(stageIndex int, status StageStatus, message string) {
	if stageIndex >= 0 && stageIndex < len(m.stages) {
		m.stages[stageIndex].Status = status
		m.stages[stageIndex].Message = message
		if status == StageRunning {
			m.currentStage = stageIndex
			m.spinner.SetMessage(m.stages[stageIndex].Name + "...")
		}
	}
}

// CompleteStage marks a stage as complete
func (m *CreatingModel) CompleteStage(stageIndex int) {
	m.SetStage(stageIndex, StageComplete, "")
}

// ErrorStage marks a stage as having an error
func (m *CreatingModel) ErrorStage(stageIndex int, err error) {
	m.SetStage(stageIndex, StageError, err.Error())
}

// View renders the view
func (m *CreatingModel) View(width, height int) string {
	m.width = width
	m.height = height

	var b strings.Builder

	// Title
	b.WriteString(styles.Title.Render("Creating Worktree"))
	b.WriteString("\n\n")

	// Stages
	for i, stage := range m.stages {
		var statusIcon string
		var line string

		switch stage.Status {
		case StagePending:
			statusIcon = "○"
			line = styles.MutedText.Render(fmt.Sprintf("%s %s", statusIcon, stage.Name))
		case StageRunning:
			statusIcon = "◐"
			line = styles.Selected.Render(fmt.Sprintf("%s %s", statusIcon, stage.Name))
		case StageComplete:
			statusIcon = "✓"
			line = styles.SuccessText.Render(fmt.Sprintf("%s %s", statusIcon, stage.Name))
		case StageError:
			statusIcon = "✘"
			line = styles.ErrorText.Render(fmt.Sprintf("%s %s", statusIcon, stage.Name))
		}

		// Stage line
		b.WriteString(line)
		b.WriteString("\n")

		// Message (if any)
		if stage.Message != "" {
			b.WriteString("  ")
			if stage.Status == StageError {
				b.WriteString(styles.ErrorText.Render(stage.Message))
			} else {
				b.WriteString(styles.Subtitle.Render(stage.Message))
			}
			b.WriteString("\n")
		}

		// Show spinner for running stage
		if stage.Status == StageRunning && i == m.currentStage {
			b.WriteString("  ")
			b.WriteString(m.spinner.View())
			b.WriteString("\n")
		}

		b.WriteString("\n")
	}

	return b.String()
}

// DeletingModel is the view for worktree deletion progress
type DeletingModel struct {
	targets     []string
	current     int
	currentPath string
	failed      []string
	width       int
	height      int
	spinner     *components.Spinner
}

// NewDeletingModel creates a new deleting view
func NewDeletingModel(targets []string) *DeletingModel {
	return &DeletingModel{
		targets: targets,
		current: 0,
		failed:  make([]string, 0),
		spinner: components.NewSpinner("Deleting worktrees..."),
	}
}

// Init initializes the model
func (m *DeletingModel) Init() tea.Cmd {
	return m.spinner.Init()
}

// Update handles messages
func (m *DeletingModel) Update(msg tea.Msg) (*DeletingModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case components.SpinnerTickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

// SetCurrent sets the current worktree being deleted
func (m *DeletingModel) SetCurrent(index int, path string) {
	m.current = index
	m.currentPath = path
}

// AddFailed adds a worktree that failed to delete
func (m *DeletingModel) AddFailed(path string) {
	m.failed = append(m.failed, path)
}

// View renders the view
func (m *DeletingModel) View(width, height int) string {
	m.width = width
	m.height = height

	var b strings.Builder

	// Title
	b.WriteString(styles.Title.Render("Deleting Worktrees"))
	b.WriteString("\n\n")

	// Progress
	b.WriteString(styles.Subtitle.Render(fmt.Sprintf("Progress: %d / %d", m.current, len(m.targets))))
	b.WriteString("\n\n")

	// Spinner
	b.WriteString(m.spinner.View())
	b.WriteString("\n\n")

	// Current path
	if m.currentPath != "" {
		displayPath := m.currentPath
		if len(displayPath) > 60 {
			displayPath = "..." + displayPath[len(displayPath)-57:]
		}
		b.WriteString(styles.MutedText.Render("Deleting: " + displayPath))
		b.WriteString("\n")
	}

	// Failed (if any)
	if len(m.failed) > 0 {
		b.WriteString("\n")
		b.WriteString(styles.ErrorText.Render(fmt.Sprintf("%d worktree(s) failed to delete", len(m.failed))))
		b.WriteString("\n")
	}

	return b.String()
}
