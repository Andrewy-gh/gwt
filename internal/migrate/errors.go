package migrate

import "fmt"

// MigrationError represents a migration execution failure
type MigrationError struct {
	Tool     string
	Command  []string
	Stderr   string
	ExitCode int
}

func (e *MigrationError) Error() string {
	return fmt.Sprintf("migration failed (%s): exit code %d", e.Tool, e.ExitCode)
}

// DetectionError represents a failure in detecting migration tools
type DetectionError struct {
	Path string
	Err  error
}

func (e *DetectionError) Error() string {
	return fmt.Sprintf("migration detection failed at %s: %v", e.Path, e.Err)
}

// ContainerNotReadyError indicates the database container isn't running
type ContainerNotReadyError struct {
	Service string
	Reason  string
}

func (e *ContainerNotReadyError) Error() string {
	return fmt.Sprintf("database container %q not ready: %s", e.Service, e.Reason)
}
