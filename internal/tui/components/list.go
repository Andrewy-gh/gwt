package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/key"
	"github.com/Andrewy-gh/gwt/internal/tui/styles"
)

// CheckboxItem represents an item in the checkbox list
type CheckboxItem struct {
	Label       string
	Description string
	Value       interface{}
	Disabled    bool
}

// CheckboxList is a multi-select list with checkboxes
type CheckboxList struct {
	Items    []CheckboxItem
	Cursor   int
	Selected map[int]bool
	Title    string
	Height   int // Viewport height for scrolling
	offset   int // Scroll offset
}

// NewCheckboxList creates a new checkbox list
func NewCheckboxList(title string, items []CheckboxItem, height int) *CheckboxList {
	return &CheckboxList{
		Title:    title,
		Items:    items,
		Cursor:   0,
		Selected: make(map[int]bool),
		Height:   height,
		offset:   0,
	}
}

// Init initializes the component
func (c *CheckboxList) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (c *CheckboxList) Update(msg tea.Msg, keys interface{}) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
			c.CursorUp()
		case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
			c.CursorDown()
		case key.Matches(msg, key.NewBinding(key.WithKeys(" ", "x"))):
			c.Toggle()
		case key.Matches(msg, key.NewBinding(key.WithKeys("a"))):
			c.SelectAll()
		case key.Matches(msg, key.NewBinding(key.WithKeys("n"))):
			c.DeselectAll()
		}
	}
	return nil
}

// View renders the component
func (c *CheckboxList) View() string {
	var b strings.Builder

	// Title
	if c.Title != "" {
		b.WriteString(styles.Title.Render(c.Title))
		b.WriteString("\n\n")
	}

	// Calculate visible range
	visibleStart := c.offset
	visibleEnd := c.offset + c.Height
	if visibleEnd > len(c.Items) {
		visibleEnd = len(c.Items)
	}

	// Render items
	for i := visibleStart; i < visibleEnd; i++ {
		item := c.Items[i]

		// Cursor
		cursor := styles.NoCursor
		if i == c.Cursor {
			cursor = styles.CursorSymbol
		}

		// Checkbox
		checkbox := styles.UncheckedBox
		if c.Selected[i] {
			checkbox = styles.CheckedBox
		}

		// Apply styles
		var line string
		if item.Disabled {
			line = fmt.Sprintf("%s %s %s",
				styles.MutedText.Render(cursor),
				styles.MutedText.Render(checkbox),
				styles.MutedText.Render(item.Label),
			)
		} else if i == c.Cursor {
			line = fmt.Sprintf("%s %s %s",
				styles.Cursor.Render(cursor),
				styles.Selected.Render(checkbox),
				styles.Selected.Render(item.Label),
			)
		} else {
			line = fmt.Sprintf("%s %s %s",
				cursor,
				checkbox,
				item.Label,
			)
		}

		b.WriteString(line)

		// Description
		if item.Description != "" {
			b.WriteString("\n  ")
			b.WriteString(styles.Subtitle.Render(item.Description))
		}

		b.WriteString("\n")
	}

	// Scroll indicators
	if c.offset > 0 {
		b.WriteString(styles.MutedText.Render("  ▲ More items above\n"))
	}
	if visibleEnd < len(c.Items) {
		b.WriteString(styles.MutedText.Render("  ▼ More items below\n"))
	}

	return b.String()
}

// CursorUp moves the cursor up
func (c *CheckboxList) CursorUp() {
	if c.Cursor > 0 {
		c.Cursor--
		// Adjust scroll offset
		if c.Cursor < c.offset {
			c.offset = c.Cursor
		}
	}
}

// CursorDown moves the cursor down
func (c *CheckboxList) CursorDown() {
	if c.Cursor < len(c.Items)-1 {
		c.Cursor++
		// Adjust scroll offset
		if c.Cursor >= c.offset+c.Height {
			c.offset = c.Cursor - c.Height + 1
		}
	}
}

// Toggle toggles the current item's selection
func (c *CheckboxList) Toggle() {
	if c.Cursor < len(c.Items) && !c.Items[c.Cursor].Disabled {
		c.Selected[c.Cursor] = !c.Selected[c.Cursor]
	}
}

// SelectAll selects all non-disabled items
func (c *CheckboxList) SelectAll() {
	for i, item := range c.Items {
		if !item.Disabled {
			c.Selected[i] = true
		}
	}
}

// DeselectAll deselects all items
func (c *CheckboxList) DeselectAll() {
	c.Selected = make(map[int]bool)
}

// GetSelected returns the selected items
func (c *CheckboxList) GetSelected() []CheckboxItem {
	var selected []CheckboxItem
	for i, isSelected := range c.Selected {
		if isSelected {
			selected = append(selected, c.Items[i])
		}
	}
	return selected
}

// GetSelectedValues returns the values of selected items
func (c *CheckboxList) GetSelectedValues() []interface{} {
	var values []interface{}
	for i, isSelected := range c.Selected {
		if isSelected {
			values = append(values, c.Items[i].Value)
		}
	}
	return values
}

// HasSelection returns true if any items are selected
func (c *CheckboxList) HasSelection() bool {
	return len(c.Selected) > 0
}
