package docker

import (
	"fmt"
	"os"
	"path/filepath"
)

// NewModeOptions configures new mode setup
type NewModeOptions struct {
	MainWorktree    string         // Path to main worktree
	NewWorktree     string         // Path to new worktree
	BranchName      string         // Branch name for suffixes
	DataDirectories []string       // Directories to copy
	ComposeConfig   *ComposeConfig // Parsed compose config
	PortOffset      int            // Port offset
}

// NewModeResult reports what was done
type NewModeResult struct {
	CopiedDirs     []string
	OverrideFile   string
	RenamedVolumes map[string]string
	RemappedPorts  map[string]int
	PortWarnings   []string
	Warnings       []string
}

// SetupNewMode sets up isolated containers for the new worktree
func SetupNewMode(opts NewModeOptions) (*NewModeResult, error) {
	result := &NewModeResult{
		RenamedVolumes: make(map[string]string),
		RemappedPorts:  make(map[string]int),
	}

	// 1. Get directories to copy
	dirs := getDataDirectoriesToCopy(opts)

	// 2. Copy data directories
	for _, dir := range dirs {
		source := filepath.Join(opts.MainWorktree, dir)
		target := filepath.Join(opts.NewWorktree, dir)

		if _, err := os.Stat(source); os.IsNotExist(err) {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Data directory not found: %s (skipping)", dir))
			continue
		}

		// Create parent directory
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return nil, fmt.Errorf("failed to create parent directory for %s: %w", dir, err)
		}

		// Copy directory
		if err := copyDir(source, target); err != nil {
			return nil, fmt.Errorf("failed to copy %s: %w", dir, err)
		}
		result.CopiedDirs = append(result.CopiedDirs, dir)
	}

	// 3. Generate override file
	overridePath := filepath.Join(opts.NewWorktree, "docker-compose.worktree.yml")
	overrideResult, err := GenerateOverride(OverrideOptions{
		BranchName:     opts.BranchName,
		OriginalConfig: opts.ComposeConfig,
		PortOffset:     opts.PortOffset,
		OutputPath:     overridePath,
	})
	if err != nil {
		return nil, err
	}

	result.OverrideFile = overrideResult.FilePath
	result.RenamedVolumes = overrideResult.RenamedVolumes
	result.RemappedPorts = overrideResult.RemappedPorts
	result.PortWarnings = overrideResult.PortWarnings

	return result, nil
}

// getDataDirectoriesToCopy returns directories to copy
// Uses config if provided, otherwise detects from compose file
func getDataDirectoriesToCopy(opts NewModeOptions) []string {
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
