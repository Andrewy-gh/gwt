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
