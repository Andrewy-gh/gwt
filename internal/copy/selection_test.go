package copy

import "testing"

func TestNewSelection(t *testing.T) {
	files := []IgnoredFile{
		{Path: ".env", IsDir: false, Size: 100},
		{Path: "node_modules", IsDir: true, Size: 5000},
		{Path: "app.log", IsDir: false, Size: 200},
	}

	matcher := NewPatternMatcher(
		[]string{".env"},  // defaults
		[]string{"*.log"}, // excludes (plus DefaultExcludes)
	)

	selection := NewSelection(files, matcher)

	// Check that .env is selected (matches default)
	// node_modules is excluded (in DefaultExcludes)
	// app.log is excluded (matches *.log)

	// We expect only .env to be in the selection (node_modules and app.log are excluded)
	if len(selection.Files) != 1 {
		t.Errorf("Expected 1 selectable file, got %d", len(selection.Files))
	}

	// Check that .env is selected
	if len(selection.Files) > 0 {
		if selection.Files[0].Path != ".env" {
			t.Errorf("Expected first file to be .env, got %s", selection.Files[0].Path)
		}
		if !selection.Files[0].Selected {
			t.Error("Expected .env to be pre-selected")
		}
	}

	// Check selected size
	if selection.SelectedSize != 100 {
		t.Errorf("Expected SelectedSize to be 100, got %d", selection.SelectedSize)
	}
}

func TestSelection_Toggle(t *testing.T) {
	files := []IgnoredFile{
		{Path: ".env", IsDir: false, Size: 100},
	}

	matcher := NewPatternMatcher(nil, nil)
	selection := NewSelection(files, matcher)

	// Initially not selected (no defaults)
	if selection.Files[0].Selected {
		t.Error("Expected file to not be selected initially")
	}
	if selection.SelectedSize != 0 {
		t.Errorf("Expected SelectedSize to be 0, got %d", selection.SelectedSize)
	}

	// Toggle on
	selection.Toggle(0)
	if !selection.Files[0].Selected {
		t.Error("Expected file to be selected after toggle")
	}
	if selection.SelectedSize != 100 {
		t.Errorf("Expected SelectedSize to be 100, got %d", selection.SelectedSize)
	}

	// Toggle off
	selection.Toggle(0)
	if selection.Files[0].Selected {
		t.Error("Expected file to not be selected after second toggle")
	}
	if selection.SelectedSize != 0 {
		t.Errorf("Expected SelectedSize to be 0, got %d", selection.SelectedSize)
	}
}

func TestSelection_SelectAll(t *testing.T) {
	files := []IgnoredFile{
		{Path: ".env", IsDir: false, Size: 100},
		{Path: "config.json", IsDir: false, Size: 200},
	}

	matcher := NewPatternMatcher(nil, nil)
	selection := NewSelection(files, matcher)

	// Initially nothing selected
	if selection.SelectedSize != 0 {
		t.Errorf("Expected SelectedSize to be 0, got %d", selection.SelectedSize)
	}

	// Select all
	selection.SelectAll()

	// Check all files are selected
	for i, file := range selection.Files {
		if !file.Selected {
			t.Errorf("Expected file %d to be selected", i)
		}
	}

	// Check selected size
	if selection.SelectedSize != 300 {
		t.Errorf("Expected SelectedSize to be 300, got %d", selection.SelectedSize)
	}
}

func TestSelection_DeselectAll(t *testing.T) {
	files := []IgnoredFile{
		{Path: ".env", IsDir: false, Size: 100},
		{Path: "config.json", IsDir: false, Size: 200},
	}

	matcher := NewPatternMatcher(
		[]string{".env", "config.json"}, // pre-select both
		nil,
	)
	selection := NewSelection(files, matcher)

	// Initially both selected
	if selection.SelectedSize != 300 {
		t.Errorf("Expected SelectedSize to be 300, got %d", selection.SelectedSize)
	}

	// Deselect all
	selection.DeselectAll()

	// Check all files are deselected
	for i, file := range selection.Files {
		if file.Selected {
			t.Errorf("Expected file %d to be deselected", i)
		}
	}

	// Check selected size
	if selection.SelectedSize != 0 {
		t.Errorf("Expected SelectedSize to be 0, got %d", selection.SelectedSize)
	}
}

func TestSelection_GetSelected(t *testing.T) {
	files := []IgnoredFile{
		{Path: ".env", IsDir: false, Size: 100},
		{Path: "config.json", IsDir: false, Size: 200},
		{Path: "app.log", IsDir: false, Size: 50},
	}

	matcher := NewPatternMatcher(
		[]string{".env"}, // only pre-select .env
		nil,
	)
	selection := NewSelection(files, matcher)

	selected := selection.GetSelected()

	// Only .env should be selected
	if len(selected) != 1 {
		t.Errorf("Expected 1 selected file, got %d", len(selected))
	}
	if len(selected) > 0 && selected[0].Path != ".env" {
		t.Errorf("Expected selected file to be .env, got %s", selected[0].Path)
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{
			name:     "bytes",
			bytes:    100,
			expected: "100 B",
		},
		{
			name:     "kilobytes",
			bytes:    1024,
			expected: "1.0 KB",
		},
		{
			name:     "kilobytes with decimal",
			bytes:    1536, // 1.5 KB
			expected: "1.5 KB",
		},
		{
			name:     "megabytes",
			bytes:    1048576, // 1 MB
			expected: "1.0 MB",
		},
		{
			name:     "gigabytes",
			bytes:    1073741824, // 1 GB
			expected: "1.0 GB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatSize(tt.bytes)
			if result != tt.expected {
				t.Errorf("FormatSize(%d) = %s, want %s", tt.bytes, result, tt.expected)
			}
		})
	}
}
