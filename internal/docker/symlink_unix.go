//go:build !windows

package docker

import "errors"

// createJunction is not available on non-Windows platforms
func createJunction(source, target string) error {
	return errors.New("junctions are only available on Windows")
}
