package filter

import (
	"testing"
	"time"

	"github.com/Andrewy-gh/gwt/internal/git"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *FilterExpr
		wantErr bool
	}{
		{
			name:  "simple contains",
			input: "branch:feature",
			want: &FilterExpr{
				Field:    "branch",
				Operator: OpContains,
				Value:    "feature",
				Negate:   false,
			},
		},
		{
			name:  "equals operator",
			input: "branch:=main",
			want: &FilterExpr{
				Field:    "branch",
				Operator: OpEquals,
				Value:    "main",
				Negate:   false,
			},
		},
		{
			name:  "not equals operator",
			input: "branch:!=main",
			want: &FilterExpr{
				Field:    "branch",
				Operator: OpNotEquals,
				Value:    "main",
				Negate:   false,
			},
		},
		{
			name:  "greater than",
			input: "age:>7d",
			want: &FilterExpr{
				Field:    "age",
				Operator: OpGreater,
				Value:    "7d",
				Negate:   false,
			},
		},
		{
			name:  "less than",
			input: "age:<30d",
			want: &FilterExpr{
				Field:    "age",
				Operator: OpLess,
				Value:    "30d",
				Negate:   false,
			},
		},
		{
			name:  "greater or equal",
			input: "commits:>=5",
			want: &FilterExpr{
				Field:    "commits",
				Operator: OpGreaterEq,
				Value:    "5",
				Negate:   false,
			},
		},
		{
			name:  "less or equal",
			input: "commits:<=10",
			want: &FilterExpr{
				Field:    "commits",
				Operator: OpLessEq,
				Value:    "10",
				Negate:   false,
			},
		},
		{
			name:  "regex match",
			input: "branch:~^feature/.*",
			want: &FilterExpr{
				Field:    "branch",
				Operator: OpRegex,
				Value:    "^feature/.*",
				Negate:   false,
			},
		},
		{
			name:  "starts with shorthand",
			input: "branch:^fix/",
			want: &FilterExpr{
				Field:    "branch",
				Operator: OpRegex,
				Value:    "^fix/",
				Negate:   false,
			},
		},
		{
			name:  "negated contains",
			input: "branch:!main",
			want: &FilterExpr{
				Field:    "branch",
				Operator: OpContains,
				Value:    "main",
				Negate:   true,
			},
		},
		{
			name:  "prefix negation",
			input: "!branch:feature",
			want: &FilterExpr{
				Field:    "branch",
				Operator: OpContains,
				Value:    "feature",
				Negate:   true,
			},
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: true,
		},
		{
			name:    "no colon",
			input:   "branch",
			wantErr: true,
		},
		{
			name:    "empty field",
			input:   ":value",
			wantErr: true,
		},
		{
			name:    "empty value",
			input:   "branch:",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if got.Field != tt.want.Field {
				t.Errorf("Field = %q, want %q", got.Field, tt.want.Field)
			}
			if got.Operator != tt.want.Operator {
				t.Errorf("Operator = %q, want %q", got.Operator, tt.want.Operator)
			}
			if got.Value != tt.want.Value {
				t.Errorf("Value = %q, want %q", got.Value, tt.want.Value)
			}
			if got.Negate != tt.want.Negate {
				t.Errorf("Negate = %v, want %v", got.Negate, tt.want.Negate)
			}
		})
	}
}

func TestParseMultiple(t *testing.T) {
	exprs := []string{"branch:feature", "status:dirty"}

	filter, err := ParseMultiple(exprs)
	if err != nil {
		t.Fatalf("ParseMultiple failed: %v", err)
	}

	if len(filter.Expressions) != 2 {
		t.Errorf("expected 2 expressions, got %d", len(filter.Expressions))
	}
}

func TestParseMultiple_InvalidExpression(t *testing.T) {
	exprs := []string{"branch:feature", "invalid"}

	_, err := ParseMultiple(exprs)
	if err == nil {
		t.Error("expected error for invalid expression")
	}
}

func TestFilterMatch_Branch(t *testing.T) {
	filter := New()
	filter.Add(FilterExpr{
		Field:    "branch",
		Operator: OpContains,
		Value:    "feature",
		Negate:   false,
	})

	tests := []struct {
		name   string
		branch string
		want   bool
	}{
		{"matches", "feature-auth", true},
		{"matches partial", "my-feature", true},
		{"case insensitive", "FEATURE-test", true},
		{"no match", "bugfix-123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &WorktreeFilterContext{
				Worktree: &git.Worktree{Branch: tt.branch},
			}
			if got := filter.Match(ctx); got != tt.want {
				t.Errorf("Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterMatch_Status(t *testing.T) {
	tests := []struct {
		name   string
		value  string
		clean  bool
		want   bool
	}{
		{"clean matches clean", "clean", true, true},
		{"clean doesn't match dirty", "clean", false, false},
		{"dirty matches dirty", "dirty", false, true},
		{"dirty doesn't match clean", "dirty", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := New()
			filter.Add(FilterExpr{
				Field:    "status",
				Operator: OpContains,
				Value:    tt.value,
			})

			ctx := &WorktreeFilterContext{
				Worktree: &git.Worktree{},
				Status:   &git.WorktreeStatus{Clean: tt.clean},
			}

			if got := filter.Match(ctx); got != tt.want {
				t.Errorf("Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterMatch_Commits(t *testing.T) {
	tests := []struct {
		name      string
		op        Operator
		threshold string
		count     int
		want      bool
	}{
		{"greater than - true", OpGreater, "5", 10, true},
		{"greater than - false", OpGreater, "5", 3, false},
		{"less than - true", OpLess, "5", 3, true},
		{"less than - false", OpLess, "5", 10, false},
		{"equals - true", OpEquals, "5", 5, true},
		{"equals - false", OpEquals, "5", 3, false},
		{"greater or equal - true", OpGreaterEq, "5", 5, true},
		{"less or equal - true", OpLessEq, "5", 5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := New()
			filter.Add(FilterExpr{
				Field:    "commits",
				Operator: tt.op,
				Value:    tt.threshold,
			})

			ctx := &WorktreeFilterContext{
				Worktree: &git.Worktree{},
				Status:   &git.WorktreeStatus{AheadCount: tt.count},
			}

			if got := filter.Match(ctx); got != tt.want {
				t.Errorf("Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterMatch_Main(t *testing.T) {
	tests := []struct {
		name   string
		value  string
		isMain bool
		want   bool
	}{
		{"true matches main", "true", true, true},
		{"true doesn't match non-main", "true", false, false},
		{"false matches non-main", "false", false, true},
		{"false doesn't match main", "false", true, false},
		{"yes matches main", "yes", true, true},
		{"no matches non-main", "no", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := New()
			filter.Add(FilterExpr{
				Field:    "main",
				Operator: OpEquals,
				Value:    tt.value,
			})

			ctx := &WorktreeFilterContext{
				Worktree: &git.Worktree{IsMain: tt.isMain},
			}

			if got := filter.Match(ctx); got != tt.want {
				t.Errorf("Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterMatch_Negation(t *testing.T) {
	filter := New()
	filter.Add(FilterExpr{
		Field:    "branch",
		Operator: OpContains,
		Value:    "main",
		Negate:   true,
	})

	tests := []struct {
		name   string
		branch string
		want   bool
	}{
		{"not main matches feature", "feature-test", true},
		{"not main excludes main", "main", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &WorktreeFilterContext{
				Worktree: &git.Worktree{Branch: tt.branch},
			}
			if got := filter.Match(ctx); got != tt.want {
				t.Errorf("Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterMatch_MultipleExpressions(t *testing.T) {
	// Filter for feature branches that are dirty
	filter := New()
	filter.Add(FilterExpr{
		Field:    "branch",
		Operator: OpContains,
		Value:    "feature",
	})
	filter.Add(FilterExpr{
		Field:    "status",
		Operator: OpContains,
		Value:    "dirty",
	})

	tests := []struct {
		name   string
		branch string
		clean  bool
		want   bool
	}{
		{"feature and dirty", "feature-auth", false, true},
		{"feature but clean", "feature-auth", true, false},
		{"not feature but dirty", "bugfix-123", false, false},
		{"not feature and clean", "bugfix-123", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &WorktreeFilterContext{
				Worktree: &git.Worktree{Branch: tt.branch},
				Status:   &git.WorktreeStatus{Clean: tt.clean},
			}
			if got := filter.Match(ctx); got != tt.want {
				t.Errorf("Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterMatch_Regex(t *testing.T) {
	filter := New()
	filter.Add(FilterExpr{
		Field:    "branch",
		Operator: OpRegex,
		Value:    "^feature/.*",
	})

	tests := []struct {
		name   string
		branch string
		want   bool
	}{
		{"matches regex", "feature/auth", true},
		{"matches full path", "feature/api/v2", true},
		{"no match prefix", "my-feature/auth", false},
		{"no match different", "bugfix/123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &WorktreeFilterContext{
				Worktree: &git.Worktree{Branch: tt.branch},
			}
			if got := filter.Match(ctx); got != tt.want {
				t.Errorf("Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterWorktrees(t *testing.T) {
	worktrees := []git.Worktree{
		{Path: "/repo/main", Branch: "main", IsMain: true},
		{Path: "/repo/feature-1", Branch: "feature-1"},
		{Path: "/repo/feature-2", Branch: "feature-2"},
		{Path: "/repo/bugfix", Branch: "bugfix-123"},
	}

	filter := New()
	filter.Add(FilterExpr{
		Field:    "branch",
		Operator: OpContains,
		Value:    "feature",
	})

	statusFunc := func(path string) *git.WorktreeStatus {
		return &git.WorktreeStatus{Clean: true}
	}

	result := filter.FilterWorktrees(worktrees, statusFunc)

	if len(result) != 2 {
		t.Errorf("expected 2 filtered worktrees, got %d", len(result))
	}

	for _, wt := range result {
		if wt.Branch != "feature-1" && wt.Branch != "feature-2" {
			t.Errorf("unexpected worktree in result: %s", wt.Branch)
		}
	}
}

func TestFilterIsEmpty(t *testing.T) {
	filter := New()
	if !filter.IsEmpty() {
		t.Error("new filter should be empty")
	}

	filter.Add(FilterExpr{Field: "branch", Value: "test"})
	if filter.IsEmpty() {
		t.Error("filter with expression should not be empty")
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Duration
		wantErr bool
	}{
		{"days", "7d", 7 * 24 * time.Hour, false},
		{"weeks", "2w", 14 * 24 * time.Hour, false},
		{"months", "1m", 30 * 24 * time.Hour, false},
		{"years", "1y", 365 * 24 * time.Hour, false},
		{"go duration", "24h", 24 * time.Hour, false},
		{"empty", "", 0, true},
		{"invalid", "abc", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDuration(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDuration(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseDuration(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestSupportedFields(t *testing.T) {
	fields := SupportedFields()
	if len(fields) == 0 {
		t.Error("expected at least some supported fields")
	}

	// Check for essential fields
	essential := []string{"branch", "status", "path"}
	for _, f := range essential {
		found := false
		for _, sf := range fields {
			if sf == f {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected %q in supported fields", f)
		}
	}
}

func TestFormatHelp(t *testing.T) {
	help := FormatHelp()
	if help == "" {
		t.Error("help should not be empty")
	}

	// Should contain some key terms
	keywords := []string{"branch", "status", "filter", "Examples"}
	for _, kw := range keywords {
		if !contains(help, kw) {
			t.Errorf("help should contain %q", kw)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
