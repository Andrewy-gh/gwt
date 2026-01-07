package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Andrewy-gh/gwt/internal/git"
	"github.com/Andrewy-gh/gwt/internal/output"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check system prerequisites and environment",
	Long: `The doctor command validates that all required tools and prerequisites
are properly installed and configured for gwt to work correctly.

It checks:
  • Git installation and version
  • Git repository status
  • Symlink support (Windows)
  • Docker and Docker Compose (optional)`,
	RunE: runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

func runDoctor(cmd *cobra.Command, args []string) error {
	output.Info("Checking prerequisites...\n")

	allPassed := true

	// Check Git installation
	if !checkGitInstalled() {
		allPassed = false
	}

	// Check Git version
	if !checkGitVersion() {
		allPassed = false
	}

	// Check if in a Git repository
	if !checkGitRepository() {
		allPassed = false
	}

	// Check if not a bare repository (only if in a repo)
	if git.IsRepository() {
		if !checkNotBareRepository() {
			allPassed = false
		}
	}

	// Check symlink support (Windows only)
	if runtime.GOOS == "windows" {
		if !checkSymlinkSupport() {
			// This is a warning, not a failure
		}
	}

	// Check Docker (optional)
	checkDocker()

	// Check Docker Compose (optional)
	checkDockerCompose()

	output.Println("")

	if allPassed {
		output.Success("All checks passed! gwt is ready to use.")
		return nil
	}

	output.Error("Some checks failed. Please address the issues above.")
	os.Exit(ExitGeneralError)
	return nil
}

func checkGitInstalled() bool {
	if !git.IsInstalled() {
		output.Error("Git is not installed or not in PATH")
		output.Println("")
		output.Println("gwt requires Git to be installed. Please install Git:")
		output.Println("  • Windows: https://git-scm.com/download/win")
		output.Println("  • macOS:   brew install git")
		output.Println("  • Linux:   apt install git / dnf install git")
		output.Println("")
		output.Println("After installing, restart your terminal and try again.")
		return false
	}
	return true
}

func checkGitVersion() bool {
	version, err := git.GetVersion()
	if err != nil {
		output.Warning(fmt.Sprintf("Could not determine Git version: %v", err))
		return true // Don't fail, just warn
	}

	output.Success(fmt.Sprintf("Git installed (%s)", version.String()))

	// Check if version is at least 2.20 (required for worktree features)
	if !version.AtLeast(2, 20) {
		output.Warning(fmt.Sprintf("Git version %s may have issues. Version 2.20+ recommended.", version.String()))
		output.Println("  Some worktree features may not work correctly with older Git versions.")
		return true // Warn but don't fail
	}

	return true
}

func checkGitRepository() bool {
	if !git.IsRepository() {
		output.Error("Not a git repository")
		output.Println("")
		output.Println("gwt must be run from within a Git repository. Navigate to your project")
		output.Println("directory and initialize a repository with:")
		output.Println("")
		output.Println("  git init")
		output.Println("")
		output.Println("Or clone an existing repository:")
		output.Println("")
		output.Println("  git clone <repository-url>")
		return false
	}

	output.Success("Git repository detected")
	return true
}

func checkNotBareRepository() bool {
	isBare, err := git.IsBareRepository()
	if err != nil {
		output.Warning(fmt.Sprintf("Could not check if bare repository: %v", err))
		return true // Don't fail
	}

	if isBare {
		output.Error("Bare repositories not supported")
		output.Println("")
		output.Println("gwt cannot be used with bare repositories. Clone the repository")
		output.Println("to a working directory instead:")
		output.Println("")
		output.Println("  git clone <repository-url> <directory>")
		return false
	}

	output.Success("Not a bare repository")
	return true
}

func checkSymlinkSupport() bool {
	// Try to create a test symlink
	tempDir := os.TempDir()
	testFile := filepath.Join(tempDir, "gwt-test-file")
	testLink := filepath.Join(tempDir, "gwt-test-link")

	// Clean up any existing test files
	os.Remove(testFile)
	os.Remove(testLink)

	// Create a test file
	f, err := os.Create(testFile)
	if err != nil {
		output.Warning("Could not test symlink support")
		return true // Don't fail
	}
	f.Close()
	defer os.Remove(testFile)

	// Try to create a symlink
	err = os.Symlink(testFile, testLink)
	if err != nil {
		output.Warning("Symlinks may require elevation")
		output.Println("  On Windows, enable Developer Mode or run as Administrator")
		output.Println("  for symlink support. This is required for database linking.")
		defer os.Remove(testLink)
		return true // Warn but don't fail
	}

	// Clean up
	os.Remove(testLink)

	output.Success("Symlink permissions available")
	return true
}

func checkDocker() bool {
	cmd := exec.Command("docker", "--version")
	cmdOutput, err := cmd.Output()
	if err != nil {
		output.Warning("Docker not found (optional)")
		return false
	}

	// Parse version from output like "Docker version 24.0.7, build afdd53b"
	versionStr := strings.TrimSpace(string(cmdOutput))
	parts := strings.Fields(versionStr)
	if len(parts) >= 3 {
		version := strings.TrimSuffix(parts[2], ",")
		output.Success(fmt.Sprintf("Docker installed (%s)", version))
	} else {
		output.Success("Docker installed")
	}

	return true
}

func checkDockerCompose() bool {
	// Try "docker compose" first (newer)
	cmd := exec.Command("docker", "compose", "version")
	cmdOutput, err := cmd.Output()

	if err != nil {
		// Try "docker-compose" (older)
		cmd = exec.Command("docker-compose", "--version")
		cmdOutput, err = cmd.Output()
	}

	if err != nil {
		output.Warning("Docker Compose not found (optional)")
		return false
	}

	// Parse version from output
	versionStr := strings.TrimSpace(string(cmdOutput))
	output.Success(fmt.Sprintf("Docker Compose available"))
	output.Verbose(versionStr)

	return true
}
