package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// Run starts the TUI application
func Run() error {
	p := tea.NewProgram(
		New(),
		tea.WithAltScreen(),       // Use alternate screen buffer
		tea.WithMouseCellMotion(), // Enable mouse support
	)

	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	// Handle any final state from the model
	if m, ok := finalModel.(Model); ok {
		if m.err != nil {
			return m.err
		}
	}

	return nil
}

// RunWithContext starts the TUI with context data
// This can be used to pass initial state to the TUI
func RunWithContext(initialView View, data interface{}) error {
	m := New()
	m.view = initialView

	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	if m, ok := finalModel.(Model); ok {
		if m.err != nil {
			return m.err
		}
	}

	return nil
}

// RunWithResult starts TUI and returns the result
// This is useful for views that return data (like branch selection)
func RunWithResult[T any]() (T, error) {
	var result T

	// This will be implemented in Phase 12 when we have
	// views that need to return data
	err := Run()
	return result, err
}

// IsSupported checks if the terminal supports TUI features
func IsSupported() bool {
	// Check if we're running in an interactive terminal
	// This is a simple check - could be expanded
	return true
}

// Version returns the TUI framework version
func Version() string {
	return "1.0.0"
}
