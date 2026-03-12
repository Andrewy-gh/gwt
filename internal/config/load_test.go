package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMinimalConfig(t *testing.T) {
	// Load minimal config
	cfg, err := Load("testdata/valid/minimal.yaml")
	if err != nil {
		t.Fatalf("Failed to load minimal config: %v", err)
	}

	// Verify minimal fields
	if len(cfg.CopyDefaults) != 1 {
		t.Errorf("Expected 1 copy_defaults, got %d", len(cfg.CopyDefaults))
	}

	if cfg.Docker.DefaultMode != "shared" {
		t.Errorf("Expected docker.default_mode 'shared', got '%s'", cfg.Docker.DefaultMode)
	}

	if cfg.Docker.PortOffset != 1 {
		t.Errorf("Expected docker.port_offset 1, got %d", cfg.Docker.PortOffset)
	}
}

func TestLoadFullConfig(t *testing.T) {
	// Load full config
	cfg, err := Load("testdata/valid/full.yaml")
	if err != nil {
		t.Fatalf("Failed to load full config: %v", err)
	}

	// Verify all fields
	if len(cfg.CopyDefaults) != 8 {
		t.Errorf("Expected 8 copy_defaults, got %d", len(cfg.CopyDefaults))
	}

	if len(cfg.CopyExclude) != 8 {
		t.Errorf("Expected 8 copy_exclude, got %d", len(cfg.CopyExclude))
	}

	if len(cfg.Docker.ComposeFiles) != 2 {
		t.Errorf("Expected 2 compose_files, got %d", len(cfg.Docker.ComposeFiles))
	}

	if len(cfg.Docker.DataDirectories) != 2 {
		t.Errorf("Expected 2 data_directories, got %d", len(cfg.Docker.DataDirectories))
	}

	if cfg.Docker.DefaultMode != "new" {
		t.Errorf("Expected docker.default_mode 'new', got '%s'", cfg.Docker.DefaultMode)
	}

	if cfg.Docker.PortOffset != 10 {
		t.Errorf("Expected docker.port_offset 10, got %d", cfg.Docker.PortOffset)
	}

	if cfg.Dependencies.AutoInstall {
		t.Error("Expected dependencies.auto_install false, got true")
	}

	if len(cfg.Dependencies.Paths) != 3 {
		t.Errorf("Expected 3 dependency paths, got %d", len(cfg.Dependencies.Paths))
	}

	if cfg.Migrations.AutoDetect {
		t.Error("Expected migrations.auto_detect false, got true")
	}

	if cfg.Migrations.Command != "make migrate-up" {
		t.Errorf("Expected migrations.command 'make migrate-up', got '%s'", cfg.Migrations.Command)
	}

	if len(cfg.Hooks.PostCreate) != 2 {
		t.Errorf("Expected 2 post_create hooks, got %d", len(cfg.Hooks.PostCreate))
	}

	if len(cfg.Hooks.PostDelete) != 1 {
		t.Errorf("Expected 1 post_delete hook, got %d", len(cfg.Hooks.PostDelete))
	}
}

func TestLoadConfigWithComments(t *testing.T) {
	// Load config with comments
	cfg, err := Load("testdata/valid/with_comments.yaml")
	if err != nil {
		t.Fatalf("Failed to load config with comments: %v", err)
	}

	// Verify it still loads correctly
	if len(cfg.CopyDefaults) != 2 {
		t.Errorf("Expected 2 copy_defaults, got %d", len(cfg.CopyDefaults))
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	// Load invalid YAML
	_, err := Load("testdata/invalid/bad_yaml.yaml")
	if err == nil {
		t.Fatal("Expected error loading invalid YAML, got nil")
	}

	// Check that it's a ConfigParseError
	_, ok := err.(*ConfigParseError)
	if !ok {
		t.Errorf("Expected ConfigParseError, got %T", err)
	}
}

func TestLoadInvalidDockerMode(t *testing.T) {
	// Load config with invalid docker mode
	_, err := Load("testdata/invalid/bad_mode.yaml")
	if err == nil {
		t.Fatal("Expected validation error for invalid docker mode, got nil")
	}

	// Check that it's a ConfigValidationError
	_, ok := err.(*ConfigValidationError)
	if !ok {
		t.Errorf("Expected ConfigValidationError, got %T", err)
	}
}

func TestLoadInvalidPortOffset(t *testing.T) {
	// Load config with invalid port offset
	_, err := Load("testdata/invalid/bad_port.yaml")
	if err == nil {
		t.Fatal("Expected validation error for invalid port offset, got nil")
	}

	// Check that it's a ConfigValidationError
	_, ok := err.(*ConfigValidationError)
	if !ok {
		t.Errorf("Expected ConfigValidationError, got %T", err)
	}
}

func TestLoadNonExistentFile(t *testing.T) {
	// Load non-existent file
	_, err := Load("testdata/nonexistent.yaml")
	if err == nil {
		t.Fatal("Expected error loading non-existent file, got nil")
	}

	// Check that it's a ConfigParseError
	_, ok := err.(*ConfigParseError)
	if !ok {
		t.Errorf("Expected ConfigParseError, got %T", err)
	}
}

func TestLoadFromDir(t *testing.T) {
	// Create a temporary directory with a config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".worktree.yaml")

	// Write a test config
	configContent := `
copy_defaults:
  - ".env"

docker:
  default_mode: "shared"
  port_offset: 5

dependencies:
  auto_install: true

migrations:
  auto_detect: true
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Load from directory
	cfg, err := LoadFromDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to load config from directory: %v", err)
	}

	// Verify it loaded correctly
	if cfg.Docker.PortOffset != 5 {
		t.Errorf("Expected port_offset 5, got %d", cfg.Docker.PortOffset)
	}
}

func TestLoadFromDirNoConfig(t *testing.T) {
	// Create a temporary directory without a config file
	tmpDir := t.TempDir()

	// Load from directory (should return defaults)
	cfg, err := LoadFromDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to load defaults: %v", err)
	}

	// Verify we got defaults
	if cfg.Docker.PortOffset != 1 {
		t.Errorf("Expected default port_offset 1, got %d", cfg.Docker.PortOffset)
	}
}

func TestFindConfigFile(t *testing.T) {
	// Create a temporary directory with a config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".worktree.yaml")

	// Write a test config
	err := os.WriteFile(configPath, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Find config file
	found, err := FindConfigFile(tmpDir)
	if err != nil {
		t.Fatalf("FindConfigFile failed: %v", err)
	}

	if found == "" {
		t.Fatal("Expected to find config file, got empty string")
	}

	// Verify the path is correct
	expectedPath := configPath
	if found != expectedPath {
		t.Errorf("Expected config path %s, got %s", expectedPath, found)
	}
}

func TestFindConfigFileNotFound(t *testing.T) {
	// Create a temporary directory without a config file
	tmpDir := t.TempDir()

	// Find config file (should return empty string)
	found, err := FindConfigFile(tmpDir)
	if err != nil {
		t.Fatalf("FindConfigFile failed: %v", err)
	}

	if found != "" {
		t.Errorf("Expected empty string for missing config, got %s", found)
	}
}

func TestConfigExists(t *testing.T) {
	// Create a temporary directory with a config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".worktree.yaml")

	// Config should not exist yet
	if ConfigExists(tmpDir) {
		t.Error("ConfigExists returned true for non-existent config")
	}

	// Write a test config
	err := os.WriteFile(configPath, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Config should now exist
	if !ConfigExists(tmpDir) {
		t.Error("ConfigExists returned false for existing config")
	}
}

func TestGetConfigPath(t *testing.T) {
	// Create a temporary directory with a config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".worktree.yaml")

	// Write a test config
	err := os.WriteFile(configPath, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Get config path
	found, err := GetConfigPath(tmpDir)
	if err != nil {
		t.Fatalf("GetConfigPath failed: %v", err)
	}

	if found != configPath {
		t.Errorf("Expected config path %s, got %s", configPath, found)
	}
}
