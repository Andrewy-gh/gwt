package copy

import (
	"errors"
	"fmt"
)

var (
	ErrNoSourceDirectory = errors.New("source directory does not exist")
	ErrTargetExists      = errors.New("target already exists")
	ErrCopyFailed        = errors.New("file copy failed")
	ErrPatternInvalid    = errors.New("invalid glob pattern")
	ErrPermissionDenied  = errors.New("permission denied")
)

// CopyError wraps an error with file context
type CopyError struct {
	Path string
	Op   string
	Err  error
}

func (e *CopyError) Error() string {
	return fmt.Sprintf("%s %s: %v", e.Op, e.Path, e.Err)
}

func (e *CopyError) Unwrap() error {
	return e.Err
}
