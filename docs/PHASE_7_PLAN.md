# Phase 7: Docker Compose Scaffolding - Implementation Plan

This document outlines the implementation plan for Phase 7 of the GWT (Git Worktree Manager) project.

---

## Overview

Phase 7 adds Docker Compose scaffolding capabilities, enabling worktrees to either share container data with the main worktree or have isolated container environments with separate volumes and ports.

### Goals

1. Implement compose file auto-detection
2. Parse compose files for services and volumes
3. Implement Shared mode (symlink data directories)
4. Implement New mode (copy data, rename volumes, generate override)
5. Add Windows symlink fallback (junction, then copy)
6. Generate `dc` helper script
7. Handle port conflict warnings

---

## File Structure

Create a new `internal/docker/` package:

```
internal/docker/
├── detect.go         # Compose file auto-detection
├── detect_test.go
├── parse.go          # Compose file parsing
├── parse_test.go
├── shared.go         # Shared mode implementation
├── shared_test.go
├── new.go            # New mode implementation
├── new_test.go
├── override.go       # Override file generation
├── override_test.go
├── helper.go         # dc helper script generation
├── helper_test.go
├── symlink.go        # Symlink/junction/copy utilities
├── symlink_test.go
└── errors.go         # Custom error types
```

Also modify:
- `internal/cli/create.go` - Integration with create workflow

---

## Task Breakdown

### Task 1: Create Package Structure and Error Types

**File:** `internal/docker/errors.go`

Define custom error types for the docker package:

```go
package docker

import (
    "errors"
    "fmt"
)

var (
    ErrNoComposeFile       = errors.New("no docker compose file found")
    ErrInvalidComposeFile  = errors.New("invalid docker compose file")
    ErrParseError          = errors.New("failed to parse compose file")
    ErrSymlinkFailed       = errors.New("symlink creation failed")
    ErrJunctionFailed      = errors.New("junction creation failed")
    ErrCopyFailed          = errors.New("directory copy failed")
    ErrOverrideExists      = errors.New("override file already exists")
    ErrPortConflict        = errors.New("port conflict detected")
)

// DockerError wraps an error with context
type DockerError struct {
    Op      string
    File    string
    Service string
    Err     error
}

func (e *DockerError) Error() string {
    if e.Service != "" {
        return fmt.Sprintf("%s %s (service: %s): %v", e.Op, e.File, e.Service, e.Err)
    }
    return fmt.Sprintf("%s %s: %v", e.Op, e.File, e.Err)
}

func (e *DockerError) Unwrap() error {
    return e.Err
}
```

---

### Task 2: Implement Compose File Auto-Detection

**File:** `internal/docker/detect.go`

Detect Docker Compose files in the repository.

**Functions to implement:**

```go
// ComposeFile represents a detected compose file
type ComposeFile struct {
    Path     string // Relative path from repo root
    FullPath string // Absolute path
    IsBase   bool   // Is this a base compose file (vs override)
}

// DetectComposeFiles finds all compose files in the given directory
// Searches for:
// - docker-compose.yml / docker-compose.yaml
// - docker-compose.*.yml / docker-compose.*.yaml
// - compose.yml / compose.yaml
// - compose.*.yml / compose.*.yaml
func DetectComposeFiles(repoPath string) ([]ComposeFile, error)

// GetBaseComposeFile returns the primary compose file (not an override)
// Priority: docker-compose.yml > docker-compose.yaml > compose.yml > compose.yaml
func GetBaseComposeFile(files []ComposeFile) *ComposeFile

// IsOverrideFile checks if a compose file is an override (has .*.yml pattern)
func IsOverrideFile(filename string) bool
```

**Detection patterns:**
```go
var composePatterns = []string{
    "docker-compose.yml",
    "docker-compose.yaml",
    "docker-compose.*.yml",
    "docker-compose.*.yaml",
    "compose.yml",
    "compose.yaml",
    "compose.*.yml",
    "compose.*.yaml",
}

// Base file priority (higher = more preferred)
var basePriority = map[string]int{
    "docker-compose.yml":  4,
    "docker-compose.yaml": 3,
    "compose.yml":         2,
    "compose.yaml":        1,
}
```

**Config override:**
If `docker.compose_files` is specified in config, use those instead of auto-detection:

```go
func DetectOrLoad(repoPath string, configFiles []string) ([]ComposeFile, error) {
    if len(configFiles) > 0 {
        return loadConfiguredFiles(repoPath, configFiles)
    }
    return DetectComposeFiles(repoPath)
}
```

---

### Task 3: Implement Compose File Parsing

**File:** `internal/docker/parse.go`

Parse compose files to extract services, volumes, and ports.

**Types to define:**

```go
// ComposeConfig represents a parsed docker-compose file
type ComposeConfig struct {
    Version  string                  `yaml:"version,omitempty"`
    Services map[string]Service      `yaml:"services"`
    Volumes  map[string]VolumeConfig `yaml:"volumes,omitempty"`
    Networks map[string]interface{}  `yaml:"networks,omitempty"`
}

// Service represents a docker-compose service
type Service struct {
    Image       string            `yaml:"image,omitempty"`
    Build       interface{}       `yaml:"build,omitempty"`
    Volumes     []string          `yaml:"volumes,omitempty"`
    Ports       []string          `yaml:"ports,omitempty"`
    Environment map[string]string `yaml:"environment,omitempty"`
    DependsOn   []string          `yaml:"depends_on,omitempty"`
}

// VolumeConfig represents a named volume configuration
type VolumeConfig struct {
    Name     string            `yaml:"name,omitempty"`
    Driver   string            `yaml:"driver,omitempty"`
    External bool              `yaml:"external,omitempty"`
    Labels   map[string]string `yaml:"labels,omitempty"`
}

// VolumeMount represents a parsed volume mount
type VolumeMount struct {
    Source      string // Volume name or host path
    Target      string // Container path
    IsNamed     bool   // True if named volume, false if bind mount
    IsReadOnly  bool   // True if read-only mount
    ServiceName string // Service that uses this volume
}

// PortMapping represents a parsed port mapping
type PortMapping struct {
    HostPort      int    // Port on host
    ContainerPort int    // Port in container
    Protocol      string // "tcp" or "udp"
    ServiceName   string // Service that exposes this port
}
```

**Functions to implement:**

```go
// ParseComposeFile parses a docker-compose file
func ParseComposeFile(path string) (*ComposeConfig, error)

// ParseComposeFiles parses multiple compose files and merges them
// Later files override earlier ones (standard docker-compose behavior)
func ParseComposeFiles(paths []string) (*ComposeConfig, error)

// ExtractVolumes extracts all volume mounts from a compose config
func ExtractVolumes(config *ComposeConfig) []VolumeMount

// ExtractPorts extracts all port mappings from a compose config
func ExtractPorts(config *ComposeConfig) []PortMapping

// ExtractNamedVolumes returns only named volumes (not bind mounts)
func ExtractNamedVolumes(config *ComposeConfig) []string

// ExtractDataDirectories returns bind mount paths that look like data directories
// Heuristics: contains "data", "db", "storage", "volumes", etc.
func ExtractDataDirectories(config *ComposeConfig) []string

// parseVolumeMount parses a volume string like "postgres_data:/var/lib/postgresql/data:ro"
func parseVolumeMount(volumeStr string, serviceName string) VolumeMount

// parsePortMapping parses a port string like "5432:5432" or "127.0.0.1:5432:5432/tcp"
func parsePortMapping(portStr string, serviceName string) PortMapping
```

**Volume string formats to handle:**
```
- "volume_name:/container/path"           # Named volume
- "volume_name:/container/path:ro"        # Named volume, read-only
- "./data:/container/path"                # Relative bind mount
- "/absolute/path:/container/path"        # Absolute bind mount
- "${VAR}:/container/path"                # Variable expansion (treat as bind mount)
```

**Port string formats to handle:**
```
- "8080"                    # Just container port
- "8080:80"                 # host:container
- "127.0.0.1:8080:80"       # ip:host:container
- "8080:80/udp"             # With protocol
- "8080-8090:80-90"         # Port range
```

---

### Task 4: Implement Symlink Utilities with Windows Fallback

**File:** `internal/docker/symlink.go`

Create symlinks with Windows fallback to junctions then copy.

**Functions to implement:**

```go
// LinkResult indicates how a link was created
type LinkResult int

const (
    LinkSymlink  LinkResult = iota // Symlink created successfully
    LinkJunction                   // Junction created (Windows only)
    LinkCopy                       // Fell back to copy
    LinkFailed                     // All methods failed
)

// LinkOptions configures link creation
type LinkOptions struct {
    Source      string // Source directory (main worktree)
    Target      string // Target path (new worktree)
    FallbackMsg string // Message to show if falling back
}

// CreateLink creates a symlink, falling back to junction then copy on Windows
// Returns the method used and any error
func CreateLink(opts LinkOptions) (LinkResult, error)

// createSymlink attempts to create a symbolic link
func createSymlink(source, target string) error

// createJunction creates a Windows junction (directory only)
// Only available on Windows
func createJunction(source, target string) error

// fallbackCopy copies the directory when symlink/junction fails
func fallbackCopy(source, target string) error

// CanCreateSymlink checks if the current process can create symlinks
func CanCreateSymlink() bool

// CanCreateJunction checks if junctions are available (Windows only)
func CanCreateJunction() bool
```

**Windows junction implementation:**
```go
// +build windows

func createJunction(source, target string) error {
    // Use mklink /J via cmd.exe
    cmd := exec.Command("cmd", "/c", "mklink", "/J", target, source)
    return cmd.Run()
}
```

**Symlink permission check:**
```go
func CanCreateSymlink() bool {
    // Create a temp file and try to symlink to it
    tmpDir := os.TempDir()
    src := filepath.Join(tmpDir, "gwt_symlink_test_src")
    dst := filepath.Join(tmpDir, "gwt_symlink_test_dst")

    // Create source file
    if err := os.WriteFile(src, []byte("test"), 0644); err != nil {
        return false
    }
    defer os.Remove(src)
    defer os.Remove(dst)

    // Try symlink
    if err := os.Symlink(src, dst); err != nil {
        return false
    }
    return true
}
```

**Developer Mode message:**
```go
const WindowsSymlinkHelp = `
Symlink creation failed. On Windows, you need one of:
  1. Run as Administrator
  2. Enable Developer Mode (Settings > Update & Security > For developers)
  3. Grant SeCreateSymbolicLinkPrivilege to your user

gwt will use directory junctions as a fallback.
`
```

---

### Task 5: Implement Shared Mode

**File:** `internal/docker/shared.go`

Implement shared mode - symlink data directories to main worktree.

**Types and functions:**

```go
// SharedModeOptions configures shared mode setup
type SharedModeOptions struct {
    MainWorktree    string   // Path to main worktree
    NewWorktree     string   // Path to new worktree
    DataDirectories []string // Directories to symlink (from config or detected)
    ComposeConfig   *ComposeConfig // Parsed compose config (for detection)
}

// SharedModeResult reports what was done
type SharedModeResult struct {
    LinkedDirs  []LinkedDirectory
    Warnings    []string
}

// LinkedDirectory represents a directory that was linked
type LinkedDirectory struct {
    Source string     // Path in main worktree
    Target string     // Path in new worktree
    Method LinkResult // How it was linked
}

// SetupSharedMode creates symlinks for data directories
func SetupSharedMode(opts SharedModeOptions) (*SharedModeResult, error)

// getDataDirectories returns directories to share
// Uses config if provided, otherwise detects from compose file
func getDataDirectories(opts SharedModeOptions) []string
```

**Implementation flow:**
```go
func SetupSharedMode(opts SharedModeOptions) (*SharedModeResult, error) {
    result := &SharedModeResult{}

    // 1. Get directories to share
    dirs := getDataDirectories(opts)
    if len(dirs) == 0 {
        result.Warnings = append(result.Warnings,
            "No data directories configured. Containers will use independent volumes.")
        return result, nil
    }

    // 2. Create symlinks for each directory
    for _, dir := range dirs {
        source := filepath.Join(opts.MainWorktree, dir)
        target := filepath.Join(opts.NewWorktree, dir)

        // Check source exists
        if _, err := os.Stat(source); os.IsNotExist(err) {
            result.Warnings = append(result.Warnings,
                fmt.Sprintf("Data directory not found: %s (skipping)", dir))
            continue
        }

        // Create parent directory in target
        if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
            return nil, fmt.Errorf("failed to create parent directory: %w", err)
        }

        // Create link
        method, err := CreateLink(LinkOptions{
            Source: source,
            Target: target,
        })
        if err != nil {
            return nil, err
        }

        result.LinkedDirs = append(result.LinkedDirs, LinkedDirectory{
            Source: source,
            Target: target,
            Method: method,
        })
    }

    return result, nil
}
```

---

### Task 6: Implement Override File Generation

**File:** `internal/docker/override.go`

Generate docker-compose.worktree.yml override file for new mode.

**Types and functions:**

```go
// OverrideOptions configures override file generation
type OverrideOptions struct {
    BranchName     string          // Branch name for suffix
    OriginalConfig *ComposeConfig  // Original compose config
    PortOffset     int             // Port offset from config
    OutputPath     string          // Where to write override file
}

// OverrideResult reports what was generated
type OverrideResult struct {
    FilePath        string
    RenamedVolumes  map[string]string // original -> renamed
    RemappedPorts   map[string]int    // original:new
    PortWarnings    []string          // Potential conflicts
}

// GenerateOverride creates a docker-compose.worktree.yml file
func GenerateOverride(opts OverrideOptions) (*OverrideResult, error)

// sanitizeBranchName converts branch name to safe suffix
// "feature/auth" -> "feature-auth"
func sanitizeBranchName(branch string) string

// renameVolume generates new volume name with branch suffix
// "postgres_data" -> "postgres_data_feature-auth"
func renameVolume(volumeName, branchSuffix string) string

// offsetPort calculates new port with offset
// Checks for common port conflicts
func offsetPort(port, offset int) int
```

**Override file template:**
```yaml
# Auto-generated by gwt - do not edit manually
# To use: docker compose -f docker-compose.yml -f docker-compose.worktree.yml up
#
# Generated for branch: feature-auth
# Generated at: 2024-01-15T10:30:00Z

volumes:
  postgres_data_feature-auth:
    name: postgres_data_feature-auth
  redis_data_feature-auth:
    name: redis_data_feature-auth

services:
  db:
    volumes:
      - postgres_data_feature-auth:/var/lib/postgresql/data
    ports:
      - "5433:5432"  # Changed from 5432 (offset: +1)

  redis:
    volumes:
      - redis_data_feature-auth:/data
    ports:
      - "6380:6379"  # Changed from 6379 (offset: +1)
```

**Implementation:**
```go
func GenerateOverride(opts OverrideOptions) (*OverrideResult, error) {
    result := &OverrideResult{
        RenamedVolumes: make(map[string]string),
        RemappedPorts:  make(map[string]int),
    }

    branchSuffix := sanitizeBranchName(opts.BranchName)
    override := &ComposeConfig{
        Services: make(map[string]Service),
        Volumes:  make(map[string]VolumeConfig),
    }

    // 1. Process named volumes
    for volName := range opts.OriginalConfig.Volumes {
        newName := renameVolume(volName, branchSuffix)
        result.RenamedVolumes[volName] = newName
        override.Volumes[newName] = VolumeConfig{
            Name: newName,
        }
    }

    // 2. Process services
    for svcName, svc := range opts.OriginalConfig.Services {
        overrideSvc := Service{}

        // Rename volume references
        for _, vol := range svc.Volumes {
            mount := parseVolumeMount(vol, svcName)
            if mount.IsNamed {
                if newName, ok := result.RenamedVolumes[mount.Source]; ok {
                    overrideSvc.Volumes = append(overrideSvc.Volumes,
                        fmt.Sprintf("%s:%s", newName, mount.Target))
                }
            }
        }

        // Offset ports
        for _, port := range svc.Ports {
            mapping := parsePortMapping(port, svcName)
            if mapping.HostPort > 0 {
                newPort := offsetPort(mapping.HostPort, opts.PortOffset)
                result.RemappedPorts[fmt.Sprintf("%d", mapping.HostPort)] = newPort

                // Check for common conflicts
                if isCommonPort(newPort) {
                    result.PortWarnings = append(result.PortWarnings,
                        fmt.Sprintf("Port %d may conflict with common services", newPort))
                }

                portStr := fmt.Sprintf("%d:%d", newPort, mapping.ContainerPort)
                overrideSvc.Ports = append(overrideSvc.Ports, portStr)
            }
        }

        if len(overrideSvc.Volumes) > 0 || len(overrideSvc.Ports) > 0 {
            override.Services[svcName] = overrideSvc
        }
    }

    // 3. Write file
    result.FilePath = opts.OutputPath
    return result, writeOverrideFile(override, opts)
}
```

**Common ports to check:**
```go
var commonPorts = map[int]string{
    80:    "HTTP",
    443:   "HTTPS",
    3000:  "Node.js/React dev server",
    3306:  "MySQL",
    5432:  "PostgreSQL",
    5433:  "PostgreSQL (alternate)",
    6379:  "Redis",
    8080:  "HTTP alternate",
    8443:  "HTTPS alternate",
    27017: "MongoDB",
}
```

---

### Task 7: Implement New Mode

**File:** `internal/docker/new.go`

Implement new mode - copy data, rename volumes, generate override.

**Types and functions:**

```go
// NewModeOptions configures new mode setup
type NewModeOptions struct {
    MainWorktree    string          // Path to main worktree
    NewWorktree     string          // Path to new worktree
    BranchName      string          // Branch name for suffixes
    DataDirectories []string        // Directories to copy
    ComposeConfig   *ComposeConfig  // Parsed compose config
    PortOffset      int             // Port offset
}

// NewModeResult reports what was done
type NewModeResult struct {
    CopiedDirs     []string
    OverrideFile   string
    RenamedVolumes map[string]string
    RemappedPorts  map[string]int
    PortWarnings   []string
    Warnings       []string
}

// SetupNewMode sets up isolated containers for the new worktree
func SetupNewMode(opts NewModeOptions) (*NewModeResult, error)
```

**Implementation flow:**
```go
func SetupNewMode(opts NewModeOptions) (*NewModeResult, error) {
    result := &NewModeResult{
        RenamedVolumes: make(map[string]string),
        RemappedPorts:  make(map[string]int),
    }

    // 1. Copy data directories
    for _, dir := range opts.DataDirectories {
        source := filepath.Join(opts.MainWorktree, dir)
        target := filepath.Join(opts.NewWorktree, dir)

        if _, err := os.Stat(source); os.IsNotExist(err) {
            result.Warnings = append(result.Warnings,
                fmt.Sprintf("Data directory not found: %s (skipping)", dir))
            continue
        }

        // Copy directory
        if err := copyDir(source, target); err != nil {
            return nil, fmt.Errorf("failed to copy %s: %w", dir, err)
        }
        result.CopiedDirs = append(result.CopiedDirs, dir)
    }

    // 2. Generate override file
    overridePath := filepath.Join(opts.NewWorktree, "docker-compose.worktree.yml")
    overrideResult, err := GenerateOverride(OverrideOptions{
        BranchName:     opts.BranchName,
        OriginalConfig: opts.ComposeConfig,
        PortOffset:     opts.PortOffset,
        OutputPath:     overridePath,
    })
    if err != nil {
        return nil, err
    }

    result.OverrideFile = overrideResult.FilePath
    result.RenamedVolumes = overrideResult.RenamedVolumes
    result.RemappedPorts = overrideResult.RemappedPorts
    result.PortWarnings = overrideResult.PortWarnings

    return result, nil
}
```

---

### Task 8: Generate dc Helper Script

**File:** `internal/docker/helper.go`

Generate a convenience script for running docker-compose with the override file.

**Functions to implement:**

```go
// HelperScriptOptions configures helper script generation
type HelperScriptOptions struct {
    WorktreePath   string   // Path to new worktree
    ComposeFiles   []string // Base compose files
    OverrideFile   string   // Override file name (or empty for shared mode)
    ShellType      string   // "bash", "powershell", or "cmd"
}

// GenerateHelperScript creates the dc helper script
func GenerateHelperScript(opts HelperScriptOptions) error

// getDefaultShell returns the default shell for the current OS
func getDefaultShell() string
```

**Bash script template (dc):**
```bash
#!/bin/bash
# Auto-generated by gwt - convenience wrapper for docker-compose
# Usage: ./dc up, ./dc down, ./dc logs, etc.

COMPOSE_FILES="-f docker-compose.yml -f docker-compose.worktree.yml"

docker compose $COMPOSE_FILES "$@"
```

**PowerShell script template (dc.ps1):**
```powershell
# Auto-generated by gwt - convenience wrapper for docker-compose
# Usage: .\dc.ps1 up, .\dc.ps1 down, .\dc.ps1 logs, etc.

$ComposeFiles = @("-f", "docker-compose.yml", "-f", "docker-compose.worktree.yml")

docker compose @ComposeFiles $args
```

**CMD batch script template (dc.cmd):**
```cmd
@echo off
REM Auto-generated by gwt - convenience wrapper for docker-compose
REM Usage: dc up, dc down, dc logs, etc.

docker compose -f docker-compose.yml -f docker-compose.worktree.yml %*
```

**Implementation:**
```go
func GenerateHelperScript(opts HelperScriptOptions) error {
    shell := opts.ShellType
    if shell == "" {
        shell = getDefaultShell()
    }

    var scriptPath, scriptContent string

    switch shell {
    case "bash":
        scriptPath = filepath.Join(opts.WorktreePath, "dc")
        scriptContent = generateBashScript(opts)
    case "powershell":
        scriptPath = filepath.Join(opts.WorktreePath, "dc.ps1")
        scriptContent = generatePowerShellScript(opts)
    case "cmd":
        scriptPath = filepath.Join(opts.WorktreePath, "dc.cmd")
        scriptContent = generateCmdScript(opts)
    }

    // Write script
    if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
        return err
    }

    return nil
}

func getDefaultShell() string {
    if runtime.GOOS == "windows" {
        // Check if running in Git Bash/MSYS
        if os.Getenv("MSYSTEM") != "" {
            return "bash"
        }
        return "cmd" // Default to cmd for broader compatibility
    }
    return "bash"
}
```

---

### Task 9: Implement Port Conflict Detection

**File:** `internal/docker/ports.go` (or add to `parse.go`)

Detect and warn about potential port conflicts.

**Functions to implement:**

```go
// PortConflict represents a potential port conflict
type PortConflict struct {
    Port        int
    Service     string
    Reason      string
    Suggestion  int
}

// CheckPortConflicts checks for potential port conflicts
func CheckPortConflicts(ports []PortMapping, offset int) []PortConflict

// IsPortInUse checks if a port is currently in use
func IsPortInUse(port int) bool

// SuggestAlternativePort suggests an available port
func SuggestAlternativePort(basePort, offset int) int
```

**Implementation:**
```go
func CheckPortConflicts(ports []PortMapping, offset int) []PortConflict {
    var conflicts []PortConflict

    for _, pm := range ports {
        newPort := pm.HostPort + offset

        // Check if port is in use
        if IsPortInUse(newPort) {
            conflicts = append(conflicts, PortConflict{
                Port:       newPort,
                Service:    pm.ServiceName,
                Reason:     "Port is currently in use",
                Suggestion: SuggestAlternativePort(pm.HostPort, offset),
            })
            continue
        }

        // Check for common port conflicts
        if desc, ok := commonPorts[newPort]; ok {
            conflicts = append(conflicts, PortConflict{
                Port:       newPort,
                Service:    pm.ServiceName,
                Reason:     fmt.Sprintf("Commonly used by %s", desc),
                Suggestion: newPort + 1,
            })
        }
    }

    return conflicts
}

func IsPortInUse(port int) bool {
    listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
    if err != nil {
        return true
    }
    listener.Close()
    return false
}
```

---

### Task 10: Integrate with Create Command

**File:** `internal/cli/create.go` (modify existing)

Add Docker setup step to the create workflow.

**New flags:**

```go
var (
    createDockerMode   string // "shared", "new", or "skip"
    createSkipDocker   bool   // Skip Docker setup entirely
)

func init() {
    // ... existing flags ...
    createCmd.Flags().StringVar(&createDockerMode, "docker-mode", "",
        "Docker setup mode: shared, new, or skip")
    createCmd.Flags().BoolVar(&createSkipDocker, "skip-docker", false,
        "Skip Docker Compose setup")
}
```

**Integration flow:**

```go
func runCreate(cmd *cobra.Command, args []string) error {
    // ... existing worktree creation code ...

    // ... file copying step (Phase 6) ...

    // Docker setup step
    if !createSkipDocker {
        if err := setupDocker(mainWorktree, newWorktreePath, cfg, branchName); err != nil {
            output.Warning(fmt.Sprintf("Docker setup failed: %v", err))
            // Non-fatal - worktree was created successfully
        }
    }

    return nil
}

func setupDocker(mainWorktree, newWorktree string, cfg *config.Config, branch string) error {
    // 1. Detect compose files
    composeFiles, err := docker.DetectOrLoad(mainWorktree, cfg.Docker.ComposeFiles)
    if err != nil {
        if errors.Is(err, docker.ErrNoComposeFile) {
            output.Info("No Docker Compose files found")
            return nil
        }
        return err
    }

    // 2. Parse compose files
    config, err := docker.ParseComposeFiles(composePaths(composeFiles))
    if err != nil {
        return err
    }

    // Show detected services
    services := make([]string, 0, len(config.Services))
    for name := range config.Services {
        services = append(services, name)
    }
    output.Info(fmt.Sprintf("Found Docker services: %s", strings.Join(services, ", ")))

    // 3. Determine mode
    mode := createDockerMode
    if mode == "" {
        mode = cfg.Docker.DefaultMode
    }
    if mode == "" {
        mode = "shared" // Default
    }

    // 4. Execute mode
    switch mode {
    case "shared":
        result, err := docker.SetupSharedMode(docker.SharedModeOptions{
            MainWorktree:    mainWorktree,
            NewWorktree:     newWorktree,
            DataDirectories: cfg.Docker.DataDirectories,
            ComposeConfig:   config,
        })
        if err != nil {
            return err
        }
        showSharedModeResult(result)

    case "new":
        result, err := docker.SetupNewMode(docker.NewModeOptions{
            MainWorktree:    mainWorktree,
            NewWorktree:     newWorktree,
            BranchName:      branch,
            DataDirectories: cfg.Docker.DataDirectories,
            ComposeConfig:   config,
            PortOffset:      cfg.Docker.PortOffset,
        })
        if err != nil {
            return err
        }
        showNewModeResult(result)

        // Generate helper script
        docker.GenerateHelperScript(docker.HelperScriptOptions{
            WorktreePath: newWorktree,
            ComposeFiles: composePaths(composeFiles),
            OverrideFile: "docker-compose.worktree.yml",
        })

    case "skip":
        output.Info("Skipping Docker setup")
    }

    return nil
}
```

---

### Task 11: Write Tests

**Test files to create:**

1. `internal/docker/detect_test.go`
   - Test compose file detection patterns
   - Test priority ordering
   - Test config override
   - Test missing files handling

2. `internal/docker/parse_test.go`
   - Test volume string parsing (named, bind, readonly)
   - Test port string parsing (various formats)
   - Test full compose file parsing
   - Test multi-file merging

3. `internal/docker/symlink_test.go`
   - Test symlink creation
   - Test junction fallback (Windows)
   - Test copy fallback
   - Test permission checking

4. `internal/docker/shared_test.go`
   - Test symlink creation for data directories
   - Test missing directory handling
   - Test warnings collection

5. `internal/docker/new_test.go`
   - Test directory copying
   - Test override generation
   - Test volume renaming
   - Test port remapping

6. `internal/docker/override_test.go`
   - Test branch name sanitization
   - Test volume name generation
   - Test port offset calculation
   - Test YAML output format

7. `internal/docker/helper_test.go`
   - Test script generation for each shell
   - Test correct file permissions

**Test patterns:**
- Use temporary directories for file operations
- Create test compose files programmatically
- Test Windows-specific behavior with build tags
- Mock network operations for port checking

---

## Dependencies

**Standard library:**
- `os`, `io`, `path/filepath` - File operations
- `fmt`, `strings` - String formatting
- `runtime` - OS detection
- `net` - Port checking
- `time` - Timestamps

**External packages (already in go.mod):**
- `gopkg.in/yaml.v3` - YAML parsing/writing

No new external dependencies required.

---

## Config Reference

Relevant config fields from `.worktree.yaml`:

```yaml
docker:
  # Compose files to consider (auto-detected if not specified)
  compose_files:
    - "docker-compose.yml"
    - "docker-compose.dev.yml"

  # Data directories to symlink (shared) or copy (new)
  data_directories:
    - "server/db-data"
    - "data/redis"

  # Default mode: "shared" or "new"
  default_mode: "shared"

  # Port offset for new containers
  port_offset: 1
```

---

## User Output Messages

### Shared Mode Success:
```
Docker Compose Setup (Shared Mode)
──────────────────────────────────
✓ Linked server/db-data → main worktree
✓ Linked data/redis → main worktree

Containers will share data with the main worktree.
Run 'docker compose up' to start services.
```

### New Mode Success:
```
Docker Compose Setup (New Mode)
───────────────────────────────
✓ Copied server/db-data (125 MB)
✓ Copied data/redis (2.3 MB)
✓ Created docker-compose.worktree.yml

Volumes renamed:
  postgres_data → postgres_data_feature-auth
  redis_data → redis_data_feature-auth

Ports remapped:
  5432 → 5433 (PostgreSQL)
  6379 → 6380 (Redis)

⚠ Port 5433 is commonly used. Check for conflicts.

Created ./dc helper script for convenience.
Run './dc up' to start services.
```

### Windows Symlink Fallback:
```
⚠ Symlink creation requires elevated permissions.
  Using directory junctions instead.

  To enable symlinks, either:
  • Run as Administrator
  • Enable Developer Mode in Windows Settings
```

---

## Edge Cases

### No Compose Files
- Display info message and continue
- Don't fail worktree creation

### Invalid Compose File
- Display error with file path
- Continue without Docker setup

### Missing Data Directories
- Warn but don't fail
- Skip that directory

### Port Already in Use
- Warn and suggest alternative
- Still generate override file
- User can manually adjust

### Symlink Permission Denied (Windows)
- Try junction
- If junction fails, copy with warning
- Display Developer Mode instructions

### External Volumes
- Skip external volumes (not renamed)
- Warn user about potential conflicts

### Variable Expansion in Compose
- Treat ${VAR} paths as bind mounts
- Don't attempt to resolve variables

---

## Implementation Order

1. **Task 1:** Create package structure and error types
2. **Task 2:** Implement compose file auto-detection
3. **Task 3:** Implement compose file parsing
4. **Task 4:** Implement symlink utilities with Windows fallback
5. **Task 5:** Implement shared mode
6. **Task 6:** Implement override file generation
7. **Task 7:** Implement new mode
8. **Task 8:** Generate dc helper script
9. **Task 9:** Implement port conflict detection
10. **Task 10:** Integrate with create command
11. **Task 11:** Write tests (throughout, not just at end)

---

## Verification Checklist

- [ ] Compose file detection finds all standard patterns
- [ ] Config override for compose_files works
- [ ] Volume strings parse correctly (named, bind, readonly)
- [ ] Port strings parse correctly (all formats)
- [ ] Symlinks work on Linux/macOS
- [ ] Junctions work on Windows without elevation
- [ ] Fallback to copy works when junctions fail
- [ ] Developer Mode message displays on Windows
- [ ] Shared mode creates correct symlinks
- [ ] New mode copies directories correctly
- [ ] Override file has correct YAML format
- [ ] Volume names include branch suffix
- [ ] Port offset applies correctly
- [ ] Port conflict warnings display
- [ ] dc helper script is executable
- [ ] dc helper works on Windows (cmd/powershell)
- [ ] --docker-mode flag works
- [ ] --skip-docker flag works
- [ ] Errors don't fail worktree creation
- [ ] Tests pass on Windows, macOS, Linux

---

## Future Enhancements (Phase 11-12)

- TUI Docker mode selection view
- Interactive port conflict resolution
- Preview of changes before applying
- Container status checking before setup
- Volume size estimation
