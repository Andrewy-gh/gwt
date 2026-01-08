# Phase 3: Configuration System - Implementation Plan

## Overview

Phase 3 implements the configuration system that allows users to customize gwt behavior. The configuration file (`.worktree.yaml`) controls file copying defaults, Docker settings, dependency installation, migrations, and hooks.

**Deliverables:**
- Configuration struct matching the spec
- Config loading with Viper (YAML support)
- Config inheritance from main worktree
- `gwt config` command (view configuration)
- `gwt config init` command (create default config file)
- Default values for all config options

**Prerequisites:**
- Phase 1 completed (CLI framework, output utilities, error handling)
- Phase 2 completed (Git operations core, repository state validation)

---

## Tasks

### 1. Define Configuration Struct

**Objective:** Create Go structs that represent the full `.worktree.yaml` configuration file.

**Files to Create:**
- `internal/config/config.go` - Configuration struct definitions

**Configuration Struct:**

```go
// Config represents the root configuration from .worktree.yaml
type Config struct {
    CopyDefaults []string          `mapstructure:"copy_defaults" yaml:"copy_defaults"`
    CopyExclude  []string          `mapstructure:"copy_exclude" yaml:"copy_exclude"`
    Docker       DockerConfig      `mapstructure:"docker" yaml:"docker"`
    Dependencies DependenciesConfig `mapstructure:"dependencies" yaml:"dependencies"`
    Migrations   MigrationsConfig  `mapstructure:"migrations" yaml:"migrations"`
    Hooks        HooksConfig       `mapstructure:"hooks" yaml:"hooks"`
}

// DockerConfig contains Docker/Compose-related settings
type DockerConfig struct {
    ComposeFiles    []string `mapstructure:"compose_files" yaml:"compose_files"`
    DataDirectories []string `mapstructure:"data_directories" yaml:"data_directories"`
    DefaultMode     string   `mapstructure:"default_mode" yaml:"default_mode"` // "shared" or "new"
    PortOffset      int      `mapstructure:"port_offset" yaml:"port_offset"`
}

// DependenciesConfig controls dependency installation behavior
type DependenciesConfig struct {
    AutoInstall bool     `mapstructure:"auto_install" yaml:"auto_install"`
    Paths       []string `mapstructure:"paths" yaml:"paths"`
}

// MigrationsConfig controls database migration behavior
type MigrationsConfig struct {
    AutoDetect bool   `mapstructure:"auto_detect" yaml:"auto_detect"`
    Command    string `mapstructure:"command" yaml:"command,omitempty"`
}

// HooksConfig contains lifecycle hooks
type HooksConfig struct {
    PostCreate []string `mapstructure:"post_create" yaml:"post_create"`
    PostDelete []string `mapstructure:"post_delete" yaml:"post_delete"`
}
```

**Implementation Notes:**
- Use both `mapstructure` tags (for Viper) and `yaml` tags (for direct YAML operations)
- Use `omitempty` for optional fields
- Ensure all fields have sensible zero values

**Acceptance Criteria:**
- [ ] All config fields from spec are represented
- [ ] Struct tags are correct for Viper and YAML
- [ ] Nested structs properly defined
- [ ] Config can be serialized to/from YAML

---

### 2. Implement Default Configuration Values

**Objective:** Define sensible defaults that match the spec and common use cases.

**Files to Create/Modify:**
- `internal/config/defaults.go` - Default configuration values

**Default Values:**

```go
// DefaultConfig returns a new Config with default values
func DefaultConfig() *Config {
    return &Config{
        CopyDefaults: []string{
            ".env",
            "**/.env",
            "**/.env.local",
            ".claude/",
            "**/*.local.md",
            "**/setenv.sh",
        },
        CopyExclude: []string{
            "node_modules",
            "vendor",
            ".venv",
            "__pycache__",
            "target",
            "dist",
            "build",
            "*.log",
        },
        Docker: DockerConfig{
            ComposeFiles:    []string{}, // Auto-detect if empty
            DataDirectories: []string{},
            DefaultMode:     "shared",
            PortOffset:      1,
        },
        Dependencies: DependenciesConfig{
            AutoInstall: true,
            Paths: []string{
                ".",
            },
        },
        Migrations: MigrationsConfig{
            AutoDetect: true,
            Command:    "", // Auto-detect if empty
        },
        Hooks: HooksConfig{
            PostCreate: []string{},
            PostDelete: []string{},
        },
    }
}
```

**Implementation Notes:**
- Defaults should be production-ready, not just examples
- Empty slices mean "use auto-detection" where applicable
- Port offset of 1 prevents conflicts (5432 -> 5433)

**Acceptance Criteria:**
- [ ] DefaultConfig() returns fully populated config
- [ ] All defaults match the spec
- [ ] Defaults are sensible for common use cases
- [ ] Auto-detection triggered when lists are empty

---

### 3. Implement Config Loading with Viper

**Objective:** Load configuration from `.worktree.yaml` using Viper with proper precedence and error handling.

**Files to Create:**
- `internal/config/load.go` - Configuration loading logic
- `internal/config/errors.go` - Config-specific error types

**Functions to Implement:**

```go
// Load reads configuration from the specified path
// If path is empty, searches in the current directory and ancestors
func Load(path string) (*Config, error)

// LoadFromDir loads configuration starting from the given directory
func LoadFromDir(dir string) (*Config, error)

// FindConfigFile searches for .worktree.yaml starting from dir
// Returns the path to the config file or empty string if not found
func FindConfigFile(dir string) (string, error)

// ConfigExists checks if a config file exists in the given directory
func ConfigExists(dir string) bool

// GetConfigPath returns the path where config would be loaded from
// Returns empty string if no config file exists
func GetConfigPath(dir string) (string, error)
```

**Config File Search Order:**
1. Explicit path provided via `--config` flag
2. `.worktree.yaml` in current directory
3. `.worktree.yaml` in repository root (detected via git)
4. `.worktree.yaml` in main worktree (for linked worktrees)

**Error Types:**

```go
// ConfigNotFoundError indicates no config file was found
type ConfigNotFoundError struct {
    SearchPath string
}

// ConfigParseError indicates the config file is invalid
type ConfigParseError struct {
    Path string
    Err  error
}

// ConfigValidationError indicates the config has invalid values
type ConfigValidationError struct {
    Field   string
    Value   interface{}
    Message string
}
```

**Implementation Notes:**
- Use Viper for loading and merging
- Support YAML format only (simplify maintenance)
- Merge loaded config with defaults (loaded values override defaults)
- Return defaults if no config file found (not an error)
- Validate config values after loading

**Acceptance Criteria:**
- [ ] Loads config from explicit path
- [ ] Searches current directory for config
- [ ] Searches repository root for config
- [ ] Merges with defaults correctly
- [ ] Returns defaults when no config exists
- [ ] Parse errors include file path and line number
- [ ] Handles malformed YAML gracefully

---

### 4. Implement Config Inheritance (Main Worktree)

**Objective:** Linked worktrees should read configuration from the main worktree by default.

**Files to Modify:**
- `internal/config/load.go` - Add inheritance logic

**Functions to Implement:**

```go
// LoadWithInheritance loads config with worktree inheritance
// If in a linked worktree, reads from main worktree first
func LoadWithInheritance(dir string) (*Config, string, error)

// GetEffectiveConfigPath returns the path where config will be loaded from
// accounting for worktree inheritance
func GetEffectiveConfigPath(dir string) (string, error)

// IsInheritedConfig checks if the loaded config came from main worktree
func IsInheritedConfig(dir string) (bool, error)
```

**Inheritance Logic:**

```
1. Check if currently in a linked worktree (git rev-parse --git-common-dir)
2. If in linked worktree:
   a. Check for .worktree.yaml in current worktree
   b. If not found, look in main worktree path
3. If in main worktree:
   a. Look for .worktree.yaml in repository root
4. Fall back to defaults if no config found anywhere
```

**Implementation Notes:**
- Use Phase 2's `git.GetMainWorktreePath()` function
- Config in linked worktree overrides main worktree config (no merging)
- Track which path the config was loaded from for display purposes
- Consider caching the loaded config for performance

**Acceptance Criteria:**
- [ ] Main worktree loads config from its own root
- [ ] Linked worktree inherits from main worktree
- [ ] Local config in linked worktree overrides inheritance
- [ ] Returns correct source path for display
- [ ] Works when run from subdirectory of worktree

---

### 5. Implement Config Validation

**Objective:** Validate configuration values to catch errors early.

**Files to Create:**
- `internal/config/validate.go` - Configuration validation

**Functions to Implement:**

```go
// Validate checks the config for invalid values
// Returns a slice of validation errors (empty if valid)
func (c *Config) Validate() []ConfigValidationError

// ValidateDockerMode checks if the docker mode is valid
func ValidateDockerMode(mode string) error

// ValidateGlobPatterns checks if glob patterns are valid
func ValidateGlobPatterns(patterns []string) []error

// ValidatePaths checks if paths are reasonable (not absolute, etc.)
func ValidatePaths(paths []string) []error
```

**Validation Rules:**

| Field | Rule |
|-------|------|
| `docker.default_mode` | Must be "shared" or "new" |
| `docker.port_offset` | Must be >= 0 and < 65535 |
| `copy_defaults` | Valid glob patterns |
| `copy_exclude` | Valid glob patterns |
| `dependencies.paths` | Relative paths only (warn for absolute) |
| `hooks.post_create` | Non-empty strings |
| `hooks.post_delete` | Non-empty strings |

**Implementation Notes:**
- Collect all errors, don't stop at first
- Distinguish warnings from errors
- Use descriptive error messages with field names

**Acceptance Criteria:**
- [ ] Validates docker.default_mode
- [ ] Validates port_offset range
- [ ] Validates glob patterns
- [ ] Warns about absolute paths
- [ ] Returns all validation errors at once

---

### 6. Implement `gwt config` Command

**Objective:** Allow users to view the current effective configuration.

**Files to Create:**
- `internal/cli/config.go` - Config command implementation

**Command Structure:**

```
gwt config           # Display current configuration
gwt config init      # Create default config file
gwt config path      # Show path to config file
```

**Subcommands:**

```go
// configCmd shows the current configuration
var configCmd = &cobra.Command{
    Use:   "config",
    Short: "View or manage configuration",
    Long:  "Display the current configuration or manage .worktree.yaml",
}

// configShowCmd (default) displays the current config
var configShowCmd = &cobra.Command{
    Use:   "show",
    Short: "Display current configuration",
}

// configInitCmd creates a default config file
var configInitCmd = &cobra.Command{
    Use:   "init",
    Short: "Create a default .worktree.yaml file",
}

// configPathCmd shows where config is loaded from
var configPathCmd = &cobra.Command{
    Use:   "path",
    Short: "Show config file path",
}
```

**Output for `gwt config`:**

```
Configuration source: /projects/myapp/.worktree.yaml

copy_defaults:
  - .env
  - **/.env
  - **/.env.local
  - .claude/

copy_exclude:
  - node_modules
  - vendor
  - .venv

docker:
  default_mode: shared
  port_offset: 1

dependencies:
  auto_install: true
  paths:
    - .
    - client
    - server

migrations:
  auto_detect: true

hooks:
  post_create:
    - echo 'Worktree ready!'
```

**Output for `gwt config path`:**

```
/projects/myapp/.worktree.yaml
```

Or if inherited:

```
/projects/myapp/.worktree.yaml (inherited from main worktree)
```

Or if using defaults:

```
No configuration file found. Using defaults.
```

**Implementation Notes:**
- `gwt config` with no subcommand should show config (like `gwt config show`)
- Use YAML format for output (familiar and copy-pasteable)
- Show source path prominently
- Indicate when using defaults vs file

**Acceptance Criteria:**
- [ ] `gwt config` displays current config in YAML
- [ ] Shows config source path
- [ ] Indicates if inherited from main worktree
- [ ] Indicates if using defaults
- [ ] `gwt config path` shows just the path
- [ ] Works with `--config` flag override

---

### 7. Implement `gwt config init` Command

**Objective:** Create a default `.worktree.yaml` file in the repository root.

**Files to Modify:**
- `internal/cli/config.go` - Add init subcommand

**Functionality:**

```go
// initConfig creates a new .worktree.yaml with defaults and comments
func initConfig(cmd *cobra.Command, args []string) error
```

**Generated File Template:**

```yaml
# gwt configuration file
# See: https://github.com/Andrewy-gh/gwt for documentation

# Files and directories to pre-select for copying
# Supports glob patterns
copy_defaults:
  - ".env"
  - "**/.env"
  - "**/.env.local"
  - ".claude/"
  - "**/*.local.md"
  - "**/setenv.sh"

# Patterns to never select by default (even if gitignored)
copy_exclude:
  - "node_modules"
  - "vendor"
  - ".venv"
  - "__pycache__"
  - "target"
  - "dist"
  - "build"
  - "*.log"

# Docker configuration
docker:
  # Compose files to consider (auto-detected if not specified)
  compose_files: []

  # Data directories that should be symlinked (shared) or copied (new)
  data_directories: []

  # Default mode: "shared" or "new"
  default_mode: "shared"

  # Port offset for new containers (e.g., 5432 -> 5433)
  port_offset: 1

# Dependency installation settings
dependencies:
  # Auto-detect and install (default: true)
  auto_install: true

  # Directories to check for package managers
  paths:
    - "."

# Migration settings
migrations:
  # Auto-detect and offer to run (default: true)
  auto_detect: true

  # Custom migration command (overrides auto-detection)
  # command: "make migrate-up"

# Post-setup hooks (run after everything else)
# Hooks run in the new worktree directory with GWT_* environment variables
hooks:
  post_create: []
  post_delete: []
```

**Flags:**

```go
--force, -f    # Overwrite existing config file
--output, -o   # Output path (default: .worktree.yaml in repo root)
```

**Implementation Notes:**
- Write to repository root by default (use `git.GetRepoRoot()`)
- Include helpful comments in generated file
- Refuse to overwrite existing file without `--force`
- Show success message with path

**Output Messages:**

Success:
```
Created .worktree.yaml in /projects/myapp

Edit this file to customize gwt behavior for this repository.
```

Already exists:
```
Configuration file already exists: /projects/myapp/.worktree.yaml

Use --force to overwrite.
```

**Acceptance Criteria:**
- [ ] Creates .worktree.yaml in repository root
- [ ] Generated file includes comments
- [ ] Generated file is valid YAML
- [ ] Refuses to overwrite without --force
- [ ] --force overwrites existing file
- [ ] Shows success message with path

---

### 8. Add Config Flag to Root Command

**Objective:** Allow users to specify an alternate config file path.

**Files to Modify:**
- `internal/cli/root.go` - Add --config flag

**Implementation:**

```go
var cfgFile string

func init() {
    rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "",
        "config file path (default: .worktree.yaml in repo root)")
}

// GetConfig returns the loaded configuration
// Called by subcommands that need config
func GetConfig() (*config.Config, error) {
    if cfgFile != "" {
        return config.Load(cfgFile)
    }
    return config.LoadWithInheritance(".")
}
```

**Implementation Notes:**
- Make `--config` a persistent flag (available to all subcommands)
- Store in package variable for access by subcommands
- Override inheritance when explicit path provided

**Acceptance Criteria:**
- [ ] `--config` flag available on all commands
- [ ] Explicit path overrides auto-detection
- [ ] Error if specified path doesn't exist
- [ ] Works with both relative and absolute paths

---

## Implementation Order

```
1. Config struct definition      ─┐
2. Default config values         ─┴─▶ Foundation for config system

3. Config loading (Viper)        ─┐
4. Config validation             ─┴─▶ Core loading functionality

5. Config inheritance            ────▶ Worktree-aware loading

6. gwt config command            ─┐
7. gwt config init command       ─┼─▶ CLI commands
8. Root --config flag            ─┘
```

---

## Dependencies

**New dependencies required:**

```
github.com/spf13/viper v1.18.2    # Configuration management
gopkg.in/yaml.v3 v3.0.1           # YAML parsing (may come with viper)
```

**Installation:**

```bash
go get github.com/spf13/viper@v1.18.2
```

---

## Testing Strategy

### Unit Tests

- `internal/config/config_test.go` - Struct marshaling tests
- `internal/config/defaults_test.go` - Default values tests
- `internal/config/load_test.go` - Loading and merging tests
- `internal/config/validate_test.go` - Validation tests

### Test Fixtures

Create test config files in `internal/config/testdata/`:

```
testdata/
├── valid/
│   ├── minimal.yaml         # Only required fields
│   ├── full.yaml            # All fields populated
│   └── with_comments.yaml   # Comments preserved
├── invalid/
│   ├── bad_yaml.yaml        # Malformed YAML
│   ├── bad_mode.yaml        # Invalid docker mode
│   └── bad_port.yaml        # Invalid port offset
└── inheritance/
    ├── main/.worktree.yaml
    └── linked/.worktree.yaml
```

### Integration Tests

**Test Scenarios:**

- [ ] Load config from current directory
- [ ] Load config from repository root
- [ ] Load config from explicit path
- [ ] Config inheritance from main worktree
- [ ] Local config overrides inherited config
- [ ] Defaults used when no config exists
- [ ] Invalid YAML produces helpful error
- [ ] Validation catches bad values
- [ ] `gwt config` displays correctly
- [ ] `gwt config init` creates valid file
- [ ] `gwt config init --force` overwrites

### Manual Testing Checklist

- [ ] `gwt config` works in main worktree
- [ ] `gwt config` works in linked worktree (shows inherited)
- [ ] `gwt config init` creates file with correct content
- [ ] `gwt config path` shows correct path
- [ ] `--config` flag works with all commands
- [ ] Config file with comments loads correctly
- [ ] Windows paths handled correctly

---

## Definition of Done

Phase 3 is complete when:

1. **Configuration Struct**
   - [ ] All fields from spec represented
   - [ ] Proper struct tags for Viper and YAML
   - [ ] Serializes correctly to/from YAML

2. **Default Values**
   - [ ] All defaults match spec
   - [ ] DefaultConfig() returns complete config
   - [ ] Defaults are sensible for common use cases

3. **Config Loading**
   - [ ] Loads from explicit path
   - [ ] Searches repository root
   - [ ] Merges with defaults correctly
   - [ ] Handles missing file gracefully

4. **Config Inheritance**
   - [ ] Main worktree loads from its root
   - [ ] Linked worktree inherits from main
   - [ ] Local config overrides inheritance
   - [ ] Reports correct config source

5. **Validation**
   - [ ] Validates all fields with rules
   - [ ] Collects all errors at once
   - [ ] Helpful error messages

6. **CLI Commands**
   - [ ] `gwt config` shows current config
   - [ ] `gwt config init` creates default file
   - [ ] `gwt config path` shows config path
   - [ ] `--config` flag works globally

7. **Code Quality**
   - [ ] Code passes `go vet`
   - [ ] Unit tests for all functions
   - [ ] Integration tests pass
   - [ ] Documentation comments on exports

---

## File Summary

| File | Purpose |
|------|---------|
| `internal/config/config.go` | Configuration struct definitions |
| `internal/config/defaults.go` | Default configuration values |
| `internal/config/load.go` | Config loading with Viper |
| `internal/config/validate.go` | Configuration validation |
| `internal/config/errors.go` | Config-specific error types |
| `internal/config/template.go` | Template for generated config file |
| `internal/cli/config.go` | Config command and subcommands |

---

## Notes

- This phase focuses on configuration only - no worktree creation logic
- The config system will be used by Phases 4-10 for their respective settings
- Consider adding config schema validation in future (JSON Schema or similar)
- Viper handles env var overrides automatically (GWT_DOCKER_DEFAULT_MODE, etc.)
- Config file should be .gitignore-able but can also be committed (user choice)
- Phase 1's root command needs updating to wire in the --config flag
