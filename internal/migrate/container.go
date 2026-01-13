package migrate

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ContainerStatus represents the state of a Docker container
type ContainerStatus struct {
	Name    string
	Running bool
	Health  string // "healthy", "unhealthy", "starting", "none"
}

// CheckDatabaseContainer verifies the database container is ready
func CheckDatabaseContainer(worktreePath string) (*ContainerStatus, error) {
	// Try to find docker-compose.yml
	composeFile := findComposeFile(worktreePath)
	if composeFile == "" {
		return nil, nil // No compose file, skip check
	}

	// Get database service name (common conventions)
	dbServices := []string{"db", "database", "postgres", "mysql", "mariadb", "mongodb"}

	for _, service := range dbServices {
		status, err := getContainerStatus(worktreePath, service)
		if err != nil {
			continue // Service doesn't exist
		}
		if status != nil {
			return status, nil
		}
	}

	return nil, nil // No database container found
}

func findComposeFile(path string) string {
	names := []string{
		"docker-compose.yml",
		"docker-compose.yaml",
		"compose.yml",
		"compose.yaml",
	}

	for _, name := range names {
		fullPath := filepath.Join(path, name)
		if _, err := os.Stat(fullPath); err == nil {
			return fullPath
		}
	}
	return ""
}

func getContainerStatus(worktreePath, service string) (*ContainerStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Use docker compose ps to check service status
	cmd := exec.CommandContext(ctx, "docker", "compose", "ps", service, "--format", "{{.State}}")
	cmd.Dir = worktreePath

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	state := strings.TrimSpace(string(output))
	if state == "" {
		return nil, nil
	}

	return &ContainerStatus{
		Name:    service,
		Running: state == "running",
		Health:  "none", // Could parse health status if needed
	}, nil
}

// WaitForContainer waits for a container to be ready with timeout
func WaitForContainer(worktreePath, service string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		status, err := getContainerStatus(worktreePath, service)
		if err == nil && status != nil && status.Running {
			return nil
		}
		time.Sleep(2 * time.Second)
	}

	return &ContainerNotReadyError{
		Service: service,
		Reason:  fmt.Sprintf("timeout after %v", timeout),
	}
}
