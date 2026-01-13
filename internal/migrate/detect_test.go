package migrate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Andrewy-gh/gwt/internal/config"
)

func TestDetectMakefile(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		expectTool bool
		expectName string
	}{
		{
			name:       "migrate target",
			content:    "migrate:\n\techo running",
			expectTool: true,
			expectName: "makefile",
		},
		{
			name:       "db-migrate target",
			content:    "db-migrate:\n\techo running",
			expectTool: true,
			expectName: "makefile",
		},
		{
			name:       "no migrate target",
			content:    "build:\n\techo building",
			expectTool: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "Makefile"), []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			tool, err := detectMakefile(dir)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.expectTool {
				if tool == nil {
					t.Error("expected tool, got nil")
				} else if tool.Name != tt.expectName {
					t.Errorf("expected name %q, got %q", tt.expectName, tool.Name)
				}
			} else if tool != nil {
				t.Errorf("expected nil tool, got %+v", tool)
			}
		})
	}
}

func TestDetectPrisma(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(dir string) error
		expectTool bool
	}{
		{
			name: "prisma/schema.prisma",
			setup: func(dir string) error {
				prismaDir := filepath.Join(dir, "prisma")
				if err := os.Mkdir(prismaDir, 0755); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(prismaDir, "schema.prisma"), []byte(""), 0644)
			},
			expectTool: true,
		},
		{
			name: "root schema.prisma",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "schema.prisma"), []byte(""), 0644)
			},
			expectTool: true,
		},
		{
			name:       "no prisma",
			setup:      func(dir string) error { return nil },
			expectTool: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := tt.setup(dir); err != nil {
				t.Fatal(err)
			}

			tool, err := detectPrisma(dir)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.expectTool && tool == nil {
				t.Error("expected tool, got nil")
			} else if !tt.expectTool && tool != nil {
				t.Errorf("expected nil tool, got %+v", tool)
			}
		})
	}
}

func TestDetectWithConfigOverride(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.MigrationsConfig{
		AutoDetect: true,
		Command:    "custom migrate --prod",
	}

	tool, err := Detect(dir, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tool == nil {
		t.Fatal("expected tool, got nil")
	}

	if tool.Name != "custom" {
		t.Errorf("expected name 'custom', got %q", tool.Name)
	}

	expectedCmd := []string{"custom", "migrate", "--prod"}
	if len(tool.Command) != len(expectedCmd) {
		t.Errorf("expected command %v, got %v", expectedCmd, tool.Command)
	}
}

func TestDetectAutoDetectDisabled(t *testing.T) {
	dir := t.TempDir()

	// Create a Makefile that would normally be detected
	os.WriteFile(filepath.Join(dir, "Makefile"), []byte("migrate:\n\techo"), 0644)

	cfg := &config.MigrationsConfig{
		AutoDetect: false,
		Command:    "", // No custom command
	}

	tool, err := Detect(dir, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tool != nil {
		t.Errorf("expected nil when auto-detect disabled, got %+v", tool)
	}
}
