package docker

import (
	"fmt"
	"os"
	"path/filepath"
)

// SharedModeOptions configures shared mode setup
type SharedModeOptions struct {
	MainWorktree    string         // Path to main worktree
	NewWorktree     string         // Path to new worktree
	DataDirectories []string       // Directories to symlink (from config or detected)
	ComposeConfig   *ComposeConfig // Parsed compose config (for detection)
}

// SharedModeResult reports what was done
type SharedModeResult struct {
	LinkedDirs []LinkedDirectory
	Warnings   []string
}

// LinkedDirectory represents a directory that was linked
type LinkedDirectory struct {
	Source string     // Path in main worktree
	Target string     // Path in new worktree
	Method LinkResult // How it was linked
}

// SetupSharedMode creates symlinks for data directories
func SetupSharedMode(opts SharedModeOptions) (*SharedModeResult, error) {
	result := &SharedModeResult{}

	// 1. Get directories to share
	dirs := getDataDirectories(opts)
	if len(dirs) == 0 {
		result.Warnings = append(result.Warnings,
			"No data directories configured. Containers will use independent volumes.")
		return result, nil
	}

	// 2. Create symlinks for each directory
	for _, dir := range dirs {
		source := filepath.Join(opts.MainWorktree, dir)
		target := filepath.Join(opts.NewWorktree, dir)

		// Check source exists
		if _, err := os.Stat(source); os.IsNotExist(err) {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Data directory not found: %s (skipping)", dir))
			continue
		}

		// Create parent directory in target
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return nil, fmt.Errorf("failed to create parent directory: %w", err)
		}

		// Create link
		method, err := CreateLink(LinkOptions{
			Source: source,
			Target: target,
		})
		if err != nil {
			return nil, err
		}

		result.LinkedDirs = append(result.LinkedDirs, LinkedDirectory{
			Source: source,
			Target: target,
			Method: method,
		})

		// Warn if had to fall back to copy
		if method == LinkCopy {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Copied %s (symlink not available)", dir))
		}
	}

	return result, nil
}

// getDataDirectories returns directories to share
// Uses config if provided, otherwise detects from compose file
func getDataDirectories(opts SharedModeOptions) []string {
	// Use configured directories if provided
	if len(opts.DataDirectories) > 0 {
		return opts.DataDirectories
	}

	// Otherwise, try to detect from compose config
	if opts.ComposeConfig != nil {
		return ExtractDataDirectories(opts.ComposeConfig)
	}

	return nil
}
