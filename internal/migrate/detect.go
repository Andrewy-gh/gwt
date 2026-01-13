package migrate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Andrewy-gh/gwt/internal/config"
)

// Detect finds migration tools in the given worktree path
func Detect(worktreePath string, cfg *config.MigrationsConfig) (*MigrationTool, error) {
	// Priority 1: Config override
	if cfg != nil && cfg.Command != "" {
		return &MigrationTool{
			Name:        "custom",
			Command:     parseCommand(cfg.Command),
			Path:        worktreePath,
			Description: "Custom migration command from config",
		}, nil
	}

	// Priority 2: Auto-detection if enabled
	if cfg == nil || cfg.AutoDetect {
		return autoDetect(worktreePath)
	}

	return nil, nil // No migrations configured
}

func autoDetect(path string) (*MigrationTool, error) {
	detectors := []func(string) (*MigrationTool, error){
		detectMakefile,
		detectPrisma,
		detectDrizzle,
		detectAlembic,
		detectRawSQL,
	}

	for _, detect := range detectors {
		tool, err := detect(path)
		if err != nil {
			return nil, err
		}
		if tool != nil {
			return tool, nil
		}
	}

	return nil, nil // No migration tool found
}

// detectMakefile checks for Makefile with migrate target
func detectMakefile(path string) (*MigrationTool, error) {
	makefilePath := filepath.Join(path, "Makefile")
	content, err := os.ReadFile(makefilePath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Check for migrate-related targets
	targets := []string{"migrate:", "db-migrate:", "db:migrate:"}
	for _, target := range targets {
		if strings.Contains(string(content), target) {
			targetName := strings.TrimSuffix(target, ":")
			return &MigrationTool{
				Name:        "makefile",
				Command:     []string{"make", targetName},
				Path:        path,
				Description: fmt.Sprintf("Makefile target: %s", targetName),
			}, nil
		}
	}
	return nil, nil
}

// detectPrisma checks for Prisma schema
func detectPrisma(path string) (*MigrationTool, error) {
	locations := []string{
		filepath.Join(path, "prisma", "schema.prisma"),
		filepath.Join(path, "schema.prisma"),
	}

	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return &MigrationTool{
				Name:        "prisma",
				Command:     []string{"npx", "prisma", "migrate", "deploy"},
				Path:        path,
				Description: "Prisma migrations",
			}, nil
		}
	}
	return nil, nil
}

// detectDrizzle checks for Drizzle config
func detectDrizzle(path string) (*MigrationTool, error) {
	configs := []string{
		filepath.Join(path, "drizzle.config.ts"),
		filepath.Join(path, "drizzle.config.js"),
	}

	for _, cfg := range configs {
		if _, err := os.Stat(cfg); err == nil {
			return &MigrationTool{
				Name:        "drizzle",
				Command:     []string{"npx", "drizzle-kit", "migrate"},
				Path:        path,
				Description: "Drizzle migrations",
			}, nil
		}
	}
	return nil, nil
}

// detectAlembic checks for Alembic (Python)
func detectAlembic(path string) (*MigrationTool, error) {
	indicators := []string{
		filepath.Join(path, "alembic.ini"),
		filepath.Join(path, "alembic"),
	}

	for _, ind := range indicators {
		info, err := os.Stat(ind)
		if err == nil {
			if info.IsDir() || strings.HasSuffix(ind, ".ini") {
				return &MigrationTool{
					Name:        "alembic",
					Command:     []string{"alembic", "upgrade", "head"},
					Path:        path,
					Description: "Alembic migrations",
				}, nil
			}
		}
	}
	return nil, nil
}

// detectRawSQL checks for SQL migration files
func detectRawSQL(path string) (*MigrationTool, error) {
	migrationDirs := []string{
		filepath.Join(path, "migrations"),
		filepath.Join(path, "db", "migrations"),
		filepath.Join(path, "sql", "migrations"),
	}

	for _, dir := range migrationDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), ".sql") {
				return &MigrationTool{
					Name:        "sql",
					Command:     nil, // Requires manual handling
					Path:        dir,
					Description: fmt.Sprintf("Raw SQL files in %s (requires manual execution)", filepath.Base(dir)),
				}, nil
			}
		}
	}
	return nil, nil
}

// parseCommand splits a command string into args
func parseCommand(cmd string) []string {
	// Handle quoted strings properly
	return strings.Fields(cmd)
}
