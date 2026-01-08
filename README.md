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

**Phase 3: Configuration System**
- **Config file support** - `.worktree.yaml` with YAML format
- **Config commands** - View, initialize, and manage configuration
- **Config inheritance** - Linked worktrees inherit from main worktree
- **Validation** - Comprehensive validation for all config fields
- **Default values** - Sensible defaults for all settings
- **Global --config flag** - Override config file path

### Planned
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

#### `gwt config`

View and manage gwt configuration.

```bash
# View current configuration
gwt config
# or
gwt config show

# Show path to config file
gwt config path

# Create default config file
gwt config init

# Overwrite existing config file
gwt config init --force

# Create config in specific location
gwt config init --output /path/to/.worktree.yaml

# Use custom config file
gwt --config /path/to/config.yaml config
```

**Configuration File (`.worktree.yaml`):**

The config file controls gwt behavior with the following sections:

- **copy_defaults**: Files/patterns to pre-select for copying (supports glob patterns)
- **copy_exclude**: Patterns to never select by default
- **docker**: Docker Compose settings (compose files, data directories, mode, port offset)
- **dependencies**: Dependency installation settings (auto-install, paths)
- **migrations**: Database migration settings (auto-detect, custom command)
- **hooks**: Lifecycle hooks (post_create, post_delete)

**Config File Search Order:**
1. Explicit path via `--config` flag
2. `.worktree.yaml` in current directory
3. `.worktree.yaml` in repository root
4. `.worktree.yaml` in main worktree (for linked worktrees)
5. Default values if no config file exists

**Example:**
```yaml
copy_defaults:
  - ".env"
  - "**/.env.local"

docker:
  default_mode: "shared"
  port_offset: 1

dependencies:
  auto_install: true
  paths:
    - "."
    - "client"
```

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
│   │   ├── config.go     # Config command
│   │   └── errors.go     # Error handling
│   ├── config/           # Configuration management (Phase 3)
│   │   ├── config.go     # Config structs
│   │   ├── defaults.go   # Default values
│   │   ├── load.go       # Config loading with Viper
│   │   ├── validate.go   # Config validation
│   │   ├── errors.go     # Config error types
│   │   ├── template.go   # Config file template
│   │   ├── *_test.go     # Config tests
│   │   └── testdata/     # Test fixtures
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
│   ├── PHASE_3_PLAN.md
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
- **Phase 3** (✓ Complete) - Configuration management, `gwt config` commands
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
- [Viper](https://github.com/spf13/viper) - Configuration management
