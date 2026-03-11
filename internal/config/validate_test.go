package config

import (
	"testing"
)

func TestValidateDockerMode(t *testing.T) {
	tests := []struct {
		mode      string
		shouldErr bool
	}{
		{"shared", false},
		{"new", false},
		{"invalid", true},
		{"", true},
		{"SHARED", true}, // case sensitive
		{"NEW", true},    // case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			err := ValidateDockerMode(tt.mode)
			if tt.shouldErr && err == nil {
				t.Errorf("Expected error for mode '%s', got nil", tt.mode)
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("Expected no error for mode '%s', got: %v", tt.mode, err)
			}
		})
	}
}

func TestValidateGlobPatterns(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		wantErr  bool
	}{
		{
			name:     "valid patterns",
			patterns: []string{"*.go", "**/*.js", ".env"},
			wantErr:  false,
		},
		{
			name:     "empty pattern",
			patterns: []string{"*.go", ""},
			wantErr:  true,
		},
		{
			name:     "invalid pattern",
			patterns: []string{"["},
			wantErr:  true,
		},
		{
			name:     "empty list",
			patterns: []string{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateGlobPatterns(tt.patterns)
			if tt.wantErr && len(errs) == 0 {
				t.Error("Expected errors, got none")
			}
			if !tt.wantErr && len(errs) > 0 {
				t.Errorf("Expected no errors, got: %v", errs)
			}
		})
	}
}

func TestValidatePaths(t *testing.T) {
	tests := []struct {
		name    string
		paths   []string
		wantErr bool
	}{
		{
			name:    "relative paths",
			paths:   []string{".", "client", "server"},
			wantErr: false,
		},
		{
			name:    "absolute paths",
			paths:   []string{"C:\\absolute\\path"},
			wantErr: true,
		},
		{
			name:    "absolute paths with forward slashes",
			paths:   []string{"C:/absolute/path"},
			wantErr: true,
		},
		{
			name:    "empty path",
			paths:   []string{""},
			wantErr: true,
		},
		{
			name:    "mixed paths",
			paths:   []string{".", "C:\\absolute"},
			wantErr: true,
		},
		{
			name:    "empty list",
			paths:   []string{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidatePaths(tt.paths)
			if tt.wantErr && len(errs) == 0 {
				t.Error("Expected errors, got none")
			}
			if !tt.wantErr && len(errs) > 0 {
				t.Errorf("Expected no errors, got: %v", errs)
			}
		})
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name:    "valid default config",
			cfg:     DefaultConfig(),
			wantErr: false,
		},
		{
			name: "invalid docker mode",
			cfg: &Config{
				Docker: DockerConfig{
					DefaultMode: "invalid",
					PortOffset:  1,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid port offset - negative",
			cfg: &Config{
				Docker: DockerConfig{
					DefaultMode: "shared",
					PortOffset:  -1,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid port offset - too large",
			cfg: &Config{
				Docker: DockerConfig{
					DefaultMode: "shared",
					PortOffset:  65535,
				},
			},
			wantErr: true,
		},
		{
			name: "empty hook command",
			cfg: &Config{
				Docker: DockerConfig{
					DefaultMode: "shared",
					PortOffset:  1,
				},
				Hooks: HooksConfig{
					PostCreate: []string{""},
				},
			},
			wantErr: true,
		},
		{
			name: "valid config with hooks",
			cfg: &Config{
				Docker: DockerConfig{
					DefaultMode: "new",
					PortOffset:  10,
				},
				Dependencies: DependenciesConfig{
					AutoInstall: true,
					Paths:       []string{"."},
				},
				Migrations: MigrationsConfig{
					AutoDetect: true,
				},
				Hooks: HooksConfig{
					PostCreate: []string{"echo 'hello'"},
					PostDelete: []string{"echo 'goodbye'"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := tt.cfg.Validate()
			if tt.wantErr && len(errs) == 0 {
				t.Error("Expected validation errors, got none")
			}
			if !tt.wantErr && len(errs) > 0 {
				t.Errorf("Expected no validation errors, got: %v", errs)
			}
		})
	}
}
