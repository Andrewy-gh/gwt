package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Andrewy-gh/gwt/internal/create"
	"github.com/Andrewy-gh/gwt/internal/output"
	"github.com/Andrewy-gh/gwt/internal/testutil"
)

func TestRunUnlock_RemovesStaleLock(t *testing.T) {
	repoPath := testutil.CreateTestRepo(t)
	lockPath, err := create.GetLockPath(repoPath)
	if err != nil {
		t.Fatalf("GetLockPath returned error: %v", err)
	}

	info := create.LockInfo{
		PID:       -1,
		Command:   "gwt create stale",
		StartTime: time.Unix(1700000000, 0).UTC(),
	}
	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}
	if err := os.WriteFile(lockPath, data, 0600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	oldOut := output.Out
	oldForce := unlockOpts.Force
	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}
	var buf bytes.Buffer
	output.Out = &buf
	unlockOpts.Force = false
	defer func() {
		output.Out = oldOut
		unlockOpts.Force = oldForce
		_ = os.Chdir(oldCwd)
	}()

	if err := os.Chdir(repoPath); err != nil {
		t.Fatalf("Chdir returned error: %v", err)
	}

	if err := runUnlock(nil, nil); err != nil {
		t.Fatalf("runUnlock returned error: %v", err)
	}

	exists, err := create.LockExists(repoPath)
	if err != nil {
		t.Fatalf("LockExists returned error: %v", err)
	}
	if exists {
		t.Fatal("expected stale lock to be removed")
	}

	if !strings.Contains(buf.String(), "Removed gwt operation lock.") {
		t.Fatalf("expected success output, got %q", buf.String())
	}
}

func TestRunUnlock_RejectsActiveLockWithoutForce(t *testing.T) {
	repoPath := testutil.CreateTestRepo(t)
	lock, err := create.AcquireLock(repoPath)
	if err != nil {
		t.Fatalf("AcquireLock returned error: %v", err)
	}
	defer lock.Release()

	oldForce := unlockOpts.Force
	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}
	unlockOpts.Force = false
	defer func() {
		unlockOpts.Force = oldForce
		_ = os.Chdir(oldCwd)
	}()

	if err := os.Chdir(repoPath); err != nil {
		t.Fatalf("Chdir returned error: %v", err)
	}

	err = runUnlock(nil, nil)
	if err == nil {
		t.Fatal("expected active lock to require --force")
	}
	if !strings.Contains(err.Error(), "rerun with --force") {
		t.Fatalf("expected force guidance in error, got %q", err.Error())
	}
}
