package tui

import (
	"strings"
	"testing"

	"github.com/Andrewy-gh/gwt/internal/create"
	"github.com/Andrewy-gh/gwt/internal/testutil"
)

func TestCreateWorktreeCmd_FailsWhenOperationLockIsHeld(t *testing.T) {
	repoPath := testutil.CreateTestRepo(t)

	lock, err := create.AcquireLock(repoPath)
	if err != nil {
		t.Fatalf("AcquireLock returned error: %v", err)
	}
	defer lock.Release()

	state := NewCreateFlowState()
	state.BranchSpec = &create.BranchSpec{
		BranchName: "feature-lock-test",
		Source:     create.BranchSourceNewFromHEAD,
	}
	state.SourceType = create.BranchSourceNewFromHEAD
	state.TargetDir = repoPath + "-feature-lock-test"

	msg := createWorktreeCmd(state, repoPath)()
	complete, ok := msg.(CreateCompleteMsg)
	if !ok {
		t.Fatalf("expected CreateCompleteMsg, got %T", msg)
	}
	if complete.Error == nil {
		t.Fatal("expected lock acquisition error")
	}
	if !strings.Contains(complete.Error.Error(), "failed to acquire lock") {
		t.Fatalf("expected lock acquisition context, got %q", complete.Error.Error())
	}
}
