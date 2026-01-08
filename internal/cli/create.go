package cli

import (
	"fmt"

	"github.com/Andrewy-gh/gwt/internal/create"
	"github.com/Andrewy-gh/gwt/internal/git"
	"github.com/Andrewy-gh/gwt/internal/output"
	"github.com/spf13/cobra"
)

// CreateOptions holds all options for the create command
type CreateOptions struct {
	Branch         string // -b, --branch: New branch name
	From           string // --from: Starting point for new branch (default: HEAD)
	Checkout       string // --checkout: Existing local branch to checkout
	Remote         string // --remote: Remote branch to checkout (creates tracking branch)
	Directory      string // -d, --directory: Override target directory name
	Force          bool   // -f, --force: Force creation even with warnings
	SkipInstall    bool   // --skip-install: Skip dependency installation
	SkipMigrations bool   // --skip-migrations: Skip running migrations
	CopyConfig     bool   // --copy-config: Copy .worktree.yaml to new worktree
}

var createOpts CreateOptions

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new worktree",
	Long: `Create a new worktree from a new or existing branch.

Examples:
  gwt create -b feature-auth          Create worktree with new branch from HEAD
  gwt create -b feature-auth --from main  Create from specific branch
  gwt create --checkout existing-branch   Use existing local branch
  gwt create --remote origin/feature      Checkout remote branch`,
	RunE: runCreate,
}

func init() {
	// Branch source flags
	createCmd.Flags().StringVarP(&createOpts.Branch, "branch", "b", "", "new branch name")
	createCmd.Flags().StringVar(&createOpts.From, "from", "", "starting point for new branch (default: HEAD)")
	createCmd.Flags().StringVar(&createOpts.Checkout, "checkout", "", "existing local branch to checkout")
	createCmd.Flags().StringVar(&createOpts.Remote, "remote", "", "remote branch to checkout")

	// Directory and behavior flags
	createCmd.Flags().StringVarP(&createOpts.Directory, "directory", "d", "", "override target directory name")
	createCmd.Flags().BoolVarP(&createOpts.Force, "force", "f", false, "force creation even with warnings")

	// Post-creation flags (features implemented in later phases)
	createCmd.Flags().BoolVar(&createOpts.SkipInstall, "skip-install", false, "skip dependency installation")
	createCmd.Flags().BoolVar(&createOpts.SkipMigrations, "skip-migrations", false, "skip running migrations")
	createCmd.Flags().BoolVar(&createOpts.CopyConfig, "copy-config", false, "copy .worktree.yaml to new worktree")

	// Mark mutually exclusive flags
	createCmd.MarkFlagsMutuallyExclusive("branch", "checkout", "remote")

	rootCmd.AddCommand(createCmd)
}

// runCreate is the main entry point for the create command
func runCreate(cmd *cobra.Command, args []string) error {
	// 1. Validate that at least one branch source is specified
	if !hasAnyBranchFlag(createOpts) {
		// No flags provided - would launch TUI (Phase 12)
		return fmt.Errorf("interactive mode not yet implemented; use --branch, --checkout, or --remote")
	}

	// 2. Validate flag combinations
	if err := validateCreateOptions(createOpts); err != nil {
		return err
	}

	// 3. Validate we're in a git repository
	repoPath, err := getRepoPath(".")
	if err != nil {
		return err
	}

	// 4. Acquire operation lock
	lock, err := acquireLock(repoPath)
	if err != nil {
		return err
	}
	defer releaseLock(lock)

	// 5. Parse branch specification
	spec, err := parseBranchSpec(createOpts)
	if err != nil {
		return err
	}

	// 6. Validate branch specification
	if err := validateBranchSpec(repoPath, spec); err != nil {
		return err
	}

	// 7. Calculate target directory
	targetDir, err := calculateTargetDirectory(repoPath, spec, createOpts.Directory)
	if err != nil {
		return err
	}

	// 8. Check directory availability
	if err := checkDirectory(targetDir); err != nil {
		return err
	}

	// 9. Create the worktree with rollback support
	result, err := createWorktreeWithRollback(repoPath, spec, targetDir)
	if err != nil {
		return err
	}

	// 10. Print success message
	printSuccessMessage(result)

	return nil
}

// hasAnyBranchFlag checks if any branch source flag was provided
func hasAnyBranchFlag(opts CreateOptions) bool {
	return opts.Branch != "" || opts.Checkout != "" || opts.Remote != ""
}

// validateCreateOptions validates the create options for invalid combinations
func validateCreateOptions(opts CreateOptions) error {
	// --from can only be used with --branch
	if opts.From != "" && opts.Branch == "" {
		return fmt.Errorf("--from can only be used with --branch")
	}

	// At least one branch source must be specified (already checked in runCreate)
	if !hasAnyBranchFlag(opts) {
		return fmt.Errorf("must specify one of: --branch, --checkout, or --remote")
	}

	return nil
}

// Helper functions for create flow
// These wrap the create package functions with proper imports and error handling

func getRepoPath(path string) (string, error) {
	// Import git package
	repoPath, err := git.GetRepoRoot(path)
	if err != nil {
		return "", fmt.Errorf("not a git repository")
	}

	// Validate not bare
	if err := git.ValidateNotBare(repoPath); err != nil {
		return "", err
	}

	return repoPath, nil
}

func acquireLock(repoPath string) (*create.OperationLock, error) {
	lock, err := create.AcquireLock(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}
	return lock, nil
}

func releaseLock(lock *create.OperationLock) {
	if lock != nil {
		if err := lock.Release(); err != nil {
			output.Warning(fmt.Sprintf("Failed to release lock: %v", err))
		}
	}
}

func parseBranchSpec(opts CreateOptions) (*create.BranchSpec, error) {
	// Convert cli.CreateOptions to create.CreateOptions
	createOpts := create.CreateOptions{
		Branch:   opts.Branch,
		From:     opts.From,
		Checkout: opts.Checkout,
		Remote:   opts.Remote,
	}

	spec, err := create.ParseBranchSpec(createOpts)
	if err != nil {
		return nil, err
	}
	output.Verbose(fmt.Sprintf("Branch source: %s", create.GetSourceDescription(spec)))
	return spec, nil
}

func validateBranchSpec(repoPath string, spec *create.BranchSpec) error {
	if err := create.ValidateBranchSpec(repoPath, spec); err != nil {
		return err
	}
	return nil
}

func calculateTargetDirectory(repoPath string, spec *create.BranchSpec, overrideDir string) (string, error) {
	if overrideDir != "" {
		// User specified directory override
		output.Verbose(fmt.Sprintf("Using custom directory: %s", overrideDir))
		return overrideDir, nil
	}

	// Get main worktree path
	mainWorktree, err := git.GetMainWorktreePath(repoPath)
	if err != nil {
		return "", fmt.Errorf("failed to get main worktree path: %w", err)
	}

	// Generate path based on branch name
	branchName := create.ResolveBranchName(spec)
	targetDir := create.GenerateWorktreePath(mainWorktree, branchName)

	output.Verbose(fmt.Sprintf("Calculated target directory: %s", targetDir))
	return targetDir, nil
}

func checkDirectory(targetDir string) error {
	if err := create.CheckDirectory(targetDir); err != nil {
		if dirErr, ok := err.(*create.DirectoryExistsError); ok {
			output.Error(fmt.Sprintf("Directory already exists: %s", dirErr.Path))
			if dirErr.IsWorktree {
				output.Info("This directory is already a git worktree")
			} else if dirErr.IsEmpty {
				output.Info("Directory exists but is empty")
			} else {
				output.Info("Directory exists and contains files")
			}
			if dirErr.SuggestedAlt != "" {
				output.Info(fmt.Sprintf("Try using: --directory %s", dirErr.SuggestedAlt))
			}
		}
		return err
	}
	return nil
}

func createWorktreeWithRollback(repoPath string, spec *create.BranchSpec, targetDir string) (*create.CreateWorktreeResult, error) {
	// Set up rollback
	rollback := create.NewRollback(repoPath)
	defer func() {
		if rollback.IsEnabled() {
			output.Verbose("Rolling back changes...")
			if err := rollback.Execute(); err != nil {
				output.Warning(fmt.Sprintf("Rollback failed: %v", err))
			}
		}
	}()

	// Create the worktree
	output.Info(fmt.Sprintf("Creating worktree at %s...", targetDir))
	result, err := create.CreateWorktree(repoPath, spec, targetDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create worktree: %w", err)
	}

	// Track for rollback
	rollback.TrackWorktree(result.Path)
	if result.IsNew {
		rollback.TrackBranch(result.Branch)
	}

	// TODO: Copy config if requested (Phase 6)
	// TODO: Run post-creation hooks (Phase 7+)

	// Success - prevent rollback
	rollback.Clear()

	return result, nil
}

func printSuccessMessage(result *create.CreateWorktreeResult) {
	output.Success("Created worktree successfully!")
	output.Println("")
	output.Info(fmt.Sprintf("  Path:   %s", result.Path))
	output.Info(fmt.Sprintf("  Branch: %s", result.Branch))
	output.Info(fmt.Sprintf("  Commit: %s", result.Commit))

	if result.IsNew {
		output.Info(fmt.Sprintf("  Source: %s", result.FromRef))
	}

	output.Println("")
	output.Info("To start working:")
	output.Info(fmt.Sprintf("  cd %s", result.Path))
}
