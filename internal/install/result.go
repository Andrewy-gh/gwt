package install

// Result represents the overall installation result
type Result struct {
	Skipped  bool            // true if installation was skipped
	Reason   string          // reason for skipping (if skipped)
	Managers []ManagerResult // results per package manager
}

// ManagerResult represents installation result for a single package manager
type ManagerResult struct {
	Manager string // Package manager name (npm, yarn, go, etc.)
	Path    string // Directory where installation was run
	Success bool   // Whether installation succeeded
	Error   error  // Error if installation failed
}

// HasErrors returns true if any installation failed
func (r *Result) HasErrors() bool {
	for _, m := range r.Managers {
		if !m.Success {
			return true
		}
	}
	return false
}

// SuccessCount returns the number of successful installations
func (r *Result) SuccessCount() int {
	count := 0
	for _, m := range r.Managers {
		if m.Success {
			count++
		}
	}
	return count
}

// ErrorCount returns the number of failed installations
func (r *Result) ErrorCount() int {
	count := 0
	for _, m := range r.Managers {
		if !m.Success {
			count++
		}
	}
	return count
}
