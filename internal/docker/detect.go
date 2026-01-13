package docker

import (
	"os"
	"path/filepath"
	"strings"
)

// ComposeFile represents a detected compose file
type ComposeFile struct {
	Path     string // Relative path from repo root
	FullPath string // Absolute path
	IsBase   bool   // Is this a base compose file (vs override)
}

var composePatterns = []string{
	"docker-compose.yml",
	"docker-compose.yaml",
	"compose.yml",
	"compose.yaml",
}

var overridePatterns = []string{
	"docker-compose.*.yml",
	"docker-compose.*.yaml",
	"compose.*.yml",
	"compose.*.yaml",
}

// Base file priority (higher = more preferred)
var basePriority = map[string]int{
	"docker-compose.yml":  4,
	"docker-compose.yaml": 3,
	"compose.yml":         2,
	"compose.yaml":        1,
}

// DetectComposeFiles finds all compose files in the given directory
// Searches for:
// - docker-compose.yml / docker-compose.yaml
// - docker-compose.*.yml / docker-compose.*.yaml
// - compose.yml / compose.yaml
// - compose.*.yml / compose.*.yaml
func DetectComposeFiles(repoPath string) ([]ComposeFile, error) {
	var files []ComposeFile

	// Check base compose files first
	for _, pattern := range composePatterns {
		fullPath := filepath.Join(repoPath, pattern)
		if _, err := os.Stat(fullPath); err == nil {
			files = append(files, ComposeFile{
				Path:     pattern,
				FullPath: fullPath,
				IsBase:   true,
			})
		}
	}

	// Check for override files
	entries, err := os.ReadDir(repoPath)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if IsOverrideFile(name) {
			fullPath := filepath.Join(repoPath, name)
			files = append(files, ComposeFile{
				Path:     name,
				FullPath: fullPath,
				IsBase:   false,
			})
		}
	}

	if len(files) == 0 {
		return nil, ErrNoComposeFile
	}

	return files, nil
}

// GetBaseComposeFile returns the primary compose file (not an override)
// Priority: docker-compose.yml > docker-compose.yaml > compose.yml > compose.yaml
func GetBaseComposeFile(files []ComposeFile) *ComposeFile {
	var bestFile *ComposeFile
	bestPriority := 0

	for i := range files {
		if !files[i].IsBase {
			continue
		}

		priority, exists := basePriority[files[i].Path]
		if exists && priority > bestPriority {
			bestPriority = priority
			bestFile = &files[i]
		}
	}

	return bestFile
}

// IsOverrideFile checks if a compose file is an override (has .*.yml pattern)
func IsOverrideFile(filename string) bool {
	// Check for docker-compose.*.yml or docker-compose.*.yaml pattern
	if strings.HasPrefix(filename, "docker-compose.") {
		parts := strings.Split(filename, ".")
		// Must have at least 3 parts: docker-compose, something, yml/yaml
		if len(parts) >= 3 {
			ext := parts[len(parts)-1]
			middle := strings.Join(parts[1:len(parts)-1], ".")
			// It's an override if there's something between docker-compose and yml/yaml
			// but not just the base file name
			if (ext == "yml" || ext == "yaml") && middle != "" && middle != "yml" && middle != "yaml" {
				return true
			}
		}
	}

	// Check for compose.*.yml or compose.*.yaml pattern
	if strings.HasPrefix(filename, "compose.") {
		parts := strings.Split(filename, ".")
		// Must have at least 3 parts: compose, something, yml/yaml
		if len(parts) >= 3 {
			ext := parts[len(parts)-1]
			middle := strings.Join(parts[1:len(parts)-1], ".")
			// It's an override if there's something between compose and yml/yaml
			if (ext == "yml" || ext == "yaml") && middle != "" && middle != "yml" && middle != "yaml" {
				return true
			}
		}
	}

	return false
}

// DetectOrLoad detects compose files or loads configured files
// If configFiles is specified, use those instead of auto-detection
func DetectOrLoad(repoPath string, configFiles []string) ([]ComposeFile, error) {
	if len(configFiles) > 0 {
		return loadConfiguredFiles(repoPath, configFiles)
	}
	return DetectComposeFiles(repoPath)
}

// loadConfiguredFiles loads compose files from configuration
func loadConfiguredFiles(repoPath string, configFiles []string) ([]ComposeFile, error) {
	var files []ComposeFile

	for _, relPath := range configFiles {
		fullPath := filepath.Join(repoPath, relPath)
		if _, err := os.Stat(fullPath); err != nil {
			if os.IsNotExist(err) {
				continue // Skip missing configured files
			}
			return nil, err
		}

		// Determine if it's a base file based on name
		isBase := !IsOverrideFile(filepath.Base(relPath))
		files = append(files, ComposeFile{
			Path:     relPath,
			FullPath: fullPath,
			IsBase:   isBase,
		})
	}

	if len(files) == 0 {
		return nil, ErrNoComposeFile
	}

	return files, nil
}
