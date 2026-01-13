package docker

import (
	"errors"
	"fmt"
)

var (
	ErrNoComposeFile      = errors.New("no docker compose file found")
	ErrInvalidComposeFile = errors.New("invalid docker compose file")
	ErrParseError         = errors.New("failed to parse compose file")
	ErrSymlinkFailed      = errors.New("symlink creation failed")
	ErrJunctionFailed     = errors.New("junction creation failed")
	ErrCopyFailed         = errors.New("directory copy failed")
	ErrOverrideExists     = errors.New("override file already exists")
	ErrPortConflict       = errors.New("port conflict detected")
)

// DockerError wraps an error with context
type DockerError struct {
	Op      string
	File    string
	Service string
	Err     error
}

func (e *DockerError) Error() string {
	if e.Service != "" {
		return fmt.Sprintf("%s %s (service: %s): %v", e.Op, e.File, e.Service, e.Err)
	}
	return fmt.Sprintf("%s %s: %v", e.Op, e.File, e.Err)
}

func (e *DockerError) Unwrap() error {
	return e.Err
}
