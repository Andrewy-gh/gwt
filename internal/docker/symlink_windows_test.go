//go:build windows

package docker

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestCreateSymlinkWithPrivilege tests that CreateSymlink produces an accessible link target.
func TestCreateSymlinkWithPrivilege(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "target")
	link := filepath.Join(tmpDir, "link")

	// Create target directory
	if err := os.Mkdir(target, 0755); err != nil {
		t.Fatalf("failed to create target: %v", err)
	}

	// Try to create symlink
	err := CreateSymlink(target, link)

	// May succeed or fail depending on Developer Mode / Admin
	if err != nil {
		if os.IsPermission(err) || strings.Contains(err.Error(), "privilege") {
			t.Skip("Symlink privilege not available (expected)")
		}
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the link target is accessible, whether CreateSymlink used a symlink,
	// junction, or directory copy fallback.
	if _, err := os.Stat(link); err != nil {
		t.Fatalf("link not created: %v", err)
	}

	// Test reading through symlink
	testFile := filepath.Join(link, "test.txt")
	if err := os.WriteFile(testFile, []byte("hello"), 0644); err != nil {
		t.Fatalf("failed to write through symlink: %v", err)
	}

	// Verify in target
	targetFile := filepath.Join(target, "test.txt")
	if data, err := os.ReadFile(targetFile); err != nil {
		t.Fatalf("file not in target: %v", err)
	} else if string(data) != "hello" {
		t.Errorf("incorrect content: %s", string(data))
	}
}

// TestCreateJunctionDirect tests direct junction creation
func TestCreateJunctionDirect(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "target")
	junction := filepath.Join(tmpDir, "junction")

	// Create target
	if err := os.Mkdir(target, 0755); err != nil {
		t.Fatalf("failed to create target: %v", err)
	}

	// Create junction using mklink /J
	err := createJunction(target, junction)
	if err != nil {
		t.Fatalf("failed to create junction: %v", err)
	}

	// Verify junction exists
	if _, err := os.Stat(junction); err != nil {
		t.Fatalf("junction not accessible: %v", err)
	}

	// Test writing through junction
	testFile := filepath.Join(junction, "test.txt")
	if err := os.WriteFile(testFile, []byte("junction test"), 0644); err != nil {
		t.Fatalf("failed to write through junction: %v", err)
	}

	// Verify in target
	targetFile := filepath.Join(target, "test.txt")
	if data, err := os.ReadFile(targetFile); err != nil {
		t.Fatalf("file not in target: %v", err)
	} else if string(data) != "junction test" {
		t.Errorf("incorrect content")
	}
}

// TestJunctionCannotSpanDrives tests junction limitation
func TestJunctionCannotSpanDrives(t *testing.T) {
	t.Skip("Requires multiple drives - manual test only")

	// This test would need two different drive letters
	// Junction from C: to D: should fail
	// Symlink would be required instead
}

// TestSymlinkFallbackChain tests the complete fallback chain
func TestSymlinkFallbackChain(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "target")
	link := filepath.Join(tmpDir, "link")

	// Create target
	if err := os.Mkdir(target, 0755); err != nil {
		t.Fatalf("failed to create target: %v", err)
	}

	// Write test file to target
	targetFile := filepath.Join(target, "test.txt")
	if err := os.WriteFile(targetFile, []byte("original"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// CreateSymlink tries: symlink -> junction -> copy
	err := CreateSymlink(target, link)
	if err != nil {
		t.Fatalf("all fallback methods failed: %v", err)
	}

	// Verify link exists by any method
	if _, err := os.Stat(link); err != nil {
		t.Fatalf("link not created: %v", err)
	}

	// Verify test file is accessible
	linkFile := filepath.Join(link, "test.txt")
	if data, err := os.ReadFile(linkFile); err != nil {
		t.Fatalf("file not accessible through link: %v", err)
	} else if string(data) != "original" {
		t.Errorf("incorrect content: %s", string(data))
	}

	// Check which method was used
	if info, err := os.Lstat(link); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			t.Logf("Used symlink (best)")
		} else if isJunction(link) {
			t.Logf("Used junction (fallback)")
		} else if info.IsDir() {
			t.Logf("Used copy (last resort)")
		}
	}
}

// TestCreateSymlinkRelativePath tests symlink with relative paths
func TestCreateSymlinkRelativePath(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "target")
	link := filepath.Join(tmpDir, "link")

	if err := os.Mkdir(target, 0755); err != nil {
		t.Fatalf("failed to create target: %v", err)
	}

	// CreateSymlink should handle both absolute and relative
	err := CreateSymlink(target, link)
	if err != nil {
		if os.IsPermission(err) || strings.Contains(err.Error(), "privilege") {
			t.Skip("Symlink privilege not available")
		}
		t.Fatalf("failed to create link: %v", err)
	}

	if _, err := os.Stat(link); err != nil {
		t.Fatalf("link not accessible: %v", err)
	}
}

// TestCreateSymlinkAlreadyExists tests that CreateSymlink replaces an existing target.
func TestCreateSymlinkAlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "target")
	replacementTarget := filepath.Join(tmpDir, "replacement-target")
	link := filepath.Join(tmpDir, "link")

	if err := os.Mkdir(target, 0755); err != nil {
		t.Fatalf("failed to create target: %v", err)
	}
	if err := os.Mkdir(replacementTarget, 0755); err != nil {
		t.Fatalf("failed to create replacement target: %v", err)
	}

	originalFile := filepath.Join(target, "original.txt")
	if err := os.WriteFile(originalFile, []byte("original"), 0644); err != nil {
		t.Fatalf("failed to seed original target: %v", err)
	}

	// Create link first time
	if err := CreateSymlink(target, link); err != nil {
		t.Skipf("Cannot create symlink: %v", err)
	}

	replacementFile := filepath.Join(replacementTarget, "replacement.txt")
	if err := os.WriteFile(replacementFile, []byte("replacement"), 0644); err != nil {
		t.Fatalf("failed to seed replacement target: %v", err)
	}

	// Try to create again; the helper removes an existing target and recreates it.
	if err := CreateSymlink(replacementTarget, link); err != nil {
		t.Fatalf("expected existing target to be replaced, got: %v", err)
	}

	if _, err := os.Stat(filepath.Join(link, "replacement.txt")); err != nil {
		t.Fatalf("expected replacement target to be accessible through recreated link: %v", err)
	}
	if _, err := os.Stat(filepath.Join(link, "original.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected recreated link to stop pointing at the original target, got err=%v", err)
	}
}

// TestRemoveSymlink tests removing symlinks/junctions
func TestRemoveSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "target")
	link := filepath.Join(tmpDir, "link")

	if err := os.Mkdir(target, 0755); err != nil {
		t.Fatalf("failed to create target: %v", err)
	}

	// Create link
	if err := CreateSymlink(target, link); err != nil {
		t.Skipf("Cannot create symlink: %v", err)
	}

	// Remove link
	// On Windows, must use os.Remove, not os.RemoveAll for symlinks/junctions
	if err := os.Remove(link); err != nil {
		t.Fatalf("failed to remove link: %v", err)
	}

	// Verify removed
	if _, err := os.Stat(link); !os.IsNotExist(err) {
		t.Errorf("link still exists after removal")
	}

	// Verify target still exists
	if _, err := os.Stat(target); err != nil {
		t.Errorf("target was removed (should still exist)")
	}
}

// TestJunctionVsSymlink tests difference between junction and symlink
func TestJunctionVsSymlink(t *testing.T) {
	tmpDir := t.TempDir()

	// Test junction creation (always works, no privilege needed)
	junctionTarget := filepath.Join(tmpDir, "junction_target")
	junctionLink := filepath.Join(tmpDir, "junction")

	if err := os.Mkdir(junctionTarget, 0755); err != nil {
		t.Fatalf("failed to create junction target: %v", err)
	}

	if err := createJunction(junctionTarget, junctionLink); err != nil {
		t.Fatalf("junction creation failed: %v", err)
	}

	// Verify junction works
	if _, err := os.Stat(junctionLink); err != nil {
		t.Errorf("junction not accessible: %v", err)
	}

	// Test symlink creation (may fail without privilege)
	symlinkTarget := filepath.Join(tmpDir, "symlink_target")
	symlinkLink := filepath.Join(tmpDir, "symlink")

	if err := os.Mkdir(symlinkTarget, 0755); err != nil {
		t.Fatalf("failed to create symlink target: %v", err)
	}

	err := os.Symlink(symlinkTarget, symlinkLink)
	if err != nil {
		if os.IsPermission(err) || strings.Contains(err.Error(), "privilege") {
			t.Logf("Symlink requires privilege (expected): %v", err)
		} else {
			t.Errorf("unexpected symlink error: %v", err)
		}
	} else {
		// Symlink created successfully
		if _, err := os.Stat(symlinkLink); err != nil {
			t.Errorf("symlink not accessible: %v", err)
		}
		t.Logf("Symlink privilege available")
	}
}

// TestHasSymlinkPrivilege tests privilege detection
func TestHasSymlinkPrivilege(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "target")
	link := filepath.Join(tmpDir, "link")

	if err := os.Mkdir(target, 0755); err != nil {
		t.Fatalf("failed to create target: %v", err)
	}

	// Try to create a test symlink
	err := os.Symlink(target, link)

	if err == nil {
		t.Logf("Symlink privilege: AVAILABLE")
		os.Remove(link)
	} else if os.IsPermission(err) || strings.Contains(err.Error(), "privilege") {
		t.Logf("Symlink privilege: NOT AVAILABLE (expected without Developer Mode / Admin)")
	} else {
		t.Errorf("unexpected error testing symlink privilege: %v", err)
	}
}

// TestPathConversion tests Windows path conversion for Docker
func TestPathConversion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantUnix string
	}{
		{
			name:     "C drive backslash",
			input:    `C:\projects\app`,
			wantUnix: "/c/projects/app",
		},
		{
			name:     "C drive forward slash",
			input:    `C:/projects/app`,
			wantUnix: "/c/projects/app",
		},
		{
			name:     "D drive",
			input:    `D:\data`,
			wantUnix: "/d/data",
		},
		{
			name:     "with spaces",
			input:    `C:\Program Files\app`,
			wantUnix: "/c/Program Files/app",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertWindowsPathForDocker(tt.input)
			if got != tt.wantUnix {
				t.Errorf("ConvertWindowsPathForDocker(%q) = %q, want %q",
					tt.input, got, tt.wantUnix)
			}
		})
	}
}

// TestMklinkCommand tests mklink command construction
func TestMklinkCommand(t *testing.T) {
	target := `C:\projects\target`
	link := `C:\projects\link`

	// Test junction command
	cmd := exec.Command("cmd", "/C", "mklink", "/J", link, target)
	if cmd.Path == "" {
		t.Error("mklink command not constructed properly")
	}

	cmdStr := strings.Join(cmd.Args, " ")
	if !strings.Contains(cmdStr, "/J") {
		t.Error("junction flag not present")
	}
	if !strings.Contains(cmdStr, "mklink") {
		t.Error("mklink not present")
	}

	t.Logf("Junction command: %s", cmdStr)
}

// Helper function to check if a path is a junction
func isJunction(path string) bool {
	// Try to read the reparse point
	// Junctions have a specific reparse tag
	// This is a simplified check
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}

	// On Windows, junctions appear as directories
	// but os.Lstat should return info about the link itself
	// This is a basic heuristic
	return info.IsDir() && info.Mode()&os.ModeSymlink == 0
}

// ConvertWindowsPathForDocker converts Windows path to Unix-style for Docker
// This would be in the actual docker package
func ConvertWindowsPathForDocker(windowsPath string) string {
	// Convert backslashes to forward slashes
	unixPath := filepath.ToSlash(windowsPath)

	// Convert drive letter: C: -> /c
	if len(unixPath) >= 2 && unixPath[1] == ':' {
		drive := strings.ToLower(string(unixPath[0]))
		unixPath = "/" + drive + unixPath[2:]
	}

	return unixPath
}
