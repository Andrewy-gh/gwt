package version

import (
	"strings"
	"testing"
)

func TestString(t *testing.T) {
	// Save original values
	oldVersion := Version
	oldCommit := Commit
	defer func() {
		Version = oldVersion
		Commit = oldCommit
	}()

	// Test with dev commit
	Version = "1.0.0"
	Commit = "dev"

	result := String()
	if result != "1.0.0" {
		t.Errorf("Expected '1.0.0', got: %s", result)
	}

	// Test with actual commit
	Commit = "abc123"

	result = String()
	if !strings.Contains(result, "1.0.0") {
		t.Errorf("Expected result to contain '1.0.0', got: %s", result)
	}
	if !strings.Contains(result, "abc123") {
		t.Errorf("Expected result to contain commit 'abc123', got: %s", result)
	}
}
