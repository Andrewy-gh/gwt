package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/key"
	"github.com/Andrewy-gh/gwt/internal/docker"
	"github.com/Andrewy-gh/gwt/internal/tui/components"
	"github.com/Andrewy-gh/gwt/internal/tui/styles"
)

// DockerModeModel is the Docker mode selection view
type DockerModeModel struct {
	radioList       *components.RadioList
	spinner         *components.Spinner
	composeDetected bool
	composeConfig   *docker.ComposeConfig
	composeFiles    []docker.ComposeFile
	selectedMode    string // "none", "shared", "new"
	repoPath        string
	complete        bool
	detecting       bool
	width           int
	height          int
	err             error
}

// NewDockerModeModel creates a new Docker mode selection view
func NewDockerModeModel(repoPath string) *DockerModeModel {
	// Radio list for Docker mode options
	modeItems := []components.RadioItem{
		{
			Label:       "None",
			Description: "Skip Docker setup (no compose file changes)",
			Value:       "none",
		},
		{
			Label:       "Shared",
			Description: "Symlink data directories (good for read-only data)",
			Value:       "shared",
		},
		{
			Label:       "New",
			Description: "Isolated containers (copy data, rename volumes)",
			Value:       "new",
		},
	}

	radioList := components.NewRadioList("Docker Mode", modeItems, 8)
	radioList.SetSelected(0) // Default to "None"

	// Spinner for detection
	spinner := components.NewSpinner("Detecting Docker Compose files...")

	m := &DockerModeModel{
		radioList: radioList,
		spinner:   spinner,
		repoPath:  repoPath,
	}

	// Start Docker detection
	m.startDetection()

	return m
}

// Init initializes the model
func (m *DockerModeModel) Init() tea.Cmd {
	return m.spinner.Init()
}

// Update handles messages
func (m *DockerModeModel) Update(msg tea.Msg) (*DockerModeModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.detecting {
			// Ignore input while detecting
			return m, nil
		}

		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			// Go back
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			// Submit selection
			selected := m.radioList.GetSelected()
			if selected != nil {
				m.selectedMode = selected.Value
				m.complete = true
			}
			return m, nil

		default:
			// Delegate to radio list
			cmd := m.radioList.Update(msg)
			cmds = append(cmds, cmd)
		}

	case components.SpinnerTickMsg:
		if m.detecting {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	// Update spinner
	if m.detecting {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the view
func (m *DockerModeModel) View(width, height int) string {
	m.width = width
	m.height = height

	var b strings.Builder

	// Title
	b.WriteString(styles.Title.Render("Create Worktree - Docker Configuration"))
	b.WriteString("\n\n")

	if m.detecting {
		// Show spinner while detecting
		b.WriteString(m.spinner.View())
		b.WriteString("\n")
	} else {
		// Description
		b.WriteString(styles.Subtitle.Render("Choose how to handle Docker Compose configuration."))
		b.WriteString("\n\n")

		// Detection result
		if m.composeDetected {
			b.WriteString(styles.SuccessText.Render("✓ Docker Compose detected"))
			b.WriteString("\n")
			for _, file := range m.composeFiles {
				b.WriteString(styles.MutedText.Render(fmt.Sprintf("  • %s", file.Path)))
				b.WriteString("\n")
			}
			b.WriteString("\n")
		} else {
			b.WriteString(styles.WarningText.Render("⚠ No Docker Compose files detected"))
			b.WriteString("\n")
			b.WriteString(styles.MutedText.Render("  You can still select a mode for future use."))
			b.WriteString("\n\n")
		}

		// Mode selector
		b.WriteString(m.radioList.View())
		b.WriteString("\n")

		// Info box for selected mode
		selected := m.radioList.GetSelected()
		if selected != nil {
			b.WriteString(m.renderModeInfo(selected.Value))
			b.WriteString("\n")
		}

		// Error message
		if m.err != nil {
			b.WriteString(styles.ErrorText.Render("✘ " + m.err.Error()))
			b.WriteString("\n\n")
		}

		// Help text
		helpItems := []string{
			"↑/↓: navigate",
			"enter: confirm",
			"esc: back",
		}

		b.WriteString(styles.Help.Render(strings.Join(helpItems, " • ")))
	}

	return b.String()
}

// renderModeInfo renders information about the selected Docker mode
func (m *DockerModeModel) renderModeInfo(mode string) string {
	var info string

	switch mode {
	case "none":
		info = "No Docker setup will be performed.\n" +
			"The docker-compose.yml will remain unchanged.\n" +
			"Use this if you don't use Docker or will configure it manually."

	case "shared":
		info = "Data directories will be symlinked to the main worktree.\n" +
			"Good for read-only data like databases (dev/test environments).\n" +
			"⚠ Warning: Changes in one worktree affect all others."
		if !m.composeDetected {
			info += "\n\n⚠ Note: No compose file detected. This mode won't do anything."
		}

	case "new":
		info = "Fully isolated Docker environment.\n" +
			"• Data directories are copied\n" +
			"• Volume names are renamed with branch suffix\n" +
			"• Port mappings are offset to avoid conflicts\n" +
			"Best for full isolation between worktrees."
		if !m.composeDetected {
			info += "\n\n⚠ Note: No compose file detected. This mode won't do anything."
		}
	}

	return styles.Box.Render(info)
}

// startDetection detects Docker Compose files
func (m *DockerModeModel) startDetection() {
	m.detecting = true
	m.spinner.Start()

	go func() {
		// Detect compose files
		files, err := docker.DetectComposeFiles(m.repoPath)
		if err != nil {
			m.err = fmt.Errorf("failed to detect Docker Compose files: %w", err)
			m.detecting = false
			m.spinner.Stop()
			return
		}

		if len(files) > 0 {
			m.composeDetected = true
			m.composeFiles = files

			// Try to parse the first compose file
			paths := docker.GetComposePaths(files)
			config, err := docker.ParseComposeFiles(paths)
			if err == nil {
				m.composeConfig = config
			}

			// If compose detected, default to "shared" mode
			m.radioList.SetSelectedByValue("shared")
		}

		m.detecting = false
		m.spinner.Stop()
	}()
}

// IsComplete returns true if the view is complete
func (m *DockerModeModel) IsComplete() bool {
	return m.complete
}

// GetSelectedMode returns the selected Docker mode
func (m *DockerModeModel) GetSelectedMode() string {
	return m.selectedMode
}

// IsComposeDetected returns whether Docker Compose was detected
func (m *DockerModeModel) IsComposeDetected() bool {
	return m.composeDetected
}

// GetComposeConfig returns the parsed compose configuration
func (m *DockerModeModel) GetComposeConfig() *docker.ComposeConfig {
	return m.composeConfig
}

// GetComposeFiles returns the detected compose files
func (m *DockerModeModel) GetComposeFiles() []docker.ComposeFile {
	return m.composeFiles
}

// Reset resets the view state
func (m *DockerModeModel) Reset() {
	m.radioList.SetSelected(0)
	m.selectedMode = ""
	m.complete = false
	m.err = nil
	m.detecting = false
	m.composeDetected = false
	m.composeConfig = nil
	m.composeFiles = nil
	m.startDetection()
}

// SetRepoPath updates the repository path
func (m *DockerModeModel) SetRepoPath(path string) {
	m.repoPath = path
	m.Reset()
}

// GetError returns the current error
func (m *DockerModeModel) GetError() error {
	return m.err
}
