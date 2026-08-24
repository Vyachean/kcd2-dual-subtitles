//go:build windows

package modinstall

import (
	"errors"
	"syscall"
)

func platformRetryableRenameError(err error) bool {
	return errors.Is(err, syscall.Errno(32)) || errors.Is(err, syscall.Errno(33))
}
