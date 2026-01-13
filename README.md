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

**Phase 4: Create Worktree (CLI)**
- **Create command** - `gwt create` with comprehensive flag support
- **Branch validation** - Git branch name validation and directory name conversion
- **Multiple branch sources** - New branch from HEAD/ref, existing local, or remote branch
- **Directory collision detection** - Smart handling of existing directories
- **Rollback on failure** - Automatic cleanup of partial worktree creation
- **Operation locking** - Prevent concurrent operations with stale lock detection
- **Safe defaults** - Worktrees placed as siblings to main worktree

**Phase 5: List & Delete Worktrees (CLI)**
- **List command** - `gwt list` with table, JSON, and simple output formats
- **Status command** - `gwt status` for detailed worktree information
- **Delete command** - `gwt delete` with comprehensive safety checks
- **Batch operations** - Delete multiple worktrees with confirmation
- **Main worktree protection** - Prevents accidental deletion of main worktree

**Phase 6: File Copying**
- **Gitignored file discovery** - Automatic detection via `git status --ignored`
- **Pattern matching** - Apply `copy_defaults` and `copy_exclude` from config
- **Smart exclusions** - Auto-exclude dependency directories (node_modules, vendor, etc.)
- **File size display** - Show sizes for informed selection
- **Progress tracking** - Real-time progress during copy operations
- **Error handling** - Collect and report errors without stopping entire operation

**Phase 7: Docker Compose Scaffolding**
- **Auto-detection** - Finds docker-compose.yml and override files automatically
- **Shared mode** - Symlinks data directories to main worktree (default)
- **New mode** - Isolated containers with renamed volumes and remapped ports
- **Windows fallback** - Automatic fallback chain: symlink → junction → copy
- **Helper scripts** - Generates `dc` wrapper script (bash/PowerShell/CMD)
- **Port conflict detection** - Warns about common port conflicts with suggestions
- **Override generation** - Creates `docker-compose.worktree.yml` with branch suffix
- **Config integration** - Supports `docker` section in `.worktree.yaml`

**Phase 8: Dependency Installation**
- **Package manager detection** - Supports npm, yarn, pnpm, bun, go, cargo, pip, poetry
- **Monorepo support** - Configure multiple paths with glob patterns
- **Lock file priority** - Prefers specific lock files (bun > pnpm > yarn > npm)
- **Streaming output** - Real-time installation progress in verbose mode
- **Timeout protection** - 5-minute default timeout prevents hanging
- **Non-fatal errors** - Worktree creation succeeds even if install fails
- **Skip flag** - `--skip-install` to bypass dependency installation
- **Auto-install** - Runs automatically after worktree creation (configurable)

### Planned
- Interactive TUI for worktree selection
- Database migration running
- Post-creation hooks
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
- **docker**: Docker Compose settings
  - `compose_files`: Specific compose files to use (auto-detected if not set)
  - `data_directories`: Directories to symlink/copy (auto-detected if not set)
  - `default_mode`: Default Docker mode (`shared` or `new`)
  - `port_offset`: Port offset for new mode (default: 1)
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
  compose_files:
    - "docker-compose.yml"
    - "docker-compose.dev.yml"
  data_directories:
    - "server/db-data"
    - "data/redis"
  default_mode: "shared"
  port_offset: 1

dependencies:
  auto_install: true
  paths:
    - "."              # Root package
    - "client"         # Frontend
    - "packages/*"     # Monorepo packages (glob pattern)
```

#### `gwt create`

Create a new worktree from a new or existing branch.

```bash
# Create worktree with new branch from HEAD
gwt create -b feature-auth

# Create from specific ref (branch, tag, or commit)
gwt create -b feature-auth --from main
gwt create -b hotfix --from v1.2.0

# Use existing local branch
gwt create --checkout existing-branch

# Checkout remote branch (creates local tracking branch)
gwt create --remote origin/feature-x

# Override directory name
gwt create -b feature-auth --directory custom-name

# Force creation (skip some validations)
gwt create -b feature-auth --force
```

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--branch` | `-b` | New branch name |
| `--from` | | Starting point for new branch (default: HEAD) |
| `--checkout` | | Existing local branch to checkout |
| `--remote` | | Remote branch to checkout (creates tracking branch) |
| `--directory` | `-d` | Override target directory name |
| `--force` | `-f` | Force creation even with warnings |
| `--skip-install` | | Skip dependency installation |
| `--skip-migrations` | | Skip running migrations |
| `--copy-config` | | Copy `.worktree.yaml` to new worktree |
| `--skip-copy` | | Skip copying gitignored files |
| `--docker-mode` | | Docker setup mode: `shared`, `new`, or `skip` |
| `--skip-docker` | | Skip Docker Compose setup |

**Branch Source Flags:**
- `--branch`, `--checkout`, and `--remote` are mutually exclusive
- `--from` only valid with `--branch`
- At least one branch source must be specified

**Behavior:**
- Worktrees are created as siblings to the main worktree: `../project-branch-name`
- Branch names with slashes are converted: `feature/auth/login` → `project-feature-auth-login`
- Directory collision detection with helpful error messages
- Automatic rollback on failure (cleans up partial worktree)
- Operation locking prevents concurrent `gwt create` commands

**Examples:**

```bash
# Main worktree at: /home/user/myapp
# Creates worktree at: /home/user/myapp-feature-auth
gwt create -b feature-auth

# With custom directory
# Creates worktree at: /home/user/my-custom-dir
gwt create -b feature-auth --directory my-custom-dir

# Docker: Shared mode (containers share data with main worktree)
gwt create -b feature-auth --docker-mode shared
# Run: docker compose up

# Docker: New mode (isolated containers)
gwt create -b feature-auth --docker-mode new
# Creates docker-compose.worktree.yml with renamed volumes and remapped ports
# Generates ./dc helper script
# Run: ./dc up

# Skip Docker setup entirely
gwt create -b feature-auth --skip-docker

# Dependency Installation: Auto-detected and installed by default
gwt create -b feature-auth
# Detects: npm/yarn/pnpm/bun (JS), go (Go), cargo (Rust), pip/poetry (Python)

# Skip dependency installation
gwt create -b feature-auth --skip-install

# Monorepo: Configure paths in .worktree.yaml
# dependencies:
#   paths:
#     - "."
#     - "apps/*"
#     - "packages/*"
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
│   │   ├── create.go     # Create command
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
│   ├── copy/             # File copying (Phase 6)
│   │   ├── discover.go   # Git ignored file discovery
│   │   ├── match.go      # Pattern matching logic
│   │   ├── copy.go       # File/directory copying
│   │   ├── selection.go  # File selection with sizes
│   │   ├── errors.go     # Custom error types
│   │   └── *_test.go     # Copy tests
│   ├── create/           # Worktree creation (Phase 4)
│   │   ├── validate.go   # Branch name validation
│   │   ├── branch.go     # Branch source handling
│   │   ├── directory.go  # Directory collision detection
│   │   ├── worktree.go   # Worktree creation orchestration
│   │   ├── rollback.go   # Rollback and cleanup
│   │   ├── lock.go       # Operation locking
│   │   └── *_test.go     # Create tests
│   ├── docker/           # Docker Compose scaffolding (Phase 7)
│   │   ├── detect.go     # Compose file auto-detection
│   │   ├── parse.go      # Compose file parsing
│   │   ├── symlink.go    # Cross-platform symlink utilities
│   │   ├── shared.go     # Shared mode implementation
│   │   ├── new.go        # New mode implementation
│   │   ├── override.go   # Override file generation
│   │   ├── helper.go     # Helper script generation
│   │   ├── ports.go      # Port conflict detection
│   │   ├── errors.go     # Custom error types
│   │   └── *_test.go     # Docker tests
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
│   ├── install/          # Dependency installation (Phase 8)
│   │   ├── detect.go     # Package manager detection
│   │   ├── install.go    # Installation orchestrator
│   │   ├── manager.go    # Package manager types
│   │   ├── result.go     # Result types
│   │   └── *_test.go     # Installation tests
│   ├── output/           # Output utilities
│   │   ├── output.go
│   │   ├── progress.go   # Progress bar display
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
│   ├── PHASE_2_SUMMARY.md
│   ├── PHASE_3_PLAN.md
│   ├── PHASE_4_PLAN.md
│   ├── PHASE_4_SUMMARY.md
│   ├── PHASE_5_PLAN.md
│   ├── PHASE_5_SUMMARY.md
│   ├── PHASE_6_PLAN.md
│   ├── PHASE_6_SUMMARY.md
│   ├── PHASE_7_PLAN.md
│   ├── PHASE_7_SUMMARY.md
│   ├── PHASE_8_PLAN.md
│   ├── PHASE_8_SUMMARY.md
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
- **Phase 4** (✓ Complete) - Create worktree CLI command with validation and rollback
- **Phase 5** (✓ Complete) - List & delete worktree CLI commands
- **Phase 6** (✓ Complete) - File copying with pattern matching and progress tracking
- **Phase 7** (✓ Complete) - Docker Compose scaffolding with shared/new modes
- **Phase 8** (✓ Complete) - Dependency installation with package manager detection
- **Phase 9+** (Planned) - Migrations, hooks, TUI

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
