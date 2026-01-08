# gwt - Git Worktree Manager

A powerful CLI tool for managing Git worktrees, making it easy to work with multiple branches simultaneously.

## Features

### Current

**Phase 1: Foundation**
- **System diagnostics** - `gwt doctor` validates prerequisites
- **Cross-platform support** - Windows, macOS, and Linux
- **Flexible output** - Verbose and quiet modes with color support

**Phase 2: Git Operations Core**
- **Complete git abstraction layer** - Robust wrapper for all git operations
- **Worktree management** - List, add, remove, prune, lock/unlock worktrees
- **Branch operations** - Create, delete, rename, list local/remote branches
- **Repository validation** - Detect repo state, worktrees, bare repos
- **Status detection** - Clean/dirty state, staged/unstaged changes, ahead/behind tracking
- **Remote operations** - Fetch, push, upstream tracking
- **Comprehensive test coverage** - 34+ unit tests ensuring reliability

### Planned
- Configuration management (Phase 3)
- CLI commands for worktree creation (Phase 4)
- Interactive TUI for worktree selection
- Docker database synchronization
- Symlink support for shared databases
- Branch cleanup utilities

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/Andrewy-gh/gwt
cd gwt

# Build the binary
go build -o gwt ./cmd/gwt

# Optional: Install to PATH
go install ./cmd/gwt
```

### Prerequisites

- **Git 2.20+** - Required for worktree features
- **Go 1.21+** - For building from source
- **Docker** (optional) - For database synchronization features
- **Docker Compose** (optional) - For multi-container setups

## Quick Start

Check that your system meets all requirements:

```bash
gwt doctor
```

Expected output:
```
Checking prerequisites...

✓ Git installed (2.43.0)
✓ Git repository detected
✓ Not a bare repository
✓ Symlink permissions available
✓ Docker installed (24.0.7)
✓ Docker Compose available

All checks passed! gwt is ready to use.
```

## Usage

### Global Flags

All commands support these global flags:

| Flag | Short | Description |
|------|-------|-------------|
| `--verbose` | `-v` | Enable verbose output |
| `--quiet` | `-q` | Suppress non-essential output |
| `--config` | `-c` | Path to config file |
| `--no-tui` | | Disable TUI, use simple prompts |
| `--help` | `-h` | Show help |
| `--version` | | Show version |

### Commands

#### `gwt doctor`

Validates system prerequisites and environment setup.

```bash
# Basic check
gwt doctor

# Verbose output
gwt -v doctor

# Quiet mode (errors only)
gwt -q doctor
```

**Checks performed:**
- Git installation and version (minimum 2.20)
- Git repository detection
- Bare repository validation
- Symlink support (Windows)
- Docker availability (optional)
- Docker Compose availability (optional)

## Development

### Project Structure

```
gwt/
├── cmd/gwt/              # Application entry point
│   └── main.go
├── internal/
│   ├── cli/              # CLI commands and framework
│   │   ├── root.go       # Root command
│   │   ├── doctor.go     # Doctor command
│   │   └── errors.go     # Error handling
│   ├── config/           # Configuration management (Phase 3)
│   ├── git/              # Git operations core (Phase 2)
│   │   ├── exec.go       # Command execution wrapper
│   │   ├── errors.go     # Git-specific error types
│   │   ├── repo.go       # Repository validation
│   │   ├── worktree.go   # Worktree operations
│   │   ├── branch.go     # Branch operations
│   │   ├── status.go     # Status detection
│   │   ├── remote.go     # Remote operations
│   │   ├── git.go        # Git version & detection
│   │   ├── *_test.go     # Comprehensive test coverage
│   ├── output/           # Output utilities
│   │   ├── output.go
│   │   └── output_test.go
│   ├── testutil/         # Test utilities
│   │   └── git.go        # Git test helpers
│   └── version/          # Version information
│       ├── version.go
│       └── version_test.go
├── docs/                 # Documentation
│   ├── GWT_SPEC.md
│   ├── IMPLEMENTATION_PHASES.md
│   ├── PHASE_1_PLAN.md
│   ├── PHASE_2_PLAN.md
│   ├── DEVELOPMENT.md
│   └── CHANGELOG.md
├── go.mod
├── go.sum
└── README.md
```

### Building

```bash
# Build for current platform
go build -o gwt ./cmd/gwt

# Build with version information
go build -ldflags "-X github.com/Andrewy-gh/gwt/internal/version.Version=1.0.0 \
  -X github.com/Andrewy-gh/gwt/internal/version.Commit=$(git rev-parse --short HEAD) \
  -X github.com/Andrewy-gh/gwt/internal/version.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o gwt ./cmd/gwt

# Cross-compile for different platforms
GOOS=linux GOARCH=amd64 go build -o gwt-linux ./cmd/gwt
GOOS=darwin GOARCH=amd64 go build -o gwt-darwin ./cmd/gwt
GOOS=windows GOARCH=amd64 go build -o gwt.exe ./cmd/gwt
```

### Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with verbose output
go test -v ./...

# Run specific package tests
go test ./internal/git
```

### Code Quality

```bash
# Run go vet
go vet ./...

# Format code
go fmt ./...

# Tidy dependencies
go mod tidy
```

## Implementation Phases

This project follows a phased implementation approach:

- **Phase 1** (✓ Complete) - Foundation, CLI framework, `gwt doctor`
- **Phase 2** (✓ Complete) - Git operations core (worktree, branch, status, remote)
- **Phase 3** (Planned) - Configuration management
- **Phase 4** (Planned) - Create worktree CLI command
- **Phase 5** (Planned) - List & delete worktree CLI commands
- **Phase 6+** (Planned) - File copying, Docker, dependencies, TUI

See [docs/IMPLEMENTATION_PHASES.md](docs/IMPLEMENTATION_PHASES.md) for details.

## Contributing

Contributions are welcome! Please ensure:

1. Code passes `go vet` and `go test`
2. New features include tests
3. Documentation is updated
4. Commit messages are clear and concise

## License

This project is open source. License TBD.

## Acknowledgments

Built with:
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Viper](https://github.com/spf13/viper) - Configuration (coming in Phase 3)
