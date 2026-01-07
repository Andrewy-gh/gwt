package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Version represents a Git version
type Version struct {
	Major int
	Minor int
	Patch int
}

// String returns the version as a string
func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// AtLeast checks if this version is at least the specified version
func (v Version) AtLeast(major, minor int) bool {
	if v.Major > major {
		return true
	}
	if v.Major == major && v.Minor >= minor {
		return true
	}
	return false
}

// IsInstalled checks if Git is installed and accessible
func IsInstalled() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// GetVersion returns the installed Git version
func GetVersion() (Version, error) {
	cmd := exec.Command("git", "--version")
	output, err := cmd.Output()
	if err != nil {
		return Version{}, fmt.Errorf("failed to get git version: %w", err)
	}

	// Parse output like "git version 2.43.0" or "git version 2.43.0.windows.1"
	versionStr := strings.TrimSpace(string(output))
	parts := strings.Fields(versionStr)
	if len(parts) < 3 {
		return Version{}, fmt.Errorf("unexpected git version output: %s", versionStr)
	}

	// Extract version numbers
	versionParts := strings.Split(parts[2], ".")
	if len(versionParts) < 2 {
		return Version{}, fmt.Errorf("unexpected version format: %s", parts[2])
	}

	major, err := strconv.Atoi(versionParts[0])
	if err != nil {
		return Version{}, fmt.Errorf("invalid major version: %s", versionParts[0])
	}

	minor, err := strconv.Atoi(versionParts[1])
	if err != nil {
		return Version{}, fmt.Errorf("invalid minor version: %s", versionParts[1])
	}

	patch := 0
	if len(versionParts) >= 3 {
		// Handle versions like "2.43.0.windows.1" - extract just the patch number
		patchStr := versionParts[2]
		if dotIdx := strings.Index(patchStr, "."); dotIdx > 0 {
			patchStr = patchStr[:dotIdx]
		}
		patch, _ = strconv.Atoi(patchStr)
	}

	return Version{Major: major, Minor: minor, Patch: patch}, nil
}

// IsRepository checks if the current directory is a Git repository
func IsRepository() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	return err == nil
}

// IsBareRepository checks if the current repository is bare
func IsBareRepository() (bool, error) {
	cmd := exec.Command("git", "rev-parse", "--is-bare-repository")
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check if bare repository: %w", err)
	}

	result := strings.TrimSpace(string(output))
	return result == "true", nil
}
