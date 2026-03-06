package docker

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectComposeFiles(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	tests := []struct {
		name           string
		files          []string
		expectError    bool
		expectCount    int
		expectBase     string
		expectOverride int
	}{
		{
			name:        "No compose files",
			files:       []string{},
			expectError: true,
			expectCount: 0,
		},
		{
			name:           "Single docker-compose.yml",
			files:          []string{"docker-compose.yml"},
			expectError:    false,
			expectCount:    1,
			expectBase:     "docker-compose.yml",
			expectOverride: 0,
		},
		{
			name:           "docker-compose.yml with override",
			files:          []string{"docker-compose.yml", "docker-compose.dev.yml"},
			expectError:    false,
			expectCount:    2,
			expectBase:     "docker-compose.yml",
			expectOverride: 1,
		},
		{
			name:           "compose.yml format",
			files:          []string{"compose.yml"},
			expectError:    false,
			expectCount:    1,
			expectBase:     "compose.yml",
			expectOverride: 0,
		},
		{
			name:           "Multiple override files",
			files:          []string{"docker-compose.yml", "docker-compose.dev.yml", "docker-compose.prod.yml"},
			expectError:    false,
			expectCount:    3,
			expectBase:     "docker-compose.yml",
			expectOverride: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test directory
			testDir := filepath.Join(tmpDir, tt.name)
			if err := os.MkdirAll(testDir, 0755); err != nil {
				t.Fatal(err)
			}

			// Create test files
			for _, f := range tt.files {
				path := filepath.Join(testDir, f)
				if err := os.WriteFile(path, []byte("version: '3'\n"), 0644); err != nil {
					t.Fatal(err)
				}
			}

			// Test detection
			files, err := DetectComposeFiles(testDir)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(files) != tt.expectCount {
				t.Errorf("Expected %d files, got %d", tt.expectCount, len(files))
			}

			// Check base file
			if tt.expectBase != "" {
				baseFile := GetBaseComposeFile(files)
				if baseFile == nil {
					t.Errorf("Expected base file %s, got nil", tt.expectBase)
				} else if baseFile.Path != tt.expectBase {
					t.Errorf("Expected base file %s, got %s", tt.expectBase, baseFile.Path)
				}
			}

			// Count override files
			overrideCount := 0
			for _, f := range files {
				if !f.IsBase {
					overrideCount++
				}
			}
			if overrideCount != tt.expectOverride {
				t.Errorf("Expected %d override files, got %d", tt.expectOverride, overrideCount)
			}
		})
	}
}

func TestGetBaseComposeFile(t *testing.T) {
	tests := []struct {
		name     string
		files    []ComposeFile
		expected string
	}{
		{
			name: "docker-compose.yml priority",
			files: []ComposeFile{
				{Path: "docker-compose.yaml", IsBase: true},
				{Path: "docker-compose.yml", IsBase: true},
			},
			expected: "docker-compose.yml",
		},
		{
			name: "compose.yml fallback",
			files: []ComposeFile{
				{Path: "compose.yml", IsBase: true},
			},
			expected: "compose.yml",
		},
		{
			name: "Only override files",
			files: []ComposeFile{
				{Path: "docker-compose.dev.yml", IsBase: false},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetBaseComposeFile(tt.files)

			if tt.expected == "" {
				if result != nil {
					t.Errorf("Expected nil, got %s", result.Path)
				}
			} else {
				if result == nil {
					t.Errorf("Expected %s, got nil", tt.expected)
				} else if result.Path != tt.expected {
					t.Errorf("Expected %s, got %s", tt.expected, result.Path)
				}
			}
		})
	}
}

func TestIsOverrideFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{"docker-compose.yml", "docker-compose.yml", false},
		{"docker-compose.yaml", "docker-compose.yaml", false},
		{"docker-compose.dev.yml", "docker-compose.dev.yml", true},
		{"docker-compose.prod.yaml", "docker-compose.prod.yaml", true},
		{"compose.yml", "compose.yml", false},
		{"compose.yaml", "compose.yaml", false},
		{"compose.dev.yml", "compose.dev.yml", true},
		{"compose.test.yaml", "compose.test.yaml", true},
		{"random.yml", "random.yml", false},
		{"Dockerfile", "Dockerfile", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsOverrideFile(tt.filename)
			if result != tt.expected {
				t.Errorf("IsOverrideFile(%s) = %v, expected %v", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestDetectOrLoad(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test compose files
	composeYml := filepath.Join(tmpDir, "docker-compose.yml")
	if err := os.WriteFile(composeYml, []byte("version: '3'\n"), 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("Auto-detection", func(t *testing.T) {
		files, err := DetectOrLoad(tmpDir, nil)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if len(files) != 1 {
			t.Errorf("Expected 1 file, got %d", len(files))
		}
	})

	t.Run("Configured files", func(t *testing.T) {
		configFiles := []string{"docker-compose.yml"}
		files, err := DetectOrLoad(tmpDir, configFiles)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if len(files) != 1 {
			t.Errorf("Expected 1 file, got %d", len(files))
		}
	})

	t.Run("Missing configured file", func(t *testing.T) {
		configFiles := []string{"missing.yml"}
		_, err := DetectOrLoad(tmpDir, configFiles)
		if err == nil {
			t.Error("Expected error for missing file")
		}
	})
}
