package modinstall

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/modarchive"
)

func TestRenamePathWithRetryRetriesPermissionFailures(t *testing.T) {
	originalRename := renamePath
	originalSleep := sleepRenameRetry
	defer func() {
		renamePath = originalRename
		sleepRenameRetry = originalSleep
	}()

	attempts := 0
	renamePath = func(string, string) error {
		attempts++
		if attempts < 3 {
			return os.ErrPermission
		}
		return nil
	}
	sleepRenameRetry = func(time.Duration) {}

	if err := renamePathWithRetry("old", "new"); err != nil {
		t.Fatalf("renamePathWithRetry() error = %v", err)
	}
	if attempts != 3 {
		t.Fatalf("rename attempts = %d, want 3", attempts)
	}
}

func TestInstallFallsBackToCopyWhenStagingRenameStaysDenied(t *testing.T) {
	documents := t.TempDir()
	modsRoot := filepath.Join(documents, ModsDirectoryName)
	target := filepath.Join(modsRoot, modarchive.ModID)
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("create previous mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(target, "old.txt"), []byte("previous"), 0o644); err != nil {
		t.Fatalf("write previous marker: %v", err)
	}

	originalRename := renamePath
	originalSleep := sleepRenameRetry
	defer func() {
		renamePath = originalRename
		sleepRenameRetry = originalSleep
	}()
	sleepRenameRetry = func(time.Duration) {}
	renamePath = func(oldPath, newPath string) error {
		if newPath == target && isTransactionStagedSource(oldPath) {
			return os.ErrPermission
		}
		return os.Rename(oldPath, newPath)
	}

	installed, err := installIntoDocumentsVersionedForLanguages(
		documents,
		[]localization.Language{localization.English, localization.Czech, localization.German},
		[]localization.DialogueRow{{ID: "id", Text: "bilingual"}},
		nil,
		"v0.3.0-test",
		false,
	)
	if err != nil {
		t.Fatalf("installIntoDocumentsVersionedForLanguages() error = %v", err)
	}
	if installed != target {
		t.Fatalf("installed path = %q, want %q", installed, target)
	}
	if _, err := os.Stat(filepath.Join(target, "old.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("previous content survived copy fallback: %v", err)
	}
	for _, pak := range []string{"English_xml.pak", "Czech_xml.pak", "German_xml.pak"} {
		if info, err := os.Stat(filepath.Join(target, "Localization", pak)); err != nil || info.IsDir() {
			t.Fatalf("copied localization PAK %q missing: info=%v err=%v", pak, info, err)
		}
	}
	assertNoInstallResidue(t, modsRoot)
}

func TestInstallCopyFallbackFailureRestoresPreviousMod(t *testing.T) {
	documents := t.TempDir()
	modsRoot := filepath.Join(documents, ModsDirectoryName)
	target := filepath.Join(modsRoot, modarchive.ModID)
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("create previous mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(target, "old.txt"), []byte("previous"), 0o644); err != nil {
		t.Fatalf("write previous marker: %v", err)
	}

	originalRename := renamePath
	originalSleep := sleepRenameRetry
	originalCopy := copyPublishPath
	defer func() {
		renamePath = originalRename
		sleepRenameRetry = originalSleep
		copyPublishPath = originalCopy
	}()
	sleepRenameRetry = func(time.Duration) {}
	renamePath = func(oldPath, newPath string) error {
		if newPath == target && isTransactionStagedSource(oldPath) {
			return os.ErrPermission
		}
		return os.Rename(oldPath, newPath)
	}
	copyPublishPath = func(string, string) error {
		return errors.New("injected copy publish failure")
	}

	_, err := installIntoDocumentsVersionedForLanguages(
		documents,
		[]localization.Language{localization.English, localization.Czech},
		[]localization.DialogueRow{{ID: "id", Text: "new"}},
		nil,
		"v0.3.0-test",
		false,
	)
	if err == nil || !strings.Contains(err.Error(), "injected copy publish failure") {
		t.Fatalf("install error = %v, want injected copy fallback failure", err)
	}
	if got, readErr := os.ReadFile(filepath.Join(target, "old.txt")); readErr != nil || string(got) != "previous" {
		t.Fatalf("previous mod was not restored: got=%q err=%v", got, readErr)
	}
	assertNoInstallResidue(t, modsRoot)
}

func isTransactionStagedSource(path string) bool {
	return filepath.Base(path) == transactionStagedDirname && strings.HasPrefix(filepath.Base(filepath.Dir(path)), installTransactionPrefix)
}

func assertNoInstallResidue(t *testing.T, modsRoot string) {
	t.Helper()
	entries, err := os.ReadDir(modsRoot)
	if err != nil {
		t.Fatalf("read mods root: %v", err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".staging-") || strings.HasSuffix(entry.Name(), ".previous") {
			t.Fatalf("installer left legacy staging/backup residue in mod root: %q", entry.Name())
		}
	}
	parentEntries, err := os.ReadDir(filepath.Dir(filepath.Clean(modsRoot)))
	if err != nil {
		t.Fatalf("read transaction parent: %v", err)
	}
	for _, entry := range parentEntries {
		if strings.HasPrefix(entry.Name(), installTransactionPrefix) {
			t.Fatalf("installer left transaction residue: %q", entry.Name())
		}
	}
}
