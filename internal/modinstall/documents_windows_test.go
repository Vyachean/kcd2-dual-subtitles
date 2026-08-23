//go:build windows

package modinstall

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestDocumentsPathResolvesCurrentWindowsKnownFolder(t *testing.T) {
	path, err := documentsPath()
	if err != nil {
		t.Fatalf("documentsPath() error = %v", err)
	}
	if strings.TrimSpace(path) == "" {
		t.Fatal("documentsPath() returned empty path")
	}
	if !filepath.IsAbs(path) {
		t.Fatalf("documentsPath() = %q, want absolute path", path)
	}
}
