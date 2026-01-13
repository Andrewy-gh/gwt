package install

import "time"

// PackageManager represents a detected package manager
type PackageManager struct {
	Name        string   // e.g., "npm", "yarn", "go", "cargo"
	Path        string   // Directory containing the package manager files
	LockFile    string   // Lock file that triggered detection (if any)
	InstallCmd  string   // Command to run (e.g., "npm install")
	InstallArgs []string // Arguments for the install command
}

// Executor interface for running package manager commands
type Executor interface {
	Install(pm PackageManager, opts InstallOptions) error
}

// InstallOptions configures the installation process
type InstallOptions struct {
	Verbose    bool                 // Enable verbose output
	Timeout    time.Duration        // Timeout for installation (0 = default 5 minutes)
	OnProgress func(line string)    // Called for each output line
}
