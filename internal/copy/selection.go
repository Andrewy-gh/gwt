package copy

import "fmt"

// SelectableFile represents a file that can be selected for copying
type SelectableFile struct {
	IgnoredFile             // Embedded file info
	Selected    bool        // Whether selected for copying
	MatchResult MatchResult // How it matched patterns
}

// Selection manages the list of selectable files
type Selection struct {
	Files        []SelectableFile
	TotalSize    int64 // Total size of all files
	SelectedSize int64 // Total size of selected files
}

// NewSelection creates a selection from discovered files and pattern matcher
func NewSelection(files []IgnoredFile, matcher *PatternMatcher) *Selection {
	selectableFiles := make([]SelectableFile, 0, len(files))
	var totalSize int64

	for _, file := range files {
		matchResult := matcher.Match(file.Path)

		// Skip excluded files entirely
		if matchResult == MatchExclude {
			continue
		}

		selectable := SelectableFile{
			IgnoredFile: file,
			Selected:    matchResult == MatchDefault, // Pre-select defaults
			MatchResult: matchResult,
		}

		selectableFiles = append(selectableFiles, selectable)
		totalSize += file.Size
	}

	// Calculate initial selected size
	var selectedSize int64
	for _, file := range selectableFiles {
		if file.Selected {
			selectedSize += file.Size
		}
	}

	return &Selection{
		Files:        selectableFiles,
		TotalSize:    totalSize,
		SelectedSize: selectedSize,
	}
}

// FilterVisible returns only files not matched by exclude patterns
func (s *Selection) FilterVisible() []SelectableFile {
	visible := make([]SelectableFile, 0, len(s.Files))
	for _, file := range s.Files {
		if file.MatchResult != MatchExclude {
			visible = append(visible, file)
		}
	}
	return visible
}

// GetSelected returns only selected files
func (s *Selection) GetSelected() []SelectableFile {
	selected := make([]SelectableFile, 0, len(s.Files))
	for _, file := range s.Files {
		if file.Selected {
			selected = append(selected, file)
		}
	}
	return selected
}

// Toggle toggles selection of a file by index
func (s *Selection) Toggle(index int) {
	if index < 0 || index >= len(s.Files) {
		return
	}

	file := &s.Files[index]
	file.Selected = !file.Selected

	// Update selected size
	if file.Selected {
		s.SelectedSize += file.Size
	} else {
		s.SelectedSize -= file.Size
	}
}

// SelectAll selects all visible files
func (s *Selection) SelectAll() {
	s.SelectedSize = 0
	for i := range s.Files {
		if s.Files[i].MatchResult != MatchExclude {
			s.Files[i].Selected = true
			s.SelectedSize += s.Files[i].Size
		}
	}
}

// DeselectAll deselects all files
func (s *Selection) DeselectAll() {
	for i := range s.Files {
		s.Files[i].Selected = false
	}
	s.SelectedSize = 0
}

// FormatSize formats bytes as human-readable string
func FormatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
