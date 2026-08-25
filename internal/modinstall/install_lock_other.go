//go:build !windows

package modinstall

func acquireInstallLock(string) (func(), error) {
	return func() {}, nil
}
