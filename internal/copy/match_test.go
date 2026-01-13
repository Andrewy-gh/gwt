package copy

import "testing"

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		path     string
		expected bool
	}{
		// Exact matches
		{
			name:     "exact match at root",
			pattern:  ".env",
			path:     ".env",
			expected: true,
		},
		{
			name:     "exact match no match",
			pattern:  ".env",
			path:     "config/.env",
			expected: false,
		},
		// Simple globs
		{
			name:     "simple glob match",
			pattern:  "*.log",
			path:     "app.log",
			expected: true,
		},
		{
			name:     "simple glob no match",
			pattern:  "*.log",
			path:     "app.txt",
			expected: false,
		},
		// Double-star patterns
		{
			name:     "double-star any depth",
			pattern:  "**/.env",
			path:     "config/.env",
			expected: true,
		},
		{
			name:     "double-star at root",
			pattern:  "**/.env",
			path:     ".env",
			expected: true,
		},
		{
			name:     "double-star deep path",
			pattern:  "**/.env",
			path:     "a/b/c/.env",
			expected: true,
		},
		// Directory name matching
		{
			name:     "directory name anywhere",
			pattern:  "node_modules",
			path:     "node_modules",
			expected: true,
		},
		{
			name:     "directory name in path",
			pattern:  "node_modules",
			path:     "foo/node_modules/bar",
			expected: true,
		},
		{
			name:     "directory name no match",
			pattern:  "node_modules",
			path:     "foo/bar",
			expected: false,
		},
		// Auto-prepend ** for simple patterns
		{
			name:     "simple pattern auto-prepend",
			pattern:  ".env",
			path:     "config/.env",
			expected: false, // Exact match only, no auto-prepend for exact patterns
		},
		{
			name:     "glob pattern with path",
			pattern:  "*.env",
			path:     "config/test.env",
			expected: false, // Simple glob doesn't match subdirs
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchPattern(tt.pattern, tt.path)
			if result != tt.expected {
				t.Errorf("matchPattern(%q, %q) = %v, want %v", tt.pattern, tt.path, result, tt.expected)
			}
		})
	}
}

func TestPatternMatcher_Match(t *testing.T) {
	tests := []struct {
		name     string
		defaults []string
		excludes []string
		path     string
		expected MatchResult
	}{
		{
			name:     "match default",
			defaults: []string{".env"},
			excludes: []string{},
			path:     ".env",
			expected: MatchDefault,
		},
		{
			name:     "match exclude",
			defaults: []string{},
			excludes: []string{"node_modules"},
			path:     "node_modules",
			expected: MatchExclude,
		},
		{
			name:     "exclude takes precedence",
			defaults: []string{".env"},
			excludes: []string{".env"},
			path:     ".env",
			expected: MatchExclude,
		},
		{
			name:     "no match",
			defaults: []string{".env"},
			excludes: []string{"node_modules"},
			path:     "foo.txt",
			expected: MatchNone,
		},
		{
			name:     "default with glob",
			defaults: []string{"**/.env"},
			excludes: []string{},
			path:     "config/.env",
			expected: MatchDefault,
		},
		{
			name:     "exclude directory in path",
			defaults: []string{},
			excludes: []string{"node_modules"},
			path:     "foo/node_modules/bar",
			expected: MatchExclude,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher := &PatternMatcher{
				Defaults: tt.defaults,
				Excludes: tt.excludes,
			}
			result := matcher.Match(tt.path)
			if result != tt.expected {
				t.Errorf("Match(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestNewPatternMatcher(t *testing.T) {
	defaults := []string{".env"}
	excludes := []string{"custom_exclude"}

	matcher := NewPatternMatcher(defaults, excludes)

	// Check defaults are set
	if len(matcher.Defaults) != 1 {
		t.Errorf("Expected 1 default, got %d", len(matcher.Defaults))
	}

	// Check excludes include both custom and default excludes
	if len(matcher.Excludes) < len(DefaultExcludes)+1 {
		t.Errorf("Expected at least %d excludes, got %d", len(DefaultExcludes)+1, len(matcher.Excludes))
	}

	// Verify default excludes are included
	hasNodeModules := false
	for _, exclude := range matcher.Excludes {
		if exclude == "node_modules" {
			hasNodeModules = true
			break
		}
	}
	if !hasNodeModules {
		t.Error("Expected default excludes to include node_modules")
	}

	// Verify custom exclude is included
	hasCustom := false
	for _, exclude := range matcher.Excludes {
		if exclude == "custom_exclude" {
			hasCustom = true
			break
		}
	}
	if !hasCustom {
		t.Error("Expected custom exclude to be included")
	}
}
