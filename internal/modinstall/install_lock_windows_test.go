//go:build windows

package modinstall

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestInstallLockRejectsConcurrentOperationAndReacquires(t *testing.T) {
	parent := t.TempDir()
	modsRoot := filepath.Join(parent, ModsDirectoryName)
	if err := os.MkdirAll(modsRoot, 0o755); err != nil {
		t.Fatalf("create mods root: %v", err)
	}

	releaseFirst, err := acquireInstallLock(modsRoot)
	if err != nil {
		t.Fatalf("first acquireInstallLock() error = %v", err)
	}
	if _, err := acquireInstallLock(modsRoot); !errors.Is(err, ErrModOperationInProgress) {
		t.Fatalf("second acquireInstallLock() error = %v, want ErrModOperationInProgress", err)
	}
	releaseFirst()

	lockPath := filepath.Join(parent, ".kcd2-dual-subtitles-operation.lock")
	info, err := os.Stat(lockPath)
	if err != nil || !info.Mode().IsRegular() {
		t.Fatalf("stable lock file missing after release: info=%v err=%v", info, err)
	}

	releaseSecond, err := acquireInstallLock(modsRoot)
	if err != nil {
		t.Fatalf("reacquireInstallLock() error = %v", err)
	}
	releaseSecond()
}
