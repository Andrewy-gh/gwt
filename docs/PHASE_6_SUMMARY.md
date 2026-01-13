# Phase 6: File Copying - Summary

**Status:** ✓ Complete
**Date Completed:** 2026-01-12

## Overview

Phase 6 implements gitignored file discovery and copying functionality for the GWT project. When creating a new worktree, GWT can now automatically discover gitignored files from the main worktree, apply pattern matching rules from configuration, and copy selected files to the new worktree with progress tracking.

This is particularly useful for copying configuration files (`.env`, `.env.local`), development tools (`.claude/`), and other untracked assets that are needed for the project to run but aren't committed to version control.

## Deliverables

### 1. Gitignored File Discovery (`internal/copy/discover.go`)

**Features:**
- Uses `git status --ignored --porcelain` to find ignored files
- Parses git output to extract file paths
- Calculates file sizes (individual files and recursive directories)
- Handles directories and files uniformly

**Core Types:**
```go
type IgnoredFile struct {
    Path  string // Relative path from repo root
    IsDir bool   // Whether this is a directory
    Size  int64  // Size in bytes (calculated recursively for dirs)
}
```

**Functions:**
```go
// Main discovery function
func DiscoverIgnored(repoPath string) ([]IgnoredFile, error)

// Parse git status output
func parseIgnoredOutput(output string) []string

// Calculate directory size recursively
func calculateDirectorySize(dirPath string) (int64, error)
```

**Git Command Used:**
```bash
git status --ignored --porcelain
```

### 2. Pattern Matching (`internal/copy/match.go`)

**Features:**
- Glob pattern matching using `github.com/bmatcuk/doublestar/v4`
- Supports multiple pattern types:
  - Exact matches: `.env`
  - Simple globs: `*.log`
  - Double-star globs: `**/.env`
  - Directory patterns: `node_modules` (matches anywhere in path)
- Exclude patterns take precedence over default patterns
- Default exclusions for common dependency directories

**Core Types:**
```go
type MatchResult int

const (
    MatchNone     MatchResult = iota // No match
    MatchDefault                      // Matched copy_defaults
    MatchExclude                      // Matched copy_exclude
)

type PatternMatcher struct {
    Defaults []string // Patterns for pre-selection
    Excludes []string // Patterns for exclusion
}
```

**Default Exclusions:**
```go
var DefaultExcludes = []string{
    "node_modules", "vendor", ".venv", "venv",
    "__pycache__", ".pycache", "target", "dist",
    "build", ".gradle", ".maven", "pkg", "bin",
    ".git", ".svn", ".hg",
}
```

**Functions:**
```go
func NewPatternMatcher(defaults, excludes []string) *PatternMatcher
func (m *PatternMatcher) Match(path string) MatchResult
func matchPattern(pattern, path string) bool
```

### 3. File Selection (`internal/copy/selection.go`)

**Features:**
- File selection model with match status
- Size calculation and formatting (B, KB, MB, GB, etc.)
- Filter visible files (excludes hidden by patterns)
- Get selected files for copying
- Toggle selection (for future TUI integration)

**Core Types:**
```go
type SelectableFile struct {
    IgnoredFile          // Embedded file info
    Selected    bool     // Whether selected for copying
    MatchResult MatchResult // How it matched patterns
}

type Selection struct {
    Files        []SelectableFile
    TotalSize    int64 // Total size of all files
    SelectedSize int64 // Total size of selected files
}
```

**Functions:**
```go
func NewSelection(files []IgnoredFile, matcher *PatternMatcher) *Selection
func (s *Selection) FilterVisible() []SelectableFile
func (s *Selection) GetSelected() []SelectableFile
func (s *Selection) Toggle(index int)
func (s *Selection) SelectAll()
func (s *Selection) DeselectAll()
func FormatSize(bytes int64) string
```

**Size Formatting Examples:**
- `42 B` - Bytes
- `1.5 KB` - Kilobytes
- `23.4 MB` - Megabytes
- `2.1 GB` - Gigabytes

### 4. File/Directory Copying (`internal/copy/copy.go`)

**Features:**
- Copy individual files with permission preservation
- Copy directories recursively
- Progress tracking via callbacks
- Non-fatal error collection (continue on errors)
- Parent directory creation as needed
- Efficient `io.Copy()` for file content

**Core Types:**
```go
type CopyProgress struct {
    CurrentFile   string
    FilesDone     int
    FilesTotal    int
    BytesDone     int64
    BytesTotal    int64
}

type CopyOptions struct {
    SourceDir    string
    TargetDir    string
    Files        []SelectableFile
    OnProgress   ProgressCallback
    PreserveMode bool
}

type CopyResult struct {
    FilesCopied  int
    BytesCopied  int64
    Errors       []CopyError
}
```

**Functions:**
```go
func Copy(opts CopyOptions) (*CopyResult, error)
func copyFile(src, dst string, preserveMode bool) error
func copyDirectory(src, dst string, preserveMode bool) error
```

### 5. Error Types (`internal/copy/errors.go`)

**Custom Error Types:**
```go
var (
    ErrNoSourceDirectory = errors.New("source directory does not exist")
    ErrTargetExists      = errors.New("target already exists")
    ErrCopyFailed        = errors.New("file copy failed")
    ErrPatternInvalid    = errors.New("invalid glob pattern")
    ErrPermissionDenied  = errors.New("permission denied")
)

type CopyError struct {
    Path string
    Op   string
    Err  error
}
```

### 6. Progress Display (`internal/output/progress.go`)

**Features:**
- Simple text-based progress bar
- Percentage display
- Current file name display
- Progress count (current/total)

**Core Types:**
```go
type ProgressBar struct {
    total   int
    current int
    width   int
}
```

**Functions:**
```go
func NewProgressBar(total, width int) *ProgressBar
func (p *ProgressBar) Update(current int, message string)
func (p *ProgressBar) Done()
```

**Progress Output Example:**
```
Copying files... [████████░░░░░░░░░░░░] 45% (23/50) config/.env
```

### 7. Integration with Create Command (`internal/cli/create.go`)

**Modified Functions:**
- `runCreate()` - Added file copying step after worktree creation
- `copyIgnoredFiles()` - New function to orchestrate file copying

**New Flag:**
```go
--skip-copy    Skip copying gitignored files
```

**Workflow:**
1. Create worktree (existing functionality)
2. Discover ignored files in main worktree
3. Create pattern matcher from config
4. Create selection and pre-select defaults
5. Copy selected files with progress tracking
6. Report results

**Integration is non-fatal:** If file copying fails, the worktree creation still succeeds. Errors are reported as warnings.

## Configuration Integration

File copying behavior is controlled by `.worktree.yaml`:

```yaml
# Patterns for files to pre-select for copying
copy_defaults:
  - ".env"
  - "**/.env"
  - "**/.env.local"
  - ".claude/"
  - "**/*.local.md"
  - "**/setenv.sh"

# Patterns for files to exclude from selection (hidden)
copy_exclude:
  - "node_modules"
  - "vendor"
  - ".venv"
  - "__pycache__"
  - "target"
  - "dist"
  - "build"
  - "*.log"
```

## Testing

### Test Files Created
- `internal/copy/discover_test.go` - 156 lines, 6 tests
- `internal/copy/match_test.go` - 206 lines, 8 tests
- `internal/copy/copy_test.go` - 262 lines, 10 tests
- `internal/copy/selection_test.go` - 207 lines, 8 tests

### Test Coverage
- Git status output parsing
- File size calculation (files and directories)
- Pattern matching (exact, glob, double-star, directory)
- Exclude precedence over defaults
- File and directory copying
- Permission preservation
- Progress callback invocation
- Error collection and handling
- Selection creation and filtering
- Size formatting

**Total New Tests:** 32

## Files Created/Modified

| File | Lines | Purpose |
|------|-------|---------|
| `internal/copy/discover.go` | 123 | Gitignored file discovery |
| `internal/copy/match.go` | 129 | Pattern matching logic |
| `internal/copy/copy.go` | 216 | File/directory copying |
| `internal/copy/selection.go` | 127 | File selection with sizes |
| `internal/copy/errors.go` | 29 | Custom error types |
| `internal/copy/discover_test.go` | 156 | Discovery tests |
| `internal/copy/match_test.go` | 206 | Pattern matching tests |
| `internal/copy/copy_test.go` | 262 | Copy operation tests |
| `internal/copy/selection_test.go` | 207 | Selection tests |
| `internal/output/progress.go` | 57 | Progress bar display |
| `internal/cli/create.go` | +98 | Integration with create command |
| **Total** | **~1,610** | **New/modified code** |

## Dependencies Added

**External Package:**
- `github.com/bmatcuk/doublestar/v4` - Glob pattern matching with `**` support

Added to `go.mod` and `go.sum`.

## Code Quality

- All code passes `go build`
- All code passes `go vet`
- 32 new tests passing
- Full project test suite passing
- Cross-platform path handling (uses `filepath` package)
- Proper error handling with custom error types
- Non-fatal error collection during copying

## Integration with Previous Phases

**Phase 1 (Foundation):**
- Uses output utilities (`output.Info()`, `output.Warning()`, `output.Success()`)
- Inherits global flags (`--verbose`, `--quiet`)

**Phase 2 (Git Operations):**
- Uses `git status --ignored` via command execution
- Leverages repository path detection

**Phase 3 (Config):**
- Reads `copy_defaults` from configuration
- Reads `copy_exclude` from configuration
- Merges default exclusions with user config

**Phase 4 (Create):**
- Integrated into `gwt create` command workflow
- Executes after successful worktree creation
- Non-fatal errors don't block worktree creation

## Design Decisions

1. **Non-fatal integration** - File copying errors don't fail worktree creation
2. **Default exclusions** - Auto-exclude dependency directories even without config
3. **Pattern precedence** - Exclude patterns override default patterns
4. **Size calculation** - Show file sizes for informed decisions (future TUI)
5. **Progress tracking** - Real-time feedback during long copy operations
6. **Permission preservation** - Maintain file permissions from source
7. **Glob library** - Use `doublestar` for robust `**` pattern support
8. **Error collection** - Continue copying on single file errors, report at end

## Usage Examples

### Basic Usage
```bash
# Create worktree with automatic file copying
gwt create -b feature-auth
# Automatically copies files matching copy_defaults

# Skip file copying
gwt create -b feature-auth --skip-copy
```

### Expected Output
```
Creating worktree...
✓ Created worktree at /path/to/project-feature-auth

Copying files...
Copying 5 files (234.5 KB)...
[████████████████████] 100% (5/5) .env

✓ Copied 5 files (234.5 KB)
```

## Edge Cases Handled

1. **No ignored files** - Reports "No gitignored files to copy" and continues
2. **No selected files** - Reports "No files selected for copying" and continues
3. **File permission errors** - Collects errors, continues with remaining files
4. **Target file exists** - Skips file and reports error
5. **Large files** - Handles efficiently with streaming copy
6. **Empty directories** - Only copies if contains selected files
7. **Symlinks** - Currently skipped to avoid circular references
8. **Cross-platform paths** - Uses `filepath` package for OS compatibility

## Known Limitations

1. **No interactive selection yet** - Files matching `copy_defaults` are automatically selected (TUI in Phase 12)
2. **Symlinks not supported** - Currently skipped during discovery/copy
3. **No overwrite option** - Existing target files are skipped
4. **No size limits** - Large files copied without warnings (could add `--max-size` flag)

## Future Enhancements (Phase 11-12)

- TUI file selection with checkboxes
- Interactive pattern editor
- File preview before copy
- Undo/rollback copied files
- Symlink handling options
- Size limits and warnings for large files
- Overwrite confirmation for existing files

## Next Steps

With Phase 6 complete, the core worktree creation workflow now includes intelligent file copying:

**Phase 7: Docker Compose Scaffolding**
- Detect Docker Compose files
- Symlink or copy data directories
- Generate port-offset overrides

**Phase 8-10: Post-Creation Features**
- Dependency installation
- Database migrations
- Lifecycle hooks

**Phase 11-12: TUI**
- Interactive file selection
- Visual progress indicators
- File size-based filtering
