package create

import (
	"strings"
	"testing"

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
