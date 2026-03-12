package create

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Andrewy-gh/gwt/internal/pathutil"
)

// ValidateBranchName checks if a branch name is valid for git
// Returns nil if valid, error with message if invalid
//
// Git branch name rules:
// - Cannot contain spaces or control characters
// - Cannot start with a dash (-)
// - Cannot contain ..
// - Cannot end with .lock
// - Cannot contain ~, ^, :, ?, *, [, \
// - Cannot contain @{
// - Cannot be a single @
func ValidateBranchName(name string) error {
	if name == "" {
		return fmt.Errorf("branch name cannot be empty")
	}

	// Cannot be a single @
	if name == "@" {
		return fmt.Errorf("branch name cannot be '@'")
	}

	// Cannot start with a dash
	if strings.HasPrefix(name, "-") {
		return fmt.Errorf("branch name cannot start with '-'")
	}

	// Cannot end with .lock
	if strings.HasSuffix(name, ".lock") {
		return fmt.Errorf("branch name cannot end with '.lock'")
	}

	// Cannot contain ..
	if strings.Contains(name, "..") {
		return fmt.Errorf("branch name cannot contain '..'")
	}

	// Cannot contain @{
	if strings.Contains(name, "@{") {
		return fmt.Errorf("branch name cannot contain '@{'")
	}

	// Cannot contain control characters or spaces
	for _, ch := range name {
		if ch < 32 || ch == 127 {
			return fmt.Errorf("branch name cannot contain control characters")
		}
		if ch == ' ' {
			return fmt.Errorf("branch name cannot contain spaces")
		}
	}

	// Cannot contain: ~, ^, :, ?, *, [, \
	invalidChars := []string{"~", "^", ":", "?", "*", "[", "\\"}
	for _, char := range invalidChars {
		if strings.Contains(name, char) {
			return fmt.Errorf("branch name cannot contain '%s'", char)
		}
	}

	// Additional check: cannot end with /
	if strings.HasSuffix(name, "/") {
		return fmt.Errorf("branch name cannot end with '/'")
	}

	// Additional check: cannot have multiple consecutive slashes
	if strings.Contains(name, "//") {
		return fmt.Errorf("branch name cannot contain consecutive slashes '//'")
	}

	// Windows-specific: Check for reserved device names
	// This applies on Windows and should be checked cross-platform for compatibility
	if filepath.Separator == '\\' { // Windows
		// Check the branch name and each component in the path
		components := strings.Split(name, "/")
		for _, component := range components {
			// Remove extension if present
			baseName := component
			if idx := strings.IndexByte(component, '.'); idx != -1 {
				baseName = component[:idx]
			}

			reservedNames := []string{"CON", "PRN", "AUX", "NUL",
				"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
				"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9"}

			upperBase := strings.ToUpper(baseName)
			for _, reserved := range reservedNames {
				if upperBase == reserved {
					return fmt.Errorf("branch name '%s' contains reserved Windows device name '%s'", name, component)
				}
			}
		}
	}

	return nil
}

// SanitizeDirectoryName converts a branch name to a valid directory name
// Example: "feature/auth/login" -> "feature-auth-login"
//
// Conversion rules:
// - Replace / with -
// - Replace \ with -
// - Remove leading/trailing dashes
// - Collapse multiple consecutive dashes
func SanitizeDirectoryName(branchName string) string {
	// Replace slashes with dashes
	name := strings.ReplaceAll(branchName, "/", "-")
	name = strings.ReplaceAll(name, "\\", "-")

	// Collapse multiple consecutive dashes
	re := regexp.MustCompile(`-+`)
	name = re.ReplaceAllString(name, "-")

	// Remove leading and trailing dashes
	name = strings.Trim(name, "-")

	return name
}

// GenerateWorktreePath generates the target directory path for a worktree
// Places worktrees as siblings to the main worktree: ../project-branch-name
//
// Example:
//
//	Main worktree: /home/user/myproject
//	Branch: feature/auth
//	Result: /home/user/myproject-feature-auth
func GenerateWorktreePath(mainWorktreePath, branchName string) string {
	// Get parent directory of main worktree
	parentDir := filepath.Dir(mainWorktreePath)

	// Get project name from main worktree
	projectName := pathutil.Base(mainWorktreePath)

	// Sanitize branch name for directory
	dirName := SanitizeDirectoryName(branchName)

	// Combine: parent/project-branch
	return filepath.Join(parentDir, projectName+"-"+dirName)
}

// ValidateDirectoryName checks if a directory name is valid for the OS
// This is a basic check for common invalid characters
func ValidateDirectoryName(name string) error {
	if name == "" {
		return fmt.Errorf("directory name cannot be empty")
	}

	// Check for invalid characters (common across Windows and Unix)
	// Windows: < > : " | ? * and control characters
	// Unix: mainly just null byte, but we'll be conservative
	invalidChars := []string{"<", ">", ":", "\"", "|", "?", "*"}
	for _, char := range invalidChars {
		if strings.Contains(name, char) {
			return fmt.Errorf("directory name cannot contain '%s'", char)
		}
	}

	// Check for control characters
	for _, ch := range name {
		if ch < 32 || ch == 127 {
			return fmt.Errorf("directory name cannot contain control characters")
		}
	}

	// Check for reserved names on Windows
	reservedNames := []string{"CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9"}

	upperName := strings.ToUpper(name)
	for _, reserved := range reservedNames {
		if upperName == reserved || strings.HasPrefix(upperName, reserved+".") {
			return fmt.Errorf("directory name '%s' is reserved on Windows", name)
		}
	}

	return nil
}
