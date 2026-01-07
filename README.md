# gwt - Git Worktree Manager

A powerful CLI tool for managing Git worktrees, making it easy to work with multiple branches simultaneously.

## Features

### Current (Phase 1)
- **System diagnostics** - `gwt doctor` validates prerequisites
- **Cross-platform support** - Windows, macOS, and Linux
- **Flexible output** - Verbose and quiet modes with color support
- **Git integration** - Smart detection of repository state

### Planned
- Interactive TUI for worktree selection
- Create and manage worktrees with ease
- Docker database synchronization
- Symlink support for shared databases
- Branch cleanup utilities
- Stash management across worktrees

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
│   ├── config/           # Configuration management (future)
│   ├── git/              # Git operations wrapper
│   │   ├── git.go
│   │   └── git_test.go
│   ├── output/           # Output utilities
│   │   ├── output.go
│   │   └── output_test.go
│   └── version/          # Version information
│       ├── version.go
│       └── version_test.go
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
- **Phase 2** (Planned) - Configuration management
- **Phase 3** (Planned) - Core worktree operations
- **Phase 4** (Planned) - TUI implementation
- **Phase 5+** (Planned) - Advanced features

See [IMPLEMENTATION_PHASES.md](IMPLEMENTATION_PHASES.md) for details.

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
