package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Andrewy-gh/gwt/internal/config"
	"github.com/Andrewy-gh/gwt/internal/create"
	"github.com/Andrewy-gh/gwt/internal/git"
	"github.com/Andrewy-gh/gwt/internal/hooks"
	"github.com/Andrewy-gh/gwt/internal/output"
	"github.com/spf13/cobra"
)

// DeleteOptions holds options for the delete command
type DeleteOptions struct {
	Force        bool // -f, --force: Skip confirmation and force delete dirty worktrees
	DeleteBranch bool // -b, --delete-branch: Also delete the branch
	DryRun       bool // --dry-run: Show what would be deleted without doing it
	SkipHooks    bool // --skip-hooks: Skip post-delete hooks
}

var deleteOpts DeleteOptions

var deleteCmd = &cobra.Command{
	Use:     "delete <branch-or-path>...",
	Aliases: []string{"rm", "remove"},
	Short:   "Delete one or more worktrees",
	Long: `Safely delete one or more worktrees with confirmation and checks.

Worktrees can be specified by branch name or path. Pre-deletion checks
are performed to warn about uncommitted changes, unmerged branches, etc.

The main worktree cannot be deleted (this cannot be overridden).

Examples:
  gwt delete feature-auth           Delete by branch name
  gwt delete /path/to/worktree      Delete by path
  gwt delete feature-1 feature-2    Batch delete
  gwt delete -f feature-auth        Force delete (skip confirmation)
  gwt delete -b feature-auth        Also delete the branch`,
	Args: cobra.MinimumNArgs(1),
	RunE: runDelete,
}

func init() {
	deleteCmd.Flags().BoolVarP(&deleteOpts.Force, "force", "f", false, "skip confirmation, force delete dirty worktrees")
	deleteCmd.Flags().BoolVarP(&deleteOpts.DeleteBranch, "delete-branch", "b", false, "also delete the branch")
	deleteCmd.Flags().BoolVar(&deleteOpts.DryRun, "dry-run", false, "show what would be deleted")
	deleteCmd.Flags().BoolVar(&deleteOpts.SkipHooks, "skip-hooks", false, "skip post-delete hooks")

	rootCmd.AddCommand(deleteCmd)
}

// CheckStatus represents the result of a pre-deletion check
type CheckStatus int

const (
	CheckPass CheckStatus = iota
	CheckWarn
	CheckBlock
)

// PreDeleteCheck represents a single pre-deletion check result
type PreDeleteCheck struct {
	Name    string
	Status  CheckStatus
	Message string
}

// DeleteTarget represents a worktree to be deleted with its check results
type DeleteTarget struct {
	Worktree *git.Worktree
	Checks   []PreDeleteCheck
	Blocked  bool
	Error    error
}

func runDelete(cmd *cobra.Command, args []string) error {
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

	// Resolve all targets
	targets := resolveDeleteTargets(repoPath, args)

	// Check if we're trying to delete the current directory
	for _, target := range targets {
		if target.Worktree != nil {
			absPath, _ := filepath.Abs(cwd)
			if strings.HasPrefix(absPath, target.Worktree.Path) {
				output.Warning(fmt.Sprintf("Cannot delete worktree containing current directory: %s", target.Worktree.Path))
				target.Blocked = true
				target.Checks = append(target.Checks, PreDeleteCheck{
					Name:    "CurrentDirectory",
					Status:  CheckBlock,
					Message: "Current directory is inside this worktree",
				})
			}
		}
	}

	// If dry-run, just show what would happen
	if deleteOpts.DryRun {
		return showDryRun(targets, repoPath)
	}

	// Filter out errored targets
	var validTargets []*DeleteTarget
	for i, t := range targets {
		if t.Error != nil {
			output.Error(fmt.Sprintf("Cannot resolve '%s': %v", args[i], t.Error))
		} else {
			validTargets = append(validTargets, t)
		}
	}

	if len(validTargets) == 0 {
		return fmt.Errorf("no valid worktrees to delete")
	}

	// Show confirmation and get approval
	if !deleteOpts.Force {
		if !confirmDeletion(validTargets) {
			output.Info("Deletion cancelled")
			return nil
		}
	}

	// Acquire lock
	lock, err := create.AcquireLock(repoPath)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer func() {
		if err := lock.Release(); err != nil {
			output.Warning(fmt.Sprintf("Failed to release lock: %v", err))
		}
	}()

	// Delete each worktree
	deletedCount := 0
	for _, target := range validTargets {
		if target.Blocked && !deleteOpts.Force {
			output.Warning(fmt.Sprintf("Skipping %s (blocked)", target.Worktree.Path))
			continue
		}

		// Can't override main worktree block
		if target.Worktree.IsMain {
			output.Error(fmt.Sprintf("Cannot delete main worktree: %s", target.Worktree.Path))
			continue
		}

		err := deleteWorktree(repoPath, target)
		if err != nil {
			output.Error(fmt.Sprintf("Failed to delete %s: %v", target.Worktree.Path, err))
			continue
		}

		deletedCount++
		output.Success(fmt.Sprintf("Deleted worktree: %s", target.Worktree.Path))

		// Delete branch if requested
		if deleteOpts.DeleteBranch && target.Worktree.Branch != "" {
			err := git.DeleteBranch(repoPath, git.DeleteBranchOptions{
				Name:  target.Worktree.Branch,
				Force: deleteOpts.Force,
			})
			if err != nil {
				output.Warning(fmt.Sprintf("Could not delete branch %s: %v", target.Worktree.Branch, err))
			} else {
				output.Success(fmt.Sprintf("Deleted branch: %s", target.Worktree.Branch))
			}
		}

		// Execute post-delete hooks (Phase 10)
		if !deleteOpts.SkipHooks {
			cfg, err := config.Load(repoPath)
			if err != nil {
				cfg = config.DefaultConfig()
			}
			mainWorktree, err := git.GetMainWorktreePath(repoPath)
			if err != nil {
				output.Warning(fmt.Sprintf("Failed to get main worktree path: %v", err))
			} else {
				if err := runPostDeleteHooks(repoPath, cfg, target.Worktree, mainWorktree); err != nil {
					output.Warning(fmt.Sprintf("Post-delete hooks had errors: %v", err))
					// Non-fatal - worktree was deleted successfully
				}
			}
		}
	}

	if deletedCount == 0 {
		return fmt.Errorf("no worktrees were deleted")
	}

	output.Info(fmt.Sprintf("Deleted %d worktree(s)", deletedCount))
	return nil
}

func resolveDeleteTargets(repoPath string, args []string) []*DeleteTarget {
	targets := make([]*DeleteTarget, 0, len(args))

	for _, arg := range args {
		target := &DeleteTarget{}

		// Try to find by path first
		if filepath.IsAbs(arg) || strings.HasPrefix(arg, ".") || strings.HasPrefix(arg, "/") {
			wt, err := git.GetWorktree(arg)
			if err == nil {
				target.Worktree = wt
				target.Checks = runPreDeleteChecks(repoPath, wt)
				target.Blocked = hasBlockingCheck(target.Checks)
				targets = append(targets, target)
				continue
			}
		}

		// Try to find by branch name
		wt, err := git.FindWorktreeByBranch(repoPath, arg)
		if err != nil {
			target.Error = fmt.Errorf("failed to search worktrees: %w", err)
			targets = append(targets, target)
			continue
		}

		if wt == nil {
			// Try as a path relative to current directory
			absPath, _ := filepath.Abs(arg)
			wt, err = git.GetWorktree(absPath)
			if err != nil {
				target.Error = fmt.Errorf("worktree not found: %s", arg)
				targets = append(targets, target)
				continue
			}
		}

		target.Worktree = wt
		target.Checks = runPreDeleteChecks(repoPath, wt)
		target.Blocked = hasBlockingCheck(target.Checks)
		targets = append(targets, target)
	}

	return targets
}

func runPreDeleteChecks(repoPath string, wt *git.Worktree) []PreDeleteCheck {
	var checks []PreDeleteCheck

	// Check if main worktree
	if wt.IsMain {
		checks = append(checks, PreDeleteCheck{
			Name:    "IsMain",
			Status:  CheckBlock,
			Message: "Main worktree cannot be deleted",
		})
	}

	// Check for uncommitted changes
	status, err := git.GetWorktreeStatus(wt.Path)
	if err == nil && !status.Clean {
		checks = append(checks, PreDeleteCheck{
			Name:    "UncommittedChanges",
			Status:  CheckWarn,
			Message: fmt.Sprintf("Has uncommitted changes (%d staged, %d unstaged, %d untracked)", status.StagedCount, status.UnstagedCount, status.UntrackedCount),
		})
	}

	// Check if branch is merged
	if wt.Branch != "" && !wt.IsMain {
		// Try to check if merged to main/master
		mainBranch := getDefaultBranch(repoPath)
		if mainBranch != "" && mainBranch != wt.Branch {
			merged, err := git.IsBranchMerged(repoPath, wt.Branch, mainBranch)
			if err == nil && !merged {
				checks = append(checks, PreDeleteCheck{
					Name:    "NotMerged",
					Status:  CheckWarn,
					Message: fmt.Sprintf("Branch not merged to %s", mainBranch),
				})
			}
		}
	}

	// Check if locked
	if wt.Locked {
		checks = append(checks, PreDeleteCheck{
			Name:    "Locked",
			Status:  CheckWarn,
			Message: "Worktree is locked",
		})
	}

	// If no issues, add a passing check
	if len(checks) == 0 {
		checks = append(checks, PreDeleteCheck{
			Name:    "OK",
			Status:  CheckPass,
			Message: "Ready to delete",
		})
	}

	return checks
}

func getDefaultBranch(repoPath string) string {
	// Check for common default branch names
	for _, name := range []string{"main", "master"} {
		exists, _ := git.LocalBranchExists(repoPath, name)
		if exists {
			return name
		}
	}
	return ""
}

func hasBlockingCheck(checks []PreDeleteCheck) bool {
	for _, c := range checks {
		if c.Status == CheckBlock {
			return true
		}
	}
	return false
}

func hasWarningCheck(checks []PreDeleteCheck) bool {
	for _, c := range checks {
		if c.Status == CheckWarn {
			return true
		}
	}
	return false
}

func showDryRun(targets []*DeleteTarget, repoPath string) error {
	output.Println("Dry run - no changes will be made")
	output.Println("")

	mainPath, _ := git.GetMainWorktreePath(repoPath)
	parentDir := filepath.Dir(mainPath)

	for _, target := range targets {
		if target.Error != nil {
			output.Error(fmt.Sprintf("Error: %v", target.Error))
			continue
		}

		wt := target.Worktree
		displayPath := wt.Path
		if relPath, err := filepath.Rel(parentDir, wt.Path); err == nil && !filepath.IsAbs(relPath) {
			displayPath = relPath
		}

		status := "OK"
		if target.Blocked {
			status = "BLOCKED"
		} else if hasWarningCheck(target.Checks) {
			status = "WARNING"
		}

		output.Println(fmt.Sprintf("  %s (%s) - %s", displayPath, wt.Branch, status))

		for _, check := range target.Checks {
			if check.Status != CheckPass {
				prefix := "  "
				if check.Status == CheckBlock {
					prefix = "  !"
				}
				output.Println(fmt.Sprintf("    %s %s", prefix, check.Message))
			}
		}
	}

	return nil
}

func confirmDeletion(targets []*DeleteTarget) bool {
	// Count blocked and warning targets
	blockedCount := 0
	warnCount := 0
	okCount := 0

	for _, t := range targets {
		if t.Blocked {
			blockedCount++
		} else if hasWarningCheck(t.Checks) {
			warnCount++
		} else {
			okCount++
		}
	}

	output.Println("Worktrees to delete:")
	output.Println("")

	headers := []string{"PATH", "BRANCH", "STATUS", "CHECKS"}
	rows := make([][]string, 0, len(targets))

	for _, t := range targets {
		wt := t.Worktree
		status := "Clean"
		if s, err := git.GetWorktreeStatus(wt.Path); err == nil && !s.Clean {
			status = "Dirty"
		}

		checkStr := "OK"
		if t.Blocked {
			checkStr = "BLOCKED"
		} else if hasWarningCheck(t.Checks) {
			// Get first warning message
			for _, c := range t.Checks {
				if c.Status == CheckWarn {
					checkStr = c.Message
					break
				}
			}
		}

		rows = append(rows, []string{wt.Path, wt.Branch, status, checkStr})
	}

	output.Table(headers, rows)
	output.Println("")

	if blockedCount > 0 {
		output.Warning(fmt.Sprintf("%d worktree(s) will be skipped (blocked)", blockedCount))
	}
	if warnCount > 0 {
		output.Warning(fmt.Sprintf("%d worktree(s) have warnings", warnCount))
	}

	deleteCount := len(targets) - blockedCount
	if deleteCount == 0 {
		output.Error("No worktrees can be deleted")
		return false
	}

	output.Println("")
	output.Printf("Delete %d worktree(s)? [y/N] ", deleteCount)

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

func deleteWorktree(repoPath string, target *DeleteTarget) error {
	wt := target.Worktree

	// If locked and force, unlock first
	if wt.Locked && deleteOpts.Force {
		if err := git.UnlockWorktree(wt.Path); err != nil {
			output.Warning(fmt.Sprintf("Could not unlock worktree: %v", err))
		}
	}

	// Delete the worktree
	opts := git.RemoveWorktreeOptions{
		Path:  wt.Path,
		Force: deleteOpts.Force,
	}

	return git.RemoveWorktree(repoPath, opts)
}

// runPostDeleteHooks executes post-delete hooks after worktree deletion
func runPostDeleteHooks(repoPath string, cfg *config.Config, worktree *git.Worktree, mainWorktree string) error {
	if cfg == nil || len(cfg.Hooks.PostDelete) == 0 {
		output.Verbose("No post-delete hooks configured")
		return nil
	}

	output.Verbose(fmt.Sprintf("Running %d post-delete hooks...", len(cfg.Hooks.PostDelete)))

	executor := hooks.NewExecutor(repoPath, cfg)
	hookResult, err := executor.Execute(hooks.ExecuteOptions{
		HookType:         hooks.HookTypePostDelete,
		WorktreeBranch:   worktree.Branch,
		MainWorktreePath: mainWorktree,
		// Note: WorktreePath is omitted since worktree is deleted
	})

	if err != nil {
		return err
	}

	if hookResult.Failed > 0 {
		for _, hookErr := range hookResult.Errors {
			output.Warning(fmt.Sprintf("Post-delete hook failed: %s", hookErr.Error()))
		}
		return fmt.Errorf("%d hooks failed", hookResult.Failed)
	}

	if hookResult.Successful > 0 {
		output.Verbose(fmt.Sprintf("Executed %d post-delete hooks", hookResult.Successful))
	}

	return nil
}
