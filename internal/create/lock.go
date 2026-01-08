package create

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Andrewy-gh/gwt/internal/git"
)

// OperationLock represents a lock for gwt operations
type OperationLock struct {
	lockFile string
	file     *os.File
}

// LockInfo contains information about a lock holder
type LockInfo struct {
	PID       int       `json:"pid"`
	Command   string    `json:"command"`
	StartTime time.Time `json:"started"`
}

// AcquireLock attempts to acquire the operation lock
// Returns error if lock is held by another process
func AcquireLock(repoPath string) (*OperationLock, error) {
	// Get git directory
	gitDir, err := git.GetGitDir(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get git directory: %w", err)
	}

	lockPath := filepath.Join(gitDir, "gwt.lock")

	// Try to create lock file with exclusive access
	// O_CREATE|O_EXCL ensures atomicity - fails if file exists
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		if os.IsExist(err) {
			// Lock exists, check if process is still running
			info, infoErr := GetLockInfo(repoPath)
			if infoErr != nil {
				return nil, fmt.Errorf("lock exists but cannot read info: %w", infoErr)
			}

			// Check if process is still running
			if isProcessRunning(info.PID) {
				return nil, fmt.Errorf("another gwt operation is in progress (PID %d, started %s)",
					info.PID, info.StartTime.Format(time.RFC3339))
			}

			// Stale lock, remove it
			os.Remove(lockPath)
			// Retry
			return AcquireLock(repoPath)
		}
		return nil, fmt.Errorf("failed to create lock file: %w", err)
	}

	// Write lock info
	info := LockInfo{
		PID:       os.Getpid(),
		Command:   strings.Join(os.Args, " "),
		StartTime: time.Now(),
	}

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(info); err != nil {
		file.Close()
		os.Remove(lockPath)
		return nil, fmt.Errorf("failed to write lock info: %w", err)
	}

	return &OperationLock{
		lockFile: lockPath,
		file:     file,
	}, nil
}

// Release releases the operation lock
func (l *OperationLock) Release() error {
	if l.file != nil {
		l.file.Close()
		l.file = nil
	}

	if l.lockFile != "" {
		err := os.Remove(l.lockFile)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove lock file: %w", err)
		}
		l.lockFile = ""
	}

	return nil
}

// IsLocked checks if operations are locked
func IsLocked(repoPath string) (bool, error) {
	gitDir, err := git.GetGitDir(repoPath)
	if err != nil {
		return false, fmt.Errorf("failed to get git directory: %w", err)
	}

	lockPath := filepath.Join(gitDir, "gwt.lock")

	// Check if lock file exists
	_, err = os.Stat(lockPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check lock file: %w", err)
	}

	// Lock file exists, check if it's stale
	info, err := GetLockInfo(repoPath)
	if err != nil {
		// Can't read lock info, assume locked
		return true, nil
	}

	// Check if process is still running
	return isProcessRunning(info.PID), nil
}

// GetLockInfo returns information about the current lock holder
func GetLockInfo(repoPath string) (*LockInfo, error) {
	gitDir, err := git.GetGitDir(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get git directory: %w", err)
	}

	lockPath := filepath.Join(gitDir, "gwt.lock")

	// Read lock file
	data, err := os.ReadFile(lockPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("lock file does not exist")
		}
		return nil, fmt.Errorf("failed to read lock file: %w", err)
	}

	// Parse JSON
	var info LockInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("failed to parse lock info: %w", err)
	}

	return &info, nil
}

// isProcessRunning checks if a process with the given PID is running
func isProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}

	// Platform-specific process check
	if runtime.GOOS == "windows" {
		// On Windows, try to open the process handle
		// This is a simplified check - a full implementation would use syscall
		// For now, we'll use a heuristic: check if we can send signal 0
		process, err := os.FindProcess(pid)
		if err != nil {
			return false
		}

		// On Windows, FindProcess always succeeds, so we need another check
		// Try to signal the process (signal 0 doesn't actually send a signal)
		// This is not perfect on Windows, but it's a reasonable heuristic
		err = process.Signal(os.Signal(nil))
		return err == nil
	}

	// On Unix, try to send signal 0 (doesn't actually send a signal, just checks if process exists)
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Signal 0 is a special case that checks if the process exists
	// without actually sending a signal
	err = process.Signal(os.Signal(nil))
	return err == nil
}

// ForceUnlock forcibly removes the lock file
// This should only be used when you're certain the lock is stale
func ForceUnlock(repoPath string) error {
	gitDir, err := git.GetGitDir(repoPath)
	if err != nil {
		return fmt.Errorf("failed to get git directory: %w", err)
	}

	lockPath := filepath.Join(gitDir, "gwt.lock")

	err = os.Remove(lockPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove lock file: %w", err)
	}

	return nil
}
