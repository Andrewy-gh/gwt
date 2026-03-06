package docker

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

// LinkResult indicates how a link was created
type LinkResult int

const (
	LinkSymlink  LinkResult = iota // Symlink created successfully
	LinkJunction                   // Junction created (Windows only)
	LinkCopy                       // Fell back to copy
	LinkFailed                     // All methods failed
)

// String returns a human-readable description of the link result
func (lr LinkResult) String() string {
	switch lr {
	case LinkSymlink:
		return "symlink"
	case LinkJunction:
		return "junction"
	case LinkCopy:
		return "copy"
	case LinkFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// LinkOptions configures link creation
type LinkOptions struct {
	Source      string // Source directory (main worktree)
	Target      string // Target path (new worktree)
	FallbackMsg string // Message to show if falling back
}

const WindowsSymlinkHelp = `
Symlink creation failed. On Windows, you need one of:
  1. Run as Administrator
  2. Enable Developer Mode (Settings > Update & Security > For developers)
  3. Grant SeCreateSymbolicLinkPrivilege to your user

gwt will use directory junctions as a fallback.
`

// CreateLink creates a symlink, falling back to junction then copy on Windows
// Returns the method used and any error
func CreateLink(opts LinkOptions) (LinkResult, error) {
	// Ensure source exists
	sourceInfo, err := os.Stat(opts.Source)
	if err != nil {
		return LinkFailed, fmt.Errorf("source does not exist: %w", err)
	}

	if !sourceInfo.IsDir() {
		return LinkFailed, fmt.Errorf("source is not a directory")
	}

	// Remove target if it exists
	if _, err := os.Lstat(opts.Target); err == nil {
		if err := os.RemoveAll(opts.Target); err != nil {
			return LinkFailed, fmt.Errorf("failed to remove existing target: %w", err)
		}
	}

	// Try symlink first
	if err := createSymlink(opts.Source, opts.Target); err == nil {
		return LinkSymlink, nil
	}

	// On Windows, try junction
	if runtime.GOOS == "windows" {
		if err := createJunction(opts.Source, opts.Target); err == nil {
			return LinkJunction, nil
		}
	}

	// Fall back to copy
	if err := fallbackCopy(opts.Source, opts.Target); err != nil {
		return LinkFailed, fmt.Errorf("all link methods failed: %w", err)
	}

	return LinkCopy, nil
}

// createSymlink attempts to create a symbolic link
func createSymlink(source, target string) error {
	return os.Symlink(source, target)
}

// fallbackCopy copies the directory when symlink/junction fails
func fallbackCopy(source, target string) error {
	return copyDir(source, target)
}

// copyDir recursively copies a directory
func copyDir(src, dst string) error {
	// Get source directory info
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	// Create destination directory
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	// Read source directory
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	// Copy each entry
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			// Recursively copy subdirectory
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// Copy file
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFile copies a single file
func copyFile(src, dst string) error {
	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Get source file info
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	// Create destination file
	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// Copy contents
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	return nil
}

// CreateSymlink creates a symlink with fallback to junction/copy on Windows
// This is a convenience wrapper around CreateLink that returns just an error
func CreateSymlink(source, target string) error {
	result, err := CreateLink(LinkOptions{
		Source: source,
		Target: target,
	})
	if result == LinkFailed {
		return err
	}
	return nil
}

// CanCreateSymlink checks if the current process can create symlinks
func CanCreateSymlink() bool {
	// Create a temp file and try to symlink to it
	tmpDir := os.TempDir()
	src := filepath.Join(tmpDir, "gwt_symlink_test_src")
	dst := filepath.Join(tmpDir, "gwt_symlink_test_dst")

	// Create source file
	if err := os.WriteFile(src, []byte("test"), 0644); err != nil {
		return false
	}
	defer os.Remove(src)
	defer os.Remove(dst)

	// Try symlink
	if err := os.Symlink(src, dst); err != nil {
		return false
	}
	return true
}

// CanCreateJunction checks if junctions are available (Windows only)
func CanCreateJunction() bool {
	if runtime.GOOS != "windows" {
		return false
	}

	// Create temp directories and try to create a junction
	tmpDir := os.TempDir()
	src := filepath.Join(tmpDir, "gwt_junction_test_src")
	dst := filepath.Join(tmpDir, "gwt_junction_test_dst")

	// Create source directory
	if err := os.MkdirAll(src, 0755); err != nil {
		return false
	}
	defer os.RemoveAll(src)
	defer os.RemoveAll(dst)

	// Try junction
	if err := createJunction(src, dst); err != nil {
		return false
	}

	return true
}
