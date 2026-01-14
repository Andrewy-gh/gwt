package components

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/Andrewy-gh/gwt/internal/tui/styles"
)

// TextInput is a styled text input component with validation
type TextInput struct {
	Input       textinput.Model
	Label       string
	Placeholder string
	Validator   func(string) error
	ErrorMsg    string
}

// NewTextInput creates a new text input component
func NewTextInput(label, placeholder string) *TextInput {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = 256

	return &TextInput{
		Input:       ti,
		Label:       label,
		Placeholder: placeholder,
	}
}

// Init initializes the component
func (t *TextInput) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages
func (t *TextInput) Update(msg tea.Msg) (*TextInput, tea.Cmd) {
	var cmd tea.Cmd
	t.Input, cmd = t.Input.Update(msg)

	// Validate on change
	if t.Validator != nil {
		if err := t.Validator(t.Input.Value()); err != nil {
			t.ErrorMsg = err.Error()
		} else {
			t.ErrorMsg = ""
		}
	}

	return t, cmd
}

// View renders the component
func (t *TextInput) View() string {
	var output string

	// Label
	if t.Label != "" {
		output += styles.Subtitle.Render(t.Label) + "\n"
	}

	// Input
	output += t.Input.View() + "\n"

	// Error message
	if t.ErrorMsg != "" {
		output += styles.ErrorText.Render("✘ " + t.ErrorMsg) + "\n"
	}

	return output
}

// Value returns the current input value
func (t *TextInput) Value() string {
	return t.Input.Value()
}

// SetValue sets the input value
func (t *TextInput) SetValue(value string) {
	t.Input.SetValue(value)
}

// Focus focuses the input
func (t *TextInput) Focus() tea.Cmd {
	return t.Input.Focus()
}

// Blur removes focus from the input
func (t *TextInput) Blur() {
	t.Input.Blur()
}

// Validate runs the validator on the current value
func (t *TextInput) Validate() error {
	if t.Validator != nil {
		if err := t.Validator(t.Input.Value()); err != nil {
			t.ErrorMsg = err.Error()
			return err
		}
	}
	t.ErrorMsg = ""
	return nil
}

// IsValid returns true if the input is valid
func (t *TextInput) IsValid() bool {
	return t.ErrorMsg == ""
}

// SetValidator sets the validation function
func (t *TextInput) SetValidator(validator func(string) error) {
	t.Validator = validator
}

// SetPromptStyle sets the prompt style
func (t *TextInput) SetPromptStyle(style string) {
	t.Input.Prompt = fmt.Sprintf("%s ", style)
}

// Focused returns true if the input is focused
func (t *TextInput) Focused() bool {
	return t.Input.Focused()
}
