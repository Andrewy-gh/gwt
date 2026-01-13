package docker

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGenerateHelperScript(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name         string
		shellType    string
		expectedFile string
		expectedContent []string
	}{
		{
			name:         "Bash script",
			shellType:    "bash",
			expectedFile: "dc",
			expectedContent: []string{
				"#!/bin/bash",
				"docker compose",
				"COMPOSE_FILES",
			},
		},
		{
			name:         "PowerShell script",
			shellType:    "powershell",
			expectedFile: "dc.ps1",
			expectedContent: []string{
				"$ComposeFiles",
				"docker compose",
				"$args",
			},
		},
		{
			name:         "CMD script",
			shellType:    "cmd",
			expectedFile: "dc.cmd",
			expectedContent: []string{
				"@echo off",
				"docker compose",
				"%*",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test directory for this shell type
			testDir := filepath.Join(tmpDir, tt.name)
			if err := os.MkdirAll(testDir, 0755); err != nil {
				t.Fatal(err)
			}

			opts := HelperScriptOptions{
				WorktreePath: testDir,
				ComposeFiles: []string{
					filepath.Join(testDir, "docker-compose.yml"),
				},
				OverrideFile: "docker-compose.worktree.yml",
				ShellType:    tt.shellType,
			}

			err := GenerateHelperScript(opts)
			if err != nil {
				t.Fatalf("GenerateHelperScript failed: %v", err)
			}

			// Check file was created
			scriptPath := filepath.Join(testDir, tt.expectedFile)
			if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
				t.Errorf("Expected file %s was not created", tt.expectedFile)
				return
			}

			// Read and check content
			content, err := os.ReadFile(scriptPath)
			if err != nil {
				t.Fatalf("Failed to read script: %v", err)
			}

			contentStr := string(content)
			for _, expected := range tt.expectedContent {
				if !strings.Contains(contentStr, expected) {
					t.Errorf("Expected content to contain %q, but it doesn't", expected)
				}
			}

			// Check if override file is referenced
			if !strings.Contains(contentStr, "docker-compose.worktree.yml") {
				t.Error("Expected override file to be referenced")
			}
		})
	}
}

func TestGenerateHelperScriptWithoutOverride(t *testing.T) {
	tmpDir := t.TempDir()

	opts := HelperScriptOptions{
		WorktreePath: tmpDir,
		ComposeFiles: []string{
			filepath.Join(tmpDir, "docker-compose.yml"),
		},
		OverrideFile: "", // No override
		ShellType:    "bash",
	}

	err := GenerateHelperScript(opts)
	if err != nil {
		t.Fatalf("GenerateHelperScript failed: %v", err)
	}

	// Check file was created
	scriptPath := filepath.Join(tmpDir, "dc")
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("Failed to read script: %v", err)
	}

	// Should still work without override file
	contentStr := string(content)
	if !strings.Contains(contentStr, "docker compose") {
		t.Error("Expected docker compose command")
	}
}

func TestGetDefaultShell(t *testing.T) {
	shell := getDefaultShell()

	if runtime.GOOS == "windows" {
		// On Windows, should be cmd or bash (if in MSYS)
		if shell != "cmd" && shell != "bash" {
			t.Errorf("Expected cmd or bash on Windows, got %s", shell)
		}
	} else {
		// On Unix-like systems, should be bash
		if shell != "bash" {
			t.Errorf("Expected bash on Unix-like systems, got %s", shell)
		}
	}
}

func TestGenerateBashScript(t *testing.T) {
	opts := HelperScriptOptions{
		WorktreePath: "/test",
		ComposeFiles: []string{
			"/test/docker-compose.yml",
			"/test/docker-compose.dev.yml",
		},
		OverrideFile: "docker-compose.worktree.yml",
	}

	script := generateBashScript(opts)

	// Check for required elements
	if !strings.Contains(script, "#!/bin/bash") {
		t.Error("Expected bash shebang")
	}
	if !strings.Contains(script, "docker compose") {
		t.Error("Expected docker compose command")
	}
	if !strings.Contains(script, "docker-compose.yml") {
		t.Error("Expected base compose file")
	}
	if !strings.Contains(script, "docker-compose.worktree.yml") {
		t.Error("Expected override file")
	}
}

func TestGeneratePowerShellScript(t *testing.T) {
	opts := HelperScriptOptions{
		WorktreePath: "C:\\test",
		ComposeFiles: []string{
			"C:\\test\\docker-compose.yml",
		},
		OverrideFile: "docker-compose.worktree.yml",
	}

	script := generatePowerShellScript(opts)

	// Check for required elements
	if !strings.Contains(script, "$ComposeFiles") {
		t.Error("Expected PowerShell array variable")
	}
	if !strings.Contains(script, "docker compose") {
		t.Error("Expected docker compose command")
	}
	if !strings.Contains(script, "$args") {
		t.Error("Expected $args for argument passing")
	}
}

func TestGenerateCmdScript(t *testing.T) {
	opts := HelperScriptOptions{
		WorktreePath: "C:\\test",
		ComposeFiles: []string{
			"C:\\test\\docker-compose.yml",
		},
		OverrideFile: "docker-compose.worktree.yml",
	}

	script := generateCmdScript(opts)

	// Check for required elements
	if !strings.Contains(script, "@echo off") {
		t.Error("Expected CMD echo off")
	}
	if !strings.Contains(script, "docker compose") {
		t.Error("Expected docker compose command")
	}
	if !strings.Contains(script, "%*") {
		t.Error("Expected %* for argument passing")
	}
}
