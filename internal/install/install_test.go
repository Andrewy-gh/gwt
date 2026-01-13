package install

import (
	"testing"

	"github.com/Andrewy-gh/gwt/internal/config"
)

func TestInstall_SkipsWhenDisabled(t *testing.T) {
	cfg := &config.DependenciesConfig{
		AutoInstall: false,
		Paths:       []string{"."},
	}

	result, err := Install(t.TempDir(), cfg, InstallOptions{})

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !result.Skipped {
		t.Error("Expected installation to be skipped")
	}
	if result.Reason != "auto_install disabled" {
		t.Errorf("Expected reason 'auto_install disabled', got '%s'", result.Reason)
	}
}

func TestInstall_SkipsWhenNoManagers(t *testing.T) {
	// Create empty temp directory
	dir := t.TempDir()

	cfg := &config.DependenciesConfig{
		AutoInstall: true,
		Paths:       []string{"."},
	}

	result, err := Install(dir, cfg, InstallOptions{})

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !result.Skipped {
		t.Error("Expected installation to be skipped")
	}
	if result.Reason != "no package managers detected" {
		t.Errorf("Expected reason 'no package managers detected', got '%s'", result.Reason)
	}
}

func TestResult_HasErrors(t *testing.T) {
	tests := []struct {
		name     string
		result   *Result
		expected bool
	}{
		{
			name: "no errors",
			result: &Result{
				Managers: []ManagerResult{
					{Manager: "npm", Success: true},
					{Manager: "go", Success: true},
				},
			},
			expected: false,
		},
		{
			name: "has errors",
			result: &Result{
				Managers: []ManagerResult{
					{Manager: "npm", Success: true},
					{Manager: "go", Success: false},
				},
			},
			expected: true,
		},
		{
			name: "all errors",
			result: &Result{
				Managers: []ManagerResult{
					{Manager: "npm", Success: false},
					{Manager: "go", Success: false},
				},
			},
			expected: true,
		},
		{
			name:     "empty result",
			result:   &Result{Managers: []ManagerResult{}},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.result.HasErrors() != tt.expected {
				t.Errorf("Expected HasErrors() = %v, got %v", tt.expected, tt.result.HasErrors())
			}
		})
	}
}

func TestResult_SuccessCount(t *testing.T) {
	result := &Result{
		Managers: []ManagerResult{
			{Manager: "npm", Success: true},
			{Manager: "go", Success: false},
			{Manager: "cargo", Success: true},
		},
	}

	expected := 2
	if result.SuccessCount() != expected {
		t.Errorf("Expected SuccessCount() = %d, got %d", expected, result.SuccessCount())
	}
}

func TestResult_ErrorCount(t *testing.T) {
	result := &Result{
		Managers: []ManagerResult{
			{Manager: "npm", Success: true},
			{Manager: "go", Success: false},
			{Manager: "cargo", Success: false},
		},
	}

	expected := 2
	if result.ErrorCount() != expected {
		t.Errorf("Expected ErrorCount() = %d, got %d", expected, result.ErrorCount())
	}
}
