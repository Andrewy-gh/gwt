package views

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
	"github.com/Andrewy-gh/gwt/internal/git"
	"github.com/Andrewy-gh/gwt/internal/tui/styles"
)

// CheckStatus represents the result of a pre-deletion check
type CheckStatus int

const (
	CheckPass CheckStatus = iota
	CheckWarn
	CheckBlock
)

// PreDeleteCheck represents a single pre-deletion check result
type PreDeleteCheck struct {
	Name    string
	Status  CheckStatus
	Message string
}

// DeleteTarget represents a worktree to be deleted with its check results
type DeleteTarget struct {
	Worktree *git.Worktree
	Checks   []PreDeleteCheck
	Blocked  bool
	Error    error
}

// Local message types for delete confirmation

// deleteChecksCompleteMsg is sent when pre-flight checks are complete
type deleteChecksCompleteMsg struct {
	targets []*DeleteTarget
	err     error
}

// DeleteConfirmModel is the delete confirmation view
type DeleteConfirmModel struct {
	repoPath      string
	worktreePaths []string
	targets       []*DeleteTarget
	loading       bool
	confirmed     bool
	cancelled     bool
	width         int
	height        int
	err           error
	blockedCount  int
	warningCount  int
}

// NewDeleteConfirmModel creates a new delete confirmation view
func NewDeleteConfirmModel(repoPath string, worktreePaths []string) *DeleteConfirmModel {
	return &DeleteConfirmModel{
		repoPath:      repoPath,
		worktreePaths: worktreePaths,
		loading:       true,
	}
}

// Init initializes the model
func (m *DeleteConfirmModel) Init() tea.Cmd {
	return m.runPreflightChecks()
}

// Update handles messages
func (m *DeleteConfirmModel) Update(msg tea.Msg) (*DeleteConfirmModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		if m.loading {
			// Ignore input while loading
			return m, nil
		}

		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("y", "Y"))):
			// Confirm deletion
			if m.blockedCount == 0 {
				m.confirmed = true
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("n", "N", "esc"))):
			// Cancel deletion
			m.cancelled = true
		}

	case deleteChecksCompleteMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.targets = msg.targets
			// Count blocked and warnings
			m.countIssues()
		}
	}

	return m, nil
}

// View renders the view
func (m *DeleteConfirmModel) View(width, height int) string {
	m.width = width
	m.height = height

	if m.err != nil {
		return m.renderError()
	}

	if m.loading {
		return m.renderLoading()
	}

	return m.renderConfirmation()
}

// renderLoading renders the loading state
func (m *DeleteConfirmModel) renderLoading() string {
	var b strings.Builder
	b.WriteString(styles.Title.Render("Delete Worktrees - Pre-flight Checks"))
	b.WriteString("\n\n")
	b.WriteString(styles.MutedText.Render("Running pre-flight checks..."))
	b.WriteString("\n")
	return b.String()
}

// renderError renders an error message
func (m *DeleteConfirmModel) renderError() string {
	var b strings.Builder
	b.WriteString(styles.Title.Render("Delete Worktrees"))
	b.WriteString("\n\n")
	b.WriteString(styles.ErrorText.Render("Error: " + m.err.Error()))
	b.WriteString("\n\n")
	b.WriteString(styles.MutedText.Render("Press esc to return"))
	b.WriteString("\n")
	return b.String()
}

// renderConfirmation renders the confirmation view
func (m *DeleteConfirmModel) renderConfirmation() string {
	var b strings.Builder

	// Title
	b.WriteString(styles.Title.Render("Delete Worktrees - Confirmation"))
	b.WriteString("\n\n")

	// Targets table
	b.WriteString(m.renderTargetsTable())
	b.WriteString("\n")

	// Summary
	b.WriteString(m.renderSummary())
	b.WriteString("\n\n")

	// Confirmation prompt
	b.WriteString(m.renderPrompt())
	b.WriteString("\n")

	return b.String()
}

// renderTargetsTable renders the table of targets with their check results
func (m *DeleteConfirmModel) renderTargetsTable() string {
	var b strings.Builder

	// Header
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.Primary).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(styles.Border)

	headers := []string{"Path", "Branch", "Status", "Issues"}
	widths := []int{35, 20, 10, 50}

	var headerCells []string
	for i, header := range headers {
		headerCells = append(headerCells, headerStyle.Width(widths[i]).Render(header))
	}
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, headerCells...))
	b.WriteString("\n")

	// Rows
	for _, target := range m.targets {
		if target.Error != nil {
			// Error row
			cells := []string{
				lipgloss.NewStyle().Width(widths[0]).Render(m.formatPath(target)),
				lipgloss.NewStyle().Width(widths[1]).Render("N/A"),
				styles.ErrorText.Width(widths[2]).Render("ERROR"),
				styles.ErrorText.Width(widths[3]).Render(target.Error.Error()),
			}
			b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, cells...))
			b.WriteString("\n")
			continue
		}

		if target.Worktree == nil {
			continue
		}

		// Path
		path := m.formatPath(target)

		// Branch
		branch := target.Worktree.Branch
		if target.Worktree.IsDetached {
			branch = fmt.Sprintf("(detached at %s)", target.Worktree.Commit)
		}
		if len(branch) > 18 {
			branch = branch[:15] + "..."
		}

		// Status indicator
		statusText, statusStyle := m.getStatusStyle(target)

		// Issues
		issues := m.formatIssues(target)

		// Build row
		cells := []string{
			lipgloss.NewStyle().Width(widths[0]).Render(path),
			lipgloss.NewStyle().Width(widths[1]).Render(branch),
			statusStyle.Width(widths[2]).Render(statusText),
			lipgloss.NewStyle().Width(widths[3]).Render(issues),
		}

		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, cells...))
		b.WriteString("\n")
	}

	return b.String()
}

// renderSummary renders the summary line
func (m *DeleteConfirmModel) renderSummary() string {
	var b strings.Builder

	totalCount := len(m.targets)
	okCount := totalCount - m.blockedCount - m.warningCount

	// Summary box
	summaryParts := []string{
		fmt.Sprintf("Total: %d", totalCount),
	}

	if okCount > 0 {
		summaryParts = append(summaryParts, styles.SuccessText.Render(fmt.Sprintf("OK: %d", okCount)))
	}

	if m.warningCount > 0 {
		summaryParts = append(summaryParts, styles.WarningText.Render(fmt.Sprintf("Warnings: %d", m.warningCount)))
	}

	if m.blockedCount > 0 {
		summaryParts = append(summaryParts, styles.ErrorText.Render(fmt.Sprintf("Blocked: %d", m.blockedCount)))
	}

	summary := strings.Join(summaryParts, "  |  ")
	b.WriteString(styles.Box.Render(summary))

	return b.String()
}

// renderPrompt renders the confirmation prompt
func (m *DeleteConfirmModel) renderPrompt() string {
	var b strings.Builder

	if m.blockedCount > 0 {
		b.WriteString(styles.ErrorText.Render("⚠ Cannot delete: some worktrees are blocked"))
		b.WriteString("\n\n")
		b.WriteString(styles.Help.Render("Press esc to return"))
	} else {
		if m.warningCount > 0 {
			b.WriteString(styles.WarningText.Render("⚠ Some worktrees have warnings"))
			b.WriteString("\n\n")
		}

		okCount := len(m.targets) - m.blockedCount - m.warningCount
		b.WriteString(styles.Title.Render(fmt.Sprintf("Delete %d worktree(s)?", okCount+m.warningCount)))
		b.WriteString("\n\n")
		b.WriteString(styles.Help.Render("Y: confirm  N/esc: cancel"))
	}

	return b.String()
}

// formatPath formats a target's path for display
func (m *DeleteConfirmModel) formatPath(target *DeleteTarget) string {
	if target.Worktree == nil {
		return "Unknown"
	}

	path := target.Worktree.Path
	if relPath, err := filepath.Rel(m.repoPath, path); err == nil {
		path = relPath
	}

	if len(path) > 33 {
		path = "..." + path[len(path)-30:]
	}

	return path
}

// getStatusStyle returns the status text and style for a target
func (m *DeleteConfirmModel) getStatusStyle(target *DeleteTarget) (string, lipgloss.Style) {
	if target.Blocked {
		return "BLOCK", styles.ErrorText
	}

	hasWarnings := false
	for _, check := range target.Checks {
		if check.Status == CheckWarn {
			hasWarnings = true
			break
		}
	}

	if hasWarnings {
		return "WARN", styles.WarningText
	}

	return "OK", styles.SuccessText
}

// formatIssues formats the issues for a target
func (m *DeleteConfirmModel) formatIssues(target *DeleteTarget) string {
	var issues []string

	for _, check := range target.Checks {
		if check.Status == CheckBlock || check.Status == CheckWarn {
			prefix := ""
			if check.Status == CheckBlock {
				prefix = "🚫 "
			} else {
				prefix = "⚠ "
			}
			issues = append(issues, prefix+check.Message)
		}
	}

	if len(issues) == 0 {
		return "No issues"
	}

	// Limit to first 2 issues, add "..." if more
	if len(issues) > 2 {
		return strings.Join(issues[:2], "; ") + "..."
	}

	return strings.Join(issues, "; ")
}

// countIssues counts blocked and warning targets
func (m *DeleteConfirmModel) countIssues() {
	m.blockedCount = 0
	m.warningCount = 0

	for _, target := range m.targets {
		if target.Blocked {
			m.blockedCount++
			continue
		}

		hasWarnings := false
		for _, check := range target.Checks {
			if check.Status == CheckWarn {
				hasWarnings = true
				break
			}
		}

		if hasWarnings {
			m.warningCount++
		}
	}
}

// runPreflightChecks performs pre-flight checks on all targets
func (m *DeleteConfirmModel) runPreflightChecks() tea.Cmd {
	return func() tea.Msg {
		targets := make([]*DeleteTarget, 0, len(m.worktreePaths))

		for _, path := range m.worktreePaths {
			target := &DeleteTarget{}

			// Get worktree info
			wt, err := git.GetWorktree(path)
			if err != nil {
				target.Error = fmt.Errorf("failed to get worktree: %w", err)
				targets = append(targets, target)
				continue
			}

			target.Worktree = wt

			// Run checks
			checks := runDeleteChecks(m.repoPath, wt)
			target.Checks = checks

			// Determine if blocked
			for _, check := range checks {
				if check.Status == CheckBlock {
					target.Blocked = true
					break
				}
			}

			targets = append(targets, target)
		}

		return deleteChecksCompleteMsg{
			targets: targets,
			err:     nil,
		}
	}
}

// runDeleteChecks performs pre-deletion checks on a worktree
func runDeleteChecks(repoPath string, wt *git.Worktree) []PreDeleteCheck {
	var checks []PreDeleteCheck

	// Check: Is this the main worktree?
	if wt.IsMain {
		checks = append(checks, PreDeleteCheck{
			Name:    "Main Worktree",
			Status:  CheckBlock,
			Message: "Cannot delete the main worktree",
		})
		return checks // No need to check further
	}

	// Check: Is this the current directory?
	cwd, err := filepath.Abs(".")
	if err == nil {
		wtPath, err2 := filepath.Abs(wt.Path)
		if err2 == nil && wtPath == cwd {
			checks = append(checks, PreDeleteCheck{
				Name:    "Current Directory",
				Status:  CheckBlock,
				Message: "Cannot delete current directory",
			})
			return checks // No need to check further
		}
	}

	// Check: Is worktree locked?
	if wt.Locked {
		checks = append(checks, PreDeleteCheck{
			Name:    "Locked",
			Status:  CheckWarn,
			Message: "Worktree is locked",
		})
	}

	// Check: Uncommitted changes
	status, err := git.GetWorktreeStatus(wt.Path)
	if err == nil && !status.Clean {
		msg := fmt.Sprintf("%d staged, %d unstaged, %d untracked",
			status.StagedCount, status.UnstagedCount, status.UntrackedCount)
		checks = append(checks, PreDeleteCheck{
			Name:    "Uncommitted Changes",
			Status:  CheckWarn,
			Message: "Has uncommitted changes: " + msg,
		})
	}

	// Check: Unpushed commits
	if err == nil && status.AheadCount > 0 {
		checks = append(checks, PreDeleteCheck{
			Name:    "Unpushed Commits",
			Status:  CheckWarn,
			Message: fmt.Sprintf("%d unpushed commit(s)", status.AheadCount),
		})
	}

	// Check: Unmerged branch (if not detached)
	if !wt.IsDetached && wt.Branch != "" {
		// Check if branch is merged into main/master
		mainBranch := "main"
		merged, err := git.IsBranchMerged(repoPath, wt.Branch, mainBranch)
		if err != nil {
			// Try master
			mainBranch = "master"
			merged, err = git.IsBranchMerged(repoPath, wt.Branch, mainBranch)
		}

		if err == nil && !merged {
			checks = append(checks, PreDeleteCheck{
				Name:    "Unmerged Branch",
				Status:  CheckWarn,
				Message: fmt.Sprintf("Branch '%s' is not merged into %s", wt.Branch, mainBranch),
			})
		}
	}

	// If no issues found, add an OK check
	if len(checks) == 0 {
		checks = append(checks, PreDeleteCheck{
			Name:    "All Checks",
			Status:  CheckPass,
			Message: "No issues found",
		})
	}

	return checks
}

// Accessors for root model

// IsConfirmed returns true if user confirmed deletion
func (m *DeleteConfirmModel) IsConfirmed() bool {
	return m.confirmed
}

// IsCancelled returns true if user cancelled
func (m *DeleteConfirmModel) IsCancelled() bool {
	return m.cancelled
}

// GetTargets returns the targets to delete
func (m *DeleteConfirmModel) GetTargets() []*DeleteTarget {
	return m.targets
}

// GetNonBlockedTargets returns only targets that are not blocked
func (m *DeleteConfirmModel) GetNonBlockedTargets() []*DeleteTarget {
	var nonBlocked []*DeleteTarget
	for _, target := range m.targets {
		if !target.Blocked && target.Error == nil {
			nonBlocked = append(nonBlocked, target)
		}
	}
	return nonBlocked
}
