package components

import (
	"fmt"
	"strings"

	"github.com/Andrewy-gh/gwt/internal/tui/styles"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// Radio button symbols
const (
	RadioSelected   = "(●)"
	RadioUnselected = "( )"
)

// RadioItem represents an item in the radio list
type RadioItem struct {
	Label       string
	Description string
	Value       string
	Disabled    bool
}

// RadioList is a single-select list with radio buttons
type RadioList struct {
	Items    []RadioItem
	Cursor   int
	Selected int // Index of selected item (-1 for none)
	Title    string
	Height   int // Viewport height for scrolling
	offset   int // Scroll offset
}

// NewRadioList creates a new radio list
func NewRadioList(title string, items []RadioItem, height int) *RadioList {
	return &RadioList{
		Title:    title,
		Items:    items,
		Cursor:   0,
		Selected: -1,
		Height:   height,
		offset:   0,
	}
}

// Init initializes the component
func (r *RadioList) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (r *RadioList) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
			r.CursorUp()
		case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
			r.CursorDown()
		case key.Matches(msg, key.NewBinding(key.WithKeys(" ", "enter"))):
			r.Select()
		}
	}
	return nil
}

// View renders the component
func (r *RadioList) View() string {
	var b strings.Builder

	// Title
	if r.Title != "" {
		b.WriteString(styles.Title.Render(r.Title))
		b.WriteString("\n\n")
	}

	// Calculate visible range
	visibleStart := r.offset
	visibleEnd := r.offset + r.Height
	if visibleEnd > len(r.Items) {
		visibleEnd = len(r.Items)
	}

	// Render items
	for i := visibleStart; i < visibleEnd; i++ {
		item := r.Items[i]

		// Cursor
		cursor := styles.NoCursor
		if i == r.Cursor {
			cursor = styles.CursorSymbol
		}

		// Radio button
		radio := RadioUnselected
		if i == r.Selected {
			radio = RadioSelected
		}

		// Apply styles
		var line string
		if item.Disabled {
			line = fmt.Sprintf("%s %s %s",
				styles.MutedText.Render(cursor),
				styles.MutedText.Render(radio),
				styles.MutedText.Render(item.Label),
			)
		} else if i == r.Cursor {
			line = fmt.Sprintf("%s %s %s",
				styles.Cursor.Render(cursor),
				styles.Selected.Render(radio),
				styles.Selected.Render(item.Label),
			)
		} else if i == r.Selected {
			line = fmt.Sprintf("%s %s %s",
				cursor,
				styles.SuccessText.Render(radio),
				item.Label,
			)
		} else {
			line = fmt.Sprintf("%s %s %s",
				cursor,
				radio,
				item.Label,
			)
		}

		b.WriteString(line)

		// Description
		if item.Description != "" {
			b.WriteString("\n    ")
			b.WriteString(styles.Subtitle.Render(item.Description))
		}

		b.WriteString("\n")
	}

	// Scroll indicators
	if r.offset > 0 {
		b.WriteString(styles.MutedText.Render("  ▲ More items above\n"))
	}
	if visibleEnd < len(r.Items) {
		b.WriteString(styles.MutedText.Render("  ▼ More items below\n"))
	}

	return b.String()
}

// CursorUp moves the cursor up
func (r *RadioList) CursorUp() {
	if r.Cursor > 0 {
		r.Cursor--
		// Adjust scroll offset
		if r.Cursor < r.offset {
			r.offset = r.Cursor
		}
	}
}

// CursorDown moves the cursor down
func (r *RadioList) CursorDown() {
	if r.Cursor < len(r.Items)-1 {
		r.Cursor++
		// Adjust scroll offset
		if r.Cursor >= r.offset+r.Height {
			r.offset = r.Cursor - r.Height + 1
		}
	}
}

// Select selects the current item
func (r *RadioList) Select() {
	if r.Cursor < len(r.Items) && !r.Items[r.Cursor].Disabled {
		r.Selected = r.Cursor
	}
}

// SetSelected sets the selected item by index
func (r *RadioList) SetSelected(index int) {
	if index >= 0 && index < len(r.Items) && !r.Items[index].Disabled {
		r.Selected = index
	}
}

// SetSelectedByValue sets the selected item by value
func (r *RadioList) SetSelectedByValue(value string) {
	for i, item := range r.Items {
		if item.Value == value && !item.Disabled {
			r.Selected = i
			return
		}
	}
}

// GetSelected returns the selected item, or nil if none selected
func (r *RadioList) GetSelected() *RadioItem {
	if r.Selected >= 0 && r.Selected < len(r.Items) {
		return &r.Items[r.Selected]
	}
	return nil
}

// GetSelectedValue returns the value of the selected item
func (r *RadioList) GetSelectedValue() string {
	if item := r.GetSelected(); item != nil {
		return item.Value
	}
	return ""
}

// HasSelection returns true if an item is selected
func (r *RadioList) HasSelection() bool {
	return r.Selected >= 0
}

// ClearSelection clears the selection
func (r *RadioList) ClearSelection() {
	r.Selected = -1
}

// SetItems updates the items list
func (r *RadioList) SetItems(items []RadioItem) {
	r.Items = items
	r.Cursor = 0
	r.Selected = -1
	r.offset = 0
}

// GetCursor returns the current cursor position
func (r *RadioList) GetCursor() int {
	return r.Cursor
}

// SetCursor sets the cursor position
func (r *RadioList) SetCursor(cursor int) {
	if cursor >= 0 && cursor < len(r.Items) {
		r.Cursor = cursor
		// Adjust offset if needed
		if r.Cursor < r.offset {
			r.offset = r.Cursor
		} else if r.Cursor >= r.offset+r.Height {
			r.offset = r.Cursor - r.Height + 1
		}
	}
}
