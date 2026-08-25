package modarchive

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
)

func TestVerifyDirectoryVersionedWithHUDForLanguagesAcceptsExactArtifact(t *testing.T) {
	directory := t.TempDir()
	rows := []localization.DialogueRow{{ID: "line", Text: "styled"}}
	targets := []localization.Language{localization.English, localization.Czech}
	hud := []byte("derived-hud")

	if err := WriteDirectoryVersionedWithHUDForLanguages(directory, targets, rows, hud, "v-test"); err != nil {
		t.Fatalf("WriteDirectoryVersionedWithHUDForLanguages() error = %v", err)
	}
	if err := VerifyDirectoryVersionedWithHUDForLanguages(directory, targets, rows, hud, "v-test"); err != nil {
		t.Fatalf("VerifyDirectoryVersionedWithHUDForLanguages() error = %v", err)
	}
}

func TestVerifyDirectoryVersionedWithHUDForLanguagesRejectsMissingHUDDataPAK(t *testing.T) {
	directory := t.TempDir()
	rows := []localization.DialogueRow{{ID: "line", Text: "styled"}}
	targets := []localization.Language{localization.English, localization.Czech}
	hud := []byte("derived-hud")

	if err := WriteDirectoryVersionedWithHUDForLanguages(directory, targets, rows, hud, "v-test"); err != nil {
		t.Fatalf("WriteDirectoryVersionedWithHUDForLanguages() error = %v", err)
	}
	if err := os.Remove(filepath.Join(directory, "Data", DataPAKFilename)); err != nil {
		t.Fatalf("remove HUD data PAK: %v", err)
	}

	err := VerifyDirectoryVersionedWithHUDForLanguages(directory, targets, rows, hud, "v-test")
	if !errors.Is(err, ErrArtifactVerification) {
		t.Fatalf("verification error = %v, want ErrArtifactVerification", err)
	}
}

func TestVerifyDirectoryVersionedForLanguagesRejectsCorruptLocalizationPAK(t *testing.T) {
	directory := t.TempDir()
	rows := []localization.DialogueRow{{ID: "line", Text: "bilingual"}}
	targets := []localization.Language{localization.English, localization.Czech}

	if err := WriteDirectoryVersionedForLanguages(directory, targets, rows, "v-test"); err != nil {
		t.Fatalf("WriteDirectoryVersionedForLanguages() error = %v", err)
	}
	path := filepath.Join(directory, "Localization", "Czech_xml.pak")
	if err := os.WriteFile(path, []byte("corrupt"), 0o644); err != nil {
		t.Fatalf("corrupt localization PAK: %v", err)
	}

	err := VerifyDirectoryVersionedForLanguages(directory, targets, rows, "v-test")
	if !errors.Is(err, ErrArtifactVerification) {
		t.Fatalf("verification error = %v, want ErrArtifactVerification", err)
	}
}

func TestVerifyDirectoryVersionedForLanguagesRejectsUnexpectedHUDOverride(t *testing.T) {
	directory := t.TempDir()
	rows := []localization.DialogueRow{{ID: "line", Text: "bilingual"}}
	targets := []localization.Language{localization.English, localization.Czech}

	if err := WriteDirectoryVersionedForLanguages(directory, targets, rows, "v-test"); err != nil {
		t.Fatalf("WriteDirectoryVersionedForLanguages() error = %v", err)
	}
	if err := os.Mkdir(filepath.Join(directory, "Data"), 0o755); err != nil {
		t.Fatalf("create unexpected Data directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(directory, "Data", DataPAKFilename), []byte("unexpected"), 0o644); err != nil {
		t.Fatalf("write unexpected HUD data PAK: %v", err)
	}

	err := VerifyDirectoryVersionedForLanguages(directory, targets, rows, "v-test")
	if !errors.Is(err, ErrArtifactVerification) {
		t.Fatalf("verification error = %v, want ErrArtifactVerification", err)
	}
}
