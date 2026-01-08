# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
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
