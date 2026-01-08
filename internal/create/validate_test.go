package create

import (
	"path/filepath"
	"testing"
)

func TestValidateBranchName(t *testing.T) {
	tests := []struct {
		name      string
		branch    string
		wantError bool
		errorMsg  string
	}{
		// Valid branch names
		{"simple name", "main", false, ""},
		{"with dash", "feature-auth", false, ""},
		{"with slash", "feature/auth", false, ""},
		{"with number", "v1.0.0", false, ""},
		{"release branch", "release/v1.0.0", false, ""},
		{"bugfix", "bugfix/issue-123", false, ""},

		// Invalid branch names
		{"empty", "", true, "empty"},
		{"single @", "@", true, "'@'"},
		{"starts with dash", "-feature", true, "start with '-'"},
		{"ends with .lock", "feature.lock", true, "end with '.lock'"},
		{"contains ..", "feature..test", true, "contain '..'"},
		{"contains @{", "feature@{test", true, "contain '@{'"},
		{"contains space", "feature auth", true, "contain spaces"},
		{"contains ~", "feature~test", true, "contain '~'"},
		{"contains ^", "feature^test", true, "contain '^'"},
		{"contains :", "feature:test", true, "contain ':'"},
		{"contains ?", "feature?test", true, "contain '?'"},
		{"contains *", "feature*test", true, "contain '*'"},
		{"contains [", "feature[test", true, "contain '['"},
		{"contains \\", "feature\\test", true, "contain '\\'"},
		{"ends with /", "feature/", true, "end with '/'"},
		{"consecutive slashes", "feature//test", true, "consecutive slashes"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBranchName(tt.branch)
			if tt.wantError {
				if err == nil {
					t.Errorf("ValidateBranchName(%q) expected error containing %q, got nil", tt.branch, tt.errorMsg)
				} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("ValidateBranchName(%q) error = %v, want error containing %q", tt.branch, err, tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateBranchName(%q) unexpected error: %v", tt.branch, err)
				}
			}
		})
	}
}

func TestSanitizeDirectoryName(t *testing.T) {
	tests := []struct {
		name     string
		branch   string
		expected string
	}{
		{"simple", "main", "main"},
		{"with dash", "feature-auth", "feature-auth"},
		{"single slash", "feature/auth", "feature-auth"},
		{"multiple slashes", "feature/auth/login", "feature-auth-login"},
		{"backslash", "feature\\auth", "feature-auth"},
		{"mixed slashes", "feature/auth\\login", "feature-auth-login"},
		{"leading slash", "/feature", "feature"},
		{"trailing slash", "feature/", "feature"},
		{"consecutive slashes", "feature//auth", "feature-auth"},
		{"multiple dashes", "feature---auth", "feature-auth"},
		{"release version", "release/v1.0.0", "release-v1.0.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeDirectoryName(tt.branch)
			if result != tt.expected {
				t.Errorf("SanitizeDirectoryName(%q) = %q, want %q", tt.branch, result, tt.expected)
			}
		})
	}
}

func TestGenerateWorktreePath(t *testing.T) {
	tests := []struct {
		name            string
		mainWorktree    string
		branch          string
		expectedSuffix  string // We'll check the suffix since paths differ by OS
		expectedProject string // Expected project name prefix
	}{
		{
			name:            "simple branch",
			mainWorktree:    "/home/user/myproject",
			branch:          "feature-auth",
			expectedProject: "myproject",
			expectedSuffix:  "myproject-feature-auth",
		},
		{
			name:            "branch with slashes",
			mainWorktree:    "/home/user/myproject",
			branch:          "feature/auth/login",
			expectedProject: "myproject",
			expectedSuffix:  "myproject-feature-auth-login",
		},
		{
			name:            "windows path",
			mainWorktree:    "C:\\Users\\user\\myproject",
			branch:          "bugfix/issue-123",
			expectedProject: "myproject",
			expectedSuffix:  "myproject-bugfix-issue-123",
		},
		{
			name:            "release version",
			mainWorktree:    "/var/www/site",
			branch:          "release/v1.0.0",
			expectedProject: "site",
			expectedSuffix:  "site-release-v1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateWorktreePath(tt.mainWorktree, tt.branch)

			// Check that the path ends with the expected suffix
			baseName := filepath.Base(result)
			if baseName != tt.expectedSuffix {
				t.Errorf("GenerateWorktreePath() basename = %q, want %q", baseName, tt.expectedSuffix)
			}

			// Check that the parent directory matches
			expectedParent := filepath.Dir(tt.mainWorktree)
			actualParent := filepath.Dir(result)
			if actualParent != expectedParent {
				t.Errorf("GenerateWorktreePath() parent = %q, want %q", actualParent, expectedParent)
			}

			// Verify it's a sibling (same parent directory)
			mainParent := filepath.Dir(tt.mainWorktree)
			resultParent := filepath.Dir(result)
			if mainParent != resultParent {
				t.Errorf("GenerateWorktreePath() not a sibling: main parent = %q, result parent = %q", mainParent, resultParent)
			}
		})
	}
}

func TestValidateDirectoryName(t *testing.T) {
	tests := []struct {
		name      string
		dirName   string
		wantError bool
		errorMsg  string
	}{
		// Valid directory names
		{"simple", "mydir", false, ""},
		{"with dash", "my-dir", false, ""},
		{"with underscore", "my_dir", false, ""},
		{"with dot", "my.dir", false, ""},
		{"with number", "dir123", false, ""},

		// Invalid directory names
		{"empty", "", true, "empty"},
		{"contains <", "my<dir", true, "contain '<'"},
		{"contains >", "my>dir", true, "contain '>'"},
		{"contains :", "my:dir", true, "contain ':'"},
		{"contains \"", "my\"dir", true, "contain '\"'"},
		{"contains |", "my|dir", true, "contain '|'"},
		{"contains ?", "my?dir", true, "contain '?'"},
		{"contains *", "my*dir", true, "contain '*'"},

		// Windows reserved names
		{"reserved CON", "CON", true, "reserved"},
		{"reserved PRN", "PRN", true, "reserved"},
		{"reserved AUX", "AUX", true, "reserved"},
		{"reserved NUL", "NUL", true, "reserved"},
		{"reserved COM1", "COM1", true, "reserved"},
		{"reserved LPT1", "LPT1", true, "reserved"},
		{"reserved with extension", "CON.txt", true, "reserved"},
		{"lowercase reserved", "con", true, "reserved"},
		{"not reserved", "CONNECT", false, ""}, // Should be valid
		{"not reserved prefix", "CONFIG", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDirectoryName(tt.dirName)
			if tt.wantError {
				if err == nil {
					t.Errorf("ValidateDirectoryName(%q) expected error containing %q, got nil", tt.dirName, tt.errorMsg)
				} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("ValidateDirectoryName(%q) error = %v, want error containing %q", tt.dirName, err, tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateDirectoryName(%q) unexpected error: %v", tt.dirName, err)
				}
			}
		})
	}
}

// Helper function to check if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
