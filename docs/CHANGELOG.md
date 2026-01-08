# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
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
