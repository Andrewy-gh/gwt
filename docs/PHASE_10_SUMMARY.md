# Phase 10: Post-Setup Hooks - Implementation Summary

**Status:** ✅ Complete

---

## Overview

Phase 10 implements lifecycle hooks that execute custom commands after worktree creation and deletion. Hooks receive context through environment variables and run with proper error handling and timeout protection.

---

## What Was Implemented

### Core Package: `internal/hooks/`

**New Files Created:**
- `errors.go` - Custom error types (HookError, ErrHookTimeout, etc.)
- `env.go` - Environment variable setup (BuildEnvironment, BuildEnvironmentMap)
- `exec.go` - Cross-platform command execution with timeout
- `hooks.go` - Hook executor with sequential execution
- `env_test.go` - Environment variable tests
- `exec_test.go` - Command execution tests
- `hooks_test.go` - Hook executor tests
- `testdata/` - Test fixtures (shell scripts)

### Integration Points

**Modified Files:**
- `internal/cli/create.go` - Added `--skip-hooks` flag and runPostCreateHooks()
- `internal/cli/delete.go` - Added `--skip-hooks` flag and runPostDeleteHooks()

---

## Key Features

### 1. Environment Variables

Hooks receive context through GWT_* environment variables:

| Variable | Description |
|----------|-------------|
| `GWT_WORKTREE_PATH` | Absolute path to the new/deleted worktree |
| `GWT_BRANCH` | Branch name |
| `GWT_MAIN_WORKTREE` | Path to main worktree |
| `GWT_REPO_PATH` | Repository root path |
| `GWT_HOOK_TYPE` | "post_create" or "post_delete" |

### 2. Cross-Platform Execution

- **Unix/Linux/macOS**: Commands run through `/bin/sh -c`
- **Windows**: Commands run through `cmd.exe /C`
- Environment variable expansion works on both platforms

### 3. Error Handling

- Hook failures are **non-fatal** - they don't prevent worktree operations
- Failed hooks are reported as warnings with detailed error messages
- Multiple hook failures are collected and reported together
- Execution continues even if individual hooks fail

### 4. Timeout Protection

- Default 5-minute timeout per hook prevents hung processes
- Timeout errors are caught and reported distinctly
- Configurable timeout in ExecOptions for testing

### 5. Sequential Execution

- Hooks execute in the order defined in config
- Predictable execution order aids debugging
- Later hooks can depend on earlier ones completing

---

## Configuration

Hooks are configured in `.worktree.yaml`:

```yaml
hooks:
  post_create:
    - "echo 'Setting up worktree in $GWT_WORKTREE_PATH'"
    - "npm install"
    - "cp .env.example .env"
    - "./scripts/setup-dev.sh"

  post_delete:
    - "docker compose -p gwt-$GWT_BRANCH down -v"
    - "echo 'Cleaned up resources for $GWT_BRANCH'"
```

### Hook Types

**post_create:**
- Runs after worktree creation, file copying, Docker setup, dependency installation, and migrations
- Working directory: new worktree path
- Access to all GWT_* environment variables including GWT_WORKTREE_PATH

**post_delete:**
- Runs after worktree deletion
- Working directory: repository root (worktree no longer exists)
- GWT_WORKTREE_PATH is omitted (worktree deleted)
- Access to GWT_BRANCH, GWT_MAIN_WORKTREE, GWT_REPO_PATH

---

## Command-Line Flags

### Create Command

```bash
# Run with hooks (default)
gwt create -b feature-auth

# Skip hooks
gwt create -b feature-auth --skip-hooks
```

### Delete Command

```bash
# Run with hooks (default)
gwt delete feature-auth

# Skip hooks
gwt delete feature-auth --skip-hooks
```

---

## Testing

### Test Coverage

**20 unit tests** covering:
- Environment variable building and merging
- Command execution success/failure
- Timeout handling
- Working directory respect
- Environment variable passing
- Cross-platform shell selection
- Hook executor initialization
- Empty/successful/failed/mixed hook execution
- Hook type detection

### Test Results

```
=== RUN   TestBuildEnvironment
--- PASS: TestBuildEnvironment
=== RUN   TestBuildEnvironmentWithEmptyValues
--- PASS: TestBuildEnvironmentWithEmptyValues
=== RUN   TestBuildEnvironmentMergesWithExisting
--- PASS: TestBuildEnvironmentMergesWithExisting
=== RUN   TestExecuteCommandSuccess
--- PASS: TestExecuteCommandSuccess
=== RUN   TestExecuteCommandTimeout
--- PASS: TestExecuteCommandTimeout
... (20 tests total)

PASS
ok  	github.com/Andrewy-gh/gwt/internal/hooks	10.756s
```

---

## Design Decisions

### 1. Non-Fatal Hook Errors

**Decision:** Hook failures don't prevent worktree operations.

**Rationale:**
- Worktree creation/deletion is the primary operation
- Hooks are supplementary convenience features
- User can re-run hooks manually if needed
- Matches Phase 6 pattern (file copy errors don't fail creation)

### 2. Sequential Execution

**Decision:** Hooks execute sequentially, not in parallel.

**Rationale:**
- Predictable execution order
- Later hooks may depend on earlier ones
- Easier debugging
- Matches user expectations from shell scripts

### 3. Shell Execution

**Decision:** Commands run through the system shell.

**Rationale:**
- Allows shell features (pipes, redirects, variable expansion)
- Matches how users would run commands manually
- Consistent with config examples like `npm install`

### 4. Timeout Protection

**Decision:** Default 5-minute timeout per hook.

**Rationale:**
- Prevents hung processes blocking GWT
- Long enough for most operations (npm install, migrations)
- Can be made configurable in future if needed

---

## Example Use Cases

### Development Environment Setup

```yaml
hooks:
  post_create:
    - "cp .env.example .env"
    - "npm install"
    - "npm run db:setup"
    - "code $GWT_WORKTREE_PATH"  # Open in VS Code
```

### Docker Cleanup

```yaml
hooks:
  post_delete:
    - "docker compose -p gwt-$GWT_BRANCH down -v"
    - "docker volume rm gwt-$GWT_BRANCH-db-data 2>/dev/null || true"
```

### Monorepo Setup

```yaml
hooks:
  post_create:
    - "cd packages/frontend && npm install"
    - "cd packages/backend && npm install"
    - "npm run bootstrap"
```

### Notification

```yaml
hooks:
  post_create:
    - "echo 'Worktree created: $GWT_BRANCH' | notify-send"
  post_delete:
    - "echo 'Worktree deleted: $GWT_BRANCH' | notify-send"
```

---

## Edge Cases Handled

| Scenario | Handling |
|----------|----------|
| Empty hook list | Skip silently, no output |
| Hook command not found | Capture error, report as warning |
| Hook produces only stderr | Show stderr, check exit code |
| Hook times out | Report timeout error, continue |
| Working directory doesn't exist | Use repo root as fallback (post_delete) |
| Permission denied | Report error, continue |
| User presses Ctrl+C | Let parent process handle interrupt |

---

## Files Changed

| File | Type | Lines | Description |
|------|------|-------|-------------|
| `internal/hooks/errors.go` | New | 35 | Error types |
| `internal/hooks/env.go` | New | 50 | Environment setup |
| `internal/hooks/exec.go` | New | 80 | Command execution |
| `internal/hooks/hooks.go` | New | 120 | Hook executor |
| `internal/hooks/env_test.go` | New | 130 | Environment tests |
| `internal/hooks/exec_test.go` | New | 170 | Execution tests |
| `internal/hooks/hooks_test.go` | New | 280 | Executor tests |
| `internal/hooks/testdata/*.sh` | New | 15 | Test fixtures |
| `internal/cli/create.go` | Modified | +40 | Integration |
| `internal/cli/delete.go` | Modified | +45 | Integration |
| **Total** | | **~965** | |

---

## Verification Checklist

- [X] `go build ./...` succeeds
- [X] `go test ./internal/hooks/...` passes (20/20 tests)
- [X] `gwt create` runs post_create hooks
- [X] `gwt create --skip-hooks` skips hooks
- [X] `gwt delete` runs post_delete hooks
- [X] `gwt delete --skip-hooks` skips hooks
- [X] Hook failures show warnings but don't fail operations
- [X] GWT_* variables are accessible in hooks
- [X] Hooks run in correct working directory
- [X] Timeouts are enforced
- [X] Works on Windows (cmd.exe)
- [X] Manual testing completed successfully

---

## What's Next

With Phase 10 complete, the core CLI functionality is now feature-complete. Next phases focus on the interactive TUI:

- **Phase 11**: TUI Framework (Bubble Tea, Lip Gloss, components)
- **Phase 12**: TUI Views (interactive selection, visual feedback)
- **Phase 13**: Integration & Polish

---

## Lessons Learned

1. **Cross-platform testing is essential** - Windows timeout command works differently than expected
2. **Test with real directories** - Hook execution requires valid working directories
3. **Environment variable syntax varies** - `$VAR` (Unix) vs `%VAR%` (Windows)
4. **Verbose output is valuable** - Helps users understand what hooks are doing
5. **Non-fatal errors increase reliability** - Hooks shouldn't block primary operations
