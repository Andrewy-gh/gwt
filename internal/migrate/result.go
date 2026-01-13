package migrate

// MigrationTool represents a detected migration tool
type MigrationTool struct {
	Name        string   // e.g., "prisma", "makefile", "alembic"
	Command     []string // Command to run migrations
	Path        string   // Directory containing the tool
	Description string   // Human-readable description
}

// Result represents the outcome of a migration run
type Result struct {
	Skipped bool           // True if migrations were skipped
	Reason  string         // Why migrations were skipped (if applicable)
	Tool    *MigrationTool // Tool that was used
	Output  string         // Combined stdout/stderr
	Success bool           // True if migration completed successfully
	Error   error          // Error if migration failed
}

// RunOptions configures migration execution
type RunOptions struct {
	WorktreePath       string // Path to the new worktree
	Verbose            bool   // Stream output in real-time
	DryRun             bool   // Show what would run without executing
	SkipContainerCheck bool   // Skip Docker container readiness check
}
