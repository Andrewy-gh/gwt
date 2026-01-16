package filter

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Parse parses a filter expression string into a Filter
// Supported formats:
//   - field:value          (contains check)
//   - field:!value         (negated contains check)
//   - field:=value         (exact match)
//   - field:!=value        (not equals)
//   - field:>value         (greater than)
//   - field:<value         (less than)
//   - field:>=value        (greater or equal)
//   - field:<=value        (less or equal)
//   - field:~regex         (regex match)
//   - field:^regex         (regex match, starts with)
func Parse(expr string) (*FilterExpr, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return nil, fmt.Errorf("empty filter expression")
	}

	// Check for negation prefix
	negate := false
	if strings.HasPrefix(expr, "!") {
		negate = true
		expr = strings.TrimPrefix(expr, "!")
	}

	// Split on first colon
	colonIdx := strings.Index(expr, ":")
	if colonIdx == -1 {
		return nil, fmt.Errorf("invalid filter format: expected 'field:value', got %q", expr)
	}

	field := strings.TrimSpace(expr[:colonIdx])
	rest := strings.TrimSpace(expr[colonIdx+1:])

	if field == "" {
		return nil, fmt.Errorf("empty field name")
	}

	if rest == "" {
		return nil, fmt.Errorf("empty filter value")
	}

	// Parse operator and value first, before checking negation
	// This ensures != is treated as "not equals" rather than "!=" being split
	op, value := parseOperatorAndValue(rest)

	// Check for negation in value (only if not already a != operator)
	if op != OpNotEquals && strings.HasPrefix(rest, "!") {
		negate = !negate
		rest = strings.TrimPrefix(rest, "!")
		op, value = parseOperatorAndValue(rest)
	}

	return &FilterExpr{
		Field:    field,
		Operator: op,
		Value:    value,
		Negate:   negate,
	}, nil
}

// ParseMultiple parses multiple filter expressions
func ParseMultiple(exprs []string) (*Filter, error) {
	filter := New()

	for _, expr := range exprs {
		parsed, err := Parse(expr)
		if err != nil {
			return nil, fmt.Errorf("invalid filter %q: %w", expr, err)
		}
		filter.Add(*parsed)
	}

	return filter, nil
}

// parseOperatorAndValue extracts the operator and value from a filter expression
func parseOperatorAndValue(s string) (Operator, string) {
	// Check for two-character operators first
	if strings.HasPrefix(s, ">=") {
		return OpGreaterEq, strings.TrimPrefix(s, ">=")
	}
	if strings.HasPrefix(s, "<=") {
		return OpLessEq, strings.TrimPrefix(s, "<=")
	}
	if strings.HasPrefix(s, "!=") {
		return OpNotEquals, strings.TrimPrefix(s, "!=")
	}

	// Check for single-character operators
	if strings.HasPrefix(s, "=") {
		return OpEquals, strings.TrimPrefix(s, "=")
	}
	if strings.HasPrefix(s, ">") {
		return OpGreater, strings.TrimPrefix(s, ">")
	}
	if strings.HasPrefix(s, "<") {
		return OpLess, strings.TrimPrefix(s, "<")
	}
	if strings.HasPrefix(s, "~") {
		return OpRegex, strings.TrimPrefix(s, "~")
	}
	if strings.HasPrefix(s, "^") {
		// ^ is shorthand for regex starts-with
		return OpRegex, "^" + strings.TrimPrefix(s, "^")
	}

	// Default: contains check
	return OpContains, s
}

// parseDuration parses a duration string like "7d", "2w", "1m"
func parseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}

	// Try custom format first: number + unit (d/w/m/y)
	re := regexp.MustCompile(`^(\d+)\s*([dwmy])$`)
	matches := re.FindStringSubmatch(s)
	if matches != nil {
		num, err := strconv.Atoi(matches[1])
		if err != nil {
			return 0, fmt.Errorf("invalid number: %s", matches[1])
		}

		unit := matches[2]
		switch unit {
		case "d":
			return time.Duration(num) * 24 * time.Hour, nil
		case "w":
			return time.Duration(num) * 7 * 24 * time.Hour, nil
		case "m":
			return time.Duration(num) * 30 * 24 * time.Hour, nil
		case "y":
			return time.Duration(num) * 365 * 24 * time.Hour, nil
		}
	}

	// Fallback to standard Go duration
	return time.ParseDuration(s)
}

// parseInt parses an integer string
func parseInt(s string) (int, error) {
	s = strings.TrimSpace(s)
	return strconv.Atoi(s)
}

// Validate validates a filter expression string without creating a filter
func Validate(expr string) error {
	_, err := Parse(expr)
	return err
}

// ValidateMultiple validates multiple filter expression strings
func ValidateMultiple(exprs []string) error {
	for _, expr := range exprs {
		if err := Validate(expr); err != nil {
			return err
		}
	}
	return nil
}

// SupportedFields returns the list of supported filter fields
func SupportedFields() []string {
	return []string{
		"branch",
		"path",
		"status",
		"age",
		"commits",
		"main",
		"locked",
		"detached",
	}
}

// FormatHelp returns a help string describing the filter syntax
func FormatHelp() string {
	return `Filter syntax:
  field:value          Contains check (case-insensitive)
  field:!value         Negated contains check
  field:=value         Exact match
  field:!=value        Not equals
  field:>value         Greater than (for numeric fields)
  field:<value         Less than (for numeric fields)
  field:>=value        Greater or equal
  field:<=value        Less or equal
  field:~regex         Regex match
  field:^prefix        Starts with (regex)

Supported fields:
  branch     Branch name
  path       Worktree path
  status     Clean or dirty (values: clean, dirty)
  age        Worktree age (values: 7d, 2w, 1m, etc.)
  commits    Unpushed commits count
  main       Is main worktree (values: true, false)
  locked     Is locked (values: true, false)
  detached   Is detached HEAD (values: true, false)

Examples:
  gwt list --filter "branch:feature"        Branches containing "feature"
  gwt list --filter "status:dirty"          Dirty worktrees only
  gwt list --filter "age:>7d"               Older than 7 days
  gwt list --filter "commits:>0"            Has unpushed commits
  gwt list --filter "branch:!main"          Not main branch
  gwt list --filter "branch:^fix/"          Branches starting with "fix/"
`
}
