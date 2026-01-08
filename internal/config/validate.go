package config

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Validate checks the config for invalid values
// Returns a slice of validation errors (empty if valid)
func (c *Config) Validate() []ConfigValidationError {
	var errors []ConfigValidationError

	// Validate docker mode
	if err := ValidateDockerMode(c.Docker.DefaultMode); err != nil {
		errors = append(errors, ConfigValidationError{
			Field:   "docker.default_mode",
			Value:   c.Docker.DefaultMode,
			Message: err.Error(),
		})
	}

	// Validate port offset
	if c.Docker.PortOffset < 0 || c.Docker.PortOffset >= 65535 {
		errors = append(errors, ConfigValidationError{
			Field:   "docker.port_offset",
			Value:   c.Docker.PortOffset,
			Message: "must be between 0 and 65534",
		})
	}

	// Validate copy_defaults glob patterns
	if errs := ValidateGlobPatterns(c.CopyDefaults); len(errs) > 0 {
		for _, err := range errs {
			errors = append(errors, ConfigValidationError{
				Field:   "copy_defaults",
				Value:   c.CopyDefaults,
				Message: err.Error(),
			})
		}
	}

	// Validate copy_exclude glob patterns
	if errs := ValidateGlobPatterns(c.CopyExclude); len(errs) > 0 {
		for _, err := range errs {
			errors = append(errors, ConfigValidationError{
				Field:   "copy_exclude",
				Value:   c.CopyExclude,
				Message: err.Error(),
			})
		}
	}

	// Validate dependency paths
	if errs := ValidatePaths(c.Dependencies.Paths); len(errs) > 0 {
		for _, err := range errs {
			errors = append(errors, ConfigValidationError{
				Field:   "dependencies.paths",
				Value:   c.Dependencies.Paths,
				Message: err.Error(),
			})
		}
	}

	// Validate hooks
	for i, hook := range c.Hooks.PostCreate {
		if strings.TrimSpace(hook) == "" {
			errors = append(errors, ConfigValidationError{
				Field:   fmt.Sprintf("hooks.post_create[%d]", i),
				Value:   hook,
				Message: "hook command cannot be empty",
			})
		}
	}

	for i, hook := range c.Hooks.PostDelete {
		if strings.TrimSpace(hook) == "" {
			errors = append(errors, ConfigValidationError{
				Field:   fmt.Sprintf("hooks.post_delete[%d]", i),
				Value:   hook,
				Message: "hook command cannot be empty",
			})
		}
	}

	return errors
}

// ValidateDockerMode checks if the docker mode is valid
func ValidateDockerMode(mode string) error {
	if mode != "shared" && mode != "new" {
		return fmt.Errorf("must be 'shared' or 'new'")
	}
	return nil
}

// ValidateGlobPatterns checks if glob patterns are valid
func ValidateGlobPatterns(patterns []string) []error {
	var errors []error

	for _, pattern := range patterns {
		// Basic validation: check for empty patterns
		if strings.TrimSpace(pattern) == "" {
			errors = append(errors, fmt.Errorf("empty glob pattern"))
			continue
		}

		// Try to match the pattern against a test string to validate syntax
		// filepath.Match will return an error if the pattern is malformed
		_, err := filepath.Match(pattern, "test")
		if err != nil {
			errors = append(errors, fmt.Errorf("invalid glob pattern '%s': %v", pattern, err))
		}
	}

	return errors
}

// ValidatePaths checks if paths are reasonable (not absolute, etc.)
func ValidatePaths(paths []string) []error {
	var errors []error

	for _, path := range paths {
		// Check for empty paths
		if strings.TrimSpace(path) == "" {
			errors = append(errors, fmt.Errorf("empty path"))
			continue
		}

		// Warn about absolute paths (but don't error)
		if filepath.IsAbs(path) {
			// This is more of a warning, but we'll include it as an error
			// In practice, you might want to separate warnings from errors
			errors = append(errors, fmt.Errorf("absolute path '%s' may not work across different systems", path))
		}
	}

	return errors
}
