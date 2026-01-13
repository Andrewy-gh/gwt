package install

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// DetectPackageManagers finds all package managers in the given paths
// relative to the worktree root
func DetectPackageManagers(worktreePath string, paths []string) ([]PackageManager, error) {
	var managers []PackageManager
	seen := make(map[string]bool) // Prevent duplicates

	for _, relPath := range paths {
		absPath := filepath.Join(worktreePath, relPath)

		// Expand glob patterns using doublestar
		matches, err := doublestar.FilepathGlob(absPath)
		if err != nil {
			continue // Skip invalid patterns
		}

		// If no matches and the path exists as a directory, use it directly
		if len(matches) == 0 {
			if info, err := os.Stat(absPath); err == nil && info.IsDir() {
				matches = []string{absPath}
			}
		}

		for _, match := range matches {
			// Ensure match is a directory
			if info, err := os.Stat(match); err != nil || !info.IsDir() {
				continue
			}

			if seen[match] {
				continue
			}

			if pm := detectInDirectory(match); pm != nil {
				seen[match] = true
				managers = append(managers, *pm)
			}
		}
	}

	return managers, nil
}

// detectInDirectory checks a single directory for package managers
// Returns the most specific package manager found (lock files preferred)
func detectInDirectory(dir string) *PackageManager {
	// Check JavaScript/Node.js (order matters - most specific first)
	if fileExists(filepath.Join(dir, "bun.lock")) {
		return &PackageManager{
			Name:        "bun",
			Path:        dir,
			LockFile:    "bun.lock",
			InstallCmd:  "bun",
			InstallArgs: []string{"install"},
		}
	}
	if fileExists(filepath.Join(dir, "pnpm-lock.yaml")) {
		return &PackageManager{
			Name:        "pnpm",
			Path:        dir,
			LockFile:    "pnpm-lock.yaml",
			InstallCmd:  "pnpm",
			InstallArgs: []string{"install"},
		}
	}
	if fileExists(filepath.Join(dir, "yarn.lock")) {
		return &PackageManager{
			Name:        "yarn",
			Path:        dir,
			LockFile:    "yarn.lock",
			InstallCmd:  "yarn",
			InstallArgs: []string{"install"},
		}
	}
	if fileExists(filepath.Join(dir, "package-lock.json")) {
		return &PackageManager{
			Name:        "npm",
			Path:        dir,
			LockFile:    "package-lock.json",
			InstallCmd:  "npm",
			InstallArgs: []string{"install"},
		}
	}
	if fileExists(filepath.Join(dir, "package.json")) {
		return &PackageManager{
			Name:        "npm",
			Path:        dir,
			LockFile:    "",
			InstallCmd:  "npm",
			InstallArgs: []string{"install"},
		}
	}

	// Check Go
	if fileExists(filepath.Join(dir, "go.mod")) {
		return &PackageManager{
			Name:        "go",
			Path:        dir,
			LockFile:    "go.sum",
			InstallCmd:  "go",
			InstallArgs: []string{"mod", "download"},
		}
	}

	// Check Rust/Cargo
	if fileExists(filepath.Join(dir, "Cargo.toml")) {
		return &PackageManager{
			Name:        "cargo",
			Path:        dir,
			LockFile:    "Cargo.lock",
			InstallCmd:  "cargo",
			InstallArgs: []string{"fetch"},
		}
	}

	// Check Python - poetry first (more specific)
	if fileExists(filepath.Join(dir, "poetry.lock")) {
		return &PackageManager{
			Name:        "poetry",
			Path:        dir,
			LockFile:    "poetry.lock",
			InstallCmd:  "poetry",
			InstallArgs: []string{"install"},
		}
	}
	if hasPoetryConfig(filepath.Join(dir, "pyproject.toml")) {
		return &PackageManager{
			Name:        "poetry",
			Path:        dir,
			LockFile:    "",
			InstallCmd:  "poetry",
			InstallArgs: []string{"install"},
		}
	}
	if fileExists(filepath.Join(dir, "requirements.txt")) {
		return &PackageManager{
			Name:        "pip",
			Path:        dir,
			LockFile:    "",
			InstallCmd:  "pip",
			InstallArgs: []string{"install", "-r", "requirements.txt"},
		}
	}

	return nil
}

// hasPoetryConfig checks if pyproject.toml contains [tool.poetry]
func hasPoetryConfig(path string) bool {
	if !fileExists(path) {
		return false
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(content), "[tool.poetry]")
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
