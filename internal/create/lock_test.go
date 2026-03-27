package create

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Andrewy-gh/gwt/internal/testutil"
)

func TestLockExistsLifecycle(t *testing.T) {
	repoPath := testutil.CreateTestRepo(t)

	exists, err := LockExists(repoPath)
	if err != nil {
		t.Fatalf("LockExists returned error: %v", err)
	}
	if exists {
		t.Fatal("expected no lock before acquisition")
	}

	lock, err := AcquireLock(repoPath)
	if err != nil {
		t.Fatalf("AcquireLock returned error: %v", err)
	}

	exists, err = LockExists(repoPath)
	if err != nil {
		t.Fatalf("LockExists returned error after acquire: %v", err)
	}
	if !exists {
		t.Fatal("expected lock after acquisition")
	}

	if err := lock.Release(); err != nil {
		t.Fatalf("Release returned error: %v", err)
	}

	exists, err = LockExists(repoPath)
	if err != nil {
		t.Fatalf("LockExists returned error after release: %v", err)
	}
	if exists {
		t.Fatal("expected no lock after release")
	}
}

func TestAcquireLockConflictIncludesRecoveryHint(t *testing.T) {
	repoPath := testutil.CreateTestRepo(t)

	lock, err := AcquireLock(repoPath)
	if err != nil {
		t.Fatalf("AcquireLock returned error: %v", err)
	}
	defer lock.Release()

	_, err = AcquireLock(repoPath)
	if err == nil {
		t.Fatal("expected second AcquireLock call to fail")
	}

	if !strings.Contains(err.Error(), "gwt unlock") {
		t.Fatalf("expected recovery hint in error, got %q", err.Error())
	}
}

func TestIsLockedReportsActiveLock(t *testing.T) {
	repoPath := testutil.CreateTestRepo(t)

	lock, err := AcquireLock(repoPath)
	if err != nil {
		t.Fatalf("AcquireLock returned error: %v", err)
	}
	defer lock.Release()

	locked, err := IsLocked(repoPath)
	if err != nil {
		t.Fatalf("IsLocked returned error: %v", err)
	}
	if !locked {
		t.Fatal("expected acquired lock to be reported as active")
	}
}

func TestIsLockedReportsStaleLock(t *testing.T) {
	repoPath := testutil.CreateTestRepo(t)

	lockPath, err := GetLockPath(repoPath)
	if err != nil {
		t.Fatalf("GetLockPath returned error: %v", err)
	}

	info := LockInfo{
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

	locked, err := IsLocked(repoPath)
	if err != nil {
		t.Fatalf("IsLocked returned error: %v", err)
	}
	if locked {
		t.Fatal("expected stale lock to be reported as inactive")
	}
}
