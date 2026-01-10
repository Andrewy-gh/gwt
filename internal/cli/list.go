package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Andrewy-gh/gwt/internal/git"
	"github.com/Andrewy-gh/gwt/internal/output"
	"github.com/spf13/cobra"
)

// ListOptions holds options for the list command
type ListOptions struct {
	JSON   bool // --json: Output as JSON array
	Simple bool // --simple: Output paths only (one per line)
	All    bool // -a, --all: Include bare/prunable worktrees
}

var listOpts ListOptions

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all worktrees",
	Long: `Display all worktrees with their status in a formatted table.

By default, shows worktree path, branch, commit, and status.
Bare and prunable worktrees are hidden unless --all is specified.

Examples:
  gwt list                  List all active worktrees
  gwt list --json           Output as JSON for scripting
  gwt list --simple         Output paths only, one per line
  gwt list --all            Include bare and prunable worktrees`,
	RunE: runList,
}

func init() {
	listCmd.Flags().BoolVar(&listOpts.JSON, "json", false, "output as JSON")
	listCmd.Flags().BoolVar(&listOpts.Simple, "simple", false, "output paths only (one per line)")
	listCmd.Flags().BoolVarP(&listOpts.All, "all", "a", false, "include bare and prunable worktrees")

	// Mark mutually exclusive output format flags
	listCmd.MarkFlagsMutuallyExclusive("json", "simple")

	rootCmd.AddCommand(listCmd)
}

// WorktreeListItem represents a worktree in list output
type WorktreeListItem struct {
	Path       string              `json:"path"`
	Branch     string              `json:"branch"`
	Commit     string              `json:"commit"`
	CommitFull string              `json:"commitFull"`
	IsMain     bool                `json:"isMain"`
	IsDetached bool                `json:"isDetached"`
	Locked     bool                `json:"locked"`
	Status     *WorktreeStatusInfo `json:"status,omitempty"`
}

// WorktreeStatusInfo represents status information for JSON output
type WorktreeStatusInfo struct {
	Clean          bool `json:"clean"`
	StagedCount    int  `json:"stagedCount"`
	UnstagedCount  int  `json:"unstagedCount"`
	UntrackedCount int  `json:"untrackedCount"`
}

func runList(cmd *cobra.Command, args []string) error {
	// Get current working directory to find the repository
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Validate we're in a git repository
	repoPath, err := git.GetRepoRoot(cwd)
	if err != nil {
		return fmt.Errorf("not a git repository")
	}

	// Get all worktrees
	worktrees, err := git.ListWorktrees(repoPath)
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	// Filter worktrees unless --all is specified
	var filtered []git.Worktree
	for _, wt := range worktrees {
		if !listOpts.All && (wt.IsBare || wt.Prunable) {
			continue
		}
		filtered = append(filtered, wt)
	}

	if len(filtered) == 0 {
		output.Info("No worktrees found")
		return nil
	}

	// Output based on format
	switch {
	case listOpts.JSON:
		return outputJSON(filtered)
	case listOpts.Simple:
		return outputSimple(filtered)
	default:
		return outputTable(filtered, repoPath)
	}
}

func outputJSON(worktrees []git.Worktree) error {
	items := make([]WorktreeListItem, 0, len(worktrees))

	for _, wt := range worktrees {
		item := WorktreeListItem{
			Path:       wt.Path,
			Branch:     wt.Branch,
			Commit:     wt.Commit,
			CommitFull: wt.CommitFull,
			IsMain:     wt.IsMain,
			IsDetached: wt.IsDetached,
			Locked:     wt.Locked,
		}

		// Get status for each worktree
		status, err := git.GetWorktreeStatus(wt.Path)
		if err == nil {
			item.Status = &WorktreeStatusInfo{
				Clean:          status.Clean,
				StagedCount:    status.StagedCount,
				UnstagedCount:  status.UnstagedCount,
				UntrackedCount: status.UntrackedCount,
			}
		}

		items = append(items, item)
	}

	return output.JSON(items)
}

func outputSimple(worktrees []git.Worktree) error {
	paths := make([]string, 0, len(worktrees))
	for _, wt := range worktrees {
		paths = append(paths, wt.Path)
	}
	output.SimpleList(paths)
	return nil
}

func outputTable(worktrees []git.Worktree, repoPath string) error {
	// Get main worktree path for relative path calculation
	mainPath, _ := git.GetMainWorktreePath(repoPath)
	parentDir := filepath.Dir(mainPath)

	headers := []string{"PATH", "BRANCH", "COMMIT", "STATUS", ""}
	rows := make([][]string, 0, len(worktrees))

	for _, wt := range worktrees {
		// Calculate relative path if possible
		displayPath := wt.Path
		if relPath, err := filepath.Rel(parentDir, wt.Path); err == nil && !filepath.IsAbs(relPath) {
			displayPath = relPath
		}

		// Format branch
		branch := wt.Branch
		if wt.IsDetached {
			branch = "(detached)"
		}

		// Get status
		statusStr := ""
		status, err := git.GetWorktreeStatus(wt.Path)
		if err == nil {
			if status.Clean {
				statusStr = "Clean"
			} else {
				statusStr = "Dirty"
			}
		}

		// Main marker
		mainMarker := ""
		if wt.IsMain {
			mainMarker = "*"
		}
		if wt.Locked {
			mainMarker += "🔒"
		}

		rows = append(rows, []string{displayPath, branch, wt.Commit, statusStr, mainMarker})
	}

	output.Table(headers, rows)
	return nil
}
