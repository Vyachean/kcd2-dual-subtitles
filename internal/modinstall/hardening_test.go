package modinstall

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/modarchive"
)

func TestInstallVersionedWritesMatchingManifestVersion(t *testing.T) {
	documents := t.TempDir()
	installed, err := installIntoDocumentsVersioned(documents, localization.Russian, []localization.DialogueRow{{ID: "id", Text: "text"}}, "v0.1.0-rc.4")
	if err != nil {
		t.Fatalf("installIntoDocumentsVersioned() error = %v", err)
	}
	manifest, err := os.ReadFile(filepath.Join(installed, modarchive.ManifestFilename))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if !strings.Contains(string(manifest), "<version>v0.1.0-rc.4</version>") {
		t.Fatalf("installed manifest version mismatch:\n%s", manifest)
	}
}

func TestInstallRollsBackOwnModWhenLoadOrderPublishFails(t *testing.T) {
	documents := t.TempDir()
	modsRoot := filepath.Join(documents, ModsDirectoryName)
	target := filepath.Join(modsRoot, modarchive.ModID)
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("create previous own mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(target, "old.txt"), []byte("previous"), 0o644); err != nil {
		t.Fatalf("write previous marker: %v", err)
	}
	modOrderPath := filepath.Join(modsRoot, ModOrderFilename)
	originalOrder := []byte("other_mod\n")
	if err := os.WriteFile(modOrderPath, originalOrder, 0o644); err != nil {
		t.Fatalf("write mod_order.txt: %v", err)
	}

	originalRename := renamePath
	defer func() { renamePath = originalRename }()
	injected := false
	renamePath = func(oldPath, newPath string) error {
		if !injected && strings.Contains(filepath.Base(oldPath), ".mod_order.txt.tmp-") && newPath == modOrderPath {
			injected = true
			return errors.New("injected load-order publish failure")
		}
		return os.Rename(oldPath, newPath)
	}

	_, err := installIntoDocuments(documents, localization.Russian, []localization.DialogueRow{{ID: "id", Text: "new"}})
	if err == nil || !strings.Contains(err.Error(), "injected load-order publish failure") {
		t.Fatalf("installIntoDocuments() error = %v, want injected load-order failure", err)
	}
	if got, readErr := os.ReadFile(filepath.Join(target, "old.txt")); readErr != nil || string(got) != "previous" {
		t.Fatalf("previous mod was not restored: got=%q err=%v", got, readErr)
	}
	gotOrder, readErr := os.ReadFile(modOrderPath)
	if readErr != nil {
		t.Fatalf("read restored load order: %v", readErr)
	}
	if string(gotOrder) != string(originalOrder) {
		t.Fatalf("load order was not restored: got %q, want %q", gotOrder, originalOrder)
	}
}
