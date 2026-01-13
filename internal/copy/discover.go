package copy

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// IgnoredFile represents a gitignored file or directory
type IgnoredFile struct {
	Path  string // Relative path from repo root
	IsDir bool   // Whether this is a directory
	Size  int64  // Size in bytes (0 for directories, calculated recursively)
}

// DiscoverIgnored finds all gitignored files in the given directory
// Returns a flat list of ignored files and directories
func DiscoverIgnored(repoPath string) ([]IgnoredFile, error) {
	// Run git status --ignored --porcelain
	cmd := exec.Command("git", "status", "--ignored", "--porcelain")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return nil, &CopyError{
			Path: repoPath,
			Op:   "git status",
			Err:  err,
		}
	}

	// Parse the output to get ignored file paths
	paths := parseIgnoredOutput(string(output))

	// Convert paths to IgnoredFile structs with size information
	var ignoredFiles []IgnoredFile
	for _, relPath := range paths {
		fullPath := filepath.Join(repoPath, relPath)

		// Get file info
		info, err := os.Stat(fullPath)
		if err != nil {
			// Skip files that can't be stat'd (might have been deleted)
			continue
		}

		ignoredFile := IgnoredFile{
			Path:  relPath,
			IsDir: info.IsDir(),
		}

		// Calculate size
		if info.IsDir() {
			// Calculate directory size recursively
			size, err := calculateDirSize(fullPath)
			if err != nil {
				// If we can't calculate size, use 0
				ignoredFile.Size = 0
			} else {
				ignoredFile.Size = size
			}
		} else {
			// File size is straightforward
			ignoredFile.Size = info.Size()
		}

		ignoredFiles = append(ignoredFiles, ignoredFile)
	}

	return ignoredFiles, nil
}

// parseIgnoredOutput parses git status --ignored --porcelain output
// Lines starting with "!! " are ignored files/directories
func parseIgnoredOutput(output string) []string {
	var paths []string
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		// Trim whitespace
		line = strings.TrimSpace(line)

		// Look for ignored files (!! prefix)
		if strings.HasPrefix(line, "!! ") {
			// Remove the "!! " prefix
			path := strings.TrimPrefix(line, "!! ")
			// Remove trailing slash if present (directory marker)
			path = strings.TrimSuffix(path, "/")

			if path != "" {
				paths = append(paths, path)
			}
		}
	}

	return paths
}

// calculateDirSize calculates the total size of a directory recursively
func calculateDirSize(dirPath string) (int64, error) {
	var size int64

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip files/dirs we can't access
			return nil
		}

		// Don't follow symlinks
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}

		if !info.IsDir() {
			size += info.Size()
		}

		return nil
	})

	return size, err
}
