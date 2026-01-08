package config

import "fmt"

// ConfigNotFoundError indicates no config file was found
type ConfigNotFoundError struct {
	SearchPath string
}

func (e *ConfigNotFoundError) Error() string {
	return fmt.Sprintf("config file not found in %s", e.SearchPath)
}

// ConfigParseError indicates the config file is invalid
type ConfigParseError struct {
	Path string
	Err  error
}

func (e *ConfigParseError) Error() string {
	return fmt.Sprintf("failed to parse config file %s: %v", e.Path, e.Err)
}

func (e *ConfigParseError) Unwrap() error {
	return e.Err
}

// ConfigValidationError indicates the config has invalid values
type ConfigValidationError struct {
	Field   string
	Value   interface{}
	Message string
}

func (e *ConfigValidationError) Error() string {
	return fmt.Sprintf("invalid config value for %s (%v): %s", e.Field, e.Value, e.Message)
}
