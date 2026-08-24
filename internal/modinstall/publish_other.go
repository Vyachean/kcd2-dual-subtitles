//go:build !windows

package modinstall

func platformRetryableRenameError(error) bool {
	return false
}
