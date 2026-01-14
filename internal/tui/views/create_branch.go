package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/key"
	"github.com/Andrewy-gh/gwt/internal/create"
	"github.com/Andrewy-gh/gwt/internal/git"
	"github.com/Andrewy-gh/gwt/internal/tui/components"
	"github.com/Andrewy-gh/gwt/internal/tui/styles"
)

// CreateBranchModel is the branch input view
type CreateBranchModel struct {
	branchInput    *components.TextInput
	sourceSelector *components.RadioList
	repoPath       string
	complete       bool
	branchSpec     *create.BranchSpec
	focusIndex     int // 0=input, 1=selector
	width          int
	height         int
	err            error
}

// NewCreateBranchModel creates a new branch creation view
func NewCreateBranchModel(repoPath string) *CreateBranchModel {
	// Text input for branch name
	branchInput := components.NewTextInput("Branch Name", "feature/my-feature")
	branchInput.Focus()
	branchInput.SetValidator(create.ValidateBranchName)

	// Radio list for branch source type
	sourceItems := []components.RadioItem{
		{
			Label:       "New from HEAD",
			Description: "Create a new branch from the current HEAD",
			Value:       "new-head",
		},
		{
			Label:       "New from reference",
			Description: "Create a new branch from a specific commit, tag, or branch",
			Value:       "new-ref",
		},
		{
			Label:       "Existing local branch",
			Description: "Checkout an existing local branch",
			Value:       "existing",
		},
		{
			Label:       "Remote branch",
			Description: "Create local tracking branch from a remote branch",
			Value:       "remote",
		},
	}
	sourceSelector := components.NewRadioList("Branch Source", sourceItems, 8)
	sourceSelector.SetSelected(0) // Default to "New from HEAD"

	return &CreateBranchModel{
		branchInput:    branchInput,
		sourceSelector: sourceSelector,
		repoPath:       repoPath,
		focusIndex:     0, // Start with input focused
	}
}

// Init initializes the model
func (m *CreateBranchModel) Init() tea.Cmd {
	return m.branchInput.Init()
}

// Update handles messages
func (m *CreateBranchModel) Update(msg tea.Msg) (*CreateBranchModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			// Return to menu
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("tab", "shift+tab"))):
			// Toggle focus between input and selector
			m.toggleFocus()
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			// Submit if valid
			if m.canSubmit() {
				if err := m.createBranchSpec(); err != nil {
					m.err = err
					return m, nil
				}
				m.complete = true
				return m, nil
			}
			return m, nil
		}
	}

	// Delegate to focused component
	var cmd tea.Cmd
	if m.focusIndex == 0 {
		m.branchInput, cmd = m.branchInput.Update(msg)
	} else {
		cmd = m.sourceSelector.Update(msg)
	}

	return m, cmd
}

// View renders the view
func (m *CreateBranchModel) View(width, height int) string {
	m.width = width
	m.height = height

	var b strings.Builder

	// Title
	b.WriteString(styles.Title.Render("Create Worktree - Branch Selection"))
	b.WriteString("\n\n")

	// Description
	b.WriteString(styles.Subtitle.Render("Enter a branch name and select the branch source type."))
	b.WriteString("\n\n")

	// Branch input
	b.WriteString(m.branchInput.View())
	b.WriteString("\n")

	// Branch source selector
	b.WriteString(m.sourceSelector.View())
	b.WriteString("\n")

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
		helpItems = append([]string{"[branch name focused]"}, helpItems...)
	} else {
		helpItems = append([]string{"[source type focused]"}, helpItems...)
	}

	b.WriteString(styles.Help.Render(strings.Join(helpItems, " • ")))

	return b.String()
}

// toggleFocus switches focus between input and selector
func (m *CreateBranchModel) toggleFocus() {
	if m.focusIndex == 0 {
		m.branchInput.Blur()
		m.focusIndex = 1
	} else {
		m.branchInput.Focus()
		m.focusIndex = 0
	}
}

// canSubmit returns true if the form can be submitted
func (m *CreateBranchModel) canSubmit() bool {
	// Branch name must be valid
	if !m.branchInput.IsValid() || m.branchInput.Value() == "" {
		return false
	}

	// Source type must be selected
	if !m.sourceSelector.HasSelection() {
		return false
	}

	return true
}

// createBranchSpec creates the BranchSpec from current input
func (m *CreateBranchModel) createBranchSpec() error {
	branchName := m.branchInput.Value()
	sourceValue := m.sourceSelector.GetSelectedValue()

	// Validate branch name
	if err := create.ValidateBranchName(branchName); err != nil {
		return fmt.Errorf("invalid branch name: %w", err)
	}

	// Create spec based on source type
	spec := &create.BranchSpec{
		BranchName: branchName,
	}

	switch sourceValue {
	case "new-head":
		spec.Source = create.BranchSourceNewFromHEAD
		// Check if branch already exists
		exists, err := git.LocalBranchExists(m.repoPath, branchName)
		if err != nil {
			return fmt.Errorf("failed to check if branch exists: %w", err)
		}
		if exists {
			return fmt.Errorf("branch '%s' already exists", branchName)
		}

	case "new-ref":
		spec.Source = create.BranchSourceNewFromRef
		// Check if branch already exists
		exists, err := git.LocalBranchExists(m.repoPath, branchName)
		if err != nil {
			return fmt.Errorf("failed to check if branch exists: %w", err)
		}
		if exists {
			return fmt.Errorf("branch '%s' already exists", branchName)
		}

	case "existing":
		spec.Source = create.BranchSourceLocalExisting
		// Check branch exists
		exists, err := git.LocalBranchExists(m.repoPath, branchName)
		if err != nil {
			return fmt.Errorf("failed to check if branch exists: %w", err)
		}
		if !exists {
			return fmt.Errorf("branch '%s' does not exist", branchName)
		}

		// Check branch not already checked out in a worktree
		wt, err := git.FindWorktreeByBranch(m.repoPath, branchName)
		if err != nil {
			return fmt.Errorf("failed to check worktrees: %w", err)
		}
		if wt != nil {
			return fmt.Errorf("branch '%s' is already checked out in %s", branchName, wt.Path)
		}

	case "remote":
		spec.Source = create.BranchSourceRemote
		// Note: We'll select the specific remote branch in the next view
		// For now, just set the local branch name
	}

	m.branchSpec = spec
	return nil
}

// IsComplete returns true if the view is complete
func (m *CreateBranchModel) IsComplete() bool {
	return m.complete
}

// GetBranchSpec returns the created branch spec
func (m *CreateBranchModel) GetBranchSpec() *create.BranchSpec {
	return m.branchSpec
}

// Reset resets the view state
func (m *CreateBranchModel) Reset() {
	m.branchInput.SetValue("")
	m.sourceSelector.SetSelected(0)
	m.complete = false
	m.branchSpec = nil
	m.focusIndex = 0
	m.err = nil
	m.branchInput.Focus()
}

// SetRepoPath updates the repository path
func (m *CreateBranchModel) SetRepoPath(path string) {
	m.repoPath = path
}

// GetError returns the current error
func (m *CreateBranchModel) GetError() error {
	return m.err
}
