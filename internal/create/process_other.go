//go:build !windows

package create

import "os"

func isProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	return process.Signal(os.Signal(nil)) == nil
}
