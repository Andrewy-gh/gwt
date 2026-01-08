package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestConfigMarshalYAML(t *testing.T) {
	cfg := DefaultConfig()

	// Marshal to YAML
	yamlData, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("Failed to marshal config to YAML: %v", err)
	}

	// Unmarshal back to Config
	var unmarshaledCfg Config
	err = yaml.Unmarshal(yamlData, &unmarshaledCfg)
	if err != nil {
		t.Fatalf("Failed to unmarshal YAML to config: %v", err)
	}

	// Verify some key fields
	if unmarshaledCfg.Docker.DefaultMode != cfg.Docker.DefaultMode {
		t.Errorf("Expected docker.default_mode %s, got %s", cfg.Docker.DefaultMode, unmarshaledCfg.Docker.DefaultMode)
	}

	if unmarshaledCfg.Docker.PortOffset != cfg.Docker.PortOffset {
		t.Errorf("Expected docker.port_offset %d, got %d", cfg.Docker.PortOffset, unmarshaledCfg.Docker.PortOffset)
	}

	if unmarshaledCfg.Dependencies.AutoInstall != cfg.Dependencies.AutoInstall {
		t.Errorf("Expected dependencies.auto_install %v, got %v", cfg.Dependencies.AutoInstall, unmarshaledCfg.Dependencies.AutoInstall)
	}

	if unmarshaledCfg.Migrations.AutoDetect != cfg.Migrations.AutoDetect {
		t.Errorf("Expected migrations.auto_detect %v, got %v", cfg.Migrations.AutoDetect, unmarshaledCfg.Migrations.AutoDetect)
	}
}

func TestConfigUnmarshalMinimal(t *testing.T) {
	yamlStr := `
copy_defaults:
  - ".env"

docker:
  default_mode: "shared"
  port_offset: 1

dependencies:
  auto_install: true

migrations:
  auto_detect: true
`

	var cfg Config
	err := yaml.Unmarshal([]byte(yamlStr), &cfg)
	if err != nil {
		t.Fatalf("Failed to unmarshal minimal YAML: %v", err)
	}

	// Verify minimal config
	if len(cfg.CopyDefaults) != 1 {
		t.Errorf("Expected 1 copy_defaults, got %d", len(cfg.CopyDefaults))
	}

	if cfg.Docker.DefaultMode != "shared" {
		t.Errorf("Expected docker.default_mode 'shared', got '%s'", cfg.Docker.DefaultMode)
	}

	if cfg.Docker.PortOffset != 1 {
		t.Errorf("Expected docker.port_offset 1, got %d", cfg.Docker.PortOffset)
	}

	if !cfg.Dependencies.AutoInstall {
		t.Error("Expected dependencies.auto_install true, got false")
	}

	if !cfg.Migrations.AutoDetect {
		t.Error("Expected migrations.auto_detect true, got false")
	}
}

func TestConfigUnmarshalFull(t *testing.T) {
	yamlStr := `
copy_defaults:
  - ".env"
  - "**/.env"
  - "**/.env.local"

copy_exclude:
  - "node_modules"
  - "vendor"

docker:
  compose_files:
    - "docker-compose.yml"
    - "docker-compose.dev.yml"
  data_directories:
    - "postgres-data"
  default_mode: "new"
  port_offset: 10

dependencies:
  auto_install: false
  paths:
    - "."
    - "client"
    - "server"

migrations:
  auto_detect: false
  command: "make migrate-up"

hooks:
  post_create:
    - "echo 'Worktree created!'"
  post_delete:
    - "echo 'Worktree deleted!'"
`

	var cfg Config
	err := yaml.Unmarshal([]byte(yamlStr), &cfg)
	if err != nil {
		t.Fatalf("Failed to unmarshal full YAML: %v", err)
	}

	// Verify full config
	if len(cfg.CopyDefaults) != 3 {
		t.Errorf("Expected 3 copy_defaults, got %d", len(cfg.CopyDefaults))
	}

	if len(cfg.CopyExclude) != 2 {
		t.Errorf("Expected 2 copy_exclude, got %d", len(cfg.CopyExclude))
	}

	if len(cfg.Docker.ComposeFiles) != 2 {
		t.Errorf("Expected 2 compose_files, got %d", len(cfg.Docker.ComposeFiles))
	}

	if len(cfg.Docker.DataDirectories) != 1 {
		t.Errorf("Expected 1 data_directories, got %d", len(cfg.Docker.DataDirectories))
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

	if len(cfg.Hooks.PostCreate) != 1 {
		t.Errorf("Expected 1 post_create hook, got %d", len(cfg.Hooks.PostCreate))
	}

	if len(cfg.Hooks.PostDelete) != 1 {
		t.Errorf("Expected 1 post_delete hook, got %d", len(cfg.Hooks.PostDelete))
	}
}
