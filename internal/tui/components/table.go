package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
	"github.com/Andrewy-gh/gwt/internal/tui/styles"
)

// Table is a styled table component for displaying data
type Table struct {
	Headers    []string
	Rows       [][]string
	Widths     []int
	Cursor     int
	Selectable bool
	Height     int // Viewport height for scrolling
	offset     int // Scroll offset
}

// NewTable creates a new table component
func NewTable(headers []string, rows [][]string, selectable bool, height int) *Table {
	t := &Table{
		Headers:    headers,
		Rows:       rows,
		Selectable: selectable,
		Height:     height,
		Cursor:     0,
		offset:     0,
	}
	t.calculateWidths()
	return t
}

// Init initializes the component
func (t *Table) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (t *Table) Update(msg tea.Msg) tea.Cmd {
	if !t.Selectable {
		return nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
			t.CursorUp()
		case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
			t.CursorDown()
		}
	}
	return nil
}

// View renders the component
func (t *Table) View() string {
	var b strings.Builder

	// Header row
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.Primary).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(styles.Border)

	var headerRow []string
	for i, header := range t.Headers {
		width := t.Widths[i]
		headerRow = append(headerRow, headerStyle.Width(width).Render(header))
	}
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, headerRow...))
	b.WriteString("\n")

	// Calculate visible range
	visibleStart := t.offset
	visibleEnd := t.offset + t.Height
	if visibleEnd > len(t.Rows) {
		visibleEnd = len(t.Rows)
	}

	// Data rows
	for i := visibleStart; i < visibleEnd; i++ {
		row := t.Rows[i]

		var cellStyle lipgloss.Style
		if t.Selectable && i == t.Cursor {
			cellStyle = lipgloss.NewStyle().
				Foreground(styles.Primary).
				Bold(true)
		} else {
			cellStyle = lipgloss.NewStyle()
		}

		var cells []string
		for j, cell := range row {
			if j < len(t.Widths) {
				width := t.Widths[j]
				cells = append(cells, cellStyle.Width(width).Render(cell))
			}
		}

		// Add cursor indicator
		if t.Selectable && i == t.Cursor {
			b.WriteString(styles.Cursor.Render("> "))
		} else {
			b.WriteString("  ")
		}

		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, cells...))
		b.WriteString("\n")
	}

	// Scroll indicators
	if t.offset > 0 {
		b.WriteString(styles.MutedText.Render("  ▲ More rows above\n"))
	}
	if visibleEnd < len(t.Rows) {
		b.WriteString(styles.MutedText.Render("  ▼ More rows below\n"))
	}

	return b.String()
}

// CursorUp moves the cursor up
func (t *Table) CursorUp() {
	if t.Cursor > 0 {
		t.Cursor--
		if t.Cursor < t.offset {
			t.offset = t.Cursor
		}
	}
}

// CursorDown moves the cursor down
func (t *Table) CursorDown() {
	if t.Cursor < len(t.Rows)-1 {
		t.Cursor++
		if t.Cursor >= t.offset+t.Height {
			t.offset = t.Cursor - t.Height + 1
		}
	}
}

// SelectedRow returns the currently selected row
func (t *Table) SelectedRow() []string {
	if t.Cursor >= 0 && t.Cursor < len(t.Rows) {
		return t.Rows[t.Cursor]
	}
	return nil
}

// SetRows updates the table rows and recalculates widths
func (t *Table) SetRows(rows [][]string) {
	t.Rows = rows
	t.Cursor = 0
	t.offset = 0
	t.calculateWidths()
}

// calculateWidths automatically calculates column widths based on content
func (t *Table) calculateWidths() {
	if len(t.Headers) == 0 {
		return
	}

	// Initialize widths with header lengths
	t.Widths = make([]int, len(t.Headers))
	for i, header := range t.Headers {
		t.Widths[i] = len(header)
	}

	// Update widths based on row content
	for _, row := range t.Rows {
		for i, cell := range row {
			if i < len(t.Widths) {
				if len(cell) > t.Widths[i] {
					t.Widths[i] = len(cell)
				}
			}
		}
	}

	// Add padding
	for i := range t.Widths {
		t.Widths[i] += 2
	}
}

// SetWidths manually sets column widths
func (t *Table) SetWidths(widths []int) {
	t.Widths = widths
}

// RowCount returns the number of rows
func (t *Table) RowCount() int {
	return len(t.Rows)
}

// IsEmpty returns true if the table has no rows
func (t *Table) IsEmpty() bool {
	return len(t.Rows) == 0
}

// SetCursor sets the cursor position
func (t *Table) SetCursor(cursor int) {
	if cursor >= 0 && cursor < len(t.Rows) {
		t.Cursor = cursor
		// Adjust offset if needed
		if t.Cursor < t.offset {
			t.offset = t.Cursor
		} else if t.Cursor >= t.offset+t.Height {
			t.offset = t.Cursor - t.Height + 1
		}
	}
}

// GetCursor returns the current cursor position
func (t *Table) GetCursor() int {
	return t.Cursor
}

// SetSelectable sets whether the table allows row selection
func (t *Table) SetSelectable(selectable bool) {
	t.Selectable = selectable
}

// FormatWithStyle renders the table with custom styling
func (t *Table) FormatWithStyle(rowStyle func(int) lipgloss.Style) string {
	var b strings.Builder

	// Header row (same as before)
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.Primary).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(styles.Border)

	var headerRow []string
	for i, header := range t.Headers {
		width := t.Widths[i]
		headerRow = append(headerRow, headerStyle.Width(width).Render(header))
	}
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, headerRow...))
	b.WriteString("\n")

	// Data rows with custom styling
	visibleStart := t.offset
	visibleEnd := t.offset + t.Height
	if visibleEnd > len(t.Rows) {
		visibleEnd = len(t.Rows)
	}

	for i := visibleStart; i < visibleEnd; i++ {
		row := t.Rows[i]
		cellStyle := rowStyle(i)

		var cells []string
		for j, cell := range row {
			if j < len(t.Widths) {
				width := t.Widths[j]
				cells = append(cells, cellStyle.Width(width).Render(cell))
			}
		}

		b.WriteString(fmt.Sprintf("  %s\n", lipgloss.JoinHorizontal(lipgloss.Top, cells...)))
	}

	return b.String()
}
