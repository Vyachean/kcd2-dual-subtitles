package modinstall

import (
	"archive/zip"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/modarchive"
)

func TestInstallIntoDocumentsCreatesExpectedTree(t *testing.T) {
	documents := t.TempDir()
	rows := []localization.DialogueRow{{ID: "id", Source: "source", Text: "Русский\\nEnglish"}}

	installed, err := installIntoDocuments(documents, localization.Russian, rows)
	if err != nil {
		t.Fatalf("installIntoDocuments() error = %v", err)
	}

	want := filepath.Join(documents, ModsDirectoryName, modarchive.ModID)
	if installed != want {
		t.Fatalf("installed path = %q, want %q", installed, want)
	}
	if info, err := os.Stat(filepath.Join(installed, modarchive.ManifestFilename)); err != nil || info.IsDir() {
		t.Fatalf("manifest missing or invalid: info=%v err=%v", info, err)
	}

	pakPath := filepath.Join(installed, "Localization", "Russian_xml.pak")
	pak, err := zip.OpenReader(pakPath)
	if err != nil {
		t.Fatalf("open installed localization PAK: %v", err)
	}
	defer pak.Close()
	if len(pak.File) != 1 || pak.File[0].Name != modarchive.LocalizationPatchArchivePath {
		t.Fatalf("installed PAK entries = %#v, want only %q", pak.File, modarchive.LocalizationPatchArchivePath)
	}
}

func TestInstallIntoDocumentsReplacesOnlyOwnMod(t *testing.T) {
	documents := t.TempDir()
	modsRoot := filepath.Join(documents, ModsDirectoryName)
	target := filepath.Join(modsRoot, modarchive.ModID)
	otherMod := filepath.Join(modsRoot, "other_mod")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("create previous own mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(target, "old.txt"), []byte("old"), 0o644); err != nil {
		t.Fatalf("write previous marker: %v", err)
	}
	if err := os.Mkdir(otherMod, 0o755); err != nil {
		t.Fatalf("create sibling mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(otherMod, "keep.txt"), []byte("keep"), 0o644); err != nil {
		t.Fatalf("write sibling marker: %v", err)
	}

	_, err := installIntoDocuments(documents, localization.Russian, []localization.DialogueRow{{ID: "id", Text: "new"}})
	if err != nil {
		t.Fatalf("installIntoDocuments() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "old.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("old own-mod content survived replacement: %v", err)
	}
	if got, err := os.ReadFile(filepath.Join(otherMod, "keep.txt")); err != nil || string(got) != "keep" {
		t.Fatalf("sibling mod changed: got=%q err=%v", got, err)
	}

	entries, err := os.ReadDir(modsRoot)
	if err != nil {
		t.Fatalf("read mods root: %v", err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".staging-") || strings.HasSuffix(entry.Name(), ".previous") {
			t.Fatalf("installer left staging/backup residue: %q", entry.Name())
		}
	}
}

func TestInstallIntoDocumentsRollsBackPreviousModWhenPublishFails(t *testing.T) {
	documents := t.TempDir()
	modsRoot := filepath.Join(documents, ModsDirectoryName)
	target := filepath.Join(modsRoot, modarchive.ModID)
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("create previous own mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(target, "old.txt"), []byte("previous"), 0o644); err != nil {
		t.Fatalf("write previous marker: %v", err)
	}

	originalRename := renamePath
	defer func() { renamePath = originalRename }()
	publishFailed := false
	renamePath = func(oldPath, newPath string) error {
		if newPath == target && !publishFailed {
			publishFailed = true
			return errors.New("injected publish failure")
		}
		return os.Rename(oldPath, newPath)
	}

	_, err := installIntoDocuments(documents, localization.Russian, []localization.DialogueRow{{ID: "id", Text: "new"}})
	if err == nil || !strings.Contains(err.Error(), "injected publish failure") {
		t.Fatalf("installIntoDocuments() error = %v, want injected failure", err)
	}
	if got, readErr := os.ReadFile(filepath.Join(target, "old.txt")); readErr != nil || string(got) != "previous" {
		t.Fatalf("previous mod was not restored: got=%q err=%v", got, readErr)
	}

	entries, readErr := os.ReadDir(modsRoot)
	if readErr != nil {
		t.Fatalf("read mods root: %v", readErr)
	}
	if len(entries) != 1 || entries[0].Name() != modarchive.ModID {
		t.Fatalf("rollback left unexpected entries: %#v", entries)
	}
}

func TestInstallIntoDocumentsRefusesSymlinkAtOwnModPath(t *testing.T) {
	documents := t.TempDir()
	modsRoot := filepath.Join(documents, ModsDirectoryName)
	if err := os.MkdirAll(modsRoot, 0o755); err != nil {
		t.Fatalf("create mods root: %v", err)
	}
	outside := t.TempDir()
	target := filepath.Join(modsRoot, modarchive.ModID)
	if err := os.Symlink(outside, target); err != nil {
		t.Skipf("symlinks unavailable in test environment: %v", err)
	}

	_, err := installIntoDocuments(documents, localization.Russian, []localization.DialogueRow{{ID: "id", Text: "new"}})
	if err == nil || !strings.Contains(err.Error(), "refusing to replace symlink") {
		t.Fatalf("installIntoDocuments() error = %v, want symlink refusal", err)
	}
}
