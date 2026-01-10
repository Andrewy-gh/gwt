package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Andrewy-gh/gwt/internal/git"
	"github.com/Andrewy-gh/gwt/internal/output"
	"github.com/spf13/cobra"
)

// StatusOptions holds options for the status command
type StatusOptions struct {
	JSON bool // --json: Output as JSON
}

var statusOpts StatusOptions

var statusCmd = &cobra.Command{
	Use:   "status [path]",
	Short: "Show worktree status",
	Long: `Show detailed status of current or specified worktree.

Displays information about the worktree including:
  - Path and branch name
  - HEAD commit with message
  - Working tree status (staged/unstaged/untracked)
  - Upstream tracking information (ahead/behind)
  - Lock status

Examples:
  gwt status              Status of current worktree
  gwt status /path/to/wt  Status of specific worktree
  gwt status --json       Output as JSON`,
	Args: cobra.MaximumNArgs(1),
	RunE: runStatus,
}

func init() {
	statusCmd.Flags().BoolVar(&statusOpts.JSON, "json", false, "output as JSON")
	rootCmd.AddCommand(statusCmd)
}

// WorktreeStatusOutput represents the full status output for JSON
type WorktreeStatusOutput struct {
	Path           string    `json:"path"`
	Branch         string    `json:"branch"`
	IsMain         bool      `json:"isMain"`
	IsDetached     bool      `json:"isDetached"`
	Locked         bool      `json:"locked"`
	Commit         string    `json:"commit"`
	CommitFull     string    `json:"commitFull"`
	CommitMessage  string    `json:"commitMessage"`
	CommitTime     time.Time `json:"commitTime"`
	Upstream       string    `json:"upstream,omitempty"`
	AheadCount     int       `json:"aheadCount"`
	BehindCount    int       `json:"behindCount"`
	Clean          bool      `json:"clean"`
	StagedCount    int       `json:"stagedCount"`
	UnstagedCount  int       `json:"unstagedCount"`
	UntrackedCount int       `json:"untrackedCount"`
}

func runStatus(cmd *cobra.Command, args []string) error {
	var targetPath string

	if len(args) > 0 {
		targetPath = args[0]
	} else {
		// Use current working directory
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		targetPath = cwd
	}

	// Validate this is a worktree
	wt, err := git.GetWorktree(targetPath)
	if err != nil {
		return fmt.Errorf("not a valid worktree: %w", err)
	}

	// Get detailed status
	status, err := git.GetWorktreeStatus(wt.Path)
	if err != nil {
		return fmt.Errorf("failed to get worktree status: %w", err)
	}

	// Get upstream branch name
	upstream := getUpstreamBranch(wt.Path)

	if statusOpts.JSON {
		return outputStatusJSON(wt, status, upstream)
	}

	return outputStatusText(wt, status, upstream)
}

func getUpstreamBranch(worktreePath string) string {
	result, err := git.RunInDir(worktreePath, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(result.Stdout)
}

func outputStatusJSON(wt *git.Worktree, status *git.WorktreeStatus, upstream string) error {
	out := WorktreeStatusOutput{
		Path:           wt.Path,
		Branch:         wt.Branch,
		IsMain:         wt.IsMain,
		IsDetached:     wt.IsDetached,
		Locked:         wt.Locked,
		Commit:         wt.Commit,
		CommitFull:     wt.CommitFull,
		CommitMessage:  status.LastCommitMsg,
		CommitTime:     status.LastCommitTime,
		Upstream:       upstream,
		AheadCount:     status.AheadCount,
		BehindCount:    status.BehindCount,
		Clean:          status.Clean,
		StagedCount:    status.StagedCount,
		UnstagedCount:  status.UnstagedCount,
		UntrackedCount: status.UntrackedCount,
	}

	return output.JSON(out)
}

func outputStatusText(wt *git.Worktree, status *git.WorktreeStatus, upstream string) error {
	// Header
	output.Println(fmt.Sprintf("Worktree:  %s", wt.Path))

	// Branch
	branch := wt.Branch
	if wt.IsDetached {
		branch = "(detached HEAD)"
	}
	if wt.IsMain {
		branch += " (main)"
	}
	output.Println(fmt.Sprintf("Branch:    %s", branch))

	// Commit info
	output.Println(fmt.Sprintf("Commit:    %s %s", wt.Commit, status.LastCommitMsg))

	// Modified time
	if !status.LastCommitTime.IsZero() {
		ago := formatTimeAgo(status.LastCommitTime)
		output.Println(fmt.Sprintf("Modified:  %s", ago))
	}

	// Lock status
	if wt.Locked {
		output.Println("Locked:    Yes")
	}

	output.Println("")

	// Working tree status
	output.Println("Status:")
	if status.Clean {
		output.Println("  Clean - no uncommitted changes")
	} else {
		if status.StagedCount > 0 {
			output.Println(fmt.Sprintf("  Staged:    %d changes", status.StagedCount))
		}
		if status.UnstagedCount > 0 {
			output.Println(fmt.Sprintf("  Unstaged:  %d changes", status.UnstagedCount))
		}
		if status.UntrackedCount > 0 {
			output.Println(fmt.Sprintf("  Untracked: %d files", status.UntrackedCount))
		}
	}

	// Upstream info
	if upstream != "" {
		output.Println("")
		output.Println(fmt.Sprintf("Upstream:  %s", upstream))
		if status.AheadCount > 0 || status.BehindCount > 0 {
			output.Println(fmt.Sprintf("  Ahead:   %d commits", status.AheadCount))
			output.Println(fmt.Sprintf("  Behind:  %d commits", status.BehindCount))
		} else {
			output.Println("  Up to date")
		}
	}

	return nil
}

func formatTimeAgo(t time.Time) string {
	duration := time.Since(t)

	switch {
	case duration < time.Minute:
		return "just now"
	case duration < time.Hour:
		mins := int(duration.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case duration < 24*time.Hour:
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case duration < 7*24*time.Hour:
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	default:
		weeks := int(duration.Hours() / 24 / 7)
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	}
}
