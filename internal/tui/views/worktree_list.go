package views

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Andrewy-gh/gwt/internal/filter"
	"github.com/Andrewy-gh/gwt/internal/git"
	"github.com/Andrewy-gh/gwt/internal/tui/components"
	"github.com/Andrewy-gh/gwt/internal/tui/styles"
)

// Local message types for worktree list

// worktreeListLoadedMsg is sent when worktrees are loaded
type worktreeListLoadedMsg struct {
	worktrees []git.Worktree
	err       error
}

// worktreeStatusLoadedMsg is sent when a worktree status is loaded
type worktreeStatusLoadedMsg struct {
	path   string
	status *git.WorktreeStatus
	err    error
}

// WorktreeListModel is the worktree list view
type WorktreeListModel struct {
	repoPath        string
	worktrees       []git.Worktree
	filteredIndices []int // Indices into worktrees that match current filter
	statuses        map[string]*git.WorktreeStatus
	selected        map[int]bool
	cursor          int
	loading         bool
	refreshing      bool
	width           int
	height          int
	offset          int // Scroll offset
	err             error
	spinner         *components.Spinner
	deleteRequested bool // Flag indicating delete was requested
	cancelRequested bool // Flag indicating cancel was requested

	// Filter/search state
	filterInput  textinput.Model
	filterActive bool
	filterExpr   string // Current filter expression
}

// NewWorktreeListModel creates a new worktree list view
func NewWorktreeListModel(repoPath string) *WorktreeListModel {
	ti := textinput.New()
	ti.Placeholder = "filter (e.g. branch:feature, status:dirty)"
	ti.CharLimit = 100
	ti.Width = 50

	return &WorktreeListModel{
		repoPath:    repoPath,
		selected:    make(map[int]bool),
		statuses:    make(map[string]*git.WorktreeStatus),
		spinner:     components.NewSpinner("Loading worktrees..."),
		loading:     true,
		filterInput: ti,
	}
}

// Init initializes the model
func (m *WorktreeListModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Init(),
		m.loadWorktrees(),
	)
}

// Update handles messages
func (m *WorktreeListModel) Update(msg tea.Msg) (*WorktreeListModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		if m.loading || m.refreshing {
			// Ignore input while loading
			return m, nil
		}

		// Handle filter mode input
		if m.filterActive {
			switch msg.String() {
			case "esc":
				// Exit filter mode but keep filter applied
				m.filterActive = false
				m.filterInput.Blur()
			case "enter":
				// Apply filter and exit filter mode
				m.filterActive = false
				m.filterInput.Blur()
				m.filterExpr = m.filterInput.Value()
				m.applyFilter()
			case "ctrl+c":
				// Clear filter completely
				m.filterActive = false
				m.filterInput.Blur()
				m.filterInput.SetValue("")
				m.filterExpr = ""
				m.clearFilter()
			default:
				// Update text input
				var cmd tea.Cmd
				m.filterInput, cmd = m.filterInput.Update(msg)
				cmds = append(cmds, cmd)
				// Live filter as user types
				m.filterExpr = m.filterInput.Value()
				m.applyFilter()
			}
			return m, tea.Batch(cmds...)
		}

		// Normal mode key handling
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("/"))):
			// Activate filter mode
			m.filterActive = true
			m.filterInput.Focus()
			return m, textinput.Blink
		case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
			m.cursorUp()
		case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
			m.cursorDown()
		case key.Matches(msg, key.NewBinding(key.WithKeys(" ", "x"))):
			m.toggleSelection()
		case key.Matches(msg, key.NewBinding(key.WithKeys("a"))):
			m.selectAll()
		case key.Matches(msg, key.NewBinding(key.WithKeys("n"))):
			m.deselectAll()
		case key.Matches(msg, key.NewBinding(key.WithKeys("d", "D"))):
			if m.hasSelection() {
				// Set flag for delete request
				m.deleteRequested = true
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("r", "R"))):
			m.refreshing = true
			cmds = append(cmds, m.loadWorktrees())
		case key.Matches(msg, key.NewBinding(key.WithKeys("c", "C"))):
			// Clear filter
			m.filterInput.SetValue("")
			m.filterExpr = ""
			m.clearFilter()
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			if m.filterExpr != "" {
				// First esc clears filter
				m.filterInput.SetValue("")
				m.filterExpr = ""
				m.clearFilter()
			} else {
				m.cancelRequested = true
			}
		}

	case worktreeListLoadedMsg:
		m.loading = false
		m.refreshing = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.worktrees = msg.worktrees
			m.applyFilter() // Re-apply filter after reload
			// Load status for each worktree
			cmds = append(cmds, m.loadStatuses())
		}

	case worktreeStatusLoadedMsg:
		if msg.err == nil && msg.status != nil {
			m.statuses[msg.path] = msg.status
			// Re-apply filter when status updates (for status-based filters)
			if m.filterExpr != "" {
				m.applyFilter()
			}
		}

	case components.SpinnerTickMsg:
		if m.loading || m.refreshing {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// View renders the view
func (m *WorktreeListModel) View(width, height int) string {
	m.width = width
	m.height = height

	if m.err != nil {
		return m.renderError()
	}

	if m.loading {
		return m.renderLoading()
	}

	if m.refreshing {
		return m.renderRefreshing()
	}

	return m.renderList()
}

// renderLoading renders the loading state
func (m *WorktreeListModel) renderLoading() string {
	var b strings.Builder
	b.WriteString(styles.Title.Render("Worktrees"))
	b.WriteString("\n\n")
	b.WriteString(m.spinner.View())
	b.WriteString("\n\n")
	return b.String()
}

// renderRefreshing renders the refreshing state
func (m *WorktreeListModel) renderRefreshing() string {
	var b strings.Builder
	b.WriteString(styles.Title.Render("Worktrees"))
	b.WriteString("\n\n")

	// Show the list with a refreshing indicator
	b.WriteString(m.renderTable())
	b.WriteString("\n\n")
	b.WriteString(styles.Selected.Render("↻ Refreshing..."))
	b.WriteString("\n")

	return b.String()
}

// renderError renders an error message
func (m *WorktreeListModel) renderError() string {
	var b strings.Builder
	b.WriteString(styles.Title.Render("Worktrees"))
	b.WriteString("\n\n")
	b.WriteString(styles.ErrorText.Render("Error: " + m.err.Error()))
	b.WriteString("\n\n")
	b.WriteString(styles.MutedText.Render("Press esc to return to menu"))
	b.WriteString("\n")
	return b.String()
}

// renderList renders the worktree list
func (m *WorktreeListModel) renderList() string {
	var b strings.Builder

	// Title
	b.WriteString(styles.Title.Render("Worktrees"))
	b.WriteString("\n\n")

	// Filter input (if active)
	if m.filterActive {
		b.WriteString("Filter: ")
		b.WriteString(m.filterInput.View())
		b.WriteString("\n\n")
	} else if m.filterExpr != "" {
		// Show current filter when not editing
		visible := m.getVisibleWorktrees()
		filterInfo := fmt.Sprintf("Filter: %s (%d/%d matches)", m.filterExpr, len(visible), len(m.worktrees))
		b.WriteString(styles.Selected.Render(filterInfo))
		b.WriteString("  ")
		b.WriteString(styles.MutedText.Render("(C: clear, /: edit)"))
		b.WriteString("\n\n")
	}

	// Empty state
	visible := m.getVisibleWorktrees()
	if len(m.worktrees) == 0 {
		b.WriteString(styles.MutedText.Render("No worktrees found"))
		b.WriteString("\n\n")
		b.WriteString(styles.Help.Render("Press esc to return to menu"))
		b.WriteString("\n")
		return b.String()
	}

	if len(visible) == 0 && m.filterExpr != "" {
		b.WriteString(styles.MutedText.Render("No worktrees match the filter"))
		b.WriteString("\n\n")
		b.WriteString(styles.Help.Render("Press C to clear filter, esc to return to menu"))
		b.WriteString("\n")
		return b.String()
	}

	// Table
	b.WriteString(m.renderTable())
	b.WriteString("\n")

	// Help text
	b.WriteString(m.renderHelp())

	return b.String()
}

// renderTable renders the worktree table
func (m *WorktreeListModel) renderTable() string {
	var b strings.Builder

	// Header
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.Primary).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(styles.Border)

	headers := []string{"  ", "Path", "Branch", "Status", "Last Commit", "Age"}
	widths := []int{4, 30, 20, 10, 40, 12}

	var headerCells []string
	for i, header := range headers {
		headerCells = append(headerCells, headerStyle.Width(widths[i]).Render(header))
	}
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, headerCells...))
	b.WriteString("\n")

	// Get visible worktrees (filtered or all)
	visible := m.getVisibleWorktrees()

	// Calculate visible range
	maxRows := m.height - 15 // Reserve space for title, header, and help
	if maxRows < 5 {
		maxRows = 5
	}

	visibleStart := m.offset
	visibleEnd := m.offset + maxRows
	if visibleEnd > len(visible) {
		visibleEnd = len(visible)
	}

	// Rows
	for i := visibleStart; i < visibleEnd; i++ {
		wt := visible[i]
		actualIdx := m.getActualIndex(i)

		// Determine row style
		var rowStyle lipgloss.Style
		if i == m.cursor {
			rowStyle = lipgloss.NewStyle().Foreground(styles.Primary).Bold(true)
		} else {
			rowStyle = lipgloss.NewStyle()
		}

		// Checkbox
		checkbox := "[ ]"
		if m.selected[actualIdx] {
			checkbox = "[✓]"
		}
		if i == m.cursor {
			checkbox = styles.Cursor.Render(checkbox)
		}

		// Path (show relative to repo or basename)
		displayPath := wt.Path
		if relPath, err := filepath.Rel(m.repoPath, wt.Path); err == nil {
			if relPath == "." {
				displayPath = "(main)"
			} else {
				displayPath = relPath
			}
		}
		if len(displayPath) > 28 {
			displayPath = "..." + displayPath[len(displayPath)-25:]
		}

		// Branch (or detached)
		branch := wt.Branch
		if wt.IsDetached {
			branch = fmt.Sprintf("(detached at %s)", wt.Commit)
		}
		if wt.Locked {
			branch += " 🔒"
		}
		if len(branch) > 18 {
			branch = branch[:15] + "..."
		}

		// Status indicator
		status := m.getStatusIndicator(wt.Path)

		// Last commit
		lastCommit := "N/A"
		if st, ok := m.statuses[wt.Path]; ok && st != nil {
			if st.LastCommitMsg != "" {
				lastCommit = st.LastCommitMsg
				if len(lastCommit) > 38 {
					lastCommit = lastCommit[:35] + "..."
				}
			}
		}

		// Age
		age := m.getWorktreeAge(wt.Path)

		// Build row
		cells := []string{
			rowStyle.Width(widths[0]).Render(checkbox),
			rowStyle.Width(widths[1]).Render(displayPath),
			rowStyle.Width(widths[2]).Render(branch),
			rowStyle.Width(widths[3]).Render(status),
			rowStyle.Width(widths[4]).Render(lastCommit),
			rowStyle.Width(widths[5]).Render(age),
		}

		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, cells...))
		b.WriteString("\n")
	}

	// Scroll indicators
	if m.offset > 0 {
		b.WriteString(styles.MutedText.Render("  ▲ More rows above"))
		b.WriteString("\n")
	}
	if visibleEnd < len(visible) {
		b.WriteString(styles.MutedText.Render("  ▼ More rows below"))
		b.WriteString("\n")
	}

	return b.String()
}

// renderHelp renders the help text
func (m *WorktreeListModel) renderHelp() string {
	var b strings.Builder

	// Selection summary
	selectedCount := len(m.selected)
	if selectedCount > 0 {
		b.WriteString(styles.Selected.Render(fmt.Sprintf("%d worktree(s) selected", selectedCount)))
		b.WriteString("\n\n")
	}

	// Key bindings
	if m.filterActive {
		b.WriteString(styles.Help.Render("Type to filter • Enter: apply • Esc: close • Ctrl+C: clear"))
		b.WriteString("\n")
		b.WriteString(styles.MutedText.Render("Filter syntax: field:value (e.g. branch:feature, status:dirty, age:>7d)"))
		b.WriteString("\n")
	} else {
		b.WriteString(styles.Help.Render("↑/k, ↓/j: navigate  space: toggle  a: select all  n: deselect"))
		b.WriteString("\n")
		b.WriteString(styles.Help.Render("D: delete selected  R: refresh  /: filter  C: clear filter  esc: back"))
		b.WriteString("\n")
	}

	return b.String()
}

// getStatusIndicator returns a status indicator for a worktree
func (m *WorktreeListModel) getStatusIndicator(path string) string {
	status, ok := m.statuses[path]
	if !ok {
		return styles.MutedText.Render("⋯") // Loading
	}

	if status == nil {
		return styles.ErrorText.Render("✘") // Error
	}

	// Check for warnings
	hasWarnings := false
	if status.AheadCount > 0 || status.BehindCount > 0 {
		hasWarnings = true
	}

	// Status with appropriate color
	if status.Clean {
		if hasWarnings {
			return styles.WarningText.Render("⚠")
		}
		return styles.SuccessText.Render("✓")
	}

	// Dirty with details
	details := []string{}
	if status.StagedCount > 0 {
		details = append(details, fmt.Sprintf("%dS", status.StagedCount))
	}
	if status.UnstagedCount > 0 {
		details = append(details, fmt.Sprintf("%dU", status.UnstagedCount))
	}
	if status.UntrackedCount > 0 {
		details = append(details, fmt.Sprintf("%d?", status.UntrackedCount))
	}

	statusText := "✘"
	if len(details) > 0 {
		statusText = strings.Join(details, " ")
	}

	return styles.ErrorText.Render(statusText)
}

// getWorktreeAge returns a human-readable age for a worktree
func (m *WorktreeListModel) getWorktreeAge(path string) string {
	status, ok := m.statuses[path]
	if !ok || status == nil {
		return "N/A"
	}

	if status.LastCommitTime.IsZero() {
		return "N/A"
	}

	age := time.Since(status.LastCommitTime)

	if age < time.Minute {
		return "just now"
	} else if age < time.Hour {
		minutes := int(age.Minutes())
		return fmt.Sprintf("%dm ago", minutes)
	} else if age < 24*time.Hour {
		hours := int(age.Hours())
		return fmt.Sprintf("%dh ago", hours)
	} else if age < 7*24*time.Hour {
		days := int(age.Hours() / 24)
		return fmt.Sprintf("%dd ago", days)
	} else if age < 30*24*time.Hour {
		weeks := int(age.Hours() / 24 / 7)
		return fmt.Sprintf("%dw ago", weeks)
	} else if age < 365*24*time.Hour {
		months := int(age.Hours() / 24 / 30)
		return fmt.Sprintf("%dmo ago", months)
	} else {
		years := int(age.Hours() / 24 / 365)
		return fmt.Sprintf("%dy ago", years)
	}
}

// Navigation methods

func (m *WorktreeListModel) cursorUp() {
	if m.cursor > 0 {
		m.cursor--
		// Adjust scroll
		if m.cursor < m.offset {
			m.offset = m.cursor
		}
	}
}

func (m *WorktreeListModel) cursorDown() {
	visible := m.getVisibleWorktrees()
	if m.cursor < len(visible)-1 {
		m.cursor++
		// Adjust scroll
		maxRows := m.height - 15
		if maxRows < 5 {
			maxRows = 5
		}
		if m.cursor >= m.offset+maxRows {
			m.offset = m.cursor - maxRows + 1
		}
	}
}

func (m *WorktreeListModel) toggleSelection() {
	visible := m.getVisibleWorktrees()
	if m.cursor < len(visible) {
		actualIdx := m.getActualIndex(m.cursor)
		if actualIdx < 0 {
			return
		}
		// Don't allow selecting the main worktree
		if m.worktrees[actualIdx].IsMain {
			return
		}
		m.selected[actualIdx] = !m.selected[actualIdx]
		if !m.selected[actualIdx] {
			delete(m.selected, actualIdx)
		}
	}
}

func (m *WorktreeListModel) selectAll() {
	visible := m.getVisibleWorktrees()
	for i := range visible {
		actualIdx := m.getActualIndex(i)
		if actualIdx < 0 {
			continue
		}
		// Don't select the main worktree
		if !m.worktrees[actualIdx].IsMain {
			m.selected[actualIdx] = true
		}
	}
}

func (m *WorktreeListModel) deselectAll() {
	m.selected = make(map[int]bool)
}

func (m *WorktreeListModel) hasSelection() bool {
	return len(m.selected) > 0
}

// Filter methods

// applyFilter applies the current filter expression to the worktree list
func (m *WorktreeListModel) applyFilter() {
	if m.filterExpr == "" {
		m.clearFilter()
		return
	}

	// Parse the filter expression
	f, err := filter.Parse(m.filterExpr)
	if err != nil {
		// Invalid filter - show all worktrees
		m.clearFilter()
		return
	}

	// Build the filter
	filterObj := filter.New()
	filterObj.Add(*f)

	// Apply filter to worktrees
	m.filteredIndices = nil
	for i, wt := range m.worktrees {
		ctx := &filter.WorktreeFilterContext{
			Worktree: &wt,
			Status:   m.statuses[wt.Path],
		}
		if filterObj.Match(ctx) {
			m.filteredIndices = append(m.filteredIndices, i)
		}
	}

	// Reset cursor position
	m.cursor = 0
	m.offset = 0
}

// clearFilter clears the current filter
func (m *WorktreeListModel) clearFilter() {
	m.filteredIndices = nil
	m.cursor = 0
	m.offset = 0
}

// getVisibleWorktrees returns the worktrees to display (filtered or all)
func (m *WorktreeListModel) getVisibleWorktrees() []git.Worktree {
	if m.filteredIndices == nil {
		return m.worktrees
	}

	visible := make([]git.Worktree, len(m.filteredIndices))
	for i, idx := range m.filteredIndices {
		visible[i] = m.worktrees[idx]
	}
	return visible
}

// getActualIndex converts a visible index to the actual worktree index
func (m *WorktreeListModel) getActualIndex(visibleIdx int) int {
	if m.filteredIndices == nil {
		return visibleIdx
	}
	if visibleIdx >= 0 && visibleIdx < len(m.filteredIndices) {
		return m.filteredIndices[visibleIdx]
	}
	return -1
}

func (m *WorktreeListModel) getSelectedPaths() []string {
	var paths []string
	for i := range m.selected {
		if i < len(m.worktrees) {
			paths = append(paths, m.worktrees[i].Path)
		}
	}
	return paths
}

// Data loading methods

func (m *WorktreeListModel) loadWorktrees() tea.Cmd {
	return func() tea.Msg {
		worktrees, err := git.ListWorktrees(m.repoPath)
		return worktreeListLoadedMsg{
			worktrees: worktrees,
			err:       err,
		}
	}
}

func (m *WorktreeListModel) loadStatuses() tea.Cmd {
	// Load statuses for all worktrees
	var cmds []tea.Cmd
	for _, wt := range m.worktrees {
		path := wt.Path
		cmds = append(cmds, func() tea.Msg {
			status, err := git.GetWorktreeStatus(path)
			return worktreeStatusLoadedMsg{
				path:   path,
				status: status,
				err:    err,
			}
		})
	}
	return tea.Batch(cmds...)
}

// Accessors for root model

// GetSelectedWorktrees returns the selected worktrees
func (m *WorktreeListModel) GetSelectedWorktrees() []git.Worktree {
	var selected []git.Worktree
	for i := range m.selected {
		if i < len(m.worktrees) {
			selected = append(selected, m.worktrees[i])
		}
	}
	return selected
}

// ShouldDelete returns true if delete was requested
func (m *WorktreeListModel) ShouldDelete() bool {
	return m.deleteRequested
}

// ShouldCancel returns true if cancel was requested
func (m *WorktreeListModel) ShouldCancel() bool {
	return m.cancelRequested
}

// GetSelectedPaths returns the paths of selected worktrees
func (m *WorktreeListModel) GetSelectedPaths() []string {
	return m.getSelectedPaths()
}

// ResetFlags resets the request flags
func (m *WorktreeListModel) ResetFlags() {
	m.deleteRequested = false
	m.cancelRequested = false
}
