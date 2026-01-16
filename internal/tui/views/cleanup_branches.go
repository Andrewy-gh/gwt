package views

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
	"github.com/Andrewy-gh/gwt/internal/git"
	"github.com/Andrewy-gh/gwt/internal/tui/styles"
)

// CleanupFilter represents the current filter mode
type CleanupFilter int

const (
	FilterAll CleanupFilter = iota
	FilterMerged
	FilterStale
)

// BranchItem represents a branch in the cleanup list
type BranchItem struct {
	Info     git.BranchCleanupInfo
	Selected bool
}

// branchesLoadedMsg is sent when branches are loaded
type branchesLoadedMsg struct {
	branches []git.BranchCleanupInfo
	err      error
}

// branchesDeletedMsg is sent when branches are deleted
type branchesDeletedMsg struct {
	count int
	err   error
}

// CleanupBranchesModel is the branch cleanup view
type CleanupBranchesModel struct {
	repoPath    string
	baseBranch  string
	staleDays   int
	branches    []BranchItem
	cursor      int
	filter      CleanupFilter
	loading     bool
	deleting    bool
	cancelled   bool
	err         error
	width       int
	height      int
	selectAll   bool
	viewOffset  int // For scrolling
}

// NewCleanupBranchesModel creates a new cleanup branches view
func NewCleanupBranchesModel(repoPath string) *CleanupBranchesModel {
	return &CleanupBranchesModel{
		repoPath:   repoPath,
		baseBranch: git.GetDefaultBranch(repoPath),
		staleDays:  30,
		filter:     FilterAll,
		loading:    true,
	}
}

// Init initializes the model
func (m *CleanupBranchesModel) Init() tea.Cmd {
	return m.loadBranches()
}

// Update handles messages
func (m *CleanupBranchesModel) Update(msg tea.Msg) (*CleanupBranchesModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		if m.loading || m.deleting {
			return m, nil
		}

		switch {
		// Navigation
		case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
			m.moveCursor(-1)
		case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
			m.moveCursor(1)
		case key.Matches(msg, key.NewBinding(key.WithKeys("pgup"))):
			m.moveCursor(-10)
		case key.Matches(msg, key.NewBinding(key.WithKeys("pgdown"))):
			m.moveCursor(10)
		case key.Matches(msg, key.NewBinding(key.WithKeys("home"))):
			m.cursor = 0
			m.viewOffset = 0
		case key.Matches(msg, key.NewBinding(key.WithKeys("end"))):
			m.cursor = len(m.getFilteredBranches()) - 1

		// Selection
		case key.Matches(msg, key.NewBinding(key.WithKeys(" ", "x"))):
			m.toggleSelection()
		case key.Matches(msg, key.NewBinding(key.WithKeys("a"))):
			m.toggleSelectAll()

		// Filters
		case key.Matches(msg, key.NewBinding(key.WithKeys("1"))):
			m.filter = FilterAll
			m.cursor = 0
			m.viewOffset = 0
		case key.Matches(msg, key.NewBinding(key.WithKeys("2"))):
			m.filter = FilterMerged
			m.cursor = 0
			m.viewOffset = 0
		case key.Matches(msg, key.NewBinding(key.WithKeys("3"))):
			m.filter = FilterStale
			m.cursor = 0
			m.viewOffset = 0

		// Actions
		case key.Matches(msg, key.NewBinding(key.WithKeys("d", "D"))):
			return m, m.deleteSelected()
		case key.Matches(msg, key.NewBinding(key.WithKeys("r", "R"))):
			m.loading = true
			return m, m.loadBranches()

		// Cancel/Back
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc", "q"))):
			m.cancelled = true
		}

	case branchesLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.branches = make([]BranchItem, len(msg.branches))
			for i, b := range msg.branches {
				m.branches[i] = BranchItem{
					Info:     b,
					Selected: false,
				}
			}
		}

	case branchesDeletedMsg:
		m.deleting = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			// Reload branches after deletion
			m.loading = true
			return m, m.loadBranches()
		}
	}

	return m, nil
}

// View renders the view
func (m *CleanupBranchesModel) View(width, height int) string {
	m.width = width
	m.height = height

	if m.err != nil {
		return m.renderError()
	}

	if m.loading {
		return m.renderLoading()
	}

	if m.deleting {
		return m.renderDeleting()
	}

	return m.renderMain()
}

// renderLoading renders the loading state
func (m *CleanupBranchesModel) renderLoading() string {
	var b strings.Builder
	b.WriteString(styles.Title.Render("Cleanup Branches"))
	b.WriteString("\n\n")
	b.WriteString(styles.MutedText.Render("Loading branches..."))
	b.WriteString("\n")
	return b.String()
}

// renderDeleting renders the deleting state
func (m *CleanupBranchesModel) renderDeleting() string {
	var b strings.Builder
	b.WriteString(styles.Title.Render("Cleanup Branches"))
	b.WriteString("\n\n")
	b.WriteString(styles.WarningText.Render("Deleting branches..."))
	b.WriteString("\n")
	return b.String()
}

// renderError renders an error message
func (m *CleanupBranchesModel) renderError() string {
	var b strings.Builder
	b.WriteString(styles.Title.Render("Cleanup Branches"))
	b.WriteString("\n\n")
	b.WriteString(styles.ErrorText.Render("Error: " + m.err.Error()))
	b.WriteString("\n\n")
	b.WriteString(styles.Help.Render("Press r to retry, esc to return"))
	b.WriteString("\n")
	return b.String()
}

// renderMain renders the main view
func (m *CleanupBranchesModel) renderMain() string {
	var b strings.Builder

	// Title
	b.WriteString(styles.Title.Render("Cleanup Branches"))
	if m.baseBranch != "" {
		b.WriteString(styles.MutedText.Render(fmt.Sprintf(" (base: %s)", m.baseBranch)))
	}
	b.WriteString("\n\n")

	// Filter tabs
	b.WriteString(m.renderFilterTabs())
	b.WriteString("\n\n")

	// Branch list
	b.WriteString(m.renderBranchList())
	b.WriteString("\n")

	// Summary
	b.WriteString(m.renderSummary())
	b.WriteString("\n\n")

	// Help
	b.WriteString(m.renderHelp())
	b.WriteString("\n")

	return b.String()
}

// renderFilterTabs renders the filter toggle tabs
func (m *CleanupBranchesModel) renderFilterTabs() string {
	activeStyle := lipgloss.NewStyle().
		Background(styles.Primary).
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(0, 1).
		Bold(true)

	inactiveStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#374151")).
		Foreground(styles.Muted).
		Padding(0, 1)

	tabs := []string{"All", "Merged", "Stale"}
	var rendered []string

	for i, tab := range tabs {
		style := inactiveStyle
		if CleanupFilter(i) == m.filter {
			style = activeStyle
		}
		rendered = append(rendered, style.Render(fmt.Sprintf("%d:%s", i+1, tab)))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, rendered...)
}

// renderBranchList renders the list of branches
func (m *CleanupBranchesModel) renderBranchList() string {
	filtered := m.getFilteredBranches()
	if len(filtered) == 0 {
		return styles.MutedText.Render("No branches match the current filter")
	}

	var b strings.Builder

	// Header
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.Primary).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(styles.Border)

	headers := []string{"", "BRANCH", "AGE", "MERGED", "STALE", "WORKTREE"}
	widths := []int{3, 30, 15, 8, 8, 10}

	var headerCells []string
	for i, header := range headers {
		headerCells = append(headerCells, headerStyle.Width(widths[i]).Render(header))
	}
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, headerCells...))
	b.WriteString("\n")

	// Calculate visible rows (reserve space for header, summary, help)
	visibleRows := m.height - 15
	if visibleRows < 5 {
		visibleRows = 5
	}
	if visibleRows > len(filtered) {
		visibleRows = len(filtered)
	}

	// Adjust view offset for scrolling
	if m.cursor >= m.viewOffset+visibleRows {
		m.viewOffset = m.cursor - visibleRows + 1
	}
	if m.cursor < m.viewOffset {
		m.viewOffset = m.cursor
	}

	// Render visible rows
	endIdx := m.viewOffset + visibleRows
	if endIdx > len(filtered) {
		endIdx = len(filtered)
	}

	for i := m.viewOffset; i < endIdx; i++ {
		item := filtered[i]
		b.WriteString(m.renderBranchRow(item, i == m.cursor, widths))
		b.WriteString("\n")
	}

	// Scroll indicator
	if len(filtered) > visibleRows {
		scrollInfo := fmt.Sprintf("(%d-%d of %d)", m.viewOffset+1, endIdx, len(filtered))
		b.WriteString(styles.MutedText.Render(scrollInfo))
	}

	return b.String()
}

// renderBranchRow renders a single branch row
func (m *CleanupBranchesModel) renderBranchRow(item BranchItem, isCursor bool, widths []int) string {
	// Checkbox
	checkbox := styles.UncheckedBox
	if item.Selected {
		checkbox = styles.CheckedBox
	}

	// Cursor
	cursorStr := styles.NoCursor
	if isCursor {
		cursorStr = styles.CursorSymbol
	}

	// Branch name
	name := item.Info.Branch.Name
	if len(name) > widths[1]-2 {
		name = name[:widths[1]-5] + "..."
	}

	// Merged indicator
	merged := ""
	if item.Info.IsMerged {
		merged = "yes"
	}

	// Stale indicator
	stale := ""
	if item.Info.IsStale {
		stale = "yes"
	}

	// Worktree indicator
	worktree := ""
	if item.Info.HasWorktree {
		worktree = "yes"
	}

	// Build row with styling
	rowStyle := lipgloss.NewStyle()
	if isCursor {
		rowStyle = rowStyle.Background(lipgloss.Color("#374151"))
	}
	if item.Selected {
		rowStyle = rowStyle.Foreground(styles.Primary)
	}

	cells := []string{
		lipgloss.NewStyle().Width(widths[0]).Render(cursorStr + checkbox),
		rowStyle.Width(widths[1]).Render(name),
		rowStyle.Width(widths[2]).Render(item.Info.AgeString),
		rowStyle.Width(widths[3]).Render(merged),
		rowStyle.Width(widths[4]).Render(stale),
		rowStyle.Width(widths[5]).Render(worktree),
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, cells...)
}

// renderSummary renders the summary line
func (m *CleanupBranchesModel) renderSummary() string {
	filtered := m.getFilteredBranches()
	selectedCount := m.getSelectedCount()

	summaryParts := []string{
		fmt.Sprintf("Total: %d", len(filtered)),
	}

	if selectedCount > 0 {
		summaryParts = append(summaryParts, styles.SuccessText.Render(fmt.Sprintf("Selected: %d", selectedCount)))
	}

	// Count merged and stale
	mergedCount := 0
	staleCount := 0
	worktreeCount := 0
	for _, item := range filtered {
		if item.Info.IsMerged {
			mergedCount++
		}
		if item.Info.IsStale {
			staleCount++
		}
		if item.Info.HasWorktree {
			worktreeCount++
		}
	}

	if mergedCount > 0 {
		summaryParts = append(summaryParts, styles.WarningText.Render(fmt.Sprintf("Merged: %d", mergedCount)))
	}
	if staleCount > 0 {
		summaryParts = append(summaryParts, styles.MutedText.Render(fmt.Sprintf("Stale: %d", staleCount)))
	}
	if worktreeCount > 0 {
		summaryParts = append(summaryParts, styles.ErrorText.Render(fmt.Sprintf("With worktree: %d", worktreeCount)))
	}

	return styles.Box.Render(strings.Join(summaryParts, "  |  "))
}

// renderHelp renders the help text
func (m *CleanupBranchesModel) renderHelp() string {
	helpText := "j/k: navigate  space: toggle  a: select all  d: delete  1/2/3: filter  r: refresh  esc: back"
	return styles.Help.Render(helpText)
}

// getFilteredBranches returns branches matching the current filter
func (m *CleanupBranchesModel) getFilteredBranches() []BranchItem {
	switch m.filter {
	case FilterMerged:
		var filtered []BranchItem
		for _, item := range m.branches {
			if item.Info.IsMerged {
				filtered = append(filtered, item)
			}
		}
		return filtered
	case FilterStale:
		var filtered []BranchItem
		for _, item := range m.branches {
			if item.Info.IsStale {
				filtered = append(filtered, item)
			}
		}
		return filtered
	default:
		return m.branches
	}
}

// getSelectedCount returns the number of selected branches
func (m *CleanupBranchesModel) getSelectedCount() int {
	count := 0
	for _, item := range m.branches {
		if item.Selected {
			count++
		}
	}
	return count
}

// moveCursor moves the cursor up or down
func (m *CleanupBranchesModel) moveCursor(delta int) {
	filtered := m.getFilteredBranches()
	if len(filtered) == 0 {
		return
	}

	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(filtered) {
		m.cursor = len(filtered) - 1
	}
}

// toggleSelection toggles the selection of the current branch
func (m *CleanupBranchesModel) toggleSelection() {
	filtered := m.getFilteredBranches()
	if len(filtered) == 0 || m.cursor >= len(filtered) {
		return
	}

	// Find the branch in the full list
	branchName := filtered[m.cursor].Info.Branch.Name
	for i := range m.branches {
		if m.branches[i].Info.Branch.Name == branchName {
			// Don't allow selecting branches with worktrees
			if m.branches[i].Info.HasWorktree {
				return
			}
			m.branches[i].Selected = !m.branches[i].Selected
			break
		}
	}
}

// toggleSelectAll toggles selection of all visible branches
func (m *CleanupBranchesModel) toggleSelectAll() {
	filtered := m.getFilteredBranches()
	if len(filtered) == 0 {
		return
	}

	// Determine if we should select or deselect
	m.selectAll = !m.selectAll

	// Apply to all filtered branches (that don't have worktrees)
	filteredNames := make(map[string]bool)
	for _, item := range filtered {
		filteredNames[item.Info.Branch.Name] = true
	}

	for i := range m.branches {
		if filteredNames[m.branches[i].Info.Branch.Name] && !m.branches[i].Info.HasWorktree {
			m.branches[i].Selected = m.selectAll
		}
	}
}

// loadBranches loads branch cleanup info
func (m *CleanupBranchesModel) loadBranches() tea.Cmd {
	return func() tea.Msg {
		staleAge := time.Duration(m.staleDays) * 24 * time.Hour
		branches, err := git.GetBranchCleanupInfo(m.repoPath, m.baseBranch, staleAge)
		if err != nil {
			return branchesLoadedMsg{err: err}
		}
		return branchesLoadedMsg{branches: branches}
	}
}

// deleteSelected deletes selected branches
func (m *CleanupBranchesModel) deleteSelected() tea.Cmd {
	var toDelete []string
	for _, item := range m.branches {
		if item.Selected && !item.Info.HasWorktree {
			toDelete = append(toDelete, item.Info.Branch.Name)
		}
	}

	if len(toDelete) == 0 {
		return nil
	}

	m.deleting = true

	return func() tea.Msg {
		// Force delete for stale branches, normal delete for merged
		err := git.DeleteBranches(m.repoPath, toDelete, true)
		if err != nil {
			return branchesDeletedMsg{count: 0, err: err}
		}
		return branchesDeletedMsg{count: len(toDelete), err: nil}
	}
}

// IsCancelled returns true if user cancelled
func (m *CleanupBranchesModel) IsCancelled() bool {
	return m.cancelled
}

// IsComplete returns true if the view is complete
func (m *CleanupBranchesModel) IsComplete() bool {
	return m.cancelled
}
