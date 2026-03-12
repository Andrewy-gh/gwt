package config

// DefaultConfig returns a new Config with default values
func DefaultConfig() *Config {
	return &Config{
		CopyDefaults: []string{
			".env",
			"**/.env",
			".env.*",
			"**/.env.*",
			"*.local",
			"**/*.local",
			"*.local.*",
			"**/*.local.*",
		},
		CopyExclude: []string{
			"node_modules",
			"vendor",
			".venv",
			"__pycache__",
			"target",
			"dist",
			"build",
			"*.log",
		},
		Docker: DockerConfig{
			ComposeFiles:    []string{}, // Auto-detect if empty
			DataDirectories: []string{},
			DefaultMode:     "shared",
			PortOffset:      1,
		},
		Dependencies: DependenciesConfig{
			AutoInstall: true,
			Paths: []string{
				".",
			},
		},
		Migrations: MigrationsConfig{
			AutoDetect: true,
			Command:    "", // Auto-detect if empty
		},
		Hooks: HooksConfig{
			PostCreate: []string{},
			PostDelete: []string{},
		},
		Performance: DefaultPerformanceConfig(),
	}
}

// DefaultPerformanceConfig returns default performance configuration
func DefaultPerformanceConfig() PerformanceConfig {
	return PerformanceConfig{
		Cache: CacheConfig{
			Enabled:      true,
			TTLStatus:    "30s",
			TTLBranches:  "5m",
			TTLWorktrees: "1m",
		},
		Concurrency: ConcurrencyConfig{
			MaxWorkers: 8,
			BatchSize:  10,
		},
	}
}
