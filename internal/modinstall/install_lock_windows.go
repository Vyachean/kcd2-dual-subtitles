//go:build windows

package modinstall

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

var (
	installLockKernel32 = syscall.NewLazyDLL("kernel32.dll")
	procLockFile        = installLockKernel32.NewProc("LockFile")
	procUnlockFile      = installLockKernel32.NewProc("UnlockFile")
)

func acquireInstallLock(modsRoot string) (func(), error) {
	lockPath := filepath.Join(filepath.Dir(filepath.Clean(modsRoot)), ".kcd2-dual-subtitles-install.lock")
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open install lock %q: %w", lockPath, err)
	}

	locked, _, lockErr := procLockFile.Call(file.Fd(), 0, 0, 1, 0)
	if locked == 0 {
		_ = file.Close()
		return nil, fmt.Errorf("%w: %v", ErrInstallInProgress, lockErr)
	}

	release := func() {
		_, _, _ = procUnlockFile.Call(file.Fd(), 0, 0, 1, 0)
		_ = file.Close()
		_ = os.Remove(lockPath)
	}
	return release, nil
}
