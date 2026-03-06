package components

import (
	"strings"

	"github.com/Andrewy-gh/gwt/internal/tui/styles"
	"github.com/charmbracelet/bubbles/key"
)

// Help is a contextual help display component
type Help struct {
	Keys     []key.Binding
	ShowFull bool
}

// NewHelp creates a new help component
func NewHelp(keys []key.Binding) *Help {
	return &Help{
		Keys:     keys,
		ShowFull: false,
	}
}

// View renders the help footer
func (h *Help) View() string {
	if h.ShowFull {
		return h.FullHelp()
	}
	return h.ShortHelp()
}

// ShortHelp renders a compact help view
func (h *Help) ShortHelp() string {
	if len(h.Keys) == 0 {
		return ""
	}

	var parts []string
	for _, k := range h.Keys {
		if k.Help().Key != "" {
			parts = append(parts, k.Help().Key+" "+k.Help().Desc)
		}
	}

	help := strings.Join(parts, " • ")
	return styles.Help.Render(help)
}

// FullHelp renders an expanded help view with all key bindings
func (h *Help) FullHelp() string {
	if len(h.Keys) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString(styles.Title.Render("Keyboard Shortcuts"))
	b.WriteString("\n\n")

	// Group keys in columns
	const keysPerColumn = 4
	for i, k := range h.Keys {
		if k.Help().Key != "" {
			b.WriteString(styles.Subtitle.Render(k.Help().Key))
			b.WriteString("  ")
			b.WriteString(k.Help().Desc)
			b.WriteString("\n")
		}

		// Add spacing between columns
		if (i+1)%keysPerColumn == 0 && i < len(h.Keys)-1 {
			b.WriteString("\n")
		}
	}

	return styles.Box.Render(b.String())
}

// Toggle toggles between short and full help
func (h *Help) Toggle() {
	h.ShowFull = !h.ShowFull
}

// SetShowFull sets whether to show full help
func (h *Help) SetShowFull(show bool) {
	h.ShowFull = show
}

// SetKeys updates the key bindings
func (h *Help) SetKeys(keys []key.Binding) {
	h.Keys = keys
}

// AddKey adds a key binding to the help
func (h *Help) AddKey(k key.Binding) {
	h.Keys = append(h.Keys, k)
}

// Clear removes all key bindings
func (h *Help) Clear() {
	h.Keys = nil
}

// ViewWithCustomFormat renders help with custom formatting
func (h *Help) ViewWithCustomFormat(separator string, prefix string) string {
	if len(h.Keys) == 0 {
		return ""
	}

	var parts []string
	for _, k := range h.Keys {
		if k.Help().Key != "" {
			parts = append(parts, k.Help().Key+" "+k.Help().Desc)
		}
	}

	help := prefix + strings.Join(parts, separator)
	return styles.Help.Render(help)
}

// RenderGroup renders a group of key bindings with a title
func RenderGroup(title string, keys []key.Binding) string {
	var b strings.Builder

	b.WriteString(styles.Subtitle.Render(title))
	b.WriteString("\n")

	for _, k := range keys {
		if k.Help().Key != "" {
			b.WriteString("  ")
			b.WriteString(styles.Cursor.Render(k.Help().Key))
			b.WriteString("  ")
			b.WriteString(k.Help().Desc)
			b.WriteString("\n")
		}
	}

	return b.String()
}

// RenderMultipleGroups renders multiple groups of key bindings
func RenderMultipleGroups(groups map[string][]key.Binding) string {
	var b strings.Builder

	b.WriteString(styles.Title.Render("Keyboard Shortcuts"))
	b.WriteString("\n\n")

	for title, keys := range groups {
		b.WriteString(RenderGroup(title, keys))
		b.WriteString("\n")
	}

	return styles.Box.Render(b.String())
}
