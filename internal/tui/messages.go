package tui

import (
	"github.com/Andrewy-gh/gwt/internal/copy"
	"github.com/Andrewy-gh/gwt/internal/create"
	"github.com/Andrewy-gh/gwt/internal/docker"
	"github.com/Andrewy-gh/gwt/internal/git"
)

// View transition messages

// SwitchViewMsg requests a view transition
type SwitchViewMsg struct {
	View View
}

// BackToPreviousViewMsg returns to the previous view
type BackToPreviousViewMsg struct{}

// BackToMenuMsg returns directly to the menu
type BackToMenuMsg struct{}

// Create flow messages

// CreateFlowNextStepMsg advances the create flow to the next step
type CreateFlowNextStepMsg struct{}

// CreateFlowCompleteMsg signals the create flow is ready for execution
type CreateFlowCompleteMsg struct{}

// CreateFlowCancelMsg cancels the create flow and returns to menu
type CreateFlowCancelMsg struct{}

// Branch validation messages

// ValidateBranchMsg requests branch name validation
type ValidateBranchMsg struct {
	BranchName string
}

// BranchValidationResultMsg contains branch validation result
type BranchValidationResultMsg struct {
	Valid bool
	Error string
	Spec  *create.BranchSpec
}

// Remote branch messages

// FetchRemotesMsg requests fetching remote branches
type FetchRemotesMsg struct {
	Remote string
	Prune  bool
}

// FetchRemotesCompleteMsg signals remote fetch is complete
type FetchRemotesCompleteMsg struct {
	Branches []git.Branch
	Error    error
}

// RemoteBranchSelectedMsg signals a remote branch was selected
type RemoteBranchSelectedMsg struct {
	Branch *git.Branch
}

// File discovery messages

// DiscoverFilesMsg requests file discovery
type DiscoverFilesMsg struct {
	RepoPath string
}

// DiscoverFilesCompleteMsg signals file discovery is complete
type DiscoverFilesCompleteMsg struct {
	Files []copy.IgnoredFile
	Error error
}

// FileSelectionCompleteMsg signals file selection is done
type FileSelectionCompleteMsg struct {
	Selection *copy.Selection
}

// Docker messages

// DetectDockerMsg requests docker compose detection
type DetectDockerMsg struct {
	RepoPath string
}

// DetectDockerCompleteMsg signals docker detection is complete
type DetectDockerCompleteMsg struct {
	Detected bool
	Config   *docker.ComposeConfig
	Files    []docker.ComposeFile
	Error    error
}

// DockerModeSelectedMsg signals a docker mode was selected
type DockerModeSelectedMsg struct {
	Mode string // "none", "shared", "new"
}

// Worktree operation messages

// StartCreateOperationMsg starts the worktree creation operation
type StartCreateOperationMsg struct{}

// CreateProgressMsg reports creation progress
type CreateProgressMsg struct {
	Stage       string // Current stage name
	StageIndex  int    // Current stage index (0-based)
	TotalStages int    // Total number of stages
	Message     string // Progress message
	BytesCopied int64  // Bytes copied (for file copy stage)
	TotalBytes  int64  // Total bytes to copy
}

// CreateCompleteMsg signals worktree creation is complete
type CreateCompleteMsg struct {
	WorktreePath string
	Error        error
}

// Delete operation messages

// StartDeleteOperationMsg starts the worktree deletion operation
type StartDeleteOperationMsg struct{}

// DeleteProgressMsg reports deletion progress
type DeleteProgressMsg struct {
	Current int    // Current worktree being deleted (1-based)
	Total   int    // Total worktrees to delete
	Path    string // Path being deleted
}

// DeleteCompleteMsg signals worktree deletion is complete
type DeleteCompleteMsg struct {
	Deleted []string // Paths that were deleted
	Failed  []DeleteFailure
	Error   error
}

// DeleteFailure represents a single deletion failure
type DeleteFailure struct {
	Path  string
	Error error
}

// Worktree list messages

// RefreshWorktreeListMsg requests refreshing the worktree list
type RefreshWorktreeListMsg struct{}

// WorktreeListCompleteMsg signals worktree list refresh is complete
type WorktreeListCompleteMsg struct {
	Worktrees []git.Worktree
	Error     error
}

// WorktreeStatusMsg contains status for a single worktree
type WorktreeStatusMsg struct {
	Path   string
	Status *git.WorktreeStatus
	Error  error
}

// WorktreesSelectedMsg signals worktrees were selected for deletion
type WorktreesSelectedMsg struct {
	Paths []string
}

// Error messages

// ErrorMsg represents a general error
type ErrorMsg struct {
	Error   error
	Context string // Additional context for the error
}

// ClearErrorMsg clears the current error
type ClearErrorMsg struct{}

// Tick message for animations

// TickMsg is used for animation updates (spinners, progress)
type TickMsg struct{}
