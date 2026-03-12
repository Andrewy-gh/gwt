package cli

import (
	"reflect"
	"testing"
)

func TestNormalizeCLIArgs_RoutesBareBranchToCreate(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "bare branch",
			input:    []string{"feature-auth"},
			expected: []string{"create", "feature-auth"},
		},
		{
			name:     "branch with persistent flag first",
			input:    []string{"--verbose", "feature-auth"},
			expected: []string{"--verbose", "create", "feature-auth"},
		},
		{
			name:     "config flag with value",
			input:    []string{"--config", "custom.yaml", "feature-auth"},
			expected: []string{"--config", "custom.yaml", "create", "feature-auth"},
		},
		{
			name:     "known subcommand unchanged",
			input:    []string{"list"},
			expected: []string{"list"},
		},
		{
			name:     "create command unchanged",
			input:    []string{"create", "feature-auth"},
			expected: []string{"create", "feature-auth"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeCLIArgs(tt.input)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Fatalf("normalizeCLIArgs(%v) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}
