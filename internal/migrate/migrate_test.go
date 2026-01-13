package migrate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Andrewy-gh/gwt/internal/config"
)

func TestRun_NoToolDetected(t *testing.T) {
	dir := t.TempDir()

	opts := RunOptions{
		WorktreePath: dir,
		Verbose:      false,
	}

	result, err := Run(opts, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Skipped {
		t.Error("expected migration to be skipped")
	}

	if result.Reason != "no migration tool detected" {
		t.Errorf("expected reason 'no migration tool detected', got %q", result.Reason)
	}
}

func TestRun_RawSQLSkipped(t *testing.T) {
	dir := t.TempDir()

	// Create migrations directory with SQL files
	migDir := filepath.Join(dir, "migrations")
	if err := os.Mkdir(migDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(migDir, "001_init.sql"), []byte("CREATE TABLE users;"), 0644); err != nil {
		t.Fatal(err)
	}

	opts := RunOptions{
		WorktreePath: dir,
		Verbose:      false,
	}

	result, err := Run(opts, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Skipped {
		t.Error("expected migration to be skipped for raw SQL")
	}

	if result.Tool == nil || result.Tool.Name != "sql" {
		t.Error("expected sql tool to be detected")
	}
}

func TestRun_DryRun(t *testing.T) {
	dir := t.TempDir()

	// Create a Makefile
	if err := os.WriteFile(filepath.Join(dir, "Makefile"), []byte("migrate:\n\techo done"), 0644); err != nil {
		t.Fatal(err)
	}

	opts := RunOptions{
		WorktreePath: dir,
		Verbose:      false,
		DryRun:       true,
	}

	result, err := Run(opts, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Skipped {
		t.Error("expected migration to be skipped in dry run mode")
	}

	if result.Tool == nil || result.Tool.Name != "makefile" {
		t.Error("expected makefile tool to be detected")
	}
}

func TestRun_CustomCommand(t *testing.T) {
	dir := t.TempDir()

	cfg := &config.MigrationsConfig{
		Command:    "echo custom migration",
		AutoDetect: true,
	}

	opts := RunOptions{
		WorktreePath:       dir,
		Verbose:            false,
		SkipContainerCheck: true, // Skip container check
	}

	result, err := Run(opts, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Skipped {
		t.Errorf("expected migration to run, got skipped: %s", result.Reason)
	}

	if !result.Success {
		t.Errorf("expected migration to succeed, got error: %v", result.Error)
	}

	if result.Tool == nil || result.Tool.Name != "custom" {
		t.Error("expected custom tool to be used")
	}
}
