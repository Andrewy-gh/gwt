// Package filter provides worktree filtering capabilities
package filter

import (
	"regexp"
	"strings"
	"time"

	"github.com/Andrewy-gh/gwt/internal/git"
)

// Operator represents a comparison operator
type Operator string

const (
	OpEquals    Operator = "="
	OpNotEquals Operator = "!="
	OpGreater   Operator = ">"
	OpLess      Operator = "<"
	OpGreaterEq Operator = ">="
	OpLessEq    Operator = "<="
	OpRegex     Operator = "~"
	OpContains  Operator = ":"
)

// FilterExpr represents a single filter expression
type FilterExpr struct {
	Field    string   // Field to filter on: "branch", "status", "age", "path", "commits", "main"
	Operator Operator // Comparison operator
	Value    string   // Value to compare against
	Negate   bool     // Whether to negate the match
}

// Filter represents a collection of filter expressions
type Filter struct {
	Expressions []FilterExpr
}

// New creates a new empty filter
func New() *Filter {
	return &Filter{
		Expressions: make([]FilterExpr, 0),
	}
}

// Add adds a filter expression
func (f *Filter) Add(expr FilterExpr) {
	f.Expressions = append(f.Expressions, expr)
}

// IsEmpty returns true if there are no filter expressions
func (f *Filter) IsEmpty() bool {
	return len(f.Expressions) == 0
}

// WorktreeFilterContext contains all data needed to filter a worktree
type WorktreeFilterContext struct {
	Worktree *git.Worktree
	Status   *git.WorktreeStatus
}

// Match checks if a worktree matches all filter expressions (AND logic)
func (f *Filter) Match(ctx *WorktreeFilterContext) bool {
	if f.IsEmpty() {
		return true
	}

	for _, expr := range f.Expressions {
		if !matchExpression(expr, ctx) {
			return false
		}
	}

	return true
}

// matchExpression checks if a single expression matches
func matchExpression(expr FilterExpr, ctx *WorktreeFilterContext) bool {
	wt := ctx.Worktree
	status := ctx.Status

	var result bool

	switch strings.ToLower(expr.Field) {
	case "branch":
		result = matchString(expr, wt.Branch)
	case "path":
		result = matchString(expr, wt.Path)
	case "status":
		result = matchStatus(expr, status)
	case "age":
		result = matchAge(expr, wt.Path)
	case "commits":
		result = matchCommits(expr, status)
	case "main":
		result = matchBool(expr, wt.IsMain)
	case "locked":
		result = matchBool(expr, wt.Locked)
	case "detached":
		result = matchBool(expr, wt.IsDetached)
	default:
		// Unknown field, don't match
		return false
	}

	if expr.Negate {
		return !result
	}
	return result
}

// matchString matches a string value
func matchString(expr FilterExpr, value string) bool {
	pattern := expr.Value

	switch expr.Operator {
	case OpEquals, OpContains:
		// Contains check (case-insensitive by default)
		return strings.Contains(strings.ToLower(value), strings.ToLower(pattern))
	case OpNotEquals:
		return !strings.Contains(strings.ToLower(value), strings.ToLower(pattern))
	case OpRegex:
		// Regex match
		re, err := regexp.Compile(pattern)
		if err != nil {
			return false
		}
		return re.MatchString(value)
	default:
		return value == pattern
	}
}

// matchStatus matches status (clean/dirty)
func matchStatus(expr FilterExpr, status *git.WorktreeStatus) bool {
	if status == nil {
		return false
	}

	statusValue := strings.ToLower(expr.Value)
	isClean := status.Clean

	switch statusValue {
	case "clean":
		return isClean
	case "dirty":
		return !isClean
	default:
		return false
	}
}

// matchAge matches worktree age
func matchAge(expr FilterExpr, worktreePath string) bool {
	// Get worktree age
	age, err := git.GetWorktreeAge(worktreePath)
	if err != nil {
		return false
	}

	// Parse the threshold duration
	threshold, err := parseDuration(expr.Value)
	if err != nil {
		return false
	}

	switch expr.Operator {
	case OpGreater:
		return age > threshold
	case OpLess:
		return age < threshold
	case OpGreaterEq:
		return age >= threshold
	case OpLessEq:
		return age <= threshold
	case OpEquals, OpContains:
		// For equality, allow some tolerance (within 1 day)
		diff := age - threshold
		if diff < 0 {
			diff = -diff
		}
		return diff < 24*time.Hour
	default:
		return false
	}
}

// matchCommits matches commit count (ahead of upstream)
func matchCommits(expr FilterExpr, status *git.WorktreeStatus) bool {
	if status == nil {
		return false
	}

	count := status.AheadCount

	// Parse the threshold
	threshold, err := parseInt(expr.Value)
	if err != nil {
		return false
	}

	switch expr.Operator {
	case OpGreater:
		return count > threshold
	case OpLess:
		return count < threshold
	case OpGreaterEq:
		return count >= threshold
	case OpLessEq:
		return count <= threshold
	case OpEquals, OpContains:
		return count == threshold
	case OpNotEquals:
		return count != threshold
	default:
		return false
	}
}

// matchBool matches a boolean value
func matchBool(expr FilterExpr, value bool) bool {
	boolVal := strings.ToLower(expr.Value)

	switch boolVal {
	case "true", "yes", "1":
		return value
	case "false", "no", "0":
		return !value
	default:
		return false
	}
}

// FilterWorktrees filters a list of worktrees using the filter
func (f *Filter) FilterWorktrees(worktrees []git.Worktree, getStatus func(path string) *git.WorktreeStatus) []git.Worktree {
	if f.IsEmpty() {
		return worktrees
	}

	result := make([]git.Worktree, 0, len(worktrees))
	for _, wt := range worktrees {
		ctx := &WorktreeFilterContext{
			Worktree: &wt,
			Status:   getStatus(wt.Path),
		}

		if f.Match(ctx) {
			result = append(result, wt)
		}
	}

	return result
}
