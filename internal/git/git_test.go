package git

import (
	"testing"
)

func TestVersionAtLeast(t *testing.T) {
	tests := []struct {
		version  Version
		major    int
		minor    int
		expected bool
	}{
		{Version{2, 43, 0}, 2, 20, true},
		{Version{2, 43, 0}, 2, 43, true},
		{Version{2, 43, 0}, 2, 44, false},
		{Version{2, 43, 0}, 3, 0, false},
		{Version{3, 0, 0}, 2, 43, true},
		{Version{2, 19, 0}, 2, 20, false},
	}

	for _, tt := range tests {
		result := tt.version.AtLeast(tt.major, tt.minor)
		if result != tt.expected {
			t.Errorf("Version %s.AtLeast(%d, %d) = %v, expected %v",
				tt.version.String(), tt.major, tt.minor, result, tt.expected)
		}
	}
}

func TestVersionString(t *testing.T) {
	v := Version{Major: 2, Minor: 43, Patch: 1}
	expected := "2.43.1"

	result := v.String()
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestIsInstalled(t *testing.T) {
	// This test assumes Git is installed (which it should be for development)
	result := IsInstalled()
	if !result {
		t.Skip("Git is not installed, skipping test")
	}
}

func TestGetVersion(t *testing.T) {
	// This test assumes Git is installed
	if !IsInstalled() {
		t.Skip("Git is not installed, skipping test")
	}

	version, err := GetVersion()
	if err != nil {
		t.Errorf("GetVersion() failed: %v", err)
	}

	// Version should be at least 2.0.0
	if version.Major < 2 {
		t.Errorf("Expected Git major version >= 2, got: %s", version.String())
	}
}
