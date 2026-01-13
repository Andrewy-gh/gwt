# Phase 10: Post-Setup Hooks - Implementation Plan

## Overview

Phase 10 implements hook execution for GWT, allowing users to run custom commands after worktree creation and deletion. Hooks are configured in `.worktree.yaml` and execute with context-aware environment variables.

**Requirements from IMPLEMENTATION_PHASES.md:**
- [ ] Implement hook execution from config
- [ ] Set up GWT_* environment variables
- [ ] Run hooks in new worktree directory
- [ ] Support post_create and post_delete hooks

---

## Current State

The groundwork for Phase 10 is **already in place**:

| Component | Status | Location |
|-----------|--------|----------|
| `HooksConfig` struct | ✓ Defined | `internal/config/config.go:33-37` |
| Default config | ✓ Includes hooks | `internal/config/defaults.go` |
| Config template | ✓ Documents hooks | `internal/config/template.go` |
| Example hooks | ✓ In project config | `.worktree.yaml` |
| Integration point | ✓ TODO placeholder | `internal/cli/create.go:286` |

---

## New Package Structure

Create `internal/hooks/` with the following files:

```
internal/hooks/
├── hooks.go          # Main hook execution orchestration
├── hooks_test.go     # Unit tests
├── env.go            # Environment variable setup (GWT_*)
├── env_test.go       # Environment tests
├── exec.go           # General command execution utility
├── exec_test.go      # Execution tests
├── errors.go         # Custom error types
└── testdata/         # Test fixtures (scripts)
```

---

## Task Breakdown

### Task 1: Create Error Types

**File:** `internal/hooks/errors.go`

```go
package hooks

import (
    "errors"
    "fmt"
)

var (
    ErrHookFailed        = errors.New("hook execution failed")
    ErrNoHooksConfigured = errors.New("no hooks configured")
    ErrInvalidHookType   = errors.New("invalid hook type")
    ErrHookTimeout       = errors.New("hook execution timed out")
)

type HookError struct {
    Command  string
    ExitCode int
    Stderr   string
    Err      error
}

func (e *HookError) Error() string {
    if e.Stderr != "" {
        return fmt.Sprintf("hook '%s' failed (exit %d): %s", e.Command, e.ExitCode, e.Stderr)
    }
    return fmt.Sprintf("hook '%s' failed (exit %d)", e.Command, e.ExitCode)
}

func (e *HookError) Unwrap() error {
    return e.Err
}
```

---

### Task 2: Implement Environment Variables

**File:** `internal/hooks/env.go`

Environment variables to expose:

| Variable | Description |
|----------|-------------|
| `GWT_WORKTREE_PATH` | Absolute path to the new/deleted worktree |
| `GWT_BRANCH` | Branch name |
| `GWT_MAIN_WORKTREE` | Path to main worktree |
| `GWT_REPO_PATH` | Repository root (same as main worktree for non-bare) |
| `GWT_HOOK_TYPE` | "post_create" or "post_delete" |

```go
package hooks

import "os"

type HookEnvironment struct {
    WorktreePath     string
    WorktreeBranch   string
    MainWorktreePath string
    RepoPath         string
    HookType         string
}

// BuildEnvironment creates GWT_* environment variables for hook execution.
// Returns a slice suitable for exec.Cmd.Env (merged with current env).
func BuildEnvironment(opts HookEnvironment) []string {
    env := os.Environ()

    gwtVars := map[string]string{
        "GWT_WORKTREE_PATH": opts.WorktreePath,
        "GWT_BRANCH":        opts.WorktreeBranch,
        "GWT_MAIN_WORKTREE": opts.MainWorktreePath,
        "GWT_REPO_PATH":     opts.RepoPath,
        "GWT_HOOK_TYPE":     opts.HookType,
    }

    for k, v := range gwtVars {
        if v != "" {
            env = append(env, k+"="+v)
        }
    }

    return env
}

// BuildEnvironmentMap returns GWT_* variables as a map (for testing/display).
func BuildEnvironmentMap(opts HookEnvironment) map[string]string {
    return map[string]string{
        "GWT_WORKTREE_PATH": opts.WorktreePath,
        "GWT_BRANCH":        opts.WorktreeBranch,
        "GWT_MAIN_WORKTREE": opts.MainWorktreePath,
        "GWT_REPO_PATH":     opts.RepoPath,
        "GWT_HOOK_TYPE":     opts.HookType,
    }
}
```

---

### Task 3: Implement Command Execution

**File:** `internal/hooks/exec.go`

General command execution utility (not git-specific):

```go
package hooks

import (
    "bytes"
    "context"
    "os/exec"
    "runtime"
    "time"
)

const DefaultTimeout = 5 * time.Minute

type ExecResult struct {
    Stdout   string
    Stderr   string
    ExitCode int
}

type ExecOptions struct {
    Command string
    Dir     string
    Env     []string
    Timeout time.Duration
}

// ExecuteCommand runs a shell command with the given options.
// On Unix, uses /bin/sh -c; on Windows, uses cmd.exe /C.
func ExecuteCommand(opts ExecOptions) (*ExecResult, error) {
    timeout := opts.Timeout
    if timeout == 0 {
        timeout = DefaultTimeout
    }

    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()

    var cmd *exec.Cmd
    if runtime.GOOS == "windows" {
        cmd = exec.CommandContext(ctx, "cmd.exe", "/C", opts.Command)
    } else {
        cmd = exec.CommandContext(ctx, "/bin/sh", "-c", opts.Command)
    }

    cmd.Dir = opts.Dir
    if len(opts.Env) > 0 {
        cmd.Env = opts.Env
    }

    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    err := cmd.Run()

    result := &ExecResult{
        Stdout: stdout.String(),
        Stderr: stderr.String(),
    }

    if ctx.Err() == context.DeadlineExceeded {
        return result, ErrHookTimeout
    }

    if exitErr, ok := err.(*exec.ExitError); ok {
        result.ExitCode = exitErr.ExitCode()
        return result, nil // Non-zero exit is not a Go error
    }

    if err != nil {
        return result, err
    }

    return result, nil
}
```

---

### Task 4: Implement Hook Executor

**File:** `internal/hooks/hooks.go`

Main orchestration for hook execution:

```go
package hooks

import (
    "fmt"

    "gwt/internal/config"
    "gwt/internal/output"
)

const (
    HookTypePostCreate = "post_create"
    HookTypePostDelete = "post_delete"
)

type Executor struct {
    repoPath string
    config   *config.Config
}

type ExecuteOptions struct {
    HookType         string
    WorktreePath     string
    WorktreeBranch   string
    MainWorktreePath string
}

type ExecuteResult struct {
    Executed   int
    Successful int
    Failed     int
    Errors     []*HookError
}

func NewExecutor(repoPath string, cfg *config.Config) *Executor {
    return &Executor{
        repoPath: repoPath,
        config:   cfg,
    }
}

// Execute runs all hooks of the specified type.
// Hooks are executed sequentially; failures are collected but don't stop execution.
func (e *Executor) Execute(opts ExecuteOptions) (*ExecuteResult, error) {
    commands := e.getCommands(opts.HookType)
    if len(commands) == 0 {
        return &ExecuteResult{}, nil
    }

    result := &ExecuteResult{
        Errors: make([]*HookError, 0),
    }

    // Build environment variables
    env := BuildEnvironment(HookEnvironment{
        WorktreePath:     opts.WorktreePath,
        WorktreeBranch:   opts.WorktreeBranch,
        MainWorktreePath: opts.MainWorktreePath,
        RepoPath:         e.repoPath,
        HookType:         opts.HookType,
    })

    // Determine working directory
    workDir := opts.WorktreePath
    if workDir == "" {
        workDir = e.repoPath // For post_delete, worktree may not exist
    }

    // Execute each hook
    for _, cmd := range commands {
        result.Executed++
        output.Verbose(fmt.Sprintf("Running hook: %s", cmd))

        execResult, err := ExecuteCommand(ExecOptions{
            Command: cmd,
            Dir:     workDir,
            Env:     env,
        })

        if err != nil {
            result.Failed++
            result.Errors = append(result.Errors, &HookError{
                Command: cmd,
                Err:     err,
            })
            continue
        }

        if execResult.ExitCode != 0 {
            result.Failed++
            result.Errors = append(result.Errors, &HookError{
                Command:  cmd,
                ExitCode: execResult.ExitCode,
                Stderr:   execResult.Stderr,
            })
            continue
        }

        result.Successful++

        // Show stdout in verbose mode if present
        if execResult.Stdout != "" {
            output.Verbose(execResult.Stdout)
        }
    }

    return result, nil
}

func (e *Executor) getCommands(hookType string) []string {
    if e.config == nil {
        return nil
    }

    switch hookType {
    case HookTypePostCreate:
        return e.config.Hooks.PostCreate
    case HookTypePostDelete:
        return e.config.Hooks.PostDelete
    default:
        return nil
    }
}

// HasHooks returns true if any hooks are configured for the given type.
func (e *Executor) HasHooks(hookType string) bool {
    return len(e.getCommands(hookType)) > 0
}
```

---

### Task 5: Integrate with Create Command

**File:** `internal/cli/create.go`

**Modification 1:** Add flag to `CreateOptions` struct (~line 27):
```go
SkipHooks bool
```

**Modification 2:** Add flag registration (~line 60):
```go
createCmd.Flags().BoolVar(&createOpts.SkipHooks, "skip-hooks", false, "skip post-creation hooks")
```

**Modification 3:** Add hook execution after file copying (~line 286):
```go
// Execute post-creation hooks (Phase 10)
if !createOpts.SkipHooks && cfg != nil && len(cfg.Hooks.PostCreate) > 0 {
    output.Info(fmt.Sprintf("Running %d post-create hooks...", len(cfg.Hooks.PostCreate)))

    executor := hooks.NewExecutor(repoPath, cfg)
    hookResult, err := executor.Execute(hooks.ExecuteOptions{
        HookType:         hooks.HookTypePostCreate,
        WorktreePath:     result.Path,
        WorktreeBranch:   result.Branch,
        MainWorktreePath: mainWorktree,
    })

    if err != nil {
        output.Warning(fmt.Sprintf("Hook execution error: %v", err))
    } else if hookResult.Failed > 0 {
        output.Warning(fmt.Sprintf("Hooks: %d succeeded, %d failed", hookResult.Successful, hookResult.Failed))
        for _, hookErr := range hookResult.Errors {
            output.Warning(fmt.Sprintf("  - %s", hookErr.Error()))
        }
    } else if hookResult.Successful > 0 {
        output.Success(fmt.Sprintf("Executed %d post-create hooks", hookResult.Successful))
    }
}
```

**Add import:**
```go
"gwt/internal/hooks"
```

---

### Task 6: Integrate with Delete Command

**File:** `internal/cli/delete.go`

**Modification 1:** Add flag to `DeleteOptions` struct (~line 21):
```go
SkipHooks bool
```

**Modification 2:** Add flag registration (~line 49):
```go
deleteCmd.Flags().BoolVar(&deleteOpts.SkipHooks, "skip-hooks", false, "skip post-delete hooks")
```

**Modification 3:** Add hook execution after successful deletion (~line 169):
```go
// Execute post-delete hooks (Phase 10)
if !deleteOpts.SkipHooks && cfg != nil && len(cfg.Hooks.PostDelete) > 0 {
    executor := hooks.NewExecutor(repoPath, cfg)
    hookResult, err := executor.Execute(hooks.ExecuteOptions{
        HookType:         hooks.HookTypePostDelete,
        WorktreeBranch:   target.Worktree.Branch,
        MainWorktreePath: mainWorktree,
    })

    if err != nil {
        output.Warning(fmt.Sprintf("Post-delete hook error: %v", err))
    } else if hookResult.Failed > 0 {
        for _, hookErr := range hookResult.Errors {
            output.Warning(fmt.Sprintf("Post-delete hook failed: %s", hookErr.Error()))
        }
    }
}
```

---

### Task 7: Write Unit Tests

**File:** `internal/hooks/env_test.go`
- Test `BuildEnvironment()` includes all GWT_* variables
- Test empty values are handled
- Test environment merging with existing env

**File:** `internal/hooks/exec_test.go`
- Test successful command execution
- Test failed command (non-zero exit)
- Test command timeout
- Test working directory is respected
- Test environment variables are passed
- Test Windows vs Unix shell selection

**File:** `internal/hooks/hooks_test.go`
- Test executor initialization
- Test empty hook list returns empty result
- Test successful hook execution
- Test failed hook error collection
- Test sequential execution
- Test `HasHooks()` method

---

### Task 8: Create Test Fixtures

**File:** `internal/hooks/testdata/echo_success.sh`
```bash
#!/bin/bash
echo "Hook executed successfully"
exit 0
```

**File:** `internal/hooks/testdata/exit_failure.sh`
```bash
#!/bin/bash
echo "This hook fails" >&2
exit 1
```

**File:** `internal/hooks/testdata/print_env.sh`
```bash
#!/bin/bash
echo "GWT_WORKTREE_PATH=$GWT_WORKTREE_PATH"
echo "GWT_BRANCH=$GWT_BRANCH"
echo "GWT_HOOK_TYPE=$GWT_HOOK_TYPE"
```

---

## Design Decisions

### 1. Non-Fatal Hook Errors

Hook failures should **not** prevent worktree operations from completing. Rationale:
- Worktree creation/deletion is the primary operation
- Hooks are supplementary convenience features
- User can re-run hooks manually if needed
- Follows Phase 6 pattern (file copy errors don't fail creation)

### 2. Sequential Execution

Hooks execute **sequentially**, not in parallel. Rationale:
- Predictable execution order
- Later hooks may depend on earlier ones
- Easier debugging
- Matches user expectations from shell scripts

### 3. Shell Execution

Commands run through the system shell (`/bin/sh` or `cmd.exe`). Rationale:
- Allows shell features (pipes, redirects, env expansion)
- Matches how users would run commands manually
- Consistent with config examples like `npm install`

### 4. Timeout Protection

Default 5-minute timeout per hook. Rationale:
- Prevents hung processes blocking GWT
- Long enough for most operations (npm install, migrations)
- Can be made configurable in future if needed

---

## Edge Cases

| Scenario | Handling |
|----------|----------|
| Empty hook list | Skip silently, no output |
| Hook command not found | Capture error, report as warning |
| Hook produces only stderr | Show stderr, check exit code |
| Hook times out | Report timeout error, continue |
| Working directory doesn't exist | Use repo root as fallback |
| Permission denied | Report error, continue |
| User presses Ctrl+C | Let parent process handle interrupt |

---

## Configuration Examples

### Basic Setup

```yaml
hooks:
  post_create:
    - "npm install"
  post_delete: []
```

### Advanced Setup

```yaml
hooks:
  post_create:
    - "echo 'Setting up $GWT_BRANCH...'"
    - "npm install"
    - "cp .env.example .env"
    - "./scripts/setup-dev.sh"
  post_delete:
    - "docker compose -p gwt-$GWT_BRANCH down -v"
    - "echo 'Cleaned up resources for $GWT_BRANCH'"
```

### Monorepo Setup

```yaml
hooks:
  post_create:
    - "cd packages/frontend && npm install"
    - "cd packages/backend && npm install"
    - "npm run bootstrap"
```

---

## Files Summary

| File | Type | Est. Lines |
|------|------|------------|
| `internal/hooks/errors.go` | New | ~35 |
| `internal/hooks/env.go` | New | ~50 |
| `internal/hooks/exec.go` | New | ~80 |
| `internal/hooks/hooks.go` | New | ~120 |
| `internal/hooks/env_test.go` | New | ~60 |
| `internal/hooks/exec_test.go` | New | ~100 |
| `internal/hooks/hooks_test.go` | New | ~150 |
| `internal/cli/create.go` | Modify | +25 |
| `internal/cli/delete.go` | Modify | +20 |

**Total:** ~640 lines of new/modified code

---

## Implementation Order

1. Create `internal/hooks/errors.go` - error types
2. Create `internal/hooks/env.go` - environment setup
3. Create `internal/hooks/exec.go` - command execution
4. Create `internal/hooks/hooks.go` - hook executor
5. Create test files and fixtures
6. Modify `internal/cli/create.go` - add integration
7. Modify `internal/cli/delete.go` - add integration
8. Run full test suite
9. Manual testing with example hooks

---

## Verification Checklist

- [ ] `go build ./...` succeeds
- [ ] `go test ./internal/hooks/...` passes
- [ ] `gwt create` runs post_create hooks
- [ ] `gwt create --skip-hooks` skips hooks
- [ ] `gwt delete` runs post_delete hooks
- [ ] `gwt delete --skip-hooks` skips hooks
- [ ] Hook failures show warnings but don't fail operations
- [ ] GWT_* variables are accessible in hooks
- [ ] Hooks run in correct working directory
- [ ] Timeouts are enforced
- [ ] Works on Windows (cmd.exe)
- [ ] Works on Unix (/bin/sh)
