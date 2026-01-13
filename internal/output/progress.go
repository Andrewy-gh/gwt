package output

import (
	"fmt"
	"strings"
)

// ProgressBar displays a simple progress bar
type ProgressBar struct {
	total   int
	current int
	width   int
}

// NewProgressBar creates a progress bar
func NewProgressBar(total, width int) *ProgressBar {
	return &ProgressBar{
		total:   total,
		current: 0,
		width:   width,
	}
}

// Update updates the progress bar
func (p *ProgressBar) Update(current int, message string) {
	p.current = current

	// Calculate percentage
	percentage := 0
	if p.total > 0 {
		percentage = (current * 100) / p.total
	}

	// Calculate filled width
	filled := 0
	if p.total > 0 {
		filled = (current * p.width) / p.total
	}

	// Build progress bar
	bar := strings.Repeat("█", filled) + strings.Repeat("░", p.width-filled)

	// Print progress line (with carriage return to overwrite)
	fmt.Printf("\rCopying files... [%s] %d%% (%d/%d) %s",
		bar,
		percentage,
		current,
		p.total,
		message,
	)
}

// Done marks progress as complete
func (p *ProgressBar) Done() {
	// Print newline to move to next line
	fmt.Println()
}
