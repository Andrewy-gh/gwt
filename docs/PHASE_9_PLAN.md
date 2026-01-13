# Phase 9: Database Migrations - Implementation Plan

A detailed plan for implementing database migration support in GWT.

---

## Overview

Phase 9 adds automatic detection and execution of database migrations when creating new worktrees. This ensures new worktrees have properly initialized databases without manual intervention.

### Goals

- Auto-detect common migration tools (Makefile, Prisma, Drizzle, Alembic, raw SQL)
- Verify database container is running before attempting migrations
- Execute migrations with real-time output streaming
- Support `--skip-migrations` flag (already exists in create command)
- Non-fatal failures (don't block worktree creation)

---

## Package Structure

```
internal/migrate/
├── detect.go          # Migration tool detection logic
├── migrate.go         # Migration execution
├── result.go          # Result types
├── errors.go          # Error types
├── container.go       # Docker container readiness checks
├── detect_test.go     # Detection tests
├── migrate_test.go    # Execution tests
└── container_test.go  # Container check tests
```

---

## Task Breakdown

### Task 1: Define Types and Errors

**File: `internal/migrate/errors.go`**

```go
package migrate

import "fmt"

// MigrationError represents a migration execution failure
type MigrationError struct {
    Tool     string
    Command  []string
    Stderr   string
    ExitCode int
}

func (e *MigrationError) Error() string {
    return fmt.Sprintf("migration failed (%s): exit code %d", e.Tool, e.ExitCode)
}

// DetectionError represents a failure in detecting migration tools
type DetectionError struct {
    Path string
    Err  error
}

func (e *DetectionError) Error() string {
    return fmt.Sprintf("migration detection failed at %s: %v", e.Path, e.Err)
}

// ContainerNotReadyError indicates the database container isn't running
type ContainerNotReadyError struct {
    Service string
    Reason  string
}

func (e *ContainerNotReadyError) Error() string {
    return fmt.Sprintf("database container %q not ready: %s", e.Service, e.Reason)
}
```

**File: `internal/migrate/result.go`**

```go
package migrate

// MigrationTool represents a detected migration tool
type MigrationTool struct {
    Name        string   // e.g., "prisma", "makefile", "alembic"
    Command     []string // Command to run migrations
    Path        string   // Directory containing the tool
    Description string   // Human-readable description
}

// Result represents the outcome of a migration run
type Result struct {
    Skipped    bool           // True if migrations were skipped
    Reason     string         // Why migrations were skipped (if applicable)
    Tool       *MigrationTool // Tool that was used
    Output     string         // Combined stdout/stderr
    Success    bool           // True if migration completed successfully
    Error      error          // Error if migration failed
}

// RunOptions configures migration execution
type RunOptions struct {
    WorktreePath    string // Path to the new worktree
    Verbose         bool   // Stream output in real-time
    DryRun          bool   // Show what would run without executing
    SkipContainerCheck bool // Skip Docker container readiness check
}
```

---

### Task 2: Implement Migration Tool Detection

**File: `internal/migrate/detect.go`**

Detection priority order (first match wins):

1. **Config override** - User-specified command in `.worktree.yaml`
2. **Makefile** - Check for `migrate` or `db-migrate` targets
3. **Prisma** - Check for `prisma/schema.prisma` or `schema.prisma`
4. **Drizzle** - Check for `drizzle.config.ts` or `drizzle.config.js`
5. **Alembic** - Check for `alembic.ini` or `alembic/` directory
6. **Raw SQL** - Check for `migrations/*.sql` files

```go
package migrate

import (
    "os"
    "path/filepath"
    "strings"

    "gwt/internal/config"
)

// Detect finds migration tools in the given worktree path
func Detect(worktreePath string, cfg *config.MigrationsConfig) (*MigrationTool, error) {
    // Priority 1: Config override
    if cfg != nil && cfg.Command != "" {
        return &MigrationTool{
            Name:        "custom",
            Command:     parseCommand(cfg.Command),
            Path:        worktreePath,
            Description: "Custom migration command from config",
        }, nil
    }

    // Priority 2: Auto-detection if enabled
    if cfg == nil || cfg.AutoDetect {
        return autoDetect(worktreePath)
    }

    return nil, nil // No migrations configured
}

func autoDetect(path string) (*MigrationTool, error) {
    detectors := []func(string) (*MigrationTool, error){
        detectMakefile,
        detectPrisma,
        detectDrizzle,
        detectAlembic,
        detectRawSQL,
    }

    for _, detect := range detectors {
        tool, err := detect(path)
        if err != nil {
            return nil, err
        }
        if tool != nil {
            return tool, nil
        }
    }

    return nil, nil // No migration tool found
}
```

**Detection Functions:**

```go
// detectMakefile checks for Makefile with migrate target
func detectMakefile(path string) (*MigrationTool, error) {
    makefilePath := filepath.Join(path, "Makefile")
    content, err := os.ReadFile(makefilePath)
    if os.IsNotExist(err) {
        return nil, nil
    }
    if err != nil {
        return nil, err
    }

    // Check for migrate-related targets
    targets := []string{"migrate:", "db-migrate:", "db:migrate:"}
    for _, target := range targets {
        if strings.Contains(string(content), target) {
            targetName := strings.TrimSuffix(target, ":")
            return &MigrationTool{
                Name:        "makefile",
                Command:     []string{"make", targetName},
                Path:        path,
                Description: fmt.Sprintf("Makefile target: %s", targetName),
            }, nil
        }
    }
    return nil, nil
}

// detectPrisma checks for Prisma schema
func detectPrisma(path string) (*MigrationTool, error) {
    locations := []string{
        filepath.Join(path, "prisma", "schema.prisma"),
        filepath.Join(path, "schema.prisma"),
    }

    for _, loc := range locations {
        if _, err := os.Stat(loc); err == nil {
            return &MigrationTool{
                Name:        "prisma",
                Command:     []string{"npx", "prisma", "migrate", "deploy"},
                Path:        path,
                Description: "Prisma migrations",
            }, nil
        }
    }
    return nil, nil
}

// detectDrizzle checks for Drizzle config
func detectDrizzle(path string) (*MigrationTool, error) {
    configs := []string{
        filepath.Join(path, "drizzle.config.ts"),
        filepath.Join(path, "drizzle.config.js"),
    }

    for _, cfg := range configs {
        if _, err := os.Stat(cfg); err == nil {
            return &MigrationTool{
                Name:        "drizzle",
                Command:     []string{"npx", "drizzle-kit", "migrate"},
                Path:        path,
                Description: "Drizzle migrations",
            }, nil
        }
    }
    return nil, nil
}

// detectAlembic checks for Alembic (Python)
func detectAlembic(path string) (*MigrationTool, error) {
    indicators := []string{
        filepath.Join(path, "alembic.ini"),
        filepath.Join(path, "alembic"),
    }

    for _, ind := range indicators {
        info, err := os.Stat(ind)
        if err == nil {
            if info.IsDir() || strings.HasSuffix(ind, ".ini") {
                return &MigrationTool{
                    Name:        "alembic",
                    Command:     []string{"alembic", "upgrade", "head"},
                    Path:        path,
                    Description: "Alembic migrations",
                }, nil
            }
        }
    }
    return nil, nil
}

// detectRawSQL checks for SQL migration files
func detectRawSQL(path string) (*MigrationTool, error) {
    migrationDirs := []string{
        filepath.Join(path, "migrations"),
        filepath.Join(path, "db", "migrations"),
        filepath.Join(path, "sql", "migrations"),
    }

    for _, dir := range migrationDirs {
        entries, err := os.ReadDir(dir)
        if err != nil {
            continue
        }

        for _, entry := range entries {
            if strings.HasSuffix(entry.Name(), ".sql") {
                return &MigrationTool{
                    Name:        "sql",
                    Command:     nil, // Requires manual handling
                    Path:        dir,
                    Description: fmt.Sprintf("Raw SQL files in %s (requires manual execution)", filepath.Base(dir)),
                }, nil
            }
        }
    }
    return nil, nil
}

// parseCommand splits a command string into args
func parseCommand(cmd string) []string {
    // Handle quoted strings properly
    return strings.Fields(cmd)
}
```

---

### Task 3: Implement Container Readiness Check

**File: `internal/migrate/container.go`**

```go
package migrate

import (
    "context"
    "fmt"
    "os/exec"
    "strings"
    "time"
)

// ContainerStatus represents the state of a Docker container
type ContainerStatus struct {
    Name    string
    Running bool
    Health  string // "healthy", "unhealthy", "starting", "none"
}

// CheckDatabaseContainer verifies the database container is ready
func CheckDatabaseContainer(worktreePath string) (*ContainerStatus, error) {
    // Try to find docker-compose.yml
    composeFile := findComposeFile(worktreePath)
    if composeFile == "" {
        return nil, nil // No compose file, skip check
    }

    // Get database service name (common conventions)
    dbServices := []string{"db", "database", "postgres", "mysql", "mariadb", "mongodb"}

    for _, service := range dbServices {
        status, err := getContainerStatus(worktreePath, service)
        if err != nil {
            continue // Service doesn't exist
        }
        if status != nil {
            return status, nil
        }
    }

    return nil, nil // No database container found
}

func findComposeFile(path string) string {
    names := []string{
        "docker-compose.yml",
        "docker-compose.yaml",
        "compose.yml",
        "compose.yaml",
    }

    for _, name := range names {
        fullPath := filepath.Join(path, name)
        if _, err := os.Stat(fullPath); err == nil {
            return fullPath
        }
    }
    return ""
}

func getContainerStatus(worktreePath, service string) (*ContainerStatus, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    // Use docker compose ps to check service status
    cmd := exec.CommandContext(ctx, "docker", "compose", "ps", service, "--format", "{{.State}}")
    cmd.Dir = worktreePath

    output, err := cmd.Output()
    if err != nil {
        return nil, err
    }

    state := strings.TrimSpace(string(output))
    if state == "" {
        return nil, nil
    }

    return &ContainerStatus{
        Name:    service,
        Running: state == "running",
        Health:  "none", // Could parse health status if needed
    }, nil
}

// WaitForContainer waits for a container to be ready with timeout
func WaitForContainer(worktreePath, service string, timeout time.Duration) error {
    deadline := time.Now().Add(timeout)

    for time.Now().Before(deadline) {
        status, err := getContainerStatus(worktreePath, service)
        if err == nil && status != nil && status.Running {
            return nil
        }
        time.Sleep(2 * time.Second)
    }

    return &ContainerNotReadyError{
        Service: service,
        Reason:  fmt.Sprintf("timeout after %v", timeout),
    }
}
```

---

### Task 4: Implement Migration Executor

**File: `internal/migrate/migrate.go`**

```go
package migrate

import (
    "bufio"
    "context"
    "fmt"
    "io"
    "os/exec"
    "strings"
    "time"

    "gwt/internal/config"
    "gwt/internal/output"
)

const defaultTimeout = 5 * time.Minute

// Run executes migrations for the given worktree
func Run(opts RunOptions, cfg *config.MigrationsConfig) (*Result, error) {
    result := &Result{}

    // Step 1: Detect migration tool
    tool, err := Detect(opts.WorktreePath, cfg)
    if err != nil {
        return nil, &DetectionError{Path: opts.WorktreePath, Err: err}
    }

    if tool == nil {
        result.Skipped = true
        result.Reason = "no migration tool detected"
        return result, nil
    }

    result.Tool = tool

    // Step 2: Handle raw SQL (no auto-execution)
    if tool.Name == "sql" {
        result.Skipped = true
        result.Reason = fmt.Sprintf("raw SQL files found in %s - manual execution required", tool.Path)
        return result, nil
    }

    // Step 3: Check container readiness (unless skipped)
    if !opts.SkipContainerCheck {
        status, err := CheckDatabaseContainer(opts.WorktreePath)
        if err != nil {
            output.Verbose(fmt.Sprintf("Container check failed: %v", err))
        }
        if status != nil && !status.Running {
            result.Skipped = true
            result.Reason = fmt.Sprintf("database container %q is not running", status.Name)
            return result, nil
        }
    }

    // Step 4: Dry run mode
    if opts.DryRun {
        result.Skipped = true
        result.Reason = fmt.Sprintf("would run: %s", strings.Join(tool.Command, " "))
        return result, nil
    }

    // Step 5: Execute migrations
    output.Info(fmt.Sprintf("Running %s migrations...", tool.Name))
    output.Verbose(fmt.Sprintf("Command: %s", strings.Join(tool.Command, " ")))

    stdout, stderr, exitCode, err := runCommand(opts, tool)
    result.Output = stdout + stderr

    if err != nil || exitCode != 0 {
        result.Success = false
        result.Error = &MigrationError{
            Tool:     tool.Name,
            Command:  tool.Command,
            Stderr:   stderr,
            ExitCode: exitCode,
        }
        return result, nil // Non-fatal, return result with error info
    }

    result.Success = true
    return result, nil
}

func runCommand(opts RunOptions, tool *MigrationTool) (stdout, stderr string, exitCode int, err error) {
    ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
    defer cancel()

    cmd := exec.CommandContext(ctx, tool.Command[0], tool.Command[1:]...)
    cmd.Dir = opts.WorktreePath

    // Set up pipes for streaming
    stdoutPipe, _ := cmd.StdoutPipe()
    stderrPipe, _ := cmd.StderrPipe()

    if err := cmd.Start(); err != nil {
        return "", "", -1, err
    }

    // Stream output if verbose
    var stdoutBuf, stderrBuf strings.Builder

    done := make(chan struct{})
    go func() {
        streamOutput(stdoutPipe, &stdoutBuf, opts.Verbose, "")
        done <- struct{}{}
    }()
    go func() {
        streamOutput(stderrPipe, &stderrBuf, opts.Verbose, "stderr: ")
        done <- struct{}{}
    }()

    // Wait for output goroutines
    <-done
    <-done

    err = cmd.Wait()
    exitCode = 0
    if exitErr, ok := err.(*exec.ExitError); ok {
        exitCode = exitErr.ExitCode()
        err = nil // Don't treat non-zero exit as error
    }

    return stdoutBuf.String(), stderrBuf.String(), exitCode, err
}

func streamOutput(r io.Reader, buf *strings.Builder, verbose bool, prefix string) {
    scanner := bufio.NewScanner(r)
    for scanner.Scan() {
        line := scanner.Text()
        buf.WriteString(line)
        buf.WriteString("\n")
        if verbose {
            output.Verbose(prefix + line)
        }
    }
}
```

---

### Task 5: Integrate into Create Command

**File: `internal/cli/create.go`** (modifications)

Add helper function after file copying section:

```go
// runMigrations executes database migrations for the new worktree
func runMigrations(worktreePath string, cfg *config.Config) error {
    var migrateCfg *config.MigrationsConfig
    if cfg != nil {
        migrateCfg = &cfg.Migrations
    }

    opts := migrate.RunOptions{
        WorktreePath: worktreePath,
        Verbose:      verboseMode,
    }

    result, err := migrate.Run(opts, migrateCfg)
    if err != nil {
        return err
    }

    if result.Skipped {
        output.Verbose(fmt.Sprintf("Migrations skipped: %s", result.Reason))
        return nil
    }

    if !result.Success {
        output.Warning(fmt.Sprintf("Migration failed: %v", result.Error))
        if result.Output != "" {
            output.Verbose("Migration output:\n" + result.Output)
        }
        return result.Error // Return error but caller treats as non-fatal
    }

    output.Success(fmt.Sprintf("Migrations completed (%s)", result.Tool.Name))
    return nil
}
```

Modify `runCreate` function to call migrations:

```go
// In runCreate(), after file copying and before hooks:

// Run migrations
if !createOpts.SkipMigrations {
    if err := runMigrations(result.Path, cfg); err != nil {
        output.Warning(fmt.Sprintf("Migration had errors: %v", err))
        // Non-fatal - continue with worktree creation
    }
}
```

---

### Task 6: Write Tests

**File: `internal/migrate/detect_test.go`**

```go
package migrate

import (
    "os"
    "path/filepath"
    "testing"

    "gwt/internal/config"
)

func TestDetectMakefile(t *testing.T) {
    tests := []struct {
        name        string
        content     string
        expectTool  bool
        expectName  string
    }{
        {
            name:       "migrate target",
            content:    "migrate:\n\techo running",
            expectTool: true,
            expectName: "makefile",
        },
        {
            name:       "db-migrate target",
            content:    "db-migrate:\n\techo running",
            expectTool: true,
            expectName: "makefile",
        },
        {
            name:       "no migrate target",
            content:    "build:\n\techo building",
            expectTool: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            dir := t.TempDir()
            if err := os.WriteFile(filepath.Join(dir, "Makefile"), []byte(tt.content), 0644); err != nil {
                t.Fatal(err)
            }

            tool, err := detectMakefile(dir)
            if err != nil {
                t.Fatalf("unexpected error: %v", err)
            }

            if tt.expectTool {
                if tool == nil {
                    t.Error("expected tool, got nil")
                } else if tool.Name != tt.expectName {
                    t.Errorf("expected name %q, got %q", tt.expectName, tool.Name)
                }
            } else if tool != nil {
                t.Errorf("expected nil tool, got %+v", tool)
            }
        })
    }
}

func TestDetectPrisma(t *testing.T) {
    tests := []struct {
        name       string
        setup      func(dir string) error
        expectTool bool
    }{
        {
            name: "prisma/schema.prisma",
            setup: func(dir string) error {
                prismaDir := filepath.Join(dir, "prisma")
                if err := os.Mkdir(prismaDir, 0755); err != nil {
                    return err
                }
                return os.WriteFile(filepath.Join(prismaDir, "schema.prisma"), []byte(""), 0644)
            },
            expectTool: true,
        },
        {
            name: "root schema.prisma",
            setup: func(dir string) error {
                return os.WriteFile(filepath.Join(dir, "schema.prisma"), []byte(""), 0644)
            },
            expectTool: true,
        },
        {
            name:       "no prisma",
            setup:      func(dir string) error { return nil },
            expectTool: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            dir := t.TempDir()
            if err := tt.setup(dir); err != nil {
                t.Fatal(err)
            }

            tool, err := detectPrisma(dir)
            if err != nil {
                t.Fatalf("unexpected error: %v", err)
            }

            if tt.expectTool && tool == nil {
                t.Error("expected tool, got nil")
            } else if !tt.expectTool && tool != nil {
                t.Errorf("expected nil tool, got %+v", tool)
            }
        })
    }
}

func TestDetectWithConfigOverride(t *testing.T) {
    dir := t.TempDir()
    cfg := &config.MigrationsConfig{
        AutoDetect: true,
        Command:    "custom migrate --prod",
    }

    tool, err := Detect(dir, cfg)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    if tool == nil {
        t.Fatal("expected tool, got nil")
    }

    if tool.Name != "custom" {
        t.Errorf("expected name 'custom', got %q", tool.Name)
    }

    expectedCmd := []string{"custom", "migrate", "--prod"}
    if len(tool.Command) != len(expectedCmd) {
        t.Errorf("expected command %v, got %v", expectedCmd, tool.Command)
    }
}

func TestDetectAutoDetectDisabled(t *testing.T) {
    dir := t.TempDir()

    // Create a Makefile that would normally be detected
    os.WriteFile(filepath.Join(dir, "Makefile"), []byte("migrate:\n\techo"), 0644)

    cfg := &config.MigrationsConfig{
        AutoDetect: false,
        Command:    "", // No custom command
    }

    tool, err := Detect(dir, cfg)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    if tool != nil {
        t.Errorf("expected nil when auto-detect disabled, got %+v", tool)
    }
}
```

---

## Configuration Reference

The config structure already exists in `internal/config/config.go`:

```yaml
# .worktree.yaml
migrations:
  auto_detect: true           # Enable/disable auto-detection (default: true)
  command: "make db-migrate"  # Override auto-detected command (optional)
```

---

## Integration Flow

```
gwt create feature/xyz
    │
    ├── 1. Create worktree (existing)
    │
    ├── 2. Copy gitignored files (Phase 6)
    │
    ├── 3. Install dependencies (Phase 8)
    │
    ├── 4. Run migrations (Phase 9)  ◄── NEW
    │   ├── Detect migration tool
    │   ├── Check database container status
    │   ├── Execute migrations with streaming output
    │   └── Report result (non-fatal on failure)
    │
    └── 5. Execute hooks (Phase 10)
```

---

## Edge Cases to Handle

1. **No migration tool detected** - Skip silently, log at verbose level
2. **Database container not running** - Warn and skip, suggest starting container
3. **Migration fails** - Warn but continue worktree creation
4. **Custom command in config** - Use exactly as specified
5. **Auto-detect disabled** - Only use config command or skip
6. **Raw SQL files** - Detect but don't auto-execute, inform user
7. **Multiple package managers in monorepo** - Run from worktree root
8. **Windows compatibility** - Use `exec.Command` not shell, handle paths

---

## Success Criteria

- [ ] `gwt create` detects and runs Makefile migrate targets
- [ ] `gwt create` detects and runs Prisma migrations
- [ ] `gwt create` detects and runs Drizzle migrations
- [ ] `gwt create` detects and runs Alembic migrations
- [ ] `gwt create` informs user of raw SQL files without executing
- [ ] `--skip-migrations` flag prevents migration execution
- [ ] Container readiness check warns if database not running
- [ ] Migration failures don't block worktree creation
- [ ] Custom migration command from config works
- [ ] Verbose mode streams migration output in real-time
- [ ] All tests pass

---

## Files to Create/Modify

**New Files:**
- `internal/migrate/errors.go`
- `internal/migrate/result.go`
- `internal/migrate/detect.go`
- `internal/migrate/migrate.go`
- `internal/migrate/container.go`
- `internal/migrate/detect_test.go`
- `internal/migrate/migrate_test.go`
- `internal/migrate/container_test.go`

**Modified Files:**
- `internal/cli/create.go` - Add migration execution call

---

## Dependencies

- Phase 3 (Config) - Required, complete
- Phase 4 (Create) - Required, complete
- Phase 7 (Docker) - Optional, for container checks
- Phase 8 (Dependencies) - Related pattern, can be parallel

---

## Estimated Complexity

- Detection logic: Medium (multiple tools to support)
- Container checks: Low (simple docker compose commands)
- Execution: Low (follows existing exec patterns)
- Integration: Low (single call point in create.go)
- Testing: Medium (need temp directories, mock scenarios)
