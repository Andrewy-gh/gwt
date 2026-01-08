package config

// Config represents the root configuration from .worktree.yaml
type Config struct {
	CopyDefaults []string             `mapstructure:"copy_defaults" yaml:"copy_defaults"`
	CopyExclude  []string             `mapstructure:"copy_exclude" yaml:"copy_exclude"`
	Docker       DockerConfig         `mapstructure:"docker" yaml:"docker"`
	Dependencies DependenciesConfig   `mapstructure:"dependencies" yaml:"dependencies"`
	Migrations   MigrationsConfig     `mapstructure:"migrations" yaml:"migrations"`
	Hooks        HooksConfig          `mapstructure:"hooks" yaml:"hooks"`
}

// DockerConfig contains Docker/Compose-related settings
type DockerConfig struct {
	ComposeFiles    []string `mapstructure:"compose_files" yaml:"compose_files"`
	DataDirectories []string `mapstructure:"data_directories" yaml:"data_directories"`
	DefaultMode     string   `mapstructure:"default_mode" yaml:"default_mode"` // "shared" or "new"
	PortOffset      int      `mapstructure:"port_offset" yaml:"port_offset"`
}

// DependenciesConfig controls dependency installation behavior
type DependenciesConfig struct {
	AutoInstall bool     `mapstructure:"auto_install" yaml:"auto_install"`
	Paths       []string `mapstructure:"paths" yaml:"paths"`
}

// MigrationsConfig controls database migration behavior
type MigrationsConfig struct {
	AutoDetect bool   `mapstructure:"auto_detect" yaml:"auto_detect"`
	Command    string `mapstructure:"command" yaml:"command,omitempty"`
}

// HooksConfig contains lifecycle hooks
type HooksConfig struct {
	PostCreate []string `mapstructure:"post_create" yaml:"post_create"`
	PostDelete []string `mapstructure:"post_delete" yaml:"post_delete"`
}
