package views

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/key"
	"github.com/Andrewy-gh/gwt/internal/git"
	"github.com/Andrewy-gh/gwt/internal/tui/components"
	"github.com/Andrewy-gh/gwt/internal/tui/styles"
)

// RemoteBranchModel is the remote branch selection view
type RemoteBranchModel struct {
	filterInput      *components.TextInput
	branchTable      *components.Table
	spinner          *components.Spinner
	branches         []git.Branch
	filteredBranches []git.Branch
	repoPath         string
	selected         *git.Branch
	complete         bool
	loading          bool
	fetching         bool
	focusIndex       int // 0=input, 1=table
	width            int
	height           int
	err              error
}

// NewRemoteBranchModel creates a new remote branch selection view
func NewRemoteBranchModel(repoPath string) *RemoteBranchModel {
	// Filter input
	filterInput := components.NewTextInput("Filter", "Search remote branches...")
	filterInput.Focus()

	// Branch table
	branchTable := components.NewTable(
		[]string{"Remote Branch", "Commit", "Age"},
		[][]string{},
		true,
		15,
	)

	// Spinner for fetch operation
	spinner := components.NewSpinner("Fetching remote branches...")

	m := &RemoteBranchModel{
		filterInput: filterInput,
		branchTable: branchTable,
		spinner:     spinner,
		repoPath:    repoPath,
		focusIndex:  0,
	}

	// Load branches initially
	m.loadBranches()

	return m
}

// Init initializes the model
func (m *RemoteBranchModel) Init() tea.Cmd {
	return tea.Batch(
		m.filterInput.Init(),
		m.spinner.Init(),
	)
}

// Update handles messages
func (m *RemoteBranchModel) Update(msg tea.Msg) (*RemoteBranchModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.fetching {
			// Ignore most input while fetching
			return m, nil
		}

		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			// Go back
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("r", "R"))):
			// Refresh - fetch from remote
			return m, m.startFetch()

		case key.Matches(msg, key.NewBinding(key.WithKeys("tab", "shift+tab"))):
			// Toggle focus between input and table
			m.toggleFocus()
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			// Select current branch
			if !m.branchTable.IsEmpty() && len(m.filteredBranches) > 0 {
				cursor := m.branchTable.GetCursor()
				if cursor < len(m.filteredBranches) {
					m.selected = &m.filteredBranches[cursor]
					m.complete = true
					return m, nil
				}
			}
			return m, nil
		}

	case components.SpinnerTickMsg:
		if m.fetching {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	// Update filter input and filter branches
	if m.focusIndex == 0 && !m.fetching {
		var cmd tea.Cmd
		m.filterInput, cmd = m.filterInput.Update(msg)
		cmds = append(cmds, cmd)

		// Filter branches based on input
		m.filterBranches()
	}

	// Update table
	if m.focusIndex == 1 && !m.fetching {
		cmd := m.branchTable.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Update spinner
	if m.fetching {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the view
func (m *RemoteBranchModel) View(width, height int) string {
	m.width = width
	m.height = height

	var b strings.Builder

	// Title
	b.WriteString(styles.Title.Render("Create Worktree - Select Remote Branch"))
	b.WriteString("\n\n")

	if m.fetching {
		// Show spinner while fetching
		b.WriteString(m.spinner.View())
		b.WriteString("\n\n")
		b.WriteString(styles.Subtitle.Render("This may take a moment..."))
		b.WriteString("\n")
	} else if m.loading {
		b.WriteString(styles.Subtitle.Render("Loading branches..."))
		b.WriteString("\n")
	} else {
		// Description
		b.WriteString(styles.Subtitle.Render("Select a remote branch to create a local tracking branch."))
		b.WriteString("\n\n")

		// Filter input
		b.WriteString(m.filterInput.View())
		b.WriteString("\n")

		// Branch count
		totalCount := len(m.branches)
		filteredCount := len(m.filteredBranches)
		if m.filterInput.Value() != "" {
			countText := fmt.Sprintf("Showing %d of %d branches", filteredCount, totalCount)
			b.WriteString(styles.MutedText.Render(countText))
			b.WriteString("\n\n")
		} else {
			countText := fmt.Sprintf("Total: %d branches", totalCount)
			b.WriteString(styles.MutedText.Render(countText))
			b.WriteString("\n\n")
		}

		// Branch table
		if m.branchTable.IsEmpty() {
			if m.filterInput.Value() != "" {
				b.WriteString(styles.MutedText.Render("No branches match your filter"))
			} else {
				b.WriteString(styles.MutedText.Render("No remote branches found"))
			}
			b.WriteString("\n")
		} else {
			b.WriteString(m.branchTable.View())
			b.WriteString("\n")
		}

		// Error message
		if m.err != nil {
			b.WriteString(styles.ErrorText.Render("✘ " + m.err.Error()))
			b.WriteString("\n\n")
		}

		// Help text
		helpItems := []string{
			"r: refresh",
			"tab: switch focus",
			"enter: select",
			"esc: back",
		}

		// Indicate which component is focused
		if m.focusIndex == 0 {
			helpItems = append([]string{"[filter focused]"}, helpItems...)
		} else {
			helpItems = append([]string{"[table focused]"}, helpItems...)
		}

		b.WriteString(styles.Help.Render(strings.Join(helpItems, " • ")))
	}

	return b.String()
}

// toggleFocus switches focus between input and table
func (m *RemoteBranchModel) toggleFocus() {
	if m.focusIndex == 0 {
		m.filterInput.Blur()
		m.focusIndex = 1
	} else {
		m.filterInput.Focus()
		m.focusIndex = 0
	}
}

// loadBranches loads remote branches from the repository
func (m *RemoteBranchModel) loadBranches() {
	m.loading = true
	m.err = nil

	branches, err := git.ListRemoteBranches(m.repoPath)
	if err != nil {
		m.err = fmt.Errorf("failed to load remote branches: %w", err)
		m.loading = false
		return
	}

	m.branches = branches
	m.filteredBranches = branches
	m.updateTable()
	m.loading = false
}

// filterBranches filters branches based on the current filter input
func (m *RemoteBranchModel) filterBranches() {
	filter := strings.ToLower(m.filterInput.Value())

	if filter == "" {
		m.filteredBranches = m.branches
	} else {
		m.filteredBranches = make([]git.Branch, 0)
		for _, branch := range m.branches {
			if strings.Contains(strings.ToLower(branch.Name), filter) {
				m.filteredBranches = append(m.filteredBranches, branch)
			}
		}
	}

	m.updateTable()
}

// updateTable updates the table with current filtered branches
func (m *RemoteBranchModel) updateTable() {
	rows := make([][]string, 0, len(m.filteredBranches))

	for _, branch := range m.filteredBranches {
		age := formatAge(branch.LastCommit)
		rows = append(rows, []string{
			branch.Name,
			branch.Commit,
			age,
		})
	}

	m.branchTable.SetRows(rows)
}

// startFetch returns a command to fetch remote branches
func (m *RemoteBranchModel) startFetch() tea.Cmd {
	return func() tea.Msg {
		m.fetching = true
		m.spinner.Start()

		// Fetch from all remotes
		err := git.Fetch(m.repoPath, "", true)
		if err != nil {
			m.err = fmt.Errorf("failed to fetch from remote: %w", err)
		} else {
			// Reload branches after successful fetch
			m.loadBranches()
		}

		m.fetching = false
		m.spinner.Stop()

		return nil
	}
}

// formatAge formats a time as a relative age string
func formatAge(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}

	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 min ago"
		}
		return fmt.Sprintf("%d mins ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	case diff < 30*24*time.Hour:
		weeks := int(diff.Hours() / 24 / 7)
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	case diff < 365*24*time.Hour:
		months := int(diff.Hours() / 24 / 30)
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	default:
		years := int(diff.Hours() / 24 / 365)
		if years == 1 {
			return "1 year ago"
		}
		return fmt.Sprintf("%d years ago", years)
	}
}

// IsComplete returns true if the view is complete
func (m *RemoteBranchModel) IsComplete() bool {
	return m.complete
}

// GetSelected returns the selected remote branch
func (m *RemoteBranchModel) GetSelected() *git.Branch {
	return m.selected
}

// Reset resets the view state
func (m *RemoteBranchModel) Reset() {
	m.filterInput.SetValue("")
	m.complete = false
	m.selected = nil
	m.focusIndex = 0
	m.err = nil
	m.fetching = false
	m.filterInput.Focus()
	m.loadBranches()
}

// SetRepoPath updates the repository path
func (m *RemoteBranchModel) SetRepoPath(path string) {
	m.repoPath = path
	m.loadBranches()
}

// GetError returns the current error
func (m *RemoteBranchModel) GetError() error {
	return m.err
}
