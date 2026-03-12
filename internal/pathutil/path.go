package pathutil

import (
	"path"
	"path/filepath"
	"strings"
)

// IsAbsAny reports whether a path is absolute for either the current OS
// or common Windows path formats used in cross-platform configs/tests.
func IsAbsAny(value string) bool {
	return filepath.IsAbs(value) || isWindowsAbs(value)
}

// Base returns the last path element for native paths and Windows-style paths.
func Base(value string) string {
	if strings.Contains(value, "\\") {
		return path.Base(strings.ReplaceAll(value, "\\", "/"))
	}

	return filepath.Base(value)
}

func isWindowsAbs(value string) bool {
	if len(value) >= 3 && isLetter(value[0]) && value[1] == ':' && (value[2] == '\\' || value[2] == '/') {
		return true
	}

	return strings.HasPrefix(value, `\\`) || strings.HasPrefix(value, `//`)
}

func isLetter(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')

}
