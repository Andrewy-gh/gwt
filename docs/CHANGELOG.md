# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial project foundation (Phase 1)
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
