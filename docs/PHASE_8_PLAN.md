# Phase 8: Dependency Installation - Implementation Plan

## Overview

Phase 8 implements automatic dependency installation for newly created worktrees. When a worktree is created, GWT will detect package managers in configured paths and run the appropriate install commands.

## Requirements

From `IMPLEMENTATION_PHASES.md`:
- [ ] Implement package manager detection (npm, yarn, pnpm, bun, go, cargo, pip, poetry)
- [ ] Support monorepo detection via config paths
- [ ] Run installations with output streaming
- [ ] Add `--skip-install` flag

## Existing Infrastructure

### Already Implemented

1. **Config struct** (`internal/config/config.go:21-25`):
   ```go
   type DependenciesConfig struct {
       AutoInstall bool     `mapstructure:"auto_install" yaml:"auto_install"`
       Paths       []string `mapstructure:"paths" yaml:"paths"`
   }
   ```

2. **Default values** (`internal/config/defaults.go:30-35`):
   ```go
   Dependencies: DependenciesConfig{
       AutoInstall: true,
       Paths:       []string{"."},
   }
   ```

3. **Config template** (`internal/config/template.go:42-50`)

4. **Path validation** (`internal/config/validate.go:54-63`)

5. **CLI flag** (`internal/cli/create.go:56-57`):
   ```go
   createCmd.Flags().BoolVar(&createOpts.SkipInstall, "skip-install", false,
       "skip dependency installation")
   ```

6. **Integration point** (`internal/cli/create.go:272-284`): After file copying, before hooks

---

## Package Structure

Create `internal/install/` with the following files:

```
internal/install/
├── detect.go          # Package manager detection
├── detect_test.go
├── manager.go         # Package manager interface and registry
├── manager_test.go
├── install.go         # Installation orchestrator
├── install_test.go
└── result.go          # Result types
```

---

## Task Breakdown

### Task 1: Create Package Manager Interface

**File:** `internal/install/manager.go`

```go
// PackageManager represents a detected package manager
type PackageManager struct {
    Name       string   // e.g., "npm", "yarn", "go", "cargo"
    Path       string   // Directory containing the package manager files
    LockFile   string   // Lock file that triggered detection (if any)
    InstallCmd string   // Command to run (e.g., "npm install")
    InstallArgs []string // Arguments for the install command
}

// Executor interface for running package manager commands
type Executor interface {
    Install(pm PackageManager, opts InstallOptions) error
}

type InstallOptions struct {
    Verbose    bool
    Timeout    time.Duration
    OnProgress func(line string) // Called for each output line
}
```

### Task 2: Implement Package Manager Detection

**File:** `internal/install/detect.go`

Detection order (prefer more specific lock files over generic):

| Manager | Detection Files | Install Command |
|---------|----------------|-----------------|
| bun | `bun.lock` | `bun install` |
| pnpm | `pnpm-lock.yaml` | `pnpm install` |
| yarn | `yarn.lock` | `yarn install` |
| npm | `package-lock.json` or `package.json` | `npm install` |
| go | `go.mod` | `go mod download` |
| cargo | `Cargo.toml` | `cargo fetch` |
| poetry | `poetry.lock` or `pyproject.toml` (with [tool.poetry]) | `poetry install` |
| pip | `requirements.txt` | `pip install -r requirements.txt` |

**Detection logic:**

```go
// DetectPackageManagers finds all package managers in the given paths
// relative to the worktree root
func DetectPackageManagers(worktreePath string, paths []string) ([]PackageManager, error) {
    var managers []PackageManager
    seen := make(map[string]bool) // Prevent duplicates

    for _, relPath := range paths {
        absPath := filepath.Join(worktreePath, relPath)

        // Expand glob patterns using doublestar
        matches, err := doublestar.FilepathGlob(absPath)
        if err != nil {
            continue // Skip invalid patterns
        }

        for _, match := range matches {
            if seen[match] {
                continue
            }

            if pm := detectInDirectory(match); pm != nil {
                seen[match] = true
                managers = append(managers, *pm)
            }
        }
    }

    return managers, nil
}
```

**Per-directory detection:**

```go
func detectInDirectory(dir string) *PackageManager {
    // Check JavaScript/Node.js (order matters - most specific first)
    if fileExists(filepath.Join(dir, "bun.lock")) {
        return &PackageManager{Name: "bun", Path: dir, LockFile: "bun.lock",
            InstallCmd: "bun", InstallArgs: []string{"install"}}
    }
    if fileExists(filepath.Join(dir, "pnpm-lock.yaml")) {
        return &PackageManager{Name: "pnpm", Path: dir, LockFile: "pnpm-lock.yaml",
            InstallCmd: "pnpm", InstallArgs: []string{"install"}}
    }
    if fileExists(filepath.Join(dir, "yarn.lock")) {
        return &PackageManager{Name: "yarn", Path: dir, LockFile: "yarn.lock",
            InstallCmd: "yarn", InstallArgs: []string{"install"}}
    }
    if fileExists(filepath.Join(dir, "package-lock.json")) {
        return &PackageManager{Name: "npm", Path: dir, LockFile: "package-lock.json",
            InstallCmd: "npm", InstallArgs: []string{"install"}}
    }
    if fileExists(filepath.Join(dir, "package.json")) {
        return &PackageManager{Name: "npm", Path: dir, LockFile: "",
            InstallCmd: "npm", InstallArgs: []string{"install"}}
    }

    // Check Go
    if fileExists(filepath.Join(dir, "go.mod")) {
        return &PackageManager{Name: "go", Path: dir, LockFile: "go.sum",
            InstallCmd: "go", InstallArgs: []string{"mod", "download"}}
    }

    // Check Rust/Cargo
    if fileExists(filepath.Join(dir, "Cargo.toml")) {
        return &PackageManager{Name: "cargo", Path: dir, LockFile: "Cargo.lock",
            InstallCmd: "cargo", InstallArgs: []string{"fetch"}}
    }

    // Check Python - poetry first (more specific)
    if fileExists(filepath.Join(dir, "poetry.lock")) {
        return &PackageManager{Name: "poetry", Path: dir, LockFile: "poetry.lock",
            InstallCmd: "poetry", InstallArgs: []string{"install"}}
    }
    if hasPoetryConfig(filepath.Join(dir, "pyproject.toml")) {
        return &PackageManager{Name: "poetry", Path: dir, LockFile: "",
            InstallCmd: "poetry", InstallArgs: []string{"install"}}
    }
    if fileExists(filepath.Join(dir, "requirements.txt")) {
        return &PackageManager{Name: "pip", Path: dir, LockFile: "",
            InstallCmd: "pip", InstallArgs: []string{"install", "-r", "requirements.txt"}}
    }

    return nil
}

// hasPoetryConfig checks if pyproject.toml contains [tool.poetry]
func hasPoetryConfig(path string) bool {
    if !fileExists(path) {
        return false
    }
    content, err := os.ReadFile(path)
    if err != nil {
        return false
    }
    return strings.Contains(string(content), "[tool.poetry]")
}
```

### Task 3: Implement Installation Executor

**File:** `internal/install/install.go`

```go
// Install runs dependency installation for all detected package managers
func Install(worktreePath string, cfg *config.DependenciesConfig, opts InstallOptions) (*Result, error) {
    if !cfg.AutoInstall {
        return &Result{Skipped: true, Reason: "auto_install disabled"}, nil
    }

    // Detect package managers
    managers, err := DetectPackageManagers(worktreePath, cfg.Paths)
    if err != nil {
        return nil, fmt.Errorf("detection failed: %w", err)
    }

    if len(managers) == 0 {
        return &Result{Skipped: true, Reason: "no package managers detected"}, nil
    }

    result := &Result{
        Managers: make([]ManagerResult, 0, len(managers)),
    }

    for _, pm := range managers {
        output.Info(fmt.Sprintf("Installing %s dependencies in %s...", pm.Name, pm.Path))

        mrResult := runInstall(pm, opts)
        result.Managers = append(result.Managers, mrResult)

        if mrResult.Success {
            output.Success(fmt.Sprintf("%s install completed", pm.Name))
        } else {
            output.Warning(fmt.Sprintf("%s install failed: %v", pm.Name, mrResult.Error))
        }
    }

    return result, nil
}

func runInstall(pm PackageManager, opts InstallOptions) ManagerResult {
    cmd := exec.Command(pm.InstallCmd, pm.InstallArgs...)
    cmd.Dir = pm.Path

    // Create pipes for streaming output
    stdout, _ := cmd.StdoutPipe()
    stderr, _ := cmd.StderrPipe()

    if err := cmd.Start(); err != nil {
        return ManagerResult{
            Manager: pm.Name,
            Path:    pm.Path,
            Success: false,
            Error:   err,
        }
    }

    // Stream output if callback provided
    var wg sync.WaitGroup
    if opts.OnProgress != nil {
        wg.Add(2)
        go streamLines(stdout, opts.OnProgress, &wg)
        go streamLines(stderr, opts.OnProgress, &wg)
    }

    // Wait with timeout
    done := make(chan error, 1)
    go func() {
        wg.Wait()
        done <- cmd.Wait()
    }()

    timeout := opts.Timeout
    if timeout == 0 {
        timeout = 5 * time.Minute // Default 5 minute timeout
    }

    select {
    case err := <-done:
        return ManagerResult{
            Manager: pm.Name,
            Path:    pm.Path,
            Success: err == nil,
            Error:   err,
        }
    case <-time.After(timeout):
        cmd.Process.Kill()
        return ManagerResult{
            Manager: pm.Name,
            Path:    pm.Path,
            Success: false,
            Error:   fmt.Errorf("installation timed out after %v", timeout),
        }
    }
}

func streamLines(r io.Reader, callback func(string), wg *sync.WaitGroup) {
    defer wg.Done()
    scanner := bufio.NewScanner(r)
    for scanner.Scan() {
        callback(scanner.Text())
    }
}
```

### Task 4: Define Result Types

**File:** `internal/install/result.go`

```go
package install

// Result represents the overall installation result
type Result struct {
    Skipped  bool            // true if installation was skipped
    Reason   string          // reason for skipping (if skipped)
    Managers []ManagerResult // results per package manager
}

// ManagerResult represents installation result for a single package manager
type ManagerResult struct {
    Manager string // Package manager name (npm, yarn, go, etc.)
    Path    string // Directory where installation was run
    Success bool   // Whether installation succeeded
    Error   error  // Error if installation failed
}

// HasErrors returns true if any installation failed
func (r *Result) HasErrors() bool {
    for _, m := range r.Managers {
        if !m.Success {
            return true
        }
    }
    return false
}

// SuccessCount returns the number of successful installations
func (r *Result) SuccessCount() int {
    count := 0
    for _, m := range r.Managers {
        if m.Success {
            count++
        }
    }
    return count
}

// ErrorCount returns the number of failed installations
func (r *Result) ErrorCount() int {
    count := 0
    for _, m := range r.Managers {
        if !m.Success {
            count++
        }
    }
    return count
}
```

### Task 5: Integrate with Create Command

**File:** `internal/cli/create.go`

Add after file copying section (around line 284):

```go
// Install dependencies
if !createOpts.SkipInstall {
    if err := installDependencies(result.Path, cfg); err != nil {
        output.Warning(fmt.Sprintf("Dependency installation had errors: %v", err))
        // Non-fatal - worktree was created successfully
    }
}
```

Add helper function:

```go
func installDependencies(worktreePath string, cfg *config.Config) error {
    opts := install.InstallOptions{
        Verbose: rootOpts.Verbose,
        Timeout: 5 * time.Minute,
    }

    if rootOpts.Verbose {
        opts.OnProgress = func(line string) {
            output.Verbose(line)
        }
    }

    result, err := install.Install(worktreePath, &cfg.Dependencies, opts)
    if err != nil {
        return err
    }

    if result.Skipped {
        output.Verbose(fmt.Sprintf("Dependency installation skipped: %s", result.Reason))
        return nil
    }

    if result.HasErrors() {
        return fmt.Errorf("%d of %d installations failed",
            result.ErrorCount(), len(result.Managers))
    }

    return nil
}
```

### Task 6: Write Unit Tests

**File:** `internal/install/detect_test.go`

```go
func TestDetectPackageManagers(t *testing.T) {
    tests := []struct {
        name     string
        files    []string // files to create in temp dir
        expected []string // expected manager names
    }{
        {
            name:     "npm with lock file",
            files:    []string{"package.json", "package-lock.json"},
            expected: []string{"npm"},
        },
        {
            name:     "yarn",
            files:    []string{"package.json", "yarn.lock"},
            expected: []string{"yarn"},
        },
        {
            name:     "pnpm",
            files:    []string{"package.json", "pnpm-lock.yaml"},
            expected: []string{"pnpm"},
        },
        {
            name:     "bun",
            files:    []string{"package.json", "bun.lock"},
            expected: []string{"bun"},
        },
        {
            name:     "go module",
            files:    []string{"go.mod"},
            expected: []string{"go"},
        },
        {
            name:     "cargo",
            files:    []string{"Cargo.toml"},
            expected: []string{"cargo"},
        },
        {
            name:     "pip",
            files:    []string{"requirements.txt"},
            expected: []string{"pip"},
        },
        {
            name:     "no package manager",
            files:    []string{"README.md"},
            expected: []string{},
        },
        // Add more test cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Create temp directory with test files
            // Run detection
            // Assert expected managers found
        })
    }
}
```

**File:** `internal/install/install_test.go`

```go
func TestInstall_SkipsWhenDisabled(t *testing.T) {
    cfg := &config.DependenciesConfig{
        AutoInstall: false,
    }

    result, err := Install("/tmp/test", cfg, InstallOptions{})

    require.NoError(t, err)
    assert.True(t, result.Skipped)
    assert.Equal(t, "auto_install disabled", result.Reason)
}

func TestInstall_SkipsWhenNoManagers(t *testing.T) {
    // Create empty temp directory
    // Run install
    // Assert skipped with reason "no package managers detected"
}
```

---

## Monorepo Configuration Example

```yaml
dependencies:
  auto_install: true
  paths:
    - "."              # Root package
    - "apps/web"       # Web app
    - "apps/api"       # API server
    - "packages/*"     # All packages in packages/ directory
```

With glob pattern expansion, this would detect package managers in:
- Root directory (e.g., root `package.json` for workspace)
- `apps/web/` (e.g., Next.js app)
- `apps/api/` (e.g., Go or Node API)
- All subdirectories of `packages/`

---

## Output Examples

### Normal output:
```
Installing npm dependencies in .
✓ npm install completed
Installing go dependencies in apps/api
✓ go install completed
```

### Verbose output:
```
Installing npm dependencies in .
npm WARN deprecated package@1.0.0
added 1234 packages in 45s
✓ npm install completed
```

### Warning on failure:
```
Installing npm dependencies in .
⚠ npm install failed: exit status 1
```

---

## Error Handling

Following Phase 6 patterns:
- Installation failures are **non-fatal** - worktree creation succeeds
- Errors are collected and reported via warnings
- Individual manager failures don't stop other installations
- Timeout prevents hanging on stuck installations (5 min default)

---

## Implementation Order

1. **Task 1:** Create `manager.go` with types and interface
2. **Task 2:** Implement `detect.go` with detection logic
3. **Task 3:** Implement `install.go` with execution
4. **Task 4:** Create `result.go` with result types
5. **Task 5:** Integrate into `create.go`
6. **Task 6:** Write comprehensive tests

---

## Dependencies

No new external dependencies required:
- File detection: `os` package
- Glob patterns: `github.com/bmatcuk/doublestar/v4` (already in go.mod)
- Command execution: `os/exec` package
- Output: existing `internal/output` package

---

## Testing Strategy

1. **Unit tests:**
   - Detection logic with mock filesystems
   - Result type methods
   - Path expansion with globs

2. **Integration tests:**
   - Actual package manager detection in temp directories
   - Command execution (can mock or use --dry-run where available)

3. **Manual testing:**
   - Test with real npm/yarn/pnpm/go projects
   - Test monorepo configurations
   - Test timeout behavior
   - Test `--skip-install` flag

---

## Future Enhancements (Out of Scope for Phase 8)

- Custom install commands per manager in config
- Parallel installation for independent managers
- Install progress bar with package count
- Retry logic for transient failures
- `--prefer-manager` flag to override detection
- Virtual environment creation for Python
