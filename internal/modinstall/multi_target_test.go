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
	if filepath.Dir(publishedFrom) == filepath.Clean(modsRoot) {
		t.Fatalf("staging directory %q is inside scanned mod root %q", publishedFrom, modsRoot)
	}
	if filepath.Dir(publishedFrom) != filepath.Dir(filepath.Clean(modsRoot)) {
		t.Fatalf("staging parent = %q, want sibling parent %q", filepath.Dir(publishedFrom), filepath.Dir(filepath.Clean(modsRoot)))
	}
	if !strings.HasPrefix(filepath.Base(publishedFrom), "."+modarchive.ModID+".staging-") {
		t.Fatalf("staging directory = %q, want tool-owned temp prefix", publishedFrom)
	}
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
