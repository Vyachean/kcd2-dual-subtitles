//go:build !windows

package modinstall

func documentsPath() (string, error) {
	return "", ErrAutomaticInstallUnsupported
}
