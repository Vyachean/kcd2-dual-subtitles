package modinstall

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateCustomModsRootAcceptsExistingDirectory(t *testing.T) {
	root := filepath.Join(t.TempDir(), "Mods")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	location, err := ValidateCustomModsRoot(root)
	if err != nil {
		t.Fatalf("ValidateCustomModsRoot() error = %v", err)
	}
	if location.Layout != InstallLayoutCustom || location.ModsRoot != root {
		t.Fatalf("location = %+v, want custom %q", location, root)
	}
}

func TestValidateCustomModsRootRejectsMissingPath(t *testing.T) {
	root := filepath.Join(t.TempDir(), "missing")
	if _, err := ValidateCustomModsRoot(root); err == nil {
		t.Fatal("ValidateCustomModsRoot() error = nil, want missing-directory error")
	}
}

func TestValidateCustomModsRootRejectsFile(t *testing.T) {
	root := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(root, []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateCustomModsRoot(root); err == nil {
		t.Fatal("ValidateCustomModsRoot() error = nil, want non-directory error")
	}
}

func TestValidateCustomModsRootRejectsSymlink(t *testing.T) {
	target := filepath.Join(t.TempDir(), "real-mods")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "linked-mods")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := ValidateCustomModsRoot(link); err == nil {
		t.Fatal("ValidateCustomModsRoot() error = nil, want symlink error")
	}
}
