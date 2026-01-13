package copy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseIgnoredOutput(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected []string
	}{
		{
			name: "single file",
			output: `!! .env
`,
			expected: []string{".env"},
		},
		{
			name: "single directory",
			output: `!! node_modules/
`,
			expected: []string{"node_modules"},
		},
		{
			name: "multiple files",
			output: `!! .env
!! config.local.json
!! app.log
`,
			expected: []string{".env", "config.local.json", "app.log"},
		},
		{
			name: "mixed files and directories",
			output: `!! .env
!! node_modules/
!! .venv/
!! *.log
`,
			expected: []string{".env", "node_modules", ".venv", "*.log"},
		},
		{
			name: "empty output",
			output: ``,
			expected: []string{},
		},
		{
			name: "non-ignored files mixed in",
			output: ` M modified.txt
!! .env
 M another.txt
!! node_modules/
`,
			expected: []string{".env", "node_modules"},
		},
		{
			name: "whitespace handling",
			output: `   !! .env
!! node_modules/
`,
			expected: []string{".env", "node_modules"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseIgnoredOutput(tt.output)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d paths, got %d", len(tt.expected), len(result))
				return
			}
			for i, path := range result {
				if path != tt.expected[i] {
					t.Errorf("Path %d: got %q, want %q", i, path, tt.expected[i])
				}
			}
		})
	}
}

func TestCalculateDirSize(t *testing.T) {
	// Create temp directory for testing
	tmpDir := t.TempDir()

	// Create directory structure
	dir := filepath.Join(tmpDir, "testdir")
	if err := os.MkdirAll(filepath.Join(dir, "subdir"), 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Create files with known sizes
	file1 := filepath.Join(dir, "file1.txt")
	file2 := filepath.Join(dir, "subdir", "file2.txt")

	content1 := strings.Repeat("a", 100) // 100 bytes
	content2 := strings.Repeat("b", 200) // 200 bytes

	if err := os.WriteFile(file1, []byte(content1), 0644); err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}
	if err := os.WriteFile(file2, []byte(content2), 0644); err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}

	// Calculate directory size
	size, err := calculateDirSize(dir)
	if err != nil {
		t.Fatalf("calculateDirSize failed: %v", err)
	}

	// Should be 100 + 200 = 300 bytes
	expected := int64(300)
	if size != expected {
		t.Errorf("Expected size %d, got %d", expected, size)
	}
}

func TestCalculateDirSize_EmptyDir(t *testing.T) {
	// Create temp directory for testing
	tmpDir := t.TempDir()

	// Create empty directory
	dir := filepath.Join(tmpDir, "empty")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Calculate directory size
	size, err := calculateDirSize(dir)
	if err != nil {
		t.Fatalf("calculateDirSize failed: %v", err)
	}

	// Should be 0 bytes
	if size != 0 {
		t.Errorf("Expected size 0, got %d", size)
	}
}

func TestDiscoverIgnored_NoGitRepo(t *testing.T) {
	// Create temp directory for testing (not a git repo)
	tmpDir := t.TempDir()

	// Should return error for non-git directory
	_, err := DiscoverIgnored(tmpDir)
	if err == nil {
		t.Error("Expected error for non-git directory, got nil")
	}
}

// Note: Testing DiscoverIgnored with an actual git repo requires
// setting up a git repository in the test, which is more complex.
// For now, we test the parsing logic separately.
