package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/key"
	"github.com/Andrewy-gh/gwt/internal/git"
	"github.com/Andrewy-gh/gwt/internal/tui/components"
	"github.com/Andrewy-gh/gwt/internal/tui/styles"
)

// CreateSourceModel is the source/ref selection view
type CreateSourceModel struct {
	refInput      *components.TextInput
	suggestions   *components.Table
	repoPath      string
	complete      bool
	startPoint    string
	focusIndex    int // 0=input, 1=suggestions
	width         int
	height        int
	err           error
	loadingSuggestions bool
}

// NewCreateSourceModel creates a new source selection view
func NewCreateSourceModel(repoPath string) *CreateSourceModel {
	// Text input for ref
	refInput := components.NewTextInput("Start Point", "main, HEAD, commit SHA, or tag")
	refInput.Focus()

	// Set up validator to check if ref exists
	refInput.SetValidator(func(value string) error {
		if value == "" {
			return nil // Allow empty while typing
		}
		// Only validate on complete input (when user presses enter)
		return nil
	})

	// Suggestions table
	suggestions := components.NewTable(
		[]string{"Type", "Name", "Commit"},
		[][]string{},
		true,
		10,
	)

	m := &CreateSourceModel{
		refInput:    refInput,
		suggestions: suggestions,
		repoPath:    repoPath,
		focusIndex:  0,
	}

	// Load suggestions
	m.loadSuggestions()

	return m
}

// Init initializes the model
func (m *CreateSourceModel) Init() tea.Cmd {
	return m.refInput.Init()
}

// Update handles messages
func (m *CreateSourceModel) Update(msg tea.Msg) (*CreateSourceModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			// Go back
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("tab", "shift+tab"))):
			// Toggle focus between input and suggestions
			m.toggleFocus()
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			if m.focusIndex == 0 {
				// Validate and submit from input
				if m.refInput.Value() != "" {
					if err := m.validateAndSubmit(); err != nil {
						m.err = err
						return m, nil
					}
					m.complete = true
					return m, nil
				}
			} else {
				// Select from suggestions table
				if !m.suggestions.IsEmpty() {
					row := m.suggestions.SelectedRow()
					if len(row) >= 2 {
						m.refInput.SetValue(row[1]) // Set the name
						if err := m.validateAndSubmit(); err != nil {
							m.err = err
							return m, nil
						}
						m.complete = true
						return m, nil
					}
				}
			}
			return m, nil
		}
	}

	// Delegate to focused component
	var cmd tea.Cmd
	if m.focusIndex == 0 {
		m.refInput, cmd = m.refInput.Update(msg)
	} else {
		cmd = m.suggestions.Update(msg)
	}

	return m, cmd
}

// View renders the view
func (m *CreateSourceModel) View(width, height int) string {
	m.width = width
	m.height = height

	var b strings.Builder

	// Title
	b.WriteString(styles.Title.Render("Create Worktree - Select Start Point"))
	b.WriteString("\n\n")

	// Description
	b.WriteString(styles.Subtitle.Render("Enter a commit SHA, branch name, or tag as the starting point."))
	b.WriteString("\n\n")

	// Ref input
	b.WriteString(m.refInput.View())
	b.WriteString("\n")

	// Suggestions section
	if m.loadingSuggestions {
		b.WriteString(styles.MutedText.Render("Loading suggestions..."))
		b.WriteString("\n")
	} else if !m.suggestions.IsEmpty() {
		b.WriteString(styles.Subtitle.Render("Quick Select:"))
		b.WriteString("\n")
		b.WriteString(m.suggestions.View())
		b.WriteString("\n")
	}

	// Error message
	if m.err != nil {
		b.WriteString(styles.ErrorText.Render("✘ " + m.err.Error()))
		b.WriteString("\n\n")
	}

	// Help text
	helpItems := []string{
		"tab: switch focus",
		"enter: continue",
		"esc: back",
	}

	// Indicate which component is focused
	if m.focusIndex == 0 {
		helpItems = append([]string{"[ref input focused]"}, helpItems...)
	} else {
		helpItems = append([]string{"[suggestions focused]"}, helpItems...)
	}

	b.WriteString(styles.Help.Render(strings.Join(helpItems, " • ")))

	return b.String()
}

// toggleFocus switches focus between input and suggestions
func (m *CreateSourceModel) toggleFocus() {
	if m.focusIndex == 0 {
		m.refInput.Blur()
		m.focusIndex = 1
	} else {
		m.refInput.Focus()
		m.focusIndex = 0
	}
}

// validateAndSubmit validates the current input and sets the start point
func (m *CreateSourceModel) validateAndSubmit() error {
	ref := m.refInput.Value()

	if ref == "" {
		return fmt.Errorf("start point cannot be empty")
	}

	// Validate ref exists
	if err := git.ValidateRef(m.repoPath, ref); err != nil {
		return fmt.Errorf("invalid start point: %w", err)
	}

	m.startPoint = ref
	return nil
}

// loadSuggestions loads recent branches, commits, and tags
func (m *CreateSourceModel) loadSuggestions() {
	m.loadingSuggestions = true

	var rows [][]string

	// Get local branches
	branches, err := git.ListLocalBranches(m.repoPath)
	if err == nil {
		for i, branch := range branches {
			if i >= 5 { // Limit to 5 branches
				break
			}
			rows = append(rows, []string{
				"Branch",
				branch.Name,
				branch.Commit,
			})
		}
	}

	// Get recent tags (we'll use git command)
	// For simplicity, just show HEAD and common refs
	rows = append(rows, []string{
		"Ref",
		"HEAD",
		"Current HEAD",
	})

	m.suggestions.SetRows(rows)
	m.loadingSuggestions = false
}

// IsComplete returns true if the view is complete
func (m *CreateSourceModel) IsComplete() bool {
	return m.complete
}

// GetStartPoint returns the selected start point
func (m *CreateSourceModel) GetStartPoint() string {
	return m.startPoint
}

// Reset resets the view state
func (m *CreateSourceModel) Reset() {
	m.refInput.SetValue("")
	m.complete = false
	m.startPoint = ""
	m.focusIndex = 0
	m.err = nil
	m.refInput.Focus()
	m.loadSuggestions()
}

// SetRepoPath updates the repository path
func (m *CreateSourceModel) SetRepoPath(path string) {
	m.repoPath = path
	m.loadSuggestions()
}

// GetError returns the current error
func (m *CreateSourceModel) GetError() error {
	return m.err
}
