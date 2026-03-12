package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Andrewy-gh/gwt/internal/create"
	"github.com/Andrewy-gh/gwt/internal/output"
)

func TestPrintSuccessMessage_IncludesLocalWorktreeState(t *testing.T) {
	var buf bytes.Buffer

	oldOut := output.Out
	output.Out = &buf
	defer func() { output.Out = oldOut }()

	result := &create.CreateWorktreeResult{
		Path:    "/tmp/worktree-feature-auth",
		Branch:  "feature-auth",
		Commit:  "abc1234",
		IsNew:   true,
		FromRef: "main",
	}

	printSuccessMessage(result)

	out := buf.String()

	if !strings.Contains(out, "Created worktree successfully!") {
		t.Fatal("expected success banner in output")
	}
	if !strings.Contains(out, "Worktree is ready locally on branch feature-auth at commit abc1234") {
		t.Fatal("expected local worktree state message in output")
	}
	if !strings.Contains(out, "  Branch: feature-auth") {
		t.Fatal("expected branch summary in output")
	}
	if !strings.Contains(out, "  Commit: abc1234") {
		t.Fatal("expected commit summary in output")
	}
}

func TestPrintSuccessMessage_OmitsCommitClauseWhenCommitUnknown(t *testing.T) {
	var buf bytes.Buffer

	oldOut := output.Out
	output.Out = &buf
	defer func() { output.Out = oldOut }()

	result := &create.CreateWorktreeResult{
		Path:   "/tmp/worktree-feature-auth",
		Branch: "feature-auth",
	}

	printSuccessMessage(result)

	out := buf.String()

	if !strings.Contains(out, "Worktree is ready locally on branch feature-auth") {
		t.Fatal("expected local worktree state message without commit")
	}
	if strings.Contains(out, "at commit") {
		t.Fatal("did not expect commit clause when commit is unknown")
	}
}

func TestNormalizeCreateOptions_UsesPositionalBranch(t *testing.T) {
	opts, err := normalizeCreateOptions(CreateOptions{}, []string{"feature-auth"})
	if err != nil {
		t.Fatalf("normalizeCreateOptions returned error: %v", err)
	}

	if opts.Branch != "feature-auth" {
		t.Fatalf("expected positional branch to be used, got %q", opts.Branch)
	}
}

func TestNormalizeCreateOptions_RejectsMixedBranchSources(t *testing.T) {
	_, err := normalizeCreateOptions(CreateOptions{Checkout: "existing"}, []string{"feature-auth"})
	if err == nil {
		t.Fatal("expected error when positional branch is mixed with checkout flag")
	}
}

func TestNormalizeDockerCopyExclude(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "relative path", input: "./docker-data", expected: "docker-data"},
		{name: "nested path", input: "var/lib/postgres", expected: "var/lib/postgres"},
		{name: "env var", input: "${DATA_DIR}", expected: ""},
		{name: "parent escape", input: "../shared-data", expected: ""},
		{name: "empty", input: "", expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeDockerCopyExclude(tt.input); got != tt.expected {
				t.Fatalf("normalizeDockerCopyExclude(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
