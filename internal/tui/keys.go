package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines the key bindings for the TUI
type KeyMap struct {
	Up          key.Binding
	Down        key.Binding
	Select      key.Binding
	Confirm     key.Binding
	Back        key.Binding
	Quit        key.Binding
	Help        key.Binding
	Toggle      key.Binding
	SelectAll   key.Binding
	DeselectAll key.Binding
}

// DefaultKeyMap provides the default key bindings
var DefaultKeyMap = KeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Select: key.NewBinding(
		key.WithKeys(" ", "x"),
		key.WithHelp("space/x", "toggle"),
	),
	Confirm: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "confirm"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc", "backspace"),
		key.WithHelp("esc", "back"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Toggle: key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "toggle"),
	),
	SelectAll: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "select all"),
	),
	DeselectAll: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "select none"),
	),
}

// ShortHelp returns a short help message
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Confirm, k.Quit}
}

// FullHelp returns a full help message
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Select, k.Confirm},
		{k.SelectAll, k.DeselectAll, k.Back, k.Quit},
		{k.Help},
	}
}
