package views

import (
	"fmt"
	"strings"

	"github.com/Andrewy-gh/gwt/internal/config"
	"github.com/Andrewy-gh/gwt/internal/copy"
	"github.com/Andrewy-gh/gwt/internal/tui/components"
	"github.com/Andrewy-gh/gwt/internal/tui/styles"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// FileSelectModel is the file selection view
type FileSelectModel struct {
	fileList     *components.CheckboxList
	spinner      *components.Spinner
	selection    *copy.Selection
	ignoredFiles []copy.IgnoredFile
	repoPath     string
	complete     bool
	loading      bool
	totalSize    string
	selectedSize string
	width        int
	height       int
	err          error
}

// NewFileSelectModel creates a new file selection view
func NewFileSelectModel(repoPath string) *FileSelectModel {
	// Create empty checkbox list (will be populated after discovery)
	fileList := components.NewCheckboxList("Select Files to Copy", []components.CheckboxItem{}, 15)

	// Spinner for file discovery
	spinner := components.NewSpinner("Discovering ignored files...")

	m := &FileSelectModel{
		fileList: fileList,
		spinner:  spinner,
		repoPath: repoPath,
	}

	// Start file discovery
	m.startDiscovery()

	return m
}

// Init initializes the model
func (m *FileSelectModel) Init() tea.Cmd {
	return m.spinner.Init()
}

// Update handles messages
func (m *FileSelectModel) Update(msg tea.Msg) (*FileSelectModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.loading {
			// Ignore most input while loading
			return m, nil
		}

		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			// Go back
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			// Complete selection
			m.finalizeSelection()
			m.complete = true
			return m, nil

		default:
			// Delegate to checkbox list
			cmd := m.fileList.Update(msg, nil)
			cmds = append(cmds, cmd)

			// Update selected size after each change
			m.updateSizes()
		}

	case components.SpinnerTickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	// Update spinner
	if m.loading {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the view
func (m *FileSelectModel) View(width, height int) string {
	m.width = width
	m.height = height

	var b strings.Builder

	// Title
	b.WriteString(styles.Title.Render("Create Worktree - Select Files to Copy"))
	b.WriteString("\n\n")

	if m.loading {
		// Show spinner while discovering files
		b.WriteString(m.spinner.View())
		b.WriteString("\n\n")
		b.WriteString(styles.Subtitle.Render("Scanning for ignored files..."))
		b.WriteString("\n")
	} else {
		// Description
		b.WriteString(styles.Subtitle.Render("Select which ignored files to copy to the new worktree."))
		b.WriteString("\n\n")

		// File list
		if len(m.fileList.Items) == 0 {
			b.WriteString(styles.MutedText.Render("No ignored files found"))
			b.WriteString("\n")
		} else {
			b.WriteString(m.fileList.View())
			b.WriteString("\n")

			// Size summary
			b.WriteString(styles.Box.Render(
				fmt.Sprintf("Total: %s  |  Selected: %s", m.totalSize, m.selectedSize),
			))
			b.WriteString("\n")
		}

		// Error message
		if m.err != nil {
			b.WriteString(styles.ErrorText.Render("✘ " + m.err.Error()))
			b.WriteString("\n\n")
		}

		// Help text
		helpItems := []string{
			"space: toggle",
			"a: select all",
			"n: select none",
			"enter: continue",
			"esc: back",
		}

		b.WriteString(styles.Help.Render(strings.Join(helpItems, " • ")))
	}

	return b.String()
}

// startDiscovery starts the file discovery process
func (m *FileSelectModel) startDiscovery() {
	m.loading = true
	m.spinner.Start()

	go func() {
		// Discover ignored files
		files, err := copy.DiscoverIgnored(m.repoPath)
		if err != nil {
			m.err = fmt.Errorf("failed to discover files: %w", err)
			m.loading = false
			m.spinner.Stop()
			return
		}

		m.ignoredFiles = files

		// Load config for defaults
		cfg, err := config.Load(m.repoPath)
		if err != nil {
			cfg = config.DefaultConfig()
		}

		// Create pattern matcher from config
		matcher := copy.NewPatternMatcher(
			cfg.CopyDefaults,
			cfg.CopyExclude,
		)

		// Create selection
		m.selection = copy.NewSelection(files, matcher)

		// Build checkbox items
		items := make([]components.CheckboxItem, 0, len(m.selection.Files))
		for i, file := range m.selection.Files {
			label := fmt.Sprintf("%s (%s)", file.Path, copy.FormatSize(file.Size))
			items = append(items, components.CheckboxItem{
				Label:       label,
				Description: "",
				Value:       i,
			})

			// Pre-select based on defaults
			if file.Selected {
				m.fileList.Selected[i] = true
			}
		}

		m.fileList.Items = items
		m.updateSizes()

		m.loading = false
		m.spinner.Stop()
	}()
}

// updateSizes recalculates total and selected sizes
func (m *FileSelectModel) updateSizes() {
	if m.selection == nil {
		m.totalSize = "0 B"
		m.selectedSize = "0 B"
		return
	}

	m.totalSize = copy.FormatSize(m.selection.TotalSize)

	// Calculate selected size
	var selectedSize int64
	for i, file := range m.selection.Files {
		if m.fileList.Selected[i] {
			selectedSize += file.Size
		}
	}
	m.selectedSize = copy.FormatSize(selectedSize)
}

// finalizeSelection updates the selection based on checkbox state
func (m *FileSelectModel) finalizeSelection() {
	if m.selection == nil {
		return
	}

	// Update selection based on checkbox state
	for i := range m.selection.Files {
		m.selection.Files[i].Selected = m.fileList.Selected[i]
	}

	// Recalculate selected size
	m.selection.SelectedSize = 0
	for _, file := range m.selection.Files {
		if file.Selected {
			m.selection.SelectedSize += file.Size
		}
	}
}

// IsComplete returns true if the view is complete
func (m *FileSelectModel) IsComplete() bool {
	return m.complete
}

// GetSelection returns the file selection
func (m *FileSelectModel) GetSelection() *copy.Selection {
	return m.selection
}

// GetIgnoredFiles returns the discovered ignored files
func (m *FileSelectModel) GetIgnoredFiles() []copy.IgnoredFile {
	return m.ignoredFiles
}

// Reset resets the view state
func (m *FileSelectModel) Reset() {
	m.fileList = components.NewCheckboxList("Select Files to Copy", []components.CheckboxItem{}, 15)
	m.selection = nil
	m.ignoredFiles = nil
	m.complete = false
	m.loading = false
	m.err = nil
	m.totalSize = "0 B"
	m.selectedSize = "0 B"
	m.startDiscovery()
}

// SetRepoPath updates the repository path
func (m *FileSelectModel) SetRepoPath(path string) {
	m.repoPath = path
	m.Reset()
}

// GetError returns the current error
func (m *FileSelectModel) GetError() error {
	return m.err
}

// ItemCount returns the number of selectable files
func (m *FileSelectModel) ItemCount() int {
	if m.selection == nil {
		return 0
	}
	return len(m.selection.Files)
}
