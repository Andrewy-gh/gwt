package styles

import "github.com/charmbracelet/lipgloss"

// Color palette
var (
	Primary   = lipgloss.Color("#7C3AED") // Purple
	Secondary = lipgloss.Color("#06B6D4") // Cyan
	Success   = lipgloss.Color("#22C55E") // Green
	Warning   = lipgloss.Color("#F59E0B") // Amber
	Error     = lipgloss.Color("#EF4444") // Red
	Muted     = lipgloss.Color("#6B7280") // Gray
	Text      = lipgloss.Color("#F9FAFB") // Light
	Border    = lipgloss.Color("#374151") // Dark gray
)

// Component styles
var (
	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(Primary).
		MarginBottom(1)

	Subtitle = lipgloss.NewStyle().
			Foreground(Muted).
			Italic(true)

	Selected = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true)

	Cursor = lipgloss.NewStyle().
		Foreground(Secondary)

	Help = lipgloss.NewStyle().
		Foreground(Muted).
		MarginTop(1)

	Box = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Border).
		Padding(1, 2)

	StatusBar = lipgloss.NewStyle().
			Background(lipgloss.Color("#1F2937")).
			Foreground(Text).
			Padding(0, 1)

	ErrorText = lipgloss.NewStyle().
			Foreground(Error)

	SuccessText = lipgloss.NewStyle().
			Foreground(Success)

	WarningText = lipgloss.NewStyle().
			Foreground(Warning)

	MutedText = lipgloss.NewStyle().
			Foreground(Muted)
)

// Checkbox symbols
const (
	CheckedBox   = "[✓]"
	UncheckedBox = "[ ]"
	CursorSymbol = ">"
	NoCursor     = " "
)
