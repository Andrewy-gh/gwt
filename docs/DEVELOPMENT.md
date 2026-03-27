# Development Guide

This guide covers development practices, architecture decisions, and implementation details for gwt.

## Getting Started

### Prerequisites

- Go 1.21 or later
- Git 2.20 or later
- Docker (optional, for testing Docker-related features)

### Initial Setup

```bash
# Clone the repository
git clone https://github.com/Andrewy-gh/gwt
cd gwt

# Install dependencies
go mod download

# Build the project
go build -o gwt ./cmd/gwt

# Run tests
go test ./...

# Run the doctor command
./gwt doctor
```

## Architecture

### Design Principles

1. **Separation of concerns** - CLI logic separated from business logic
2. **Testability** - All core functions are unit testable
3. **Cross-platform** - No platform-specific code except where necessary
4. **User-friendly errors** - All errors include actionable guidance

### Package Structure

#### `cmd/gwt`
Application entry point. Minimal logic - just calls `cli.Execute()`.

#### `internal/cli`
CLI commands and framework integration:
- `root.go` - Root command, global flags, and flag management
- `doctor.go` - System diagnostics command
- `errors.go` - Custom error types and error handling utilities

#### `internal/git`
Git operations wrapper:
- Abstracts all `git` command execution
- Provides type-safe interfaces
- Handles version detection and compatibility

#### `internal/output`
Output utilities for consistent messaging:
- Color-coded output with symbols
- Respects `--verbose` and `--quiet` flags
- Automatic terminal detection for color support

#### `internal/version`
Version information management:
- Version string formatting
- Build-time variable injection via ldflags

## Code Standards

### Error Handling

Always provide context and actionable guidance:

```go
// Bad
return fmt.Errorf("git not found")

// Good
return cli.GitNotFoundError()
```

Use custom error types with exit codes:

```go
return cli.NewExitError(err, cli.ExitGitNotFound, "Git is required")
```

### Output Functions

Use the output package for all user-facing messages:

```go
// Success messages
output.Success("Git repository detected")

// Warnings (don't stop execution)
output.Warning("Git version may have issues")

// Errors (critical issues)
output.Error("Not a git repository")

// Informational messages (suppressed in quiet mode)
output.Info("Checking prerequisites...")

// Debug/verbose messages (only shown with --verbose)
output.Verbose(fmt.Sprintf("Git version: %s", version))
```

### Testing

#### Unit Tests

Every package should have tests covering core functionality:

```go
func TestVersionAtLeast(t *testing.T) {
    v := git.Version{Major: 2, Minor: 43, Patch: 0}

    if !v.AtLeast(2, 20) {
        t.Errorf("Expected version to be at least 2.20")
    }
}
```

#### Table-Driven Tests

Use table-driven tests for multiple scenarios:

```go
func TestVersionComparison(t *testing.T) {
    tests := []struct {
        version  git.Version
        major    int
        minor    int
        expected bool
    }{
        {git.Version{2, 43, 0}, 2, 20, true},
        {git.Version{2, 19, 0}, 2, 20, false},
    }

    for _, tt := range tests {
        result := tt.version.AtLeast(tt.major, tt.minor)
        if result != tt.expected {
            t.Errorf("Version %s.AtLeast(%d, %d) = %v, want %v",
                tt.version, tt.major, tt.minor, result, tt.expected)
        }
    }
}
```

#### Test Coverage

Aim for:
- 80%+ coverage for business logic
- 100% coverage for critical paths (error handling, version checks)
- Integration tests for commands

## Building and Releasing

### Development Build

```bash
go build -o gwt ./cmd/gwt
```

### Production Build

```bash
# Build with version information
VERSION=1.0.0
COMMIT=$(git rev-parse --short HEAD)
BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)

go build \
  -ldflags "-X github.com/Andrewy-gh/gwt/internal/version.Version=${VERSION} \
            -X github.com/Andrewy-gh/gwt/internal/version.Commit=${COMMIT} \
            -X github.com/Andrewy-gh/gwt/internal/version.BuildDate=${BUILD_DATE}" \
  -o gwt \
  ./cmd/gwt
```

### Cross-Platform Builds

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o dist/gwt-linux-amd64 ./cmd/gwt

# macOS (Intel)
GOOS=darwin GOARCH=amd64 go build -o dist/gwt-darwin-amd64 ./cmd/gwt

# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o dist/gwt-darwin-arm64 ./cmd/gwt

# Windows
GOOS=windows GOARCH=amd64 go build -o dist/gwt-windows-amd64.exe ./cmd/gwt
```

## Debugging

### Verbose Mode

Enable verbose output to see detailed execution:

```bash
gwt -v doctor
```

### Go Debugging

Use Delve for debugging:

```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug the application
dlv debug ./cmd/gwt -- doctor
```

## Common Tasks

### Adding a New Command

1. Create a new file in `internal/cli/`:

```go
package cli

import (
    "github.com/spf13/cobra"
    "github.com/Andrewy-gh/gwt/internal/output"
)

var myCmd = &cobra.Command{
    Use:   "mycommand",
    Short: "Brief description",
    Long:  `Detailed description`,
    RunE:  runMyCommand,
}

func init() {
    rootCmd.AddCommand(myCmd)
    // Add command-specific flags here
}

func runMyCommand(cmd *cobra.Command, args []string) error {
    output.Info("Running my command...")
    return nil
}
```

2. Add tests in `internal/cli/mycommand_test.go`
3. Update documentation

### Adding a New Flag

Add to `internal/cli/root.go`:

```go
var myFlag string

func init() {
    // For command-specific flag
    myCmd.Flags().StringVar(&myFlag, "myflag", "", "description")

    // For global flag
    rootCmd.PersistentFlags().StringVar(&myFlag, "myflag", "", "description")
}
```

### Adding Git Operations

Add to `internal/git/git.go`:

```go
// MyOperation performs a git operation
func MyOperation() (string, error) {
    cmd := exec.Command("git", "my-args")
    output, err := cmd.Output()
    if err != nil {
        return "", fmt.Errorf("failed to perform operation: %w", err)
    }
    return strings.TrimSpace(string(output)), nil
}
```

Add tests in `internal/git/git_test.go`.

## Troubleshooting

### Build Issues

**Problem:** Import cycle errors

**Solution:** Ensure packages don't circularly depend on each other. The dependency flow should be:
- `cmd/gwt` → `internal/cli`
- `internal/cli` → `internal/output`, `internal/git`
- `internal/output` → (no internal dependencies)

### Test Issues

**Problem:** Tests fail on Windows with path separators

**Solution:** Use `filepath.Join()` instead of hardcoded paths:

```go
// Bad
path := "dir/file.txt"

// Good
path := filepath.Join("dir", "file.txt")
```

### Platform-Specific Code

Use build tags when necessary:

```go
//go:build windows
// +build windows

package mypackage

// Windows-specific implementation
```

## Resources

- [Cobra Documentation](https://github.com/spf13/cobra)
- [Go Testing](https://go.dev/doc/tutorial/add-a-test)
- [Effective Go](https://go.dev/doc/effective_go)
- [GWT Specification](GWT_SPEC.md)
