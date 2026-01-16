package cli

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Andrewy-gh/gwt/internal/git"
	"github.com/Andrewy-gh/gwt/internal/output"
	"github.com/spf13/cobra"
)

// CleanupOptions holds options for the cleanup command
type CleanupOptions struct {
	ListOnly   bool     // -l, --list: Only list branches, don't delete
	Merged     bool     // -m, --merged: Target merged branches
	Stale      string   // -s, --stale: Target branches older than duration (e.g., "30d", "2w")
	DryRun     bool     // -n, --dry-run: Show what would be deleted
	Force      bool     // -f, --force: Delete without confirmation
	Exclude    []string // -e, --exclude: Branches to exclude
	BaseBranch string   // -b, --base: Base branch for merge detection
}

var cleanupOpts CleanupOptions

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Clean up merged or stale branches",
	Long: `Remove branches that have been merged or haven't been updated recently.

This command helps maintain a clean branch list by identifying and removing
branches that are no longer needed:
- Merged branches: Branches that have been merged into the base branch
- Stale branches: Branches with no commits for a specified period

Examples:
  gwt cleanup --list               List all candidates for cleanup
  gwt cleanup --merged             Delete merged branches (with confirmation)
  gwt cleanup --stale 30d          Delete branches older than 30 days
  gwt cleanup --merged --dry-run   Preview what would be deleted
  gwt cleanup --merged --force     Delete without confirmation
  gwt cleanup --merged -e main,develop  Exclude specific branches`,
	RunE: runCleanup,
}

func init() {
	cleanupCmd.Flags().BoolVarP(&cleanupOpts.ListOnly, "list", "l", false, "only list branches, don't delete")
	cleanupCmd.Flags().BoolVarP(&cleanupOpts.Merged, "merged", "m", false, "target merged branches")
	cleanupCmd.Flags().StringVarP(&cleanupOpts.Stale, "stale", "s", "", "target branches older than duration (e.g., 30d, 2w, 6m)")
	cleanupCmd.Flags().BoolVarP(&cleanupOpts.DryRun, "dry-run", "n", false, "show what would be deleted without doing it")
	cleanupCmd.Flags().BoolVarP(&cleanupOpts.Force, "force", "f", false, "delete without confirmation")
	cleanupCmd.Flags().StringSliceVarP(&cleanupOpts.Exclude, "exclude", "e", nil, "branches to exclude (comma-separated)")
	cleanupCmd.Flags().StringVarP(&cleanupOpts.BaseBranch, "base", "b", "", "base branch for merge detection (default: main or master)")

	rootCmd.AddCommand(cleanupCmd)
}

func runCleanup(cmd *cobra.Command, args []string) error {
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

	// Validate options
	if !cleanupOpts.ListOnly && !cleanupOpts.Merged && cleanupOpts.Stale == "" {
		// Default to list mode if no action specified
		cleanupOpts.ListOnly = true
	}

	// Parse stale duration if provided
	var staleDuration time.Duration
	if cleanupOpts.Stale != "" {
		var err error
		staleDuration, err = parseDuration(cleanupOpts.Stale)
		if err != nil {
			return fmt.Errorf("invalid stale duration: %w", err)
		}
	}

	// Get branch cleanup info
	baseBranch := cleanupOpts.BaseBranch
	if baseBranch == "" {
		baseBranch = git.GetDefaultBranch(repoPath)
	}

	// Default stale duration for info (30 days)
	infoDuration := staleDuration
	if infoDuration == 0 {
		infoDuration = 30 * 24 * time.Hour
	}

	branches, err := git.GetBranchCleanupInfo(repoPath, baseBranch, infoDuration)
	if err != nil {
		return fmt.Errorf("failed to get branch info: %w", err)
	}

	// Build exclude map
	excludeMap := make(map[string]bool)
	for _, e := range cleanupOpts.Exclude {
		excludeMap[strings.TrimSpace(e)] = true
	}
	// Always exclude the base branch
	if baseBranch != "" {
		excludeMap[baseBranch] = true
	}

	// Filter branches based on criteria
	var candidates []git.BranchCleanupInfo
	for _, b := range branches {
		// Skip excluded branches
		if excludeMap[b.Branch.Name] {
			continue
		}

		// Skip branches with worktrees
		if b.HasWorktree {
			continue
		}

		// Apply filters
		if cleanupOpts.ListOnly {
			// In list mode, show all branches with their status
			candidates = append(candidates, b)
		} else if cleanupOpts.Merged && b.IsMerged {
			candidates = append(candidates, b)
		} else if cleanupOpts.Stale != "" && b.IsStale {
			candidates = append(candidates, b)
		}
	}

	if len(candidates) == 0 {
		if cleanupOpts.ListOnly {
			output.Info("No branches found for cleanup")
		} else {
			output.Info("No branches match the cleanup criteria")
		}
		return nil
	}

	// List mode - just show the branches
	if cleanupOpts.ListOnly {
		return showBranchList(candidates, baseBranch)
	}

	// Dry run - show what would be deleted
	if cleanupOpts.DryRun {
		return showDryRunCleanup(candidates)
	}

	// Confirm deletion
	if !cleanupOpts.Force {
		if !confirmCleanup(candidates) {
			output.Info("Cleanup cancelled")
			return nil
		}
	}

	// Delete branches
	return deleteBranches(repoPath, candidates)
}

// parseDuration parses a duration string like "30d", "2w", "6m"
func parseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}

	// Try our custom format first: number + unit (d/w/m/y)
	// This prevents "6m" from being parsed as 6 minutes instead of 6 months
	re := regexp.MustCompile(`^(\d+)\s*([dwmy])$`)
	matches := re.FindStringSubmatch(s)
	if matches != nil {
		num, err := strconv.Atoi(matches[1])
		if err != nil {
			return 0, fmt.Errorf("invalid number: %s", matches[1])
		}

		unit := matches[2]
		switch unit {
		case "d":
			return time.Duration(num) * 24 * time.Hour, nil
		case "w":
			return time.Duration(num) * 7 * 24 * time.Hour, nil
		case "m":
			return time.Duration(num) * 30 * 24 * time.Hour, nil
		case "y":
			return time.Duration(num) * 365 * 24 * time.Hour, nil
		}
	}

	// Fallback to standard Go duration (for "24h", "1h30m", etc.)
	if d, err := time.ParseDuration(s); err == nil {
		return d, nil
	}

	return 0, fmt.Errorf("invalid duration format: %s (use format like 30d, 2w, 6m, 1y)", s)
}

func showBranchList(branches []git.BranchCleanupInfo, baseBranch string) error {
	if baseBranch != "" {
		output.Println(fmt.Sprintf("Branches (base: %s):", baseBranch))
	} else {
		output.Println("Branches:")
	}
	output.Println("")

	headers := []string{"BRANCH", "AGE", "MERGED", "STALE", "WORKTREE"}
	rows := make([][]string, 0, len(branches))

	for _, b := range branches {
		merged := ""
		if b.IsMerged {
			merged = "yes"
		}

		stale := ""
		if b.IsStale {
			stale = "yes"
		}

		worktree := ""
		if b.HasWorktree {
			worktree = "yes"
		}

		rows = append(rows, []string{
			b.Branch.Name,
			b.AgeString,
			merged,
			stale,
			worktree,
		})
	}

	output.Table(headers, rows)
	output.Println("")
	output.Info(fmt.Sprintf("Found %d branch(es)", len(branches)))

	return nil
}

func showDryRunCleanup(branches []git.BranchCleanupInfo) error {
	output.Println("Dry run - these branches would be deleted:")
	output.Println("")

	for _, b := range branches {
		status := ""
		if b.IsMerged {
			status = " (merged)"
		} else if b.IsStale {
			status = fmt.Sprintf(" (stale: %s)", b.AgeString)
		}
		output.Println(fmt.Sprintf("  %s%s", b.Branch.Name, status))
	}

	output.Println("")
	output.Info(fmt.Sprintf("Would delete %d branch(es)", len(branches)))
	return nil
}

func confirmCleanup(branches []git.BranchCleanupInfo) bool {
	output.Println("Branches to delete:")
	output.Println("")

	headers := []string{"BRANCH", "AGE", "REASON"}
	rows := make([][]string, 0, len(branches))

	for _, b := range branches {
		reason := ""
		if b.IsMerged {
			reason = "merged"
		} else if b.IsStale {
			reason = "stale"
		}

		rows = append(rows, []string{
			b.Branch.Name,
			b.AgeString,
			reason,
		})
	}

	output.Table(headers, rows)
	output.Println("")

	output.Printf("Delete %d branch(es)? [y/N] ", len(branches))

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

func deleteBranches(repoPath string, branches []git.BranchCleanupInfo) error {
	branchNames := make([]string, 0, len(branches))
	for _, b := range branches {
		branchNames = append(branchNames, b.Branch.Name)
	}

	// For merged branches, we can use -d (safe delete)
	// For stale branches, we need -D (force delete) since they may not be merged
	allMerged := true
	for _, b := range branches {
		if !b.IsMerged {
			allMerged = false
			break
		}
	}

	force := !allMerged || cleanupOpts.Force

	err := git.DeleteBranches(repoPath, branchNames, force)
	if err != nil {
		output.Error(fmt.Sprintf("Some branches could not be deleted: %v", err))
		// Count successful deletions
		var deleted int
		for _, name := range branchNames {
			exists, _ := git.LocalBranchExists(repoPath, name)
			if !exists {
				deleted++
			}
		}
		if deleted > 0 {
			output.Success(fmt.Sprintf("Deleted %d branch(es)", deleted))
		}
		return nil
	}

	output.Success(fmt.Sprintf("Deleted %d branch(es)", len(branchNames)))
	return nil
}
