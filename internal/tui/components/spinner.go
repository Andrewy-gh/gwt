package components

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/Andrewy-gh/gwt/internal/tui/styles"
)

// SpinnerStyle defines different spinner animation styles
type SpinnerStyle int

const (
	SpinnerDots SpinnerStyle = iota
	SpinnerLine
	SpinnerMiniDot
	SpinnerJump
	SpinnerPulse
	SpinnerPoints
	SpinnerGlobe
	SpinnerMoon
	SpinnerMonkey
)

// Spinner is an animated loading indicator
type Spinner struct {
	spinner spinner.Model
	Message string
	Active  bool
}

// NewSpinner creates a new spinner with the given message
func NewSpinner(message string) *Spinner {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styles.Selected

	return &Spinner{
		spinner: s,
		Message: message,
		Active:  true,
	}
}

// NewSpinnerWithStyle creates a new spinner with a specific style
func NewSpinnerWithStyle(message string, style SpinnerStyle) *Spinner {
	s := spinner.New()
	s.Style = styles.Selected

	switch style {
	case SpinnerDots:
		s.Spinner = spinner.Dot
	case SpinnerLine:
		s.Spinner = spinner.Line
	case SpinnerMiniDot:
		s.Spinner = spinner.MiniDot
	case SpinnerJump:
		s.Spinner = spinner.Jump
	case SpinnerPulse:
		s.Spinner = spinner.Pulse
	case SpinnerPoints:
		s.Spinner = spinner.Points
	case SpinnerGlobe:
		s.Spinner = spinner.Globe
	case SpinnerMoon:
		s.Spinner = spinner.Moon
	case SpinnerMonkey:
		s.Spinner = spinner.Monkey
	default:
		s.Spinner = spinner.Dot
	}

	return &Spinner{
		spinner: s,
		Message: message,
		Active:  true,
	}
}

// Init initializes the spinner and starts the animation
func (s *Spinner) Init() tea.Cmd {
	return s.spinner.Tick
}

// Update handles messages
func (s *Spinner) Update(msg tea.Msg) (*Spinner, tea.Cmd) {
	if !s.Active {
		return s, nil
	}

	var cmd tea.Cmd
	s.spinner, cmd = s.spinner.Update(msg)
	return s, cmd
}

// View renders the spinner
func (s *Spinner) View() string {
	if !s.Active {
		return ""
	}

	if s.Message != "" {
		return s.spinner.View() + " " + s.Message
	}
	return s.spinner.View()
}

// Start activates the spinner
func (s *Spinner) Start() tea.Cmd {
	s.Active = true
	return s.spinner.Tick
}

// Stop deactivates the spinner
func (s *Spinner) Stop() {
	s.Active = false
}

// SetMessage updates the spinner message
func (s *Spinner) SetMessage(message string) {
	s.Message = message
}

// IsActive returns whether the spinner is active
func (s *Spinner) IsActive() bool {
	return s.Active
}

// Tick returns the tick command for the spinner
func (s *Spinner) Tick() tea.Cmd {
	return s.spinner.Tick
}

// TickMsg is a message to trigger spinner animation
type SpinnerTickMsg time.Time

// SpinnerCmd returns a command that sends tick messages for spinner animation
func SpinnerCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return SpinnerTickMsg(t)
	})
}

// StageSpinner shows a spinner with stages (e.g., "Step 1/4: Creating worktree...")
type StageSpinner struct {
	spinner      spinner.Model
	CurrentStage int
	TotalStages  int
	StageName    string
	Active       bool
}

// NewStageSpinner creates a new stage-based spinner
func NewStageSpinner(totalStages int) *StageSpinner {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styles.Selected

	return &StageSpinner{
		spinner:      s,
		CurrentStage: 0,
		TotalStages:  totalStages,
		Active:       true,
	}
}

// Init initializes the stage spinner
func (s *StageSpinner) Init() tea.Cmd {
	return s.spinner.Tick
}

// Update handles messages
func (s *StageSpinner) Update(msg tea.Msg) (*StageSpinner, tea.Cmd) {
	if !s.Active {
		return s, nil
	}

	var cmd tea.Cmd
	s.spinner, cmd = s.spinner.Update(msg)
	return s, cmd
}

// View renders the stage spinner
func (s *StageSpinner) View() string {
	if !s.Active {
		return ""
	}

	stageInfo := styles.MutedText.Render(
		"[" + itoa(s.CurrentStage) + "/" + itoa(s.TotalStages) + "]",
	)

	return s.spinner.View() + " " + stageInfo + " " + s.StageName
}

// SetStage updates the current stage
func (s *StageSpinner) SetStage(stage int, name string) {
	s.CurrentStage = stage
	s.StageName = name
}

// NextStage advances to the next stage
func (s *StageSpinner) NextStage(name string) {
	s.CurrentStage++
	s.StageName = name
}

// Start activates the stage spinner
func (s *StageSpinner) Start() tea.Cmd {
	s.Active = true
	return s.spinner.Tick
}

// Stop deactivates the stage spinner
func (s *StageSpinner) Stop() {
	s.Active = false
}

// IsComplete returns true if all stages are done
func (s *StageSpinner) IsComplete() bool {
	return s.CurrentStage >= s.TotalStages
}

// itoa is a simple int to string helper
func itoa(i int) string {
	if i == 0 {
		return "0"
	}

	var result []byte
	negative := i < 0
	if negative {
		i = -i
	}

	for i > 0 {
		result = append([]byte{byte('0' + i%10)}, result...)
		i /= 10
	}

	if negative {
		result = append([]byte{'-'}, result...)
	}

	return string(result)
}
