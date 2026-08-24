package modinstall

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
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
