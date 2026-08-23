//go:build !windows

package modinstall

import (
	"errors"
	"testing"
)

func TestDocumentsPathRejectsAutomaticInstallOutsideWindows(t *testing.T) {
	path, err := documentsPath()
	if path != "" {
		t.Fatalf("documentsPath() path = %q, want empty", path)
	}
	if !errors.Is(err, ErrAutomaticInstallUnsupported) {
		t.Fatalf("documentsPath() error = %v, want ErrAutomaticInstallUnsupported", err)
	}
}
