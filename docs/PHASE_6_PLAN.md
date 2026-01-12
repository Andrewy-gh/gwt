# Phase 6: File Copying - Implementation Plan

This document outlines the implementation plan for Phase 6 of the GWT (Git Worktree Manager) project.

---

## Overview

Phase 6 adds the ability to discover and copy gitignored files from the main worktree to newly created worktrees. This is useful for copying configuration files, environment files, and other untracked assets that are needed for the project to run.

### Goals

1. Implement gitignored file discovery via `git status --ignored`
2. Build file/directory copy with progress tracking
3. Apply `copy_defaults` and `copy_exclude` patterns from config
4. Auto-exclude dependency directories by default
5. Show file sizes in selection

---

## File Structure

Create a new `internal/copy/` package:

```
internal/copy/
├── discover.go       # Git ignored file discovery
├── discover_test.go
├── match.go          # Pattern matching logic
├── match_test.go
├── copy.go           # File/directory copying
├── copy_test.go
├── selection.go      # File selection with sizes
├── selection_test.go
└── errors.go         # Custom error types
```

---

## Task Breakdown

### Task 1: Create Package Structure and Error Types

**File:** `internal/copy/errors.go`

Define custom error types for the copy package:

```go
package copy

import "errors"

var (
    ErrNoSourceDirectory    = errors.New("source directory does not exist")
    ErrTargetExists         = errors.New("target already exists")
    ErrCopyFailed           = errors.New("file copy failed")
    ErrPatternInvalid       = errors.New("invalid glob pattern")
    ErrPermissionDenied     = errors.New("permission denied")
)

// CopyError wraps an error with file context
type CopyError struct {
    Path string
    Op   string
    Err  error
}

func (e *CopyError) Error() string {
    return fmt.Sprintf("%s %s: %v", e.Op, e.Path, e.Err)
}

func (e *CopyError) Unwrap() error {
    return e.Err
}
```

---

### Task 2: Implement Gitignored File Discovery

**File:** `internal/copy/discover.go`

Use `git status --ignored --porcelain` to find gitignored files.

**Functions to implement:**

```go
// IgnoredFile represents a gitignored file or directory
type IgnoredFile struct {
    Path     string // Relative path from repo root
    IsDir    bool   // Whether this is a directory
    Size     int64  // Size in bytes (0 for directories, calculated recursively)
}

// DiscoverIgnored finds all gitignored files in the given directory
// Returns a flat list of ignored files and directories
func DiscoverIgnored(repoPath string) ([]IgnoredFile, error)

// parseIgnoredOutput parses git status --ignored --porcelain output
// Lines starting with "!! " are ignored files/directories
func parseIgnoredOutput(output string) []string
```

**Git command:**
```bash
git status --ignored --porcelain
```

Output format:
- `!! path/to/file` - ignored file
- `!! path/to/dir/` - ignored directory (trailing slash)

**Implementation notes:**
- Run command in repo root directory
- Parse output line by line, filtering for `!!` prefix
- For each path, check if it's a file or directory using `os.Stat()`
- Calculate size for files directly, recursively for directories
- Handle symlinks appropriately (skip or follow based on config)

---

### Task 3: Implement Pattern Matching

**File:** `internal/copy/match.go`

Implement glob pattern matching for `copy_defaults` and `copy_exclude`.

**Functions to implement:**

```go
// MatchResult indicates how a file matched patterns
type MatchResult int

const (
    MatchNone     MatchResult = iota // No match
    MatchDefault                      // Matched copy_defaults (pre-selected)
    MatchExclude                      // Matched copy_exclude (hidden)
)

// PatternMatcher handles glob pattern matching
type PatternMatcher struct {
    Defaults []string // Patterns for pre-selection
    Excludes []string // Patterns for exclusion
}

// NewPatternMatcher creates a matcher from config
func NewPatternMatcher(defaults, excludes []string) *PatternMatcher

// Match checks if a path matches any patterns
// Returns the match result (exclude takes precedence over default)
func (m *PatternMatcher) Match(path string) MatchResult

// matchPattern checks if a single pattern matches the path
// Supports:
// - Exact matches: ".env"
// - Simple globs: "*.log"
// - Double-star globs: "**/.env"
// - Directory patterns: "node_modules" (matches anywhere in path)
func matchPattern(pattern, path string) bool
```

**Pattern matching rules:**
1. Exact match: `".env"` matches only `.env` at root
2. Simple glob: `"*.log"` matches `app.log`, `error.log`
3. Double-star: `"**/.env"` matches `.env` at any depth
4. Directory name: `"node_modules"` matches if anywhere in path
5. Exclude patterns take precedence over default patterns

**Implementation options:**
- Option A: Use `github.com/bmatcuk/doublestar/v4` for full glob support
- Option B: Implement simple matching with `filepath.Match()` + custom `**` handling

Recommendation: Use `doublestar` library for robust pattern matching.

---

### Task 4: Implement File Selection with Sizes

**File:** `internal/copy/selection.go`

Build the selection model for files to copy.

**Types and functions:**

```go
// SelectableFile represents a file that can be selected for copying
type SelectableFile struct {
    IgnoredFile          // Embedded file info
    Selected    bool     // Whether selected for copying
    MatchResult MatchResult // How it matched patterns
}

// Selection manages the list of selectable files
type Selection struct {
    Files      []SelectableFile
    TotalSize  int64 // Total size of all files
    SelectedSize int64 // Total size of selected files
}

// NewSelection creates a selection from discovered files and pattern matcher
func NewSelection(files []IgnoredFile, matcher *PatternMatcher) *Selection

// FilterVisible returns only files not matched by exclude patterns
func (s *Selection) FilterVisible() []SelectableFile

// GetSelected returns only selected files
func (s *Selection) GetSelected() []SelectableFile

// Toggle toggles selection of a file by index
func (s *Selection) Toggle(index int)

// SelectAll selects all visible files
func (s *Selection) SelectAll()

// DeselectAll deselects all files
func (s *Selection) DeselectAll()

// FormatSize formats bytes as human-readable string
func FormatSize(bytes int64) string
```

**Size formatting:**
```go
func FormatSize(bytes int64) string {
    const unit = 1024
    if bytes < unit {
        return fmt.Sprintf("%d B", bytes)
    }
    div, exp := int64(unit), 0
    for n := bytes / unit; n >= unit; n /= unit {
        div *= unit
        exp++
    }
    return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
```

---

### Task 5: Implement File/Directory Copying

**File:** `internal/copy/copy.go`

Build the copy engine with progress tracking.

**Types and functions:**

```go
// CopyProgress reports progress during copying
type CopyProgress struct {
    CurrentFile   string
    FilesDone     int
    FilesTotal    int
    BytesDone     int64
    BytesTotal    int64
}

// ProgressCallback is called during copying to report progress
type ProgressCallback func(progress CopyProgress)

// CopyOptions configures the copy operation
type CopyOptions struct {
    SourceDir    string           // Source directory (main worktree)
    TargetDir    string           // Target directory (new worktree)
    Files        []SelectableFile // Files to copy
    OnProgress   ProgressCallback // Progress callback (optional)
    PreserveMode bool             // Preserve file permissions
}

// CopyResult reports the result of a copy operation
type CopyResult struct {
    FilesCopied  int
    BytesCopied  int64
    Errors       []CopyError // Non-fatal errors encountered
}

// Copy copies selected files from source to target
func Copy(opts CopyOptions) (*CopyResult, error)

// copyFile copies a single file
func copyFile(src, dst string, preserveMode bool) error

// copyDirectory copies a directory recursively
func copyDirectory(src, dst string, preserveMode bool) error
```

**Implementation notes:**
- Create parent directories as needed with `os.MkdirAll()`
- Use `io.Copy()` for efficient file copying
- Preserve file permissions if `PreserveMode` is true
- Handle symlinks (skip, follow, or recreate based on config)
- Collect non-fatal errors and continue copying
- Call progress callback after each file

**Progress tracking pattern:**
```go
for i, file := range opts.Files {
    if opts.OnProgress != nil {
        opts.OnProgress(CopyProgress{
            CurrentFile: file.Path,
            FilesDone:   i,
            FilesTotal:  len(opts.Files),
            BytesDone:   bytesCopied,
            BytesTotal:  totalBytes,
        })
    }
    // Copy file...
    bytesCopied += file.Size
}
```

---

### Task 6: Integrate with Create Command

**File:** `internal/cli/create.go` (modify existing)

Add file copying step to the create workflow.

**New flags:**

```go
type CreateOptions struct {
    // ... existing fields
    CopyFiles     bool   // Whether to copy gitignored files (default: true)
    SkipCopy      bool   // Skip file copying entirely
}
```

**Integration point:**

After worktree creation succeeds, before returning:

```go
func runCreate(cmd *cobra.Command, args []string) error {
    // ... existing worktree creation code ...

    // File copying step
    if !createOpts.SkipCopy {
        if err := copyIgnoredFiles(mainWorktree, newWorktreePath, cfg); err != nil {
            output.Warning(fmt.Sprintf("Failed to copy files: %v", err))
            // Non-fatal - worktree was created successfully
        }
    }

    return nil
}

func copyIgnoredFiles(source, target string, cfg *config.Config) error {
    // 1. Discover ignored files
    ignored, err := copy.DiscoverIgnored(source)
    if err != nil {
        return err
    }

    if len(ignored) == 0 {
        output.Info("No gitignored files to copy")
        return nil
    }

    // 2. Create pattern matcher from config
    matcher := copy.NewPatternMatcher(cfg.CopyDefaults, cfg.CopyExclude)

    // 3. Create selection (pre-select defaults)
    selection := copy.NewSelection(ignored, matcher)

    // 4. Show selection (for now, auto-select defaults; TUI in Phase 12)
    visible := selection.FilterVisible()
    selected := selection.GetSelected()

    if len(selected) == 0 {
        output.Info("No files selected for copying")
        return nil
    }

    // 5. Show summary and copy
    output.Info(fmt.Sprintf("Copying %d files (%s)...",
        len(selected), copy.FormatSize(selection.SelectedSize)))

    result, err := copy.Copy(copy.CopyOptions{
        SourceDir:  source,
        TargetDir:  target,
        Files:      selected,
        OnProgress: showProgress,
        PreserveMode: true,
    })

    if err != nil {
        return err
    }

    output.Success(fmt.Sprintf("Copied %d files (%s)",
        result.FilesCopied, copy.FormatSize(result.BytesCopied)))

    return nil
}
```

---

### Task 7: Add CLI Flag Support

**File:** `internal/cli/create.go` (modify existing)

Add `--skip-copy` and `--copy-files` flags:

```go
func init() {
    // ... existing flags ...
    createCmd.Flags().BoolVar(&createOpts.SkipCopy, "skip-copy", false,
        "Skip copying gitignored files")
    createCmd.Flags().StringSliceVar(&createOpts.CopyFiles, "copy", nil,
        "Additional files to copy (can be used multiple times)")
}
```

---

### Task 8: Add Progress Display

**File:** `internal/output/progress.go` (new file)

Simple progress display for CLI:

```go
package output

import (
    "fmt"
    "strings"
)

// ProgressBar displays a simple progress bar
type ProgressBar struct {
    total   int
    current int
    width   int
}

// NewProgressBar creates a progress bar
func NewProgressBar(total, width int) *ProgressBar

// Update updates the progress bar
func (p *ProgressBar) Update(current int, message string)

// Done marks progress as complete
func (p *ProgressBar) Done()
```

**Simple progress output:**
```
Copying files... [████████░░░░░░░░░░░░] 45% (23/50) config/.env
```

For Phase 6, a simple text-based progress indicator is sufficient. The TUI progress will be implemented in Phase 11-12.

---

### Task 9: Write Tests

**Test files to create:**

1. `internal/copy/discover_test.go`
   - Test `git status --ignored` output parsing
   - Test file size calculation
   - Test directory handling

2. `internal/copy/match_test.go`
   - Test exact pattern matching
   - Test glob pattern matching (`*.log`)
   - Test double-star patterns (`**/.env`)
   - Test directory name matching
   - Test exclude precedence over default

3. `internal/copy/copy_test.go`
   - Test single file copy
   - Test directory copy
   - Test permission preservation
   - Test progress callback
   - Test error collection

4. `internal/copy/selection_test.go`
   - Test selection creation
   - Test filtering visible files
   - Test toggle/select/deselect
   - Test size calculations

**Test patterns:**
- Use `testutil.NewTempRepo()` for git repository tests
- Create temporary directories for file copy tests
- Test cross-platform path handling

---

## Dependencies

**External packages (add to go.mod):**

```
github.com/bmatcuk/doublestar/v4  # Glob pattern matching with ** support
```

**Standard library:**
- `os`, `io`, `path/filepath` - File operations
- `fmt`, `strings` - String formatting
- `time` - Timestamps

---

## Config Reference

Relevant config fields from `.worktree.yaml`:

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

---

## Default Exclusions

Auto-exclude these dependency directories by default (even without config):

```go
var DefaultExcludes = []string{
    "node_modules",
    "vendor",
    ".venv",
    "venv",
    "__pycache__",
    ".pycache",
    "target",          // Rust, Java
    "dist",
    "build",
    ".gradle",
    ".maven",
    "pkg",             // Go
    "bin",
    ".git",            // Never copy .git
    ".svn",
    ".hg",
}
```

These are merged with user-configured `copy_exclude` patterns.

---

## Edge Cases

### Symlinks
- By default, skip symlinks to avoid circular references
- Could add `--follow-symlinks` flag if needed

### Large Files
- Show warning for files over 100MB
- Consider adding `--max-size` flag to skip large files

### Permission Errors
- Collect errors and continue copying other files
- Report permission errors at the end
- Don't fail entire operation for single file errors

### Path Conflicts
- If target file already exists, skip and warn
- Could add `--overwrite` flag if needed

### Empty Directories
- Don't copy empty ignored directories
- Only copy directories that contain selected files

---

## Implementation Order

1. **Task 1:** Create package structure and error types
2. **Task 3:** Implement pattern matching (needed for discovery filtering)
3. **Task 2:** Implement gitignored file discovery
4. **Task 4:** Implement file selection with sizes
5. **Task 5:** Implement file/directory copying
6. **Task 8:** Add progress display
7. **Task 6:** Integrate with create command
8. **Task 7:** Add CLI flag support
9. **Task 9:** Write tests (throughout, not just at end)

---

## Verification Checklist

- [ ] `git status --ignored` parsing works correctly
- [ ] Pattern matching handles all glob variants
- [ ] Exclude patterns take precedence over defaults
- [ ] File sizes are calculated correctly
- [ ] Directory sizes include all contents
- [ ] Copy preserves file permissions
- [ ] Progress is displayed during copy
- [ ] Errors are collected and reported
- [ ] `--skip-copy` flag works
- [ ] Default exclusions are applied
- [ ] Tests pass on Windows, macOS, Linux
- [ ] Large files are handled gracefully
- [ ] Symlinks don't cause infinite loops

---

## Future Enhancements (Phase 11-12)

- TUI file selection with checkboxes
- Interactive pattern editor
- File preview before copy
- Undo/rollback copied files
