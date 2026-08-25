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

func TestInstallIntoDocumentsVersionedForLanguagesWritesEveryTarget(t *testing.T) {
	documents := t.TempDir()
	rows := []localization.DialogueRow{{ID: "line", Text: "bilingual"}}
	targets := []localization.Language{localization.English, localization.Czech, localization.German}

	installed, err := installIntoDocumentsVersionedForLanguages(documents, targets, rows, nil, "v0.3.0-test", false)
	if err != nil {
		t.Fatalf("installIntoDocumentsVersionedForLanguages() error = %v", err)
	}
	for _, pak := range []string{"English_xml.pak", "Czech_xml.pak", "German_xml.pak"} {
		path := filepath.Join(installed, "Localization", pak)
		if info, err := os.Stat(path); err != nil || info.IsDir() {
			t.Fatalf("localization target %q missing: info=%v err=%v", path, info, err)
		}
	}
}

func TestInstallStagesOutsideScannedModRoot(t *testing.T) {
	originalRename := renamePath
	defer func() { renamePath = originalRename }()

	parent := t.TempDir()
	modsRoot := filepath.Join(parent, ModsDirectoryName)
	var publishedFrom string
	renamePath = func(oldPath, newPath string) error {
		if filepath.Clean(newPath) == filepath.Join(modsRoot, modarchive.ModID) {
			publishedFrom = oldPath
		}
		return os.Rename(oldPath, newPath)
	}

	rows := []localization.DialogueRow{{ID: "line", Text: "bilingual"}}
	targets := []localization.Language{localization.English, localization.Czech}
	if _, err := installIntoModsRootVersionedForLanguages(modsRoot, targets, rows, nil, "v-test", false); err != nil {
		t.Fatalf("installIntoModsRootVersionedForLanguages() error = %v", err)
	}
	if publishedFrom == "" {
		t.Fatal("did not observe staged publication rename")
	}
	if pathIsWithin(modsRoot, publishedFrom) {
		t.Fatalf("staging directory %q is inside scanned mod root %q", publishedFrom, modsRoot)
	}
	transactionRoot := filepath.Dir(publishedFrom)
	if filepath.Dir(transactionRoot) != filepath.Dir(filepath.Clean(modsRoot)) {
		t.Fatalf("transaction parent = %q, want sibling parent %q", filepath.Dir(transactionRoot), filepath.Dir(filepath.Clean(modsRoot)))
	}
	if !strings.HasPrefix(filepath.Base(transactionRoot), installTransactionPrefix) {
		t.Fatalf("transaction directory = %q, want prefix %q", transactionRoot, installTransactionPrefix)
	}
	if filepath.Base(publishedFrom) != transactionStagedDirname {
		t.Fatalf("published source = %q, want transaction staged directory", publishedFrom)
	}
	assertNoInstallTransactions(t, parent)
}

func TestInstallRemovesLegacyScannedStagingDirectory(t *testing.T) {
	parent := t.TempDir()
	modsRoot := filepath.Join(parent, ModsDirectoryName)
	if err := os.MkdirAll(modsRoot, 0o755); err != nil {
		t.Fatalf("create mod root: %v", err)
	}
	legacy := filepath.Join(modsRoot, "."+modarchive.ModID+".staging-3747685633")
	if err := os.MkdirAll(filepath.Join(legacy, "Localization"), 0o755); err != nil {
		t.Fatalf("create leaked staging directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(legacy, modarchive.ManifestFilename), []byte("stale duplicate mod"), 0o644); err != nil {
		t.Fatalf("write leaked staging manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "Localization", "English_xml.pak"), []byte("stale localization"), 0o644); err != nil {
		t.Fatalf("write leaked staging localization: %v", err)
	}

	rows := []localization.DialogueRow{{ID: "line", Text: "bilingual"}}
	targets := []localization.Language{localization.English, localization.Czech}
	if _, err := installIntoModsRootVersionedForLanguages(modsRoot, targets, rows, nil, "v-test", false); err != nil {
		t.Fatalf("installIntoModsRootVersionedForLanguages() error = %v", err)
	}
	if _, err := os.Stat(legacy); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("legacy scanned staging directory survived install: %v", err)
	}
	entries, err := os.ReadDir(modsRoot)
	if err != nil {
		t.Fatalf("read mod root: %v", err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "."+modarchive.ModID+".staging-") {
			t.Fatalf("tool staging directory remains visible to KCD2 scan: %q", entry.Name())
		}
	}
}

func TestInstallKeepsRecoverableTransactionWhenRollbackIsBlocked(t *testing.T) {
	originalRename := renamePath
	originalSleep := sleepRenameRetry
	defer func() {
		renamePath = originalRename
		sleepRenameRetry = originalSleep
	}()
	sleepRenameRetry = func(time.Duration) {}

	parent := t.TempDir()
	modsRoot := filepath.Join(parent, ModsDirectoryName)
	target := filepath.Join(modsRoot, modarchive.ModID)
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("create previous install: %v", err)
	}
	if err := os.WriteFile(filepath.Join(target, "previous.txt"), []byte("previous"), 0o644); err != nil {
		t.Fatalf("write previous sentinel: %v", err)
	}

	var transactionRoot string
	renamePath = func(oldPath, newPath string) error {
		if filepath.Base(oldPath) == transactionStagedDirname && newPath == target {
			transactionRoot = filepath.Dir(oldPath)
			return errors.New("injected publication failure")
		}
		if filepath.Base(oldPath) == transactionPreviousName && newPath == target {
			transactionRoot = filepath.Dir(oldPath)
			return os.ErrPermission
		}
		return os.Rename(oldPath, newPath)
	}

	rows := []localization.DialogueRow{{ID: "line", Text: "replacement"}}
	targets := []localization.Language{localization.English, localization.Czech}
	_, err := installIntoModsRootVersionedForLanguages(modsRoot, targets, rows, nil, "v-test", false)
	if err == nil || !strings.Contains(err.Error(), "injected publication failure") {
		t.Fatalf("install error = %v, want publication failure", err)
	}
	if transactionRoot == "" {
		t.Fatal("did not capture install transaction root")
	}
	if info, statErr := os.Stat(transactionRoot); statErr != nil || !info.IsDir() {
		t.Fatalf("recoverable transaction was discarded: info=%v err=%v", info, statErr)
	}
	previous := filepath.Join(transactionRoot, transactionPreviousName)
	if got, readErr := os.ReadFile(filepath.Join(previous, "previous.txt")); readErr != nil || string(got) != "previous" {
		t.Fatalf("previous install was discarded: data=%q err=%v", got, readErr)
	}

	renamePath = originalRename
	if err := recoverInstallTransactions(modsRoot); err != nil {
		t.Fatalf("recoverInstallTransactions() error = %v", err)
	}
	if got, readErr := os.ReadFile(filepath.Join(target, "previous.txt")); readErr != nil || string(got) != "previous" {
		t.Fatalf("recovery did not restore previous install: data=%q err=%v", got, readErr)
	}
	if _, statErr := os.Stat(transactionRoot); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("transaction survived successful recovery: %v", statErr)
	}
}

func pathIsWithin(root, candidate string) bool {
	relative, err := filepath.Rel(filepath.Clean(root), filepath.Clean(candidate))
	if err != nil {
		return false
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func assertNoInstallTransactions(t *testing.T, parent string) {
	t.Helper()
	entries, err := os.ReadDir(parent)
	if err != nil {
		t.Fatalf("read transaction parent: %v", err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), installTransactionPrefix) {
			t.Fatalf("install transaction survived successful publication: %q", entry.Name())
		}
	}
}
