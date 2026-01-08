package config

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig() returned nil")
	}

	// Test CopyDefaults
	if len(cfg.CopyDefaults) == 0 {
		t.Error("Expected non-empty CopyDefaults")
	}

	expectedCopyDefaults := []string{
		".env",
		"**/.env",
		"**/.env.local",
		".claude/",
		"**/*.local.md",
		"**/setenv.sh",
	}

	if len(cfg.CopyDefaults) != len(expectedCopyDefaults) {
		t.Errorf("Expected %d copy defaults, got %d", len(expectedCopyDefaults), len(cfg.CopyDefaults))
	}

	for i, expected := range expectedCopyDefaults {
		if i >= len(cfg.CopyDefaults) {
			break
		}
		if cfg.CopyDefaults[i] != expected {
			t.Errorf("Expected copy_defaults[%d] = %s, got %s", i, expected, cfg.CopyDefaults[i])
		}
	}

	// Test CopyExclude
	if len(cfg.CopyExclude) == 0 {
		t.Error("Expected non-empty CopyExclude")
	}

	expectedCopyExclude := []string{
		"node_modules",
		"vendor",
		".venv",
		"__pycache__",
		"target",
		"dist",
		"build",
		"*.log",
	}

	if len(cfg.CopyExclude) != len(expectedCopyExclude) {
		t.Errorf("Expected %d copy exclude patterns, got %d", len(expectedCopyExclude), len(cfg.CopyExclude))
	}

	// Test Docker defaults
	if cfg.Docker.DefaultMode != "shared" {
		t.Errorf("Expected docker.default_mode 'shared', got '%s'", cfg.Docker.DefaultMode)
	}

	if cfg.Docker.PortOffset != 1 {
		t.Errorf("Expected docker.port_offset 1, got %d", cfg.Docker.PortOffset)
	}

	if len(cfg.Docker.ComposeFiles) != 0 {
		t.Errorf("Expected empty compose_files for auto-detect, got %d files", len(cfg.Docker.ComposeFiles))
	}

	if len(cfg.Docker.DataDirectories) != 0 {
		t.Errorf("Expected empty data_directories, got %d directories", len(cfg.Docker.DataDirectories))
	}

	// Test Dependencies defaults
	if !cfg.Dependencies.AutoInstall {
		t.Error("Expected dependencies.auto_install true, got false")
	}

	if len(cfg.Dependencies.Paths) != 1 {
		t.Errorf("Expected 1 dependency path, got %d", len(cfg.Dependencies.Paths))
	}

	if len(cfg.Dependencies.Paths) > 0 && cfg.Dependencies.Paths[0] != "." {
		t.Errorf("Expected dependency path '.', got '%s'", cfg.Dependencies.Paths[0])
	}

	// Test Migrations defaults
	if !cfg.Migrations.AutoDetect {
		t.Error("Expected migrations.auto_detect true, got false")
	}

	if cfg.Migrations.Command != "" {
		t.Errorf("Expected empty migration command for auto-detect, got '%s'", cfg.Migrations.Command)
	}

	// Test Hooks defaults
	if len(cfg.Hooks.PostCreate) != 0 {
		t.Errorf("Expected empty post_create hooks, got %d", len(cfg.Hooks.PostCreate))
	}

	if len(cfg.Hooks.PostDelete) != 0 {
		t.Errorf("Expected empty post_delete hooks, got %d", len(cfg.Hooks.PostDelete))
	}
}

func TestDefaultConfigIsValid(t *testing.T) {
	cfg := DefaultConfig()

	// Validate the default config
	errors := cfg.Validate()
	if len(errors) > 0 {
		t.Errorf("Default config has validation errors:")
		for _, err := range errors {
			t.Errorf("  - %v", err)
		}
	}
}
