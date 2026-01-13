package migrate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindComposeFile(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(dir string) error
		expectFile bool
	}{
		{
			name: "docker-compose.yml",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte("version: '3'"), 0644)
			},
			expectFile: true,
		},
		{
			name: "compose.yml",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "compose.yml"), []byte("version: '3'"), 0644)
			},
			expectFile: true,
		},
		{
			name:       "no compose file",
			setup:      func(dir string) error { return nil },
			expectFile: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := tt.setup(dir); err != nil {
				t.Fatal(err)
			}

			result := findComposeFile(dir)

			if tt.expectFile && result == "" {
				t.Error("expected to find compose file, got empty string")
			} else if !tt.expectFile && result != "" {
				t.Errorf("expected no compose file, got %q", result)
			}
		})
	}
}

func TestCheckDatabaseContainer_NoComposeFile(t *testing.T) {
	dir := t.TempDir()

	status, err := CheckDatabaseContainer(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if status != nil {
		t.Errorf("expected nil status when no compose file exists, got %+v", status)
	}
}
