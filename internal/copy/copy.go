package copy

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CopyProgress reports progress during copying
type CopyProgress struct {
	CurrentFile string
	FilesDone   int
	FilesTotal  int
	BytesDone   int64
	BytesTotal  int64
}

// ProgressCallback is called during copying to report progress
type ProgressCallback func(progress CopyProgress)

// CopyOptions configures the copy operation
type CopyOptions struct {
	SourceDir    string            // Source directory (main worktree)
	TargetDir    string            // Target directory (new worktree)
	Files        []SelectableFile  // Files to copy
	OnProgress   ProgressCallback  // Progress callback (optional)
	PreserveMode bool              // Preserve file permissions
}

// CopyResult reports the result of a copy operation
type CopyResult struct {
	FilesCopied int
	BytesCopied int64
	Errors      []CopyError // Non-fatal errors encountered
}

// Copy copies selected files from source to target
func Copy(opts CopyOptions) (*CopyResult, error) {
	result := &CopyResult{}

	// Calculate total bytes for progress tracking
	var totalBytes int64
	for _, file := range opts.Files {
		if file.Selected {
			totalBytes += file.Size
		}
	}

	var bytesCopied int64
	filesDone := 0

	// Copy each selected file
	for _, file := range opts.Files {
		if !file.Selected {
			continue
		}

		// Report progress before copying
		if opts.OnProgress != nil {
			opts.OnProgress(CopyProgress{
				CurrentFile: file.Path,
				FilesDone:   filesDone,
				FilesTotal:  len(opts.Files),
				BytesDone:   bytesCopied,
				BytesTotal:  totalBytes,
			})
		}

		// Build full paths
		srcPath := filepath.Join(opts.SourceDir, file.Path)
		dstPath := filepath.Join(opts.TargetDir, file.Path)

		// Copy file or directory
		var err error
		if file.IsDir {
			err = copyDirectory(srcPath, dstPath, opts.PreserveMode)
		} else {
			err = copyFile(srcPath, dstPath, opts.PreserveMode)
		}

		if err != nil {
			// Collect non-fatal errors
			result.Errors = append(result.Errors, CopyError{
				Path: file.Path,
				Op:   "copy",
				Err:  err,
			})
		} else {
			result.FilesCopied++
			result.BytesCopied += file.Size
			bytesCopied += file.Size
		}

		filesDone++
	}

	// Final progress report
	if opts.OnProgress != nil {
		opts.OnProgress(CopyProgress{
			CurrentFile: "",
			FilesDone:   filesDone,
			FilesTotal:  len(opts.Files),
			BytesDone:   bytesCopied,
			BytesTotal:  totalBytes,
		})
	}

	return result, nil
}

// copyFile copies a single file
func copyFile(src, dst string, preserveMode bool) error {
	// Check if source exists
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("source file error: %w", err)
	}

	// Skip symlinks
	if srcInfo.Mode()&os.ModeSymlink != 0 {
		return nil
	}

	// Check if target already exists
	if _, err := os.Stat(dst); err == nil {
		// Target exists, skip
		return nil
	}

	// Create parent directory
	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("create parent directory: %w", err)
	}

	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer srcFile.Close()

	// Create destination file
	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create destination: %w", err)
	}
	defer dstFile.Close()

	// Copy contents
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("copy contents: %w", err)
	}

	// Preserve permissions if requested
	if preserveMode {
		if err := os.Chmod(dst, srcInfo.Mode()); err != nil {
			// Non-fatal, just continue
			return nil
		}
	}

	return nil
}

// copyDirectory copies a directory recursively
func copyDirectory(src, dst string, preserveMode bool) error {
	// Check if source exists
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("source directory error: %w", err)
	}

	// Create destination directory
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return fmt.Errorf("create destination directory: %w", err)
	}

	// Walk the source directory
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip symlinks
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}

		// Calculate relative path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		// Build destination path
		dstPath := filepath.Join(dst, relPath)

		// Copy directory or file
		if info.IsDir() {
			// Create directory
			if err := os.MkdirAll(dstPath, info.Mode()); err != nil {
				return fmt.Errorf("create directory %s: %w", relPath, err)
			}
		} else {
			// Copy file
			if err := copyFile(path, dstPath, preserveMode); err != nil {
				// Non-fatal, continue with other files
				return nil
			}
		}

		return nil
	})
}
