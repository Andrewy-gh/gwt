//go:build windows

package docker

import (
	"os/exec"
)

// createJunction creates a Windows junction (directory only)
// Only available on Windows
func createJunction(source, target string) error {
	// Use mklink /J via cmd.exe
	cmd := exec.Command("cmd", "/c", "mklink", "/J", target, source)
	return cmd.Run()
}
