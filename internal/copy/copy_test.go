package copy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyFile(t *testing.T) {
	// Create temp directory for testing
	tmpDir := t.TempDir()

	// Create source file
	srcPath := filepath.Join(tmpDir, "source.txt")
	content := []byte("test content")
	if err := os.WriteFile(srcPath, content, 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Copy file
	dstPath := filepath.Join(tmpDir, "dest.txt")
	if err := copyFile(srcPath, dstPath, false); err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}

	// Verify destination exists and has same content
	dstContent, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}

	if string(dstContent) != string(content) {
		t.Errorf("Content mismatch: got %q, want %q", string(dstContent), string(content))
	}
}

func TestCopyFile_PreserveMode(t *testing.T) {
	// Create temp directory for testing
	tmpDir := t.TempDir()

	// Create source file with specific permissions
	srcPath := filepath.Join(tmpDir, "source.txt")
	content := []byte("test content")
	if err := os.WriteFile(srcPath, content, 0755); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Copy file with mode preservation
	dstPath := filepath.Join(tmpDir, "dest.txt")
	if err := copyFile(srcPath, dstPath, true); err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}

	// Verify permissions are preserved
	srcInfo, _ := os.Stat(srcPath)
	dstInfo, _ := os.Stat(dstPath)

	if srcInfo.Mode() != dstInfo.Mode() {
		t.Errorf("Mode mismatch: got %v, want %v", dstInfo.Mode(), srcInfo.Mode())
	}
}

func TestCopyFile_AlreadyExists(t *testing.T) {
	// Create temp directory for testing
	tmpDir := t.TempDir()

	// Create source file
	srcPath := filepath.Join(tmpDir, "source.txt")
	if err := os.WriteFile(srcPath, []byte("source"), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Create destination file
	dstPath := filepath.Join(tmpDir, "dest.txt")
	if err := os.WriteFile(dstPath, []byte("existing"), 0644); err != nil {
		t.Fatalf("Failed to create destination file: %v", err)
	}

	// Copy should not overwrite (returns nil, skips)
	if err := copyFile(srcPath, dstPath, false); err != nil {
		t.Fatalf("copyFile should not error on existing file: %v", err)
	}

	// Verify destination still has original content
	content, _ := os.ReadFile(dstPath)
	if string(content) != "existing" {
		t.Errorf("Existing file was overwritten")
	}
}

func TestCopyDirectory(t *testing.T) {
	// Create temp directory for testing
	tmpDir := t.TempDir()

	// Create source directory with files
	srcDir := filepath.Join(tmpDir, "source")
	if err := os.MkdirAll(filepath.Join(srcDir, "subdir"), 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	// Create files
	if err := os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("file1"), 0644); err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "subdir", "file2.txt"), []byte("file2"), 0644); err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}

	// Copy directory
	dstDir := filepath.Join(tmpDir, "dest")
	if err := copyDirectory(srcDir, dstDir, false); err != nil {
		t.Fatalf("copyDirectory failed: %v", err)
	}

	// Verify destination directory structure
	if _, err := os.Stat(dstDir); os.IsNotExist(err) {
		t.Error("Destination directory does not exist")
	}
	if _, err := os.Stat(filepath.Join(dstDir, "subdir")); os.IsNotExist(err) {
		t.Error("Destination subdirectory does not exist")
	}

	// Verify files
	content1, err := os.ReadFile(filepath.Join(dstDir, "file1.txt"))
	if err != nil || string(content1) != "file1" {
		t.Errorf("file1.txt not copied correctly")
	}

	content2, err := os.ReadFile(filepath.Join(dstDir, "subdir", "file2.txt"))
	if err != nil || string(content2) != "file2" {
		t.Errorf("file2.txt not copied correctly")
	}
}

func TestCopy_Progress(t *testing.T) {
	// Create temp directory for testing
	tmpDir := t.TempDir()

	// Create source files
	srcDir := filepath.Join(tmpDir, "source")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	file1 := filepath.Join(srcDir, "file1.txt")
	file2 := filepath.Join(srcDir, "file2.txt")
	if err := os.WriteFile(file1, []byte("content1"), 0644); err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}
	if err := os.WriteFile(file2, []byte("content2"), 0644); err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}

	// Create destination directory
	dstDir := filepath.Join(tmpDir, "dest")

	// Prepare files to copy
	files := []SelectableFile{
		{
			IgnoredFile: IgnoredFile{Path: "file1.txt", IsDir: false, Size: 8},
			Selected:    true,
		},
		{
			IgnoredFile: IgnoredFile{Path: "file2.txt", IsDir: false, Size: 8},
			Selected:    true,
		},
	}

	// Track progress callbacks
	var progressCalls int
	progressCallback := func(progress CopyProgress) {
		progressCalls++
		if progress.FilesTotal != 2 {
			t.Errorf("Expected FilesTotal=2, got %d", progress.FilesTotal)
		}
	}

	// Copy with progress tracking
	result, err := Copy(CopyOptions{
		SourceDir:    srcDir,
		TargetDir:    dstDir,
		Files:        files,
		OnProgress:   progressCallback,
		PreserveMode: false,
	})

	if err != nil {
		t.Fatalf("Copy failed: %v", err)
	}

	// Verify result
	if result.FilesCopied != 2 {
		t.Errorf("Expected 2 files copied, got %d", result.FilesCopied)
	}

	// Verify progress was called (at least once per file + final call)
	if progressCalls < 2 {
		t.Errorf("Expected at least 2 progress calls, got %d", progressCalls)
	}
}

func TestCopy_SkipUnselected(t *testing.T) {
	// Create temp directory for testing
	tmpDir := t.TempDir()

	// Create source files
	srcDir := filepath.Join(tmpDir, "source")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	file1 := filepath.Join(srcDir, "file1.txt")
	file2 := filepath.Join(srcDir, "file2.txt")
	if err := os.WriteFile(file1, []byte("content1"), 0644); err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}
	if err := os.WriteFile(file2, []byte("content2"), 0644); err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}

	// Create destination directory
	dstDir := filepath.Join(tmpDir, "dest")

	// Prepare files (only file1 selected)
	files := []SelectableFile{
		{
			IgnoredFile: IgnoredFile{Path: "file1.txt", IsDir: false, Size: 8},
			Selected:    true,
		},
		{
			IgnoredFile: IgnoredFile{Path: "file2.txt", IsDir: false, Size: 8},
			Selected:    false, // Not selected
		},
	}

	// Copy
	result, err := Copy(CopyOptions{
		SourceDir:    srcDir,
		TargetDir:    dstDir,
		Files:        files,
		PreserveMode: false,
	})

	if err != nil {
		t.Fatalf("Copy failed: %v", err)
	}

	// Verify only file1 was copied
	if result.FilesCopied != 1 {
		t.Errorf("Expected 1 file copied, got %d", result.FilesCopied)
	}

	// Verify file1 exists
	if _, err := os.Stat(filepath.Join(dstDir, "file1.txt")); os.IsNotExist(err) {
		t.Error("file1.txt should exist")
	}

	// Verify file2 does not exist
	if _, err := os.Stat(filepath.Join(dstDir, "file2.txt")); !os.IsNotExist(err) {
		t.Error("file2.txt should not exist")
	}
}
