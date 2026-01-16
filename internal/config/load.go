package config

import (
	"os"
	"path/filepath"

	"github.com/Andrewy-gh/gwt/internal/git"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

const configFileName = ".worktree.yaml"

// Load reads configuration from the specified path
// If path is empty, searches in the current directory and ancestors
func Load(path string) (*Config, error) {
	if path == "" {
		return LoadFromDir(".")
	}

	// Check if the path is a directory or file
	info, err := os.Stat(path)
	if err != nil {
		return nil, &ConfigParseError{
			Path: path,
			Err:  err,
		}
	}

	if info.IsDir() {
		return LoadFromDir(path)
	}

	// Load from specific file
	return loadFromFile(path)
}

// LoadFromDir loads configuration starting from the given directory
func LoadFromDir(dir string) (*Config, error) {
	configPath, err := FindConfigFile(dir)
	if err != nil {
		return nil, err
	}

	// If no config file found, return defaults
	if configPath == "" {
		return DefaultConfig(), nil
	}

	return loadFromFile(configPath)
}

// loadFromFile loads configuration from a specific file
func loadFromFile(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)

	// Read the config file
	if err := v.ReadInConfig(); err != nil {
		return nil, &ConfigParseError{
			Path: path,
			Err:  err,
		}
	}

	// Unmarshal into an empty config struct
	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, &ConfigParseError{
			Path: path,
			Err:  err,
		}
	}

	// Apply defaults for fields that weren't set
	cfg = mergeWithDefaults(cfg)

	// Validate the config
	if validationErrors := cfg.Validate(); len(validationErrors) > 0 {
		// Return the first validation error
		return nil, &validationErrors[0]
	}

	return cfg, nil
}

// mergeWithDefaults applies default values to fields that weren't set in the config file
func mergeWithDefaults(cfg *Config) *Config {
	defaults := DefaultConfig()

	// Merge slices - only use defaults if the config slice is empty
	if len(cfg.CopyDefaults) == 0 {
		cfg.CopyDefaults = defaults.CopyDefaults
	}
	if len(cfg.CopyExclude) == 0 {
		cfg.CopyExclude = defaults.CopyExclude
	}

	// Docker config
	if cfg.Docker.DefaultMode == "" {
		cfg.Docker.DefaultMode = defaults.Docker.DefaultMode
	}
	// Port offset of 0 means not set, use default
	if cfg.Docker.PortOffset == 0 {
		cfg.Docker.PortOffset = defaults.Docker.PortOffset
	}

	// Dependencies
	// Note: AutoInstall defaults to false if not set, which we want
	if len(cfg.Dependencies.Paths) == 0 {
		cfg.Dependencies.Paths = defaults.Dependencies.Paths
	}

	// Migrations
	// Note: AutoDetect defaults to false if not set, which we want

	// Hooks - empty slices are valid, so don't replace them

	return cfg
}

// FindConfigFile searches for .worktree.yaml starting from dir
// Returns the path to the config file or empty string if not found
func FindConfigFile(dir string) (string, error) {
	// Convert to absolute path
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}

	// Check if we're in a git repository
	if !git.IsGitRepository(absDir) {
		// Not in a git repo, just check current directory
		configPath := filepath.Join(absDir, configFileName)
		if fileExists(configPath) {
			return configPath, nil
		}
		return "", nil
	}

	// Get the repository root
	repoRoot, err := git.GetRepoRoot(absDir)
	if err != nil {
		// If we can't get repo root, just check current directory
		configPath := filepath.Join(absDir, configFileName)
		if fileExists(configPath) {
			return configPath, nil
		}
		return "", nil
	}

	// Check in repository root
	configPath := filepath.Join(repoRoot, configFileName)
	if fileExists(configPath) {
		return configPath, nil
	}

	return "", nil
}

// ConfigExists checks if a config file exists in the given directory
func ConfigExists(dir string) bool {
	configPath, _ := FindConfigFile(dir)
	return configPath != ""
}

// GetConfigPath returns the path where config would be loaded from
// Returns empty string if no config file exists
func GetConfigPath(dir string) (string, error) {
	return FindConfigFile(dir)
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// LoadWithInheritance loads config with worktree inheritance
// If in a linked worktree, reads from main worktree first
// Returns the config, the path it was loaded from, and any error
func LoadWithInheritance(dir string) (*Config, string, error) {
	// Convert to absolute path
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, "", err
	}

	// Check if we're in a git repository
	if !git.IsGitRepository(absDir) {
		// Not in a git repo, use normal loading
		cfg, err := LoadFromDir(absDir)
		if err != nil {
			return nil, "", err
		}

		// Try to determine the source path
		configPath, _ := FindConfigFile(absDir)
		return cfg, configPath, nil
	}

	// Check if we're in a linked worktree
	isLinked, err := git.IsWorktree(absDir)
	if err != nil {
		// If we can't determine, fall back to normal loading
		cfg, err := LoadFromDir(absDir)
		if err != nil {
			return nil, "", err
		}
		configPath, _ := FindConfigFile(absDir)
		return cfg, configPath, nil
	}

	// If in a linked worktree, check for local config first
	if isLinked {
		localConfigPath := filepath.Join(absDir, configFileName)
		if fileExists(localConfigPath) {
			cfg, err := loadFromFile(localConfigPath)
			return cfg, localConfigPath, err
		}

		// No local config, try to inherit from main worktree
		mainPath, err := git.GetMainWorktreePath(absDir)
		if err == nil {
			mainConfigPath := filepath.Join(mainPath, configFileName)
			if fileExists(mainConfigPath) {
				cfg, err := loadFromFile(mainConfigPath)
				return cfg, mainConfigPath, err
			}
		}
	}

	// For main worktree or if no inherited config found, use normal loading
	cfg, err := LoadFromDir(absDir)
	if err != nil {
		return nil, "", err
	}

	configPath, _ := FindConfigFile(absDir)
	return cfg, configPath, nil
}

// GetEffectiveConfigPath returns the path where config will be loaded from
// accounting for worktree inheritance
func GetEffectiveConfigPath(dir string) (string, error) {
	_, path, err := LoadWithInheritance(dir)
	return path, err
}

// IsInheritedConfig checks if the loaded config came from main worktree
func IsInheritedConfig(dir string) (bool, error) {
	// Convert to absolute path
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return false, err
	}

	// Check if we're in a linked worktree
	isLinked, err := git.IsWorktree(absDir)
	if err != nil {
		return false, err
	}

	if !isLinked {
		return false, nil // Not a linked worktree, so not inherited
	}

	// Check if local config exists
	localConfigPath := filepath.Join(absDir, configFileName)
	if fileExists(localConfigPath) {
		return false, nil // Has local config, so not inherited
	}

	// Check if main worktree config exists
	mainPath, err := git.GetMainWorktreePath(absDir)
	if err != nil {
		return false, err
	}

	mainConfigPath := filepath.Join(mainPath, configFileName)
	if fileExists(mainConfigPath) {
		return true, nil // Using main worktree config
	}

	return false, nil // No config at all, using defaults
}

// Save writes the configuration to a file at the specified directory
// If the config file doesn't exist, it creates a new one
func Save(dir string, cfg *Config) error {
	// Convert to absolute path
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	// Get the repository root if we're in a git repo
	if git.IsGitRepository(absDir) {
		repoRoot, err := git.GetRepoRoot(absDir)
		if err == nil {
			absDir = repoRoot
		}
	}

	// Determine config file path
	configPath := filepath.Join(absDir, configFileName)

	// Validate the config before saving
	if validationErrors := cfg.Validate(); len(validationErrors) > 0 {
		return &validationErrors[0]
	}

	// Marshal to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return &ConfigParseError{
			Path: configPath,
			Err:  err,
		}
	}

	// Write to file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return &ConfigParseError{
			Path: configPath,
			Err:  err,
		}
	}

	return nil
}
