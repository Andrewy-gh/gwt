package copy

import (
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// MatchResult indicates how a file matched patterns
type MatchResult int

const (
	MatchNone    MatchResult = iota // No match
	MatchDefault                    // Matched copy_defaults (pre-selected)
	MatchExclude                    // Matched copy_exclude (hidden)
)

// DefaultExcludes are dependency directories excluded by default
var DefaultExcludes = []string{
	"node_modules",
	"vendor",
	".venv",
	"venv",
	"__pycache__",
	".pycache",
	"target", // Rust, Java
	"dist",
	"build",
	".gradle",
	".maven",
	"pkg", // Go
	"bin",
	".git", // Never copy .git
	".svn",
	".hg",
}

// PatternMatcher handles glob pattern matching
type PatternMatcher struct {
	Defaults []string // Patterns for pre-selection
	Excludes []string // Patterns for exclusion
}

// NewPatternMatcher creates a matcher from config
func NewPatternMatcher(defaults, excludes []string) *PatternMatcher {
	// Merge user excludes with default excludes
	allExcludes := make([]string, 0, len(excludes)+len(DefaultExcludes))
	allExcludes = append(allExcludes, DefaultExcludes...)
	allExcludes = append(allExcludes, excludes...)

	return &PatternMatcher{
		Defaults: defaults,
		Excludes: allExcludes,
	}
}

// Match checks if a path matches any patterns
// Returns the match result (exclude takes precedence over default)
func (m *PatternMatcher) Match(path string) MatchResult {
	// Normalize path to use forward slashes
	path = filepath.ToSlash(path)

	// Check excludes first (they take precedence)
	for _, pattern := range m.Excludes {
		if matchPattern(pattern, path) {
			return MatchExclude
		}
	}

	// Check defaults
	for _, pattern := range m.Defaults {
		if matchPattern(pattern, path) {
			return MatchDefault
		}
	}

	return MatchNone
}

// matchPattern checks if a single pattern matches the path
// Supports:
// - Exact matches: ".env"
// - Simple globs: "*.log"
// - Double-star globs: "**/.env"
// - Directory patterns: "node_modules" (matches anywhere in path)
func matchPattern(pattern, path string) bool {
	// Normalize pattern to use forward slashes
	pattern = filepath.ToSlash(pattern)

	// Try exact match first
	if pattern == path {
		return true
	}

	// Try doublestar match for glob patterns
	matched, err := doublestar.Match(pattern, path)
	if err == nil && matched {
		return true
	}

	// Check if pattern is a simple directory name (no / or * and no file extension)
	// that should match anywhere in path
	// e.g., "node_modules" should match "foo/node_modules/bar"
	// but ".env" should NOT match "config/.env" (needs exact match or **/. env)
	if !strings.Contains(pattern, "/") && !strings.Contains(pattern, "*") && !isFilename(pattern) {
		parts := strings.Split(path, "/")
		for _, part := range parts {
			if part == pattern {
				return true
			}
		}
	}

	return false
}

// isFilename checks if a pattern looks like a filename (has extension or starts with .)
func isFilename(pattern string) bool {
	// If it starts with a dot, it's likely a dotfile (.env, .gitignore, etc.)
	if strings.HasPrefix(pattern, ".") {
		return true
	}
	// If it has an extension, it's a file (config.json, app.log, etc.)
	if strings.Contains(pattern, ".") {
		return true
	}
	return false
}
