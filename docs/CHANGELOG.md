# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Phase 6: File Copying**
  - New `internal/copy/` package for gitignored file management
  - Gitignored file discovery via `git status --ignored --porcelain`
  - Pattern matching with `doublestar` library for glob support
    - Exact matches: `.env`
    - Simple globs: `*.log`
    - Double-star globs: `**/.env`
    - Directory patterns: `node_modules`
  - File selection with size calculation and formatting
  - Default exclusions for common dependency directories
  - File and directory copying with progress tracking
  - Integration with `gwt create` command
  - `--skip-copy` flag to bypass file copying
  - Progress display in `internal/output/progress.go`
  - Comprehensive test coverage (discover, match, copy, selection)
  - Files created:
    - `internal/copy/discover.go` - Git ignored file discovery
    - `internal/copy/match.go` - Pattern matching logic
    - `internal/copy/copy.go` - File/directory copying
    - `internal/copy/selection.go` - File selection with sizes
    - `internal/copy/errors.go` - Custom error types
    - `internal/output/progress.go` - Progress bar display
    - Complete test files for all modules
- **Phase 5: List & Delete Worktrees (CLI)**
  - `gwt list` command with multiple output formats
    - Table output with PATH, BRANCH, COMMIT, STATUS columns
    - `--json` flag for JSON array output
    - `--simple` flag for one path per line (scripting)
    - `--all` / `-a` flag to include bare/prunable worktrees
    - Alias: `ls`
  - `gwt status` command for detailed worktree information
    - Shows path, branch, commit, last modified time
    - Working tree status (staged/unstaged/untracked counts)
    - Upstream tracking info (ahead/behind counts)
    - `--json` flag for JSON output
  - `gwt delete` command with safety checks
    - Delete by branch name or path
    - Pre-deletion checks: main worktree, uncommitted changes, unmerged branch, locked status
    - Batch deletion with confirmation prompt
    - `--force` / `-f` flag to skip confirmation and force delete dirty worktrees
    - `--delete-branch` / `-b` flag to also delete the branch
    - `--dry-run` flag to preview what would be deleted
    - Main worktree protection (cannot be overridden)
    - Aliases: `rm`, `remove`
  - New output utilities in `internal/output/`:
    - `JSON()` function for formatted JSON output
    - `SimpleList()` function for line-per-item output
  - Comprehensive test coverage for all new commands
- **Phase 4: Create Worktree (CLI)**
  - `gwt create` command with comprehensive flag support
  - Branch source detection: new branch from HEAD/ref, existing local, or remote branch
  - Branch name validation according to git rules (no spaces, special chars, etc.)
  - Directory name sanitization: converts branch names to valid directory names
  - Sibling directory placement: worktrees created as `../project-branch-name`
  - Directory collision detection with helpful error messages and suggestions
  - Rollback functionality: automatic cleanup on failure (removes worktree, branch, directory)
  - Operation locking: prevents concurrent `gwt create` operations with stale lock detection
  - Command flags:
    - `--branch` / `-b`: Create new branch from HEAD or specified ref
    - `--from`: Starting point for new branch (commit, branch, tag)
    - `--checkout`: Use existing local branch
    - `--remote`: Checkout remote branch (creates local tracking branch)
    - `--directory` / `-d`: Override target directory name
    - `--force` / `-f`: Force creation even with warnings
    - `--skip-install`, `--skip-migrations`, `--copy-config`: Future feature flags
  - New package `internal/create/`:
    - `validate.go`: Branch and directory name validation
    - `branch.go`: Branch source parsing and validation
    - `directory.go`: Directory collision detection
    - `worktree.go`: Worktree creation orchestration
    - `rollback.go`: Cleanup and rollback logic
    - `lock.go`: Concurrent operation locking
  - Comprehensive test coverage with unit tests
- **Phase 3: Configuration System**
  - Configuration struct definitions with YAML and mapstructure tags
  - Viper-based config loading from `.worktree.yaml`
  - Config file search order: explicit path → current dir → repo root → main worktree
  - Config inheritance: linked worktrees inherit config from main worktree
  - Validation for all config fields (docker mode, port offset, glob patterns, paths)
  - Default configuration values for all settings
  - Config file template with helpful comments
  - `gwt config` command to view current configuration
  - `gwt config init` command to create default config file (with `--force` and `--output` flags)
  - `gwt config path` command to show config file path
  - Global `--config` flag to override config file path
  - Comprehensive unit tests with test fixtures
  - Support for the following config sections:
    - `copy_defaults`: Files/patterns to pre-select for copying (glob support)
    - `copy_exclude`: Patterns to never select by default
    - `docker`: Docker Compose settings (compose files, data directories, mode, port offset)
    - `dependencies`: Dependency installation settings (auto-install, paths)
    - `migrations`: Database migration settings (auto-detect, custom command)
    - `hooks`: Lifecycle hooks (post_create, post_delete)
- **Phase 2: Git Operations Core**
  - Git command execution wrapper with timeout support and verbose logging
  - Comprehensive error types (GitError, NotARepoError, WorktreeError, BranchError, RemoteError)
  - Repository validation utilities (IsGitRepository, IsWorktree, GetRepoRoot, GetCurrentBranch, etc.)
  - Worktree operations: list, add, remove, prune, lock/unlock
  - Branch operations: list (local/remote), create, delete, rename, set upstream
  - Status detection: clean/dirty state, staged/unstaged/untracked counts, ahead/behind upstream
  - Remote operations: list, fetch, push, upstream tracking
  - Test utilities for creating temporary git repositories
  - 34+ unit tests with comprehensive coverage
- **Phase 1: Project Foundation**
  - Cobra CLI framework with root command
  - Global flags: `--verbose`, `--quiet`, `--config`, `--no-tui`
  - `gwt doctor` command for system prerequisite validation
    - Git installation and version check (minimum 2.20)
    - Git repository detection
    - Bare repository validation
    - Symlink support check (Windows)
    - Docker and Docker Compose detection (optional)
  - Output utilities with color support
  - Error handling with custom exit codes
  - Unit tests for core packages
  - Cross-platform support (Windows, macOS, Linux)

## [0.1.0] - 2025-01-06

### Added
- Project initialization
- Basic project structure
- Development documentation
