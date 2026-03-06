package tui

import (
	"github.com/Andrewy-gh/gwt/internal/copy"
	"github.com/Andrewy-gh/gwt/internal/create"
	"github.com/Andrewy-gh/gwt/internal/docker"
	"github.com/Andrewy-gh/gwt/internal/git"
)

// CreateFlowState holds accumulated state across create flow steps
type CreateFlowState struct {
	// Step 1: Branch input
	BranchSpec  *create.BranchSpec
	BranchInput string
	SourceType  create.BranchSource

	// Step 2: Source selection (conditional - only for BranchSourceNewFromRef)
	StartPoint string

	// Step 3: Remote branch selection (conditional - only for BranchSourceRemote)
	SelectedRemote *git.Branch

	// Step 4: File selection
	TargetDir     string
	IgnoredFiles  []copy.IgnoredFile
	FileSelection *copy.Selection

	// Step 5: Docker mode
	DockerMode      string // "none", "shared", "new"
	ComposeDetected bool
	ComposeConfig   *docker.ComposeConfig
	ComposeFiles    []docker.ComposeFile

	// Navigation tracking
	CurrentStep int
	TotalSteps  int
	ViewHistory []View
}

// NewCreateFlowState creates a new create flow state
func NewCreateFlowState() *CreateFlowState {
	return &CreateFlowState{
		CurrentStep: 1,
		TotalSteps:  3, // Default: branch -> files -> docker (may increase if source/remote needed)
		ViewHistory: make([]View, 0, 5),
		DockerMode:  "none",
	}
}

// Reset clears the flow state for a new flow
func (s *CreateFlowState) Reset() {
	s.BranchSpec = nil
	s.BranchInput = ""
	s.SourceType = create.BranchSourceNewFromHEAD
	s.StartPoint = ""
	s.SelectedRemote = nil
	s.TargetDir = ""
	s.IgnoredFiles = nil
	s.FileSelection = nil
	s.DockerMode = "none"
	s.ComposeDetected = false
	s.ComposeConfig = nil
	s.ComposeFiles = nil
	s.CurrentStep = 1
	s.TotalSteps = 3
	s.ViewHistory = s.ViewHistory[:0]
}

// PushView adds a view to the history
func (s *CreateFlowState) PushView(view View) {
	s.ViewHistory = append(s.ViewHistory, view)
}

// PopView removes and returns the last view from history
func (s *CreateFlowState) PopView() (View, bool) {
	if len(s.ViewHistory) == 0 {
		return ViewMenu, false
	}
	lastIdx := len(s.ViewHistory) - 1
	view := s.ViewHistory[lastIdx]
	s.ViewHistory = s.ViewHistory[:lastIdx]
	return view, true
}

// PreviousView returns the previous view without removing it
func (s *CreateFlowState) PreviousView() View {
	if len(s.ViewHistory) == 0 {
		return ViewMenu
	}
	return s.ViewHistory[len(s.ViewHistory)-1]
}

// HasHistory returns true if there's view history
func (s *CreateFlowState) HasHistory() bool {
	return len(s.ViewHistory) > 0
}

// CalculateTotalSteps determines the total steps based on branch source
func (s *CreateFlowState) CalculateTotalSteps() {
	switch s.SourceType {
	case create.BranchSourceNewFromHEAD:
		// Branch -> Files -> Docker
		s.TotalSteps = 3
	case create.BranchSourceNewFromRef:
		// Branch -> Source -> Files -> Docker
		s.TotalSteps = 4
	case create.BranchSourceLocalExisting:
		// Branch -> Files -> Docker
		s.TotalSteps = 3
	case create.BranchSourceRemote:
		// Branch -> Remote -> Files -> Docker
		s.TotalSteps = 4
	}
}

// NextView returns the next view based on current state
func (s *CreateFlowState) NextView() View {
	if s.BranchSpec == nil {
		return ViewCreateBranch
	}

	switch s.SourceType {
	case create.BranchSourceNewFromHEAD:
		// Branch -> Files -> Docker
		if s.FileSelection == nil {
			return ViewFileSelect
		}
		return ViewDockerMode

	case create.BranchSourceNewFromRef:
		// Branch -> Source -> Files -> Docker
		if s.StartPoint == "" {
			return ViewCreateSource
		}
		if s.FileSelection == nil {
			return ViewFileSelect
		}
		return ViewDockerMode

	case create.BranchSourceLocalExisting:
		// Branch -> Files -> Docker
		if s.FileSelection == nil {
			return ViewFileSelect
		}
		return ViewDockerMode

	case create.BranchSourceRemote:
		// Branch -> Remote -> Files -> Docker
		if s.SelectedRemote == nil {
			return ViewRemoteBranch
		}
		if s.FileSelection == nil {
			return ViewFileSelect
		}
		return ViewDockerMode
	}

	return ViewDockerMode
}

// IsComplete returns true if all required steps are done
func (s *CreateFlowState) IsComplete() bool {
	if s.BranchSpec == nil {
		return false
	}

	switch s.SourceType {
	case create.BranchSourceNewFromRef:
		if s.StartPoint == "" {
			return false
		}
	case create.BranchSourceRemote:
		if s.SelectedRemote == nil {
			return false
		}
	}

	// File selection and docker mode are always the final steps
	// Docker mode has a default, so we just check if we've passed file selection
	return s.FileSelection != nil
}

// StepName returns a human-readable name for the current step
func (s *CreateFlowState) StepName() string {
	switch s.CurrentStep {
	case 1:
		return "Branch"
	case 2:
		if s.SourceType == create.BranchSourceNewFromRef {
			return "Source"
		}
		if s.SourceType == create.BranchSourceRemote {
			return "Remote"
		}
		return "Files"
	case 3:
		if s.SourceType == create.BranchSourceNewFromRef || s.SourceType == create.BranchSourceRemote {
			return "Files"
		}
		return "Docker"
	case 4:
		return "Docker"
	default:
		return "Unknown"
	}
}

// Summary returns a summary of the current flow state
func (s *CreateFlowState) Summary() string {
	if s.BranchSpec == nil {
		return "No branch selected"
	}
	return create.GetSourceDescription(s.BranchSpec)
}

// DeleteFlowState holds state for the delete flow
type DeleteFlowState struct {
	// Selected worktrees
	SelectedPaths []string
	Worktrees     []git.Worktree

	// Pre-flight check results
	CheckResults map[string]*DeleteCheckResult

	// Confirmation
	Confirmed bool
}

// DeleteCheckResult contains pre-flight check results for a worktree
type DeleteCheckResult struct {
	Path        string
	CanDelete   bool     // False if blocked
	Blocked     bool     // True if deletion is blocked
	BlockReason string   // Reason for blocking
	Warnings    []string // Non-blocking warnings
	HasChanges  bool     // Has uncommitted changes
	IsUnmerged  bool     // Has unmerged changes
	IsLocked    bool     // Worktree is locked
	IsMain      bool     // Is main worktree
	IsCurrent   bool     // Is current working directory
}

// NewDeleteFlowState creates a new delete flow state
func NewDeleteFlowState() *DeleteFlowState {
	return &DeleteFlowState{
		SelectedPaths: make([]string, 0),
		CheckResults:  make(map[string]*DeleteCheckResult),
	}
}

// Reset clears the delete flow state
func (s *DeleteFlowState) Reset() {
	s.SelectedPaths = s.SelectedPaths[:0]
	s.Worktrees = nil
	s.CheckResults = make(map[string]*DeleteCheckResult)
	s.Confirmed = false
}

// AddCheck adds a check result for a path
func (s *DeleteFlowState) AddCheck(result *DeleteCheckResult) {
	s.CheckResults[result.Path] = result
}

// GetCheck returns the check result for a path
func (s *DeleteFlowState) GetCheck(path string) *DeleteCheckResult {
	return s.CheckResults[path]
}

// BlockedCount returns the number of blocked deletions
func (s *DeleteFlowState) BlockedCount() int {
	count := 0
	for _, result := range s.CheckResults {
		if result.Blocked {
			count++
		}
	}
	return count
}

// WarningCount returns the number of deletions with warnings
func (s *DeleteFlowState) WarningCount() int {
	count := 0
	for _, result := range s.CheckResults {
		if !result.Blocked && len(result.Warnings) > 0 {
			count++
		}
	}
	return count
}

// DeletablePaths returns paths that can be deleted (not blocked)
func (s *DeleteFlowState) DeletablePaths() []string {
	paths := make([]string, 0, len(s.SelectedPaths))
	for _, path := range s.SelectedPaths {
		if result, ok := s.CheckResults[path]; ok && !result.Blocked {
			paths = append(paths, path)
		}
	}
	return paths
}

// HasBlocked returns true if any deletions are blocked
func (s *DeleteFlowState) HasBlocked() bool {
	return s.BlockedCount() > 0
}

// HasWarnings returns true if any deletions have warnings
func (s *DeleteFlowState) HasWarnings() bool {
	return s.WarningCount() > 0
}

// AllBlocked returns true if all selected paths are blocked
func (s *DeleteFlowState) AllBlocked() bool {
	return s.BlockedCount() == len(s.SelectedPaths)
}
