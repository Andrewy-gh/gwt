package config

// DefaultConfigTemplate is the template for a default .worktree.yaml file
const DefaultConfigTemplate = `# gwt configuration file
# See: https://github.com/Andrewy-gh/gwt for documentation

# Files and directories to pre-select for copying
# Supports glob patterns
copy_defaults:
  - ".env"
  - "**/.env"
  - ".env.*"
  - "**/.env.*"
  - "*.local"
  - "**/*.local"
  - "*.local.*"
  - "**/*.local.*"

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
`
