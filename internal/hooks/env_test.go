package hooks

import (
	"os"
	"strings"
	"testing"
)

func TestBuildEnvironment(t *testing.T) {
	opts := HookEnvironment{
		WorktreePath:     "/path/to/worktree",
		WorktreeBranch:   "feature/test",
		MainWorktreePath: "/path/to/main",
		RepoPath:         "/path/to/repo",
		HookType:         "post_create",
	}

	env := BuildEnvironment(opts)

	// Check that we got environment variables back
	if len(env) == 0 {
		t.Fatal("Expected environment variables, got empty slice")
	}

	// Convert to map for easier checking
	envMap := make(map[string]string)
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	// Check GWT_* variables
	tests := []struct {
		key      string
		expected string
	}{
		{"GWT_WORKTREE_PATH", "/path/to/worktree"},
		{"GWT_BRANCH", "feature/test"},
		{"GWT_MAIN_WORKTREE", "/path/to/main"},
		{"GWT_REPO_PATH", "/path/to/repo"},
		{"GWT_HOOK_TYPE", "post_create"},
	}

	for _, tt := range tests {
		if got := envMap[tt.key]; got != tt.expected {
			t.Errorf("Expected %s=%s, got %s", tt.key, tt.expected, got)
		}
	}
}

func TestBuildEnvironmentWithEmptyValues(t *testing.T) {
	opts := HookEnvironment{
		WorktreePath:   "/path/to/worktree",
		WorktreeBranch: "main",
		// Others empty
	}

	env := BuildEnvironment(opts)

	// Convert to map
	envMap := make(map[string]string)
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	// Check that non-empty values are present
	if envMap["GWT_WORKTREE_PATH"] != "/path/to/worktree" {
		t.Errorf("Expected GWT_WORKTREE_PATH to be set")
	}
	if envMap["GWT_BRANCH"] != "main" {
		t.Errorf("Expected GWT_BRANCH to be set")
	}
}

func TestBuildEnvironmentMergesWithExisting(t *testing.T) {
	// Set a test env variable
	testKey := "TEST_EXISTING_VAR"
	testValue := "test_value"
	os.Setenv(testKey, testValue)
	defer os.Unsetenv(testKey)

	opts := HookEnvironment{
		WorktreePath: "/path/to/worktree",
	}

	env := BuildEnvironment(opts)

	// Check that existing env var is still present
	found := false
	for _, e := range env {
		if strings.HasPrefix(e, testKey+"=") {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected existing environment variable %s to be preserved", testKey)
	}
}

func TestBuildEnvironmentMap(t *testing.T) {
	opts := HookEnvironment{
		WorktreePath:     "/path/to/worktree",
		WorktreeBranch:   "feature/test",
		MainWorktreePath: "/path/to/main",
		RepoPath:         "/path/to/repo",
		HookType:         "post_delete",
	}

	envMap := BuildEnvironmentMap(opts)

	tests := []struct {
		key      string
		expected string
	}{
		{"GWT_WORKTREE_PATH", "/path/to/worktree"},
		{"GWT_BRANCH", "feature/test"},
		{"GWT_MAIN_WORKTREE", "/path/to/main"},
		{"GWT_REPO_PATH", "/path/to/repo"},
		{"GWT_HOOK_TYPE", "post_delete"},
	}

	for _, tt := range tests {
		if got := envMap[tt.key]; got != tt.expected {
			t.Errorf("Expected %s=%s, got %s", tt.key, tt.expected, got)
		}
	}
}
