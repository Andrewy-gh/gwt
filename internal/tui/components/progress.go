package components

import (
	"fmt"
	"strings"

	"github.com/Andrewy-gh/gwt/internal/tui/styles"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

// ProgressBar is a visual progress indicator
type ProgressBar struct {
	progress    progress.Model
	Current     int64
	Total       int64
	Label       string
	ShowPercent bool
	ShowCount   bool
	Width       int
}

// NewProgressBar creates a new progress bar
func NewProgressBar(width int) *ProgressBar {
	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(width),
		progress.WithoutPercentage(),
	)

	return &ProgressBar{
		progress:    p,
		Current:     0,
		Total:       100,
		ShowPercent: true,
		ShowCount:   true,
		Width:       width,
	}
}

// NewProgressBarWithColors creates a progress bar with custom colors
func NewProgressBarWithColors(width int, filled, empty string) *ProgressBar {
	p := progress.New(
		progress.WithSolidFill(filled),
		progress.WithWidth(width),
		progress.WithoutPercentage(),
	)

	return &ProgressBar{
		progress:    p,
		Current:     0,
		Total:       100,
		ShowPercent: true,
		ShowCount:   true,
		Width:       width,
	}
}

// Init initializes the progress bar
func (p *ProgressBar) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (p *ProgressBar) Update(msg tea.Msg) (*ProgressBar, tea.Cmd) {
	var cmd tea.Cmd
	model, cmd := p.progress.Update(msg)
	p.progress = model.(progress.Model)
	return p, cmd
}

// View renders the progress bar
func (p *ProgressBar) View() string {
	var b strings.Builder

	// Label
	if p.Label != "" {
		b.WriteString(p.Label)
		b.WriteString("\n")
	}

	// Calculate percentage
	var percent float64
	if p.Total > 0 {
		percent = float64(p.Current) / float64(p.Total)
	}

	// Progress bar
	b.WriteString(p.progress.ViewAs(percent))

	// Stats line
	var stats []string

	if p.ShowPercent {
		stats = append(stats, fmt.Sprintf("%.0f%%", percent*100))
	}

	if p.ShowCount {
		stats = append(stats, fmt.Sprintf("%s / %s",
			formatBytes(p.Current),
			formatBytes(p.Total),
		))
	}

	if len(stats) > 0 {
		b.WriteString("\n")
		b.WriteString(styles.MutedText.Render(strings.Join(stats, " - ")))
	}

	return b.String()
}

// SetProgress updates the current progress
func (p *ProgressBar) SetProgress(current, total int64) {
	p.Current = current
	p.Total = total
}

// SetCurrent updates only the current value
func (p *ProgressBar) SetCurrent(current int64) {
	p.Current = current
}

// SetTotal updates only the total value
func (p *ProgressBar) SetTotal(total int64) {
	p.Total = total
}

// SetLabel sets the progress bar label
func (p *ProgressBar) SetLabel(label string) {
	p.Label = label
}

// Increment increases current by the given amount
func (p *ProgressBar) Increment(amount int64) {
	p.Current += amount
	if p.Current > p.Total {
		p.Current = p.Total
	}
}

// Percent returns the current percentage (0.0 to 1.0)
func (p *ProgressBar) Percent() float64 {
	if p.Total == 0 {
		return 0
	}
	return float64(p.Current) / float64(p.Total)
}

// IsComplete returns true if progress is 100%
func (p *ProgressBar) IsComplete() bool {
	return p.Current >= p.Total
}

// Reset resets the progress bar
func (p *ProgressBar) Reset() {
	p.Current = 0
}

// formatBytes formats bytes as human-readable string
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// FileProgress shows progress for file operations
type FileProgress struct {
	ProgressBar
	CurrentFile string
	FilesDone   int
	TotalFiles  int
}

// NewFileProgress creates a new file progress indicator
func NewFileProgress(width int) *FileProgress {
	return &FileProgress{
		ProgressBar: *NewProgressBar(width),
	}
}

// View renders the file progress
func (f *FileProgress) View() string {
	var b strings.Builder

	// Current file
	if f.CurrentFile != "" {
		b.WriteString(styles.Subtitle.Render("Copying: "))
		b.WriteString(truncatePath(f.CurrentFile, f.Width-10))
		b.WriteString("\n")
	}

	// Progress bar
	b.WriteString(f.ProgressBar.View())

	// Files count
	if f.TotalFiles > 0 {
		b.WriteString("\n")
		fileProgress := fmt.Sprintf("Files: %d / %d", f.FilesDone, f.TotalFiles)
		b.WriteString(styles.MutedText.Render(fileProgress))
	}

	return b.String()
}

// SetFile updates the current file being processed
func (f *FileProgress) SetFile(path string) {
	f.CurrentFile = path
}

// SetFileProgress updates the file counts
func (f *FileProgress) SetFileProgress(done, total int) {
	f.FilesDone = done
	f.TotalFiles = total
}

// IncrementFile increments the files done counter
func (f *FileProgress) IncrementFile() {
	f.FilesDone++
}

// truncatePath truncates a path to fit within maxLen
func truncatePath(path string, maxLen int) string {
	if len(path) <= maxLen {
		return path
	}
	if maxLen <= 3 {
		return "..."
	}
	return "..." + path[len(path)-(maxLen-3):]
}

// MultiStageProgress shows progress across multiple stages
type MultiStageProgress struct {
	Stages       []Stage
	CurrentStage int
	Width        int
}

// Stage represents a single stage in a multi-stage process
type Stage struct {
	Name     string
	Status   StageStatus
	Message  string
	Progress *ProgressBar // Optional progress for this stage
}

// StageStatus represents the status of a stage
type StageStatus int

const (
	StagePending StageStatus = iota
	StageRunning
	StageComplete
	StageFailed
	StageSkipped
)

// NewMultiStageProgress creates a new multi-stage progress indicator
func NewMultiStageProgress(stageNames []string, width int) *MultiStageProgress {
	stages := make([]Stage, len(stageNames))
	for i, name := range stageNames {
		stages[i] = Stage{
			Name:   name,
			Status: StagePending,
		}
	}

	return &MultiStageProgress{
		Stages:       stages,
		CurrentStage: 0,
		Width:        width,
	}
}

// View renders the multi-stage progress
func (m *MultiStageProgress) View() string {
	var b strings.Builder

	for i, stage := range m.Stages {
		// Status indicator and styled line
		var line string

		switch stage.Status {
		case StagePending:
			line = styles.MutedText.Render(fmt.Sprintf("○ %s", stage.Name))
		case StageRunning:
			line = styles.Selected.Render(fmt.Sprintf("◐ %s", stage.Name))
		case StageComplete:
			line = styles.SuccessText.Render(fmt.Sprintf("● %s", stage.Name))
		case StageFailed:
			line = styles.ErrorText.Render(fmt.Sprintf("✘ %s", stage.Name))
		case StageSkipped:
			line = styles.MutedText.Render(fmt.Sprintf("○ %s", stage.Name))
		}

		b.WriteString(line)

		// Message
		if stage.Message != "" {
			b.WriteString(" - ")
			b.WriteString(styles.MutedText.Render(stage.Message))
		}

		b.WriteString("\n")

		// Stage progress bar (if running and has progress)
		if stage.Status == StageRunning && stage.Progress != nil {
			b.WriteString("  ")
			b.WriteString(stage.Progress.View())
			b.WriteString("\n")
		}

		// Connector line (except for last item)
		if i < len(m.Stages)-1 {
			if stage.Status == StageComplete || stage.Status == StageRunning {
				b.WriteString(styles.MutedText.Render("│\n"))
			} else {
				b.WriteString(styles.MutedText.Render("│\n"))
			}
		}
	}

	return b.String()
}

// StartStage marks a stage as running
func (m *MultiStageProgress) StartStage(index int) {
	if index >= 0 && index < len(m.Stages) {
		m.CurrentStage = index
		m.Stages[index].Status = StageRunning
	}
}

// CompleteStage marks a stage as complete
func (m *MultiStageProgress) CompleteStage(index int, message string) {
	if index >= 0 && index < len(m.Stages) {
		m.Stages[index].Status = StageComplete
		m.Stages[index].Message = message
	}
}

// FailStage marks a stage as failed
func (m *MultiStageProgress) FailStage(index int, message string) {
	if index >= 0 && index < len(m.Stages) {
		m.Stages[index].Status = StageFailed
		m.Stages[index].Message = message
	}
}

// SkipStage marks a stage as skipped
func (m *MultiStageProgress) SkipStage(index int, message string) {
	if index >= 0 && index < len(m.Stages) {
		m.Stages[index].Status = StageSkipped
		m.Stages[index].Message = message
	}
}

// SetStageMessage updates a stage's message
func (m *MultiStageProgress) SetStageMessage(index int, message string) {
	if index >= 0 && index < len(m.Stages) {
		m.Stages[index].Message = message
	}
}

// SetStageProgress sets a progress bar for a stage
func (m *MultiStageProgress) SetStageProgress(index int, progress *ProgressBar) {
	if index >= 0 && index < len(m.Stages) {
		m.Stages[index].Progress = progress
	}
}

// NextStage advances to the next stage
func (m *MultiStageProgress) NextStage() {
	if m.CurrentStage < len(m.Stages)-1 {
		m.CompleteStage(m.CurrentStage, "")
		m.CurrentStage++
		m.StartStage(m.CurrentStage)
	}
}

// IsComplete returns true if all stages are complete
func (m *MultiStageProgress) IsComplete() bool {
	for _, stage := range m.Stages {
		if stage.Status != StageComplete && stage.Status != StageSkipped {
			return false
		}
	}
	return true
}

// HasFailed returns true if any stage has failed
func (m *MultiStageProgress) HasFailed() bool {
	for _, stage := range m.Stages {
		if stage.Status == StageFailed {
			return true
		}
	}
	return false
}
