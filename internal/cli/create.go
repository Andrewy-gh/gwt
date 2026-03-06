package cli

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Andrewy-gh/gwt/internal/config"
	"github.com/Andrewy-gh/gwt/internal/copy"
	"github.com/Andrewy-gh/gwt/internal/create"
	"github.com/Andrewy-gh/gwt/internal/docker"
	"github.com/Andrewy-gh/gwt/internal/git"
	"github.com/Andrewy-gh/gwt/internal/hooks"
	"github.com/Andrewy-gh/gwt/internal/install"
	"github.com/Andrewy-gh/gwt/internal/migrate"
	"github.com/Andrewy-gh/gwt/internal/output"
	"github.com/Andrewy-gh/gwt/internal/tui"
	"github.com/spf13/cobra"
)

// CreateOptions holds all options for the create command
type CreateOptions struct {
	Branch         string   // -b, --branch: New branch name
	From           string   // --from: Starting point for new branch (default: HEAD)
	Checkout       string   // --checkout: Existing local branch to checkout
	Remote         string   // --remote: Remote branch to checkout (creates tracking branch)
	Directory      string   // -d, --directory: Override target directory name
	Force          bool     // -f, --force: Force creation even with warnings
	SkipInstall    bool     // --skip-install: Skip dependency installation
	SkipMigrations bool     // --skip-migrations: Skip running migrations
	CopyConfig     bool     // --copy-config: Copy .worktree.yaml to new worktree
	SkipCopy       bool     // --skip-copy: Skip copying gitignored files
	CopyFiles      []string // --copy: Additional files to copy (can be used multiple times)
	DockerMode     string   // --docker-mode: Docker setup mode: shared, new, or skip
	SkipDocker     bool     // --skip-docker: Skip Docker Compose setup
	SkipHooks      bool     // --skip-hooks: Skip post-creation hooks
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
	createCmd.Flags().BoolVar(&createOpts.SkipCopy, "skip-copy", false, "skip copying gitignored files")
	createCmd.Flags().StringSliceVar(&createOpts.CopyFiles, "copy", nil, "additional files to copy (can be used multiple times)")

	// Docker setup flags (Phase 7)
	createCmd.Flags().StringVar(&createOpts.DockerMode, "docker-mode", "", "Docker setup mode: shared, new, or skip")
	createCmd.Flags().BoolVar(&createOpts.SkipDocker, "skip-docker", false, "skip Docker Compose setup")

	// Hook execution flags (Phase 10)
	createCmd.Flags().BoolVar(&createOpts.SkipHooks, "skip-hooks", false, "skip post-creation hooks")

	// Mark mutually exclusive flags
	createCmd.MarkFlagsMutuallyExclusive("branch", "checkout", "remote")

	rootCmd.AddCommand(createCmd)
}

// runCreate is the main entry point for the create command
func runCreate(cmd *cobra.Command, args []string) error {
	// 1. Validate we're in a git repository (needed for both TUI and CLI)
	repoPath, err := getRepoPath(".")
	if err != nil {
		return err
	}

	// 2. Validate that at least one branch source is specified
	if !hasAnyBranchFlag(createOpts) {
		// No flags provided - launch TUI if enabled
		if GetNoTUI() {
			return fmt.Errorf("no branch specified; use --branch, --checkout, or --remote")
		}
		// Launch TUI with repo path
		return tui.Run(repoPath)
	}

	// 3. Validate flag combinations
	if err := validateCreateOptions(createOpts); err != nil {
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
	output.Success("Worktree created successfully")

	// Track for rollback
	rollback.TrackWorktree(result.Path)
	if result.IsNew {
		rollback.TrackBranch(result.Branch)
	}

	// Copy gitignored files (Phase 6)
	if !createOpts.SkipCopy {
		// Get main worktree path
		mainWorktree, err := git.GetMainWorktreePath(repoPath)
		if err != nil {
			output.Warning(fmt.Sprintf("Failed to get main worktree path: %v", err))
		} else {
			if err := copyIgnoredFiles(mainWorktree, result.Path, repoPath); err != nil {
				output.Warning(fmt.Sprintf("Failed to copy files: %v", err))
				// Non-fatal - worktree was created successfully
			}
		}
	}

	// Docker setup (Phase 7)
	if !createOpts.SkipDocker {
		// Get main worktree path
		mainWorktree, err := git.GetMainWorktreePath(repoPath)
		if err != nil {
			output.Warning(fmt.Sprintf("Failed to get main worktree path: %v", err))
		} else {
			if err := setupDocker(mainWorktree, result.Path, repoPath, result.Branch); err != nil {
				output.Warning(fmt.Sprintf("Docker setup failed: %v", err))
				// Non-fatal - worktree was created successfully
			}
		}
	}

	// Install dependencies (Phase 8)
	if !createOpts.SkipInstall {
		cfg, err := config.Load(repoPath)
		if err != nil {
			// If no config, use default config
			cfg = config.DefaultConfig()
		}
		if err := installDependencies(result.Path, cfg); err != nil {
			output.Warning(fmt.Sprintf("Dependency installation had errors: %v", err))
			// Non-fatal - worktree was created successfully
		}
	}

	// Run migrations (Phase 9)
	if !createOpts.SkipMigrations {
		cfg, err := config.Load(repoPath)
		if err != nil {
			// If no config, use default config
			cfg = config.DefaultConfig()
		}
		if err := runMigrations(result.Path, cfg); err != nil {
			output.Warning(fmt.Sprintf("Migration had errors: %v", err))
			// Non-fatal - worktree was created successfully
		}
	}

	// Execute post-creation hooks (Phase 10)
	if !createOpts.SkipHooks {
		cfg, err := config.Load(repoPath)
		if err != nil {
			// If no config, use default config
			cfg = config.DefaultConfig()
		}
		mainWorktree, err := git.GetMainWorktreePath(repoPath)
		if err != nil {
			output.Warning(fmt.Sprintf("Failed to get main worktree path: %v", err))
		} else {
			if err := runPostCreateHooks(repoPath, cfg, result, mainWorktree); err != nil {
				output.Warning(fmt.Sprintf("Post-create hooks had errors: %v", err))
				// Non-fatal - worktree was created successfully
			}
		}
	}

	// Success - prevent rollback
	rollback.Clear()

	return result, nil
}

func copyIgnoredFiles(source, target string, repoPath string) error {
	// 1. Load config
	cfg, err := config.Load(repoPath)
	if err != nil {
		// If no config, use empty config (will still use default excludes)
		cfg = &config.Config{}
	}

	// 2. Discover ignored files
	ignored, err := copy.DiscoverIgnored(source)
	if err != nil {
		return err
	}

	if len(ignored) == 0 {
		output.Info("No gitignored files to copy")
		return nil
	}

	// 3. Create pattern matcher from config
	matcher := copy.NewPatternMatcher(cfg.CopyDefaults, cfg.CopyExclude)

	// 4. Create selection (pre-select defaults)
	selection := copy.NewSelection(ignored, matcher)

	// 5. Get selected files
	selected := selection.GetSelected()

	if len(selected) == 0 {
		output.Info("No files selected for copying")
		return nil
	}

	// 6. Show summary and copy
	output.Info(fmt.Sprintf("Copying %d files (%s)...",
		len(selected), copy.FormatSize(selection.SelectedSize)))

	// Create progress bar
	progressBar := output.NewProgressBar(len(selected), 20)

	result, err := copy.Copy(copy.CopyOptions{
		SourceDir: source,
		TargetDir: target,
		Files:     selected,
		OnProgress: func(progress copy.CopyProgress) {
			progressBar.Update(progress.FilesDone, progress.CurrentFile)
		},
		PreserveMode: true,
	})

	progressBar.Done()

	if err != nil {
		return err
	}

	// Report any non-fatal errors
	if len(result.Errors) > 0 {
		for _, copyErr := range result.Errors {
			output.Warning(fmt.Sprintf("Failed to copy %s: %v", copyErr.Path, copyErr.Err))
		}
	}

	output.Success(fmt.Sprintf("Copied %d files (%s)",
		result.FilesCopied, copy.FormatSize(result.BytesCopied)))

	return nil
}

func printSuccessMessage(result *create.CreateWorktreeResult) {
	output.Success("Created worktree successfully!")
	output.Success(fmt.Sprintf("Worktree is ready locally on branch %s at %s", result.Branch, result.Commit))
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

// setupDocker sets up Docker Compose for the new worktree
func setupDocker(mainWorktree, newWorktree, repoPath, branchName string) error {
	// 1. Load config
	cfg, err := config.Load(repoPath)
	if err != nil {
		// If no config, use empty config
		cfg = &config.Config{}
	}

	// 2. Detect compose files
	composeFiles, err := docker.DetectOrLoad(mainWorktree, cfg.Docker.ComposeFiles)
	if err != nil {
		if errors.Is(err, docker.ErrNoComposeFile) {
			output.Verbose("No Docker Compose files found")
			return nil
		}
		return err
	}

	// 3. Parse compose files
	composePaths := docker.GetComposePaths(composeFiles)
	composeConfig, err := docker.ParseComposeFiles(composePaths)
	if err != nil {
		return err
	}

	// Show detected services
	services := make([]string, 0, len(composeConfig.Services))
	for name := range composeConfig.Services {
		services = append(services, name)
	}
	output.Info(fmt.Sprintf("Found Docker services: %s", strings.Join(services, ", ")))

	// 4. Determine mode
	mode := createOpts.DockerMode
	if mode == "" {
		mode = cfg.Docker.DefaultMode
	}
	if mode == "" {
		mode = "shared" // Default
	}

	// 5. Execute mode
	switch mode {
	case "shared":
		return setupSharedMode(mainWorktree, newWorktree, cfg, composeConfig)

	case "new":
		return setupNewMode(mainWorktree, newWorktree, branchName, cfg, composeConfig, composePaths)

	case "skip":
		output.Info("Skipping Docker setup")
		return nil

	default:
		return fmt.Errorf("invalid docker mode: %s (must be 'shared', 'new', or 'skip')", mode)
	}
}

// setupSharedMode sets up Docker in shared mode (symlink data directories)
func setupSharedMode(mainWorktree, newWorktree string, cfg *config.Config, composeConfig *docker.ComposeConfig) error {
	result, err := docker.SetupSharedMode(docker.SharedModeOptions{
		MainWorktree:    mainWorktree,
		NewWorktree:     newWorktree,
		DataDirectories: cfg.Docker.DataDirectories,
		ComposeConfig:   composeConfig,
	})
	if err != nil {
		return err
	}

	// Display results
	output.Println("")
	output.Info("Docker Compose Setup (Shared Mode)")
	output.Info("──────────────────────────────────")

	for _, linkedDir := range result.LinkedDirs {
		relPath := strings.TrimPrefix(linkedDir.Target, newWorktree)
		relPath = strings.TrimPrefix(relPath, "/")
		relPath = strings.TrimPrefix(relPath, "\\")

		methodStr := ""
		switch linkedDir.Method {
		case docker.LinkSymlink:
			methodStr = "symlink"
		case docker.LinkJunction:
			methodStr = "junction"
		case docker.LinkCopy:
			methodStr = "copied"
		}

		output.Success(fmt.Sprintf("Linked %s (%s)", relPath, methodStr))
	}

	// Display warnings
	for _, warning := range result.Warnings {
		output.Warning(warning)
	}

	if len(result.LinkedDirs) > 0 {
		output.Println("")
		output.Info("Containers will share data with the main worktree.")
	}
	output.Info("Run 'docker compose up' to start services.")

	return nil
}

// setupNewMode sets up Docker in new mode (isolated containers)
func setupNewMode(mainWorktree, newWorktree, branchName string, cfg *config.Config, composeConfig *docker.ComposeConfig, composePaths []string) error {
	// Default port offset if not configured
	portOffset := cfg.Docker.PortOffset
	if portOffset == 0 {
		portOffset = 1
	}

	result, err := docker.SetupNewMode(docker.NewModeOptions{
		MainWorktree:    mainWorktree,
		NewWorktree:     newWorktree,
		BranchName:      branchName,
		DataDirectories: cfg.Docker.DataDirectories,
		ComposeConfig:   composeConfig,
		PortOffset:      portOffset,
	})
	if err != nil {
		return err
	}

	// Display results
	output.Println("")
	output.Info("Docker Compose Setup (New Mode)")
	output.Info("───────────────────────────────")

	// Show copied directories
	for _, dir := range result.CopiedDirs {
		output.Success(fmt.Sprintf("Copied %s", dir))
	}

	if result.OverrideFile != "" {
		output.Success("Created docker-compose.worktree.yml")
	}

	// Show renamed volumes
	if len(result.RenamedVolumes) > 0 {
		output.Println("")
		output.Info("Volumes renamed:")
		for oldName, newName := range result.RenamedVolumes {
			output.Info(fmt.Sprintf("  %s → %s", oldName, newName))
		}
	}

	// Show remapped ports
	if len(result.RemappedPorts) > 0 {
		output.Println("")
		output.Info("Ports remapped:")
		for oldPort, newPort := range result.RemappedPorts {
			output.Info(fmt.Sprintf("  %s → %d", oldPort, newPort))
		}
	}

	// Show warnings
	for _, warning := range result.Warnings {
		output.Warning(warning)
	}
	for _, warning := range result.PortWarnings {
		output.Warning(warning)
	}

	// Generate helper script
	if err := docker.GenerateHelperScript(docker.HelperScriptOptions{
		WorktreePath: newWorktree,
		ComposeFiles: composePaths,
		OverrideFile: "docker-compose.worktree.yml",
	}); err != nil {
		output.Warning(fmt.Sprintf("Failed to generate helper script: %v", err))
	} else {
		output.Println("")
		output.Success("Created ./dc helper script for convenience.")
		output.Info("Run './dc up' to start services.")
	}

	return nil
}

// installDependencies installs dependencies for the newly created worktree
func installDependencies(worktreePath string, cfg *config.Config) error {
	opts := install.InstallOptions{
		Verbose: GetVerbose(),
		Timeout: 5 * time.Minute,
	}

	if GetVerbose() {
		opts.OnProgress = func(line string) {
			output.Verbose(line)
		}
	}

	result, err := install.Install(worktreePath, &cfg.Dependencies, opts)
	if err != nil {
		return err
	}

	if result.Skipped {
		output.Verbose(fmt.Sprintf("Dependency installation skipped: %s", result.Reason))
		return nil
	}

	if result.HasErrors() {
		return fmt.Errorf("%d of %d installations failed",
			result.ErrorCount(), len(result.Managers))
	}

	return nil
}

// runMigrations executes database migrations for the new worktree
func runMigrations(worktreePath string, cfg *config.Config) error {
	var migrateCfg *config.MigrationsConfig
	if cfg != nil {
		migrateCfg = &cfg.Migrations
	}

	opts := migrate.RunOptions{
		WorktreePath: worktreePath,
		Verbose:      GetVerbose(),
	}

	result, err := migrate.Run(opts, migrateCfg)
	if err != nil {
		return err
	}

	if result.Skipped {
		output.Verbose(fmt.Sprintf("Migrations skipped: %s", result.Reason))
		return nil
	}

	if !result.Success {
		output.Warning(fmt.Sprintf("Migration failed: %v", result.Error))
		if result.Output != "" {
			output.Verbose("Migration output:\n" + result.Output)
		}
		return result.Error // Return error but caller treats as non-fatal
	}

	output.Success(fmt.Sprintf("Migrations completed (%s)", result.Tool.Name))
	return nil
}

// runPostCreateHooks executes post-creation hooks for the new worktree
func runPostCreateHooks(repoPath string, cfg *config.Config, result *create.CreateWorktreeResult, mainWorktree string) error {
	if cfg == nil || len(cfg.Hooks.PostCreate) == 0 {
		output.Verbose("No post-create hooks configured")
		return nil
	}

	output.Info(fmt.Sprintf("Running %d post-create hooks...", len(cfg.Hooks.PostCreate)))

	executor := hooks.NewExecutor(repoPath, cfg)
	hookResult, err := executor.Execute(hooks.ExecuteOptions{
		HookType:         hooks.HookTypePostCreate,
		WorktreePath:     result.Path,
		WorktreeBranch:   result.Branch,
		MainWorktreePath: mainWorktree,
	})

	if err != nil {
		output.Warning(fmt.Sprintf("Hook execution error: %v", err))
		return err
	}

	if hookResult.Failed > 0 {
		output.Warning(fmt.Sprintf("Hooks: %d succeeded, %d failed", hookResult.Successful, hookResult.Failed))
		for _, hookErr := range hookResult.Errors {
			output.Warning(fmt.Sprintf("  - %s", hookErr.Error()))
		}
		return fmt.Errorf("%d hooks failed", hookResult.Failed)
	}

	if hookResult.Successful > 0 {
		output.Success(fmt.Sprintf("Executed %d post-create hooks", hookResult.Successful))
	}

	return nil
}
