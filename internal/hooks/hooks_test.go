package hooks

import (
	"runtime"
	"testing"

	"github.com/Andrewy-gh/gwt/internal/config"
)

func TestNewExecutor(t *testing.T) {
	cfg := &config.Config{}
	executor := NewExecutor("/path/to/repo", cfg)

	if executor == nil {
		t.Fatal("Expected executor to be created")
	}

	if executor.repoPath != "/path/to/repo" {
		t.Errorf("Expected repoPath to be '/path/to/repo', got %s", executor.repoPath)
	}

	if executor.config != cfg {
		t.Error("Expected config to be set")
	}
}

func TestExecutorExecuteEmptyHooks(t *testing.T) {
	cfg := &config.Config{
		Hooks: config.HooksConfig{
			PostCreate: []string{},
			PostDelete: []string{},
		},
	}

	executor := NewExecutor("/path/to/repo", cfg)
	result, err := executor.Execute(ExecuteOptions{
		HookType:     HookTypePostCreate,
		WorktreePath: "/path/to/worktree",
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.Executed != 0 {
		t.Errorf("Expected 0 hooks executed, got %d", result.Executed)
	}

	if result.Successful != 0 {
		t.Errorf("Expected 0 successful hooks, got %d", result.Successful)
	}

	if result.Failed != 0 {
		t.Errorf("Expected 0 failed hooks, got %d", result.Failed)
	}
}

func TestExecutorExecuteSuccessfulHook(t *testing.T) {
	var cmd string
	if runtime.GOOS == "windows" {
		cmd = "echo test"
	} else {
		cmd = "echo 'test'"
	}

	cfg := &config.Config{
		Hooks: config.HooksConfig{
			PostCreate: []string{cmd},
		},
	}

	tmpDir := t.TempDir()

	executor := NewExecutor(tmpDir, cfg)
	result, err := executor.Execute(ExecuteOptions{
		HookType:     HookTypePostCreate,
		WorktreePath: tmpDir,
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.Executed != 1 {
		t.Errorf("Expected 1 hook executed, got %d", result.Executed)
	}

	if result.Successful != 1 {
		t.Errorf("Expected 1 successful hook, got %d", result.Successful)
	}

	if result.Failed != 0 {
		t.Errorf("Expected 0 failed hooks, got %d", result.Failed)
	}
}

func TestExecutorExecuteFailedHook(t *testing.T) {
	var cmd string
	if runtime.GOOS == "windows" {
		cmd = "exit 1"
	} else {
		cmd = "exit 1"
	}

	cfg := &config.Config{
		Hooks: config.HooksConfig{
			PostCreate: []string{cmd},
		},
	}

	tmpDir := t.TempDir()

	executor := NewExecutor(tmpDir, cfg)
	result, err := executor.Execute(ExecuteOptions{
		HookType:     HookTypePostCreate,
		WorktreePath: tmpDir,
	})

	if err != nil {
		t.Fatalf("Expected no error (failures are in result), got %v", err)
	}

	if result.Executed != 1 {
		t.Errorf("Expected 1 hook executed, got %d", result.Executed)
	}

	if result.Successful != 0 {
		t.Errorf("Expected 0 successful hooks, got %d", result.Successful)
	}

	if result.Failed != 1 {
		t.Errorf("Expected 1 failed hook, got %d", result.Failed)
	}

	if len(result.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(result.Errors))
	}
}

func TestExecutorExecuteMultipleHooks(t *testing.T) {
	var cmd1, cmd2 string
	if runtime.GOOS == "windows" {
		cmd1 = "echo test1"
		cmd2 = "echo test2"
	} else {
		cmd1 = "echo 'test1'"
		cmd2 = "echo 'test2'"
	}

	cfg := &config.Config{
		Hooks: config.HooksConfig{
			PostCreate: []string{cmd1, cmd2},
		},
	}

	tmpDir := t.TempDir()

	executor := NewExecutor(tmpDir, cfg)
	result, err := executor.Execute(ExecuteOptions{
		HookType:     HookTypePostCreate,
		WorktreePath: tmpDir,
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.Executed != 2 {
		t.Errorf("Expected 2 hooks executed, got %d", result.Executed)
	}

	if result.Successful != 2 {
		t.Errorf("Expected 2 successful hooks, got %d", result.Successful)
	}
}

func TestExecutorExecuteMixedResults(t *testing.T) {
	var cmd1, cmd2, cmd3 string
	if runtime.GOOS == "windows" {
		cmd1 = "echo success1"
		cmd2 = "exit 1"
		cmd3 = "echo success2"
	} else {
		cmd1 = "echo 'success1'"
		cmd2 = "exit 1"
		cmd3 = "echo 'success2'"
	}

	cfg := &config.Config{
		Hooks: config.HooksConfig{
			PostCreate: []string{cmd1, cmd2, cmd3},
		},
	}

	tmpDir := t.TempDir()

	executor := NewExecutor(tmpDir, cfg)
	result, err := executor.Execute(ExecuteOptions{
		HookType:     HookTypePostCreate,
		WorktreePath: tmpDir,
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.Executed != 3 {
		t.Errorf("Expected 3 hooks executed, got %d", result.Executed)
	}

	if result.Successful != 2 {
		t.Errorf("Expected 2 successful hooks, got %d", result.Successful)
	}

	if result.Failed != 1 {
		t.Errorf("Expected 1 failed hook, got %d", result.Failed)
	}

	if len(result.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(result.Errors))
	}
}

func TestExecutorHasHooks(t *testing.T) {
	cfg := &config.Config{
		Hooks: config.HooksConfig{
			PostCreate: []string{"echo test"},
			PostDelete: []string{},
		},
	}

	executor := NewExecutor("/path/to/repo", cfg)

	if !executor.HasHooks(HookTypePostCreate) {
		t.Error("Expected post_create hooks to exist")
	}

	if executor.HasHooks(HookTypePostDelete) {
		t.Error("Expected post_delete hooks to not exist")
	}

	if executor.HasHooks("invalid_type") {
		t.Error("Expected invalid hook type to have no hooks")
	}
}

func TestExecutorGetCommands(t *testing.T) {
	cfg := &config.Config{
		Hooks: config.HooksConfig{
			PostCreate: []string{"cmd1", "cmd2"},
			PostDelete: []string{"cmd3"},
		},
	}

	executor := NewExecutor("/path/to/repo", cfg)

	postCreate := executor.getCommands(HookTypePostCreate)
	if len(postCreate) != 2 {
		t.Errorf("Expected 2 post_create commands, got %d", len(postCreate))
	}

	postDelete := executor.getCommands(HookTypePostDelete)
	if len(postDelete) != 1 {
		t.Errorf("Expected 1 post_delete command, got %d", len(postDelete))
	}

	invalid := executor.getCommands("invalid")
	if len(invalid) != 0 {
		t.Errorf("Expected 0 commands for invalid type, got %d", len(invalid))
	}
}

func TestExecutorNilConfig(t *testing.T) {
	executor := NewExecutor("/path/to/repo", nil)

	if executor.HasHooks(HookTypePostCreate) {
		t.Error("Expected no hooks with nil config")
	}

	cmds := executor.getCommands(HookTypePostCreate)
	if cmds != nil {
		t.Error("Expected nil commands with nil config")
	}
}
