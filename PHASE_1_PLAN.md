# Phase 1: Project Foundation - Implementation Plan

## Overview

Phase 1 establishes the foundation for the GWT (Git Worktree Manager) CLI tool. This phase focuses on project structure, CLI framework setup, and the `gwt doctor` diagnostic command.

**Deliverables:**
- Working Go project with proper module structure
- Cobra-based CLI with root command
- `gwt doctor` command for prerequisite validation
- Global flags and basic error handling
- Output utilities for consistent messaging

---

## Tasks

### 1. Initialize Go Module and Project Structure

**Objective:** Set up the Go module and create the directory structure defined in the specification.

**Steps:**
1. Initialize Go module: `go mod init github.com/yourusername/gwt`
2. Create directory structure:
   ```
   gwt/
   ├── cmd/gwt/main.go
   ├── internal/
   │   ├── cli/
   │   ├── config/
   │   ├── git/
   │   ├── output/
   │   └── version/
   ├── go.mod
   └── go.sum
   ```
3. Add initial dependencies to `go.mod`

**Files to Create:**
- `cmd/gwt/main.go` - Application entry point
- `internal/version/version.go` - Version information

**Acceptance Criteria:**
- [ ] `go build ./...` succeeds
- [ ] `go mod tidy` produces no errors
- [ ] Directory structure matches specification

---

### 2. Set Up Cobra CLI Framework

**Objective:** Configure Cobra as the CLI framework with the root command.

**Steps:**
1. Install Cobra: `go get github.com/spf13/cobra`
2. Create root command in `internal/cli/root.go`
3. Wire root command to main.go
4. Configure command metadata (use, short, long descriptions)

**Files to Create:**
- `internal/cli/root.go` - Root command definition and initialization

**Root Command Behavior:**
- Running `gwt` with no args should display help (TUI comes in Phase 11+)
- Display version on `--version`
- Show help on `--help` or `-h`

**Acceptance Criteria:**
- [ ] `gwt --help` displays usage information
- [ ] `gwt --version` displays version string
- [ ] Unknown commands produce helpful error messages

---

### 3. Implement Global Flags

**Objective:** Add global flags that apply to all commands.

**Flags to Implement:**

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--verbose` | `-v` | bool | false | Enable verbose output |
| `--quiet` | `-q` | bool | false | Suppress non-essential output |
| `--config` | `-c` | string | "" | Path to config file |
| `--no-tui` | | bool | false | Disable TUI, use simple prompts |
| `--help` | `-h` | bool | | Show help (Cobra built-in) |
| `--version` | | bool | | Show version (Cobra built-in) |

**Files to Modify:**
- `internal/cli/root.go` - Add persistent flags

**Implementation Notes:**
- `--verbose` and `--quiet` are mutually exclusive
- Store flag values in a shared context or global config struct
- `--no-tui` is a placeholder for future TUI functionality

**Acceptance Criteria:**
- [ ] All flags are recognized and parsed correctly
- [ ] `--verbose` and `--quiet` conflict produces error
- [ ] Flag values accessible to subcommands

---

### 4. Create Output Utilities

**Objective:** Build consistent output helpers for user messaging.

**Files to Create:**
- `internal/output/output.go` - Output utility functions

**Functions to Implement:**

```go
// Success prints a success message with checkmark
func Success(msg string)           // ✓ msg

// Warning prints a warning message
func Warning(msg string)           // ⚠ msg

// Error prints an error message
func Error(msg string)             // ✗ msg

// Info prints an informational message
func Info(msg string)              // msg (no prefix)

// Verbose prints only if verbose mode is enabled
func Verbose(msg string)

// Print prints a message (respects quiet mode)
func Print(msg string)

// Printf prints formatted message
func Printf(format string, args ...interface{})

// Table prints tabular data
func Table(headers []string, rows [][]string)
```

**Implementation Notes:**
- Respect `--quiet` flag (suppress Info, Success, Warning)
- Respect `--verbose` flag (enable Verbose messages)
- Use colors/symbols only if terminal supports them
- Consider `os.Stdout` vs `os.Stderr` for different message types

**Acceptance Criteria:**
- [ ] All output functions work correctly
- [ ] Quiet mode suppresses non-essential output
- [ ] Verbose mode shows additional details
- [ ] Colors disabled when not in terminal

---

### 5. Create Basic Error Handling

**Objective:** Establish error handling patterns and custom error types.

**Files to Create:**
- `internal/cli/errors.go` - Error handling utilities

**Error Types:**

```go
// ExitError wraps an error with an exit code
type ExitError struct {
    Err      error
    ExitCode int
}

// Common exit codes
const (
    ExitSuccess         = 0
    ExitGeneralError    = 1
    ExitGitNotFound     = 2
    ExitNotGitRepo      = 3
    ExitConfigError     = 4
)
```

**Error Messages:**
- Provide actionable guidance when possible
- Include links to documentation for common issues
- Show stack traces only in verbose mode

**Acceptance Criteria:**
- [ ] Errors display user-friendly messages
- [ ] Exit codes are consistent and documented
- [ ] Verbose mode shows additional error context

---

### 6. Implement `gwt doctor` Command

**Objective:** Create a diagnostic command that validates system prerequisites.

**Files to Create:**
- `internal/cli/doctor.go` - Doctor command implementation
- `internal/git/git.go` - Git command wrapper (minimal for Phase 1)

**Checks to Perform:**

| Check | Pass | Fail |
|-------|------|------|
| Git installed | ✓ Git installed (2.43.0) | ✗ Git is not installed |
| Git version | ✓ Git version supported | ⚠ Git version X.Y may have issues |
| Git repository | ✓ Git repository detected | ✗ Not a git repository |
| Not bare repo | ✓ Not a bare repository | ✗ Bare repositories not supported |
| Symlink support | ✓ Symlink permissions available | ⚠ Symlinks may require elevation |
| Docker installed | ✓ Docker installed (24.0.7) | ⚠ Docker not found (optional) |
| Docker Compose | ✓ Docker Compose available | ⚠ Docker Compose not found (optional) |

**Output Format:**
```
$ gwt doctor
Checking prerequisites...

✓ Git installed (2.43.0)
✓ Git repository detected
✓ Not a bare repository
✓ Symlink permissions available
✓ Docker installed (24.0.7)
✓ Docker Compose available

All checks passed! gwt is ready to use.
```

**Failure Output Example:**
```
$ gwt doctor
Checking prerequisites...

✗ Git is not installed or not in PATH

gwt requires Git to be installed. Please install Git:
  • Windows: https://git-scm.com/download/win
  • macOS:   brew install git
  • Linux:   apt install git / dnf install git

After installing, restart your terminal and try again.
```

**Implementation Details:**

1. **Git Check:**
   - Run `git --version`
   - Parse version string
   - Minimum version: Git 2.20 (worktree features)

2. **Repository Check:**
   - Run `git rev-parse --git-dir`
   - Check exit code

3. **Bare Repo Check:**
   - Run `git rev-parse --is-bare-repository`
   - Check if output is "true"

4. **Symlink Check (Windows):**
   - Attempt to create a test symlink
   - Clean up test files
   - Report Developer Mode status if needed

5. **Docker Check:**
   - Run `docker --version`
   - Run `docker compose version` or `docker-compose --version`
   - These are optional (warnings, not errors)

**Acceptance Criteria:**
- [ ] All checks execute without hanging
- [ ] Pass/fail status clearly indicated
- [ ] Helpful remediation messages on failure
- [ ] Exit code reflects overall status
- [ ] Works on Windows, macOS, and Linux

---

## Implementation Order

```
1. Initialize Go module       ─┐
2. Create directory structure ─┴─▶ Can build empty project

3. Set up Cobra root command  ─┐
4. Add global flags           ─┴─▶ Basic CLI works

5. Create output utilities    ─┐
6. Create error handling      ─┴─▶ Consistent messaging

7. Implement gwt doctor       ────▶ First functional command
```

---

## Dependencies

```
github.com/spf13/cobra v1.8.0      # CLI framework
github.com/spf13/pflag v1.0.5      # Flag parsing (Cobra dependency)
```

**Note:** Viper will be added in Phase 3 for configuration management.

---

## Testing Strategy

### Unit Tests
- `internal/output/output_test.go` - Test output functions
- `internal/cli/doctor_test.go` - Test individual checks (with mocks)

### Integration Tests
- Run `gwt doctor` in actual git repository
- Run `gwt doctor` outside git repository (expect failure)
- Test on Windows with/without Developer Mode

### Manual Testing Checklist
- [ ] `gwt` shows help
- [ ] `gwt --version` shows version
- [ ] `gwt --help` shows detailed help
- [ ] `gwt doctor` passes in valid git repo
- [ ] `gwt doctor` fails gracefully outside git repo
- [ ] `gwt -v doctor` shows verbose output
- [ ] `gwt -q doctor` shows minimal output
- [ ] `gwt unknowncommand` shows helpful error

---

## Definition of Done

Phase 1 is complete when:

1. **Project Structure**
   - [x] Go module initialized
   - [x] Directory structure created
   - [x] Project builds without errors

2. **CLI Framework**
   - [x] Root command implemented
   - [x] Global flags working
   - [x] Help and version display correctly

3. **Output System**
   - [x] Output utilities implemented
   - [x] Verbose/quiet modes work
   - [x] Consistent message formatting

4. **Doctor Command**
   - [x] All prerequisite checks implemented
   - [x] Clear pass/fail indicators
   - [x] Helpful error messages with remediation steps
   - [x] Works on all target platforms

5. **Code Quality**
   - [x] Code passes `go vet`
   - [x] Code passes `golint` or `staticcheck`
   - [x] Unit tests for core functions
   - [x] No hardcoded paths or platform-specific assumptions

---

## File Summary

| File | Purpose |
|------|---------|
| `cmd/gwt/main.go` | Entry point, initializes and runs CLI |
| `internal/cli/root.go` | Root command, global flags |
| `internal/cli/doctor.go` | Doctor command implementation |
| `internal/cli/errors.go` | Error types and handling |
| `internal/output/output.go` | Output utility functions |
| `internal/git/git.go` | Git command wrapper (minimal) |
| `internal/version/version.go` | Version information |

---

## Notes

- This phase intentionally keeps the git wrapper minimal - just enough for `doctor`
- TUI is not implemented in this phase (placeholder flag only)
- Configuration file loading comes in Phase 3
- The `--config` flag is defined but non-functional until Phase 3
