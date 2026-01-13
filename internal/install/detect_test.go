package install

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectInDirectory(t *testing.T) {
	tests := []struct {
		name         string
		files        []string // files to create in temp dir
		expectedName string   // expected manager name
		expectedLock string   // expected lock file
	}{
		{
			name:         "bun with lock file",
			files:        []string{"package.json", "bun.lock"},
			expectedName: "bun",
			expectedLock: "bun.lock",
		},
		{
			name:         "pnpm with lock file",
			files:        []string{"package.json", "pnpm-lock.yaml"},
			expectedName: "pnpm",
			expectedLock: "pnpm-lock.yaml",
		},
		{
			name:         "yarn with lock file",
			files:        []string{"package.json", "yarn.lock"},
			expectedName: "yarn",
			expectedLock: "yarn.lock",
		},
		{
			name:         "npm with lock file",
			files:        []string{"package.json", "package-lock.json"},
			expectedName: "npm",
			expectedLock: "package-lock.json",
		},
		{
			name:         "npm without lock file",
			files:        []string{"package.json"},
			expectedName: "npm",
			expectedLock: "",
		},
		{
			name:         "go module",
			files:        []string{"go.mod"},
			expectedName: "go",
			expectedLock: "go.sum",
		},
		{
			name:         "cargo",
			files:        []string{"Cargo.toml"},
			expectedName: "cargo",
			expectedLock: "Cargo.lock",
		},
		{
			name:         "poetry with lock file",
			files:        []string{"poetry.lock"},
			expectedName: "poetry",
			expectedLock: "poetry.lock",
		},
		{
			name:         "pip with requirements",
			files:        []string{"requirements.txt"},
			expectedName: "pip",
			expectedLock: "",
		},
		{
			name:         "no package manager",
			files:        []string{"README.md"},
			expectedName: "",
			expectedLock: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			dir := t.TempDir()

			// Create test files
			for _, file := range tt.files {
				path := filepath.Join(dir, file)
				if err := os.WriteFile(path, []byte("test content"), 0644); err != nil {
					t.Fatalf("Failed to create test file %s: %v", file, err)
				}
			}

			// Run detection
			pm := detectInDirectory(dir)

			// Check expectations
			if tt.expectedName == "" {
				if pm != nil {
					t.Errorf("Expected no package manager, got %s", pm.Name)
				}
			} else {
				if pm == nil {
					t.Fatalf("Expected package manager %s, got nil", tt.expectedName)
				}
				if pm.Name != tt.expectedName {
					t.Errorf("Expected manager %s, got %s", tt.expectedName, pm.Name)
				}
				if pm.LockFile != tt.expectedLock {
					t.Errorf("Expected lock file %s, got %s", tt.expectedLock, pm.LockFile)
				}
				if pm.Path != dir {
					t.Errorf("Expected path %s, got %s", dir, pm.Path)
				}
			}
		})
	}
}

func TestDetectInDirectory_Priority(t *testing.T) {
	// Test that more specific lock files take priority
	tests := []struct {
		name         string
		files        []string
		expectedName string
	}{
		{
			name:         "bun over pnpm",
			files:        []string{"package.json", "bun.lock", "pnpm-lock.yaml"},
			expectedName: "bun",
		},
		{
			name:         "pnpm over yarn",
			files:        []string{"package.json", "pnpm-lock.yaml", "yarn.lock"},
			expectedName: "pnpm",
		},
		{
			name:         "yarn over npm",
			files:        []string{"package.json", "yarn.lock", "package-lock.json"},
			expectedName: "yarn",
		},
		{
			name:         "npm lock over package.json",
			files:        []string{"package.json", "package-lock.json"},
			expectedName: "npm",
		},
		{
			name:         "poetry lock over pyproject.toml",
			files:        []string{"poetry.lock", "pyproject.toml"},
			expectedName: "poetry",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()

			// Create all files
			for _, file := range tt.files {
				path := filepath.Join(dir, file)
				if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
					t.Fatalf("Failed to create file %s: %v", file, err)
				}
			}

			pm := detectInDirectory(dir)
			if pm == nil {
				t.Fatalf("Expected package manager, got nil")
			}
			if pm.Name != tt.expectedName {
				t.Errorf("Expected %s, got %s", tt.expectedName, pm.Name)
			}
		})
	}
}

func TestHasPoetryConfig(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "has poetry config",
			content:  "[tool.poetry]\nname = \"myproject\"",
			expected: true,
		},
		{
			name:     "no poetry config",
			content:  "[tool.pytest]\nname = \"myproject\"",
			expected: false,
		},
		{
			name:     "empty file",
			content:  "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "pyproject.toml")

			if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			result := hasPoetryConfig(path)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}

	// Test non-existent file
	t.Run("non-existent file", func(t *testing.T) {
		result := hasPoetryConfig("/nonexistent/pyproject.toml")
		if result {
			t.Error("Expected false for non-existent file")
		}
	})
}

func TestDetectPackageManagers(t *testing.T) {
	// Create a temp directory structure
	dir := t.TempDir()

	// Create subdirectories
	os.MkdirAll(filepath.Join(dir, "app1"), 0755)
	os.MkdirAll(filepath.Join(dir, "app2"), 0755)
	os.MkdirAll(filepath.Join(dir, "packages", "pkg1"), 0755)
	os.MkdirAll(filepath.Join(dir, "packages", "pkg2"), 0755)

	// Create package manager files
	os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(dir, "yarn.lock"), []byte(""), 0644)
	os.WriteFile(filepath.Join(dir, "app1", "go.mod"), []byte(""), 0644)
	os.WriteFile(filepath.Join(dir, "app2", "Cargo.toml"), []byte(""), 0644)
	os.WriteFile(filepath.Join(dir, "packages", "pkg1", "package.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(dir, "packages", "pkg2", "package.json"), []byte("{}"), 0644)

	tests := []struct {
		name          string
		paths         []string
		expectedCount int
		expectedNames []string
	}{
		{
			name:          "single directory",
			paths:         []string{"."},
			expectedCount: 1,
			expectedNames: []string{"yarn"},
		},
		{
			name:          "multiple directories",
			paths:         []string{".", "app1", "app2"},
			expectedCount: 3,
			expectedNames: []string{"yarn", "go", "cargo"},
		},
		{
			name:          "glob pattern",
			paths:         []string{"packages/*"},
			expectedCount: 2,
			expectedNames: []string{"npm", "npm"},
		},
		{
			name:          "no matches",
			paths:         []string{"nonexistent"},
			expectedCount: 0,
			expectedNames: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			managers, err := DetectPackageManagers(dir, tt.paths)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(managers) != tt.expectedCount {
				t.Errorf("Expected %d managers, got %d", tt.expectedCount, len(managers))
			}

			// Check manager names match (order may vary with glob patterns)
			names := make(map[string]int)
			for _, m := range managers {
				names[m.Name]++
			}

			expectedNames := make(map[string]int)
			for _, n := range tt.expectedNames {
				expectedNames[n]++
			}

			for name, count := range expectedNames {
				if names[name] != count {
					t.Errorf("Expected %d instances of %s, got %d", count, name, names[name])
				}
			}
		})
	}
}

func TestDetectPackageManagers_NoDuplicates(t *testing.T) {
	dir := t.TempDir()

	// Create package.json
	os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0644)

	// Detect with duplicate paths
	managers, err := DetectPackageManagers(dir, []string{".", ".", "."})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(managers) != 1 {
		t.Errorf("Expected 1 manager (deduplication), got %d", len(managers))
	}
}
