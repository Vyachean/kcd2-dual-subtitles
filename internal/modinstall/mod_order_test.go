package modinstall

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/modarchive"
)

func TestInstallIntoDocumentsDoesNotCreateModOrder(t *testing.T) {
	documents := t.TempDir()
	if _, err := installIntoDocuments(documents, localization.Russian, []localization.DialogueRow{{ID: "id", Text: "text"}}); err != nil {
		t.Fatalf("installIntoDocuments() error = %v", err)
	}
	path := filepath.Join(documents, ModsDirectoryName, ModOrderFilename)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("mod_order.txt exists after install without pre-existing load order: %v", err)
	}
}

func TestInstallIntoDocumentsAppendsMissingModOrderEntry(t *testing.T) {
	documents := t.TempDir()
	modsRoot := filepath.Join(documents, ModsDirectoryName)
	if err := os.MkdirAll(modsRoot, 0o755); err != nil {
		t.Fatalf("create mods root: %v", err)
	}
	modOrderPath := filepath.Join(modsRoot, ModOrderFilename)
	original := []byte("first_mod\r\nsecond_mod\r\n")
	if err := os.WriteFile(modOrderPath, original, 0o644); err != nil {
		t.Fatalf("write mod_order.txt: %v", err)
	}

	if _, err := installIntoDocuments(documents, localization.Russian, []localization.DialogueRow{{ID: "id", Text: "text"}}); err != nil {
		t.Fatalf("installIntoDocuments() error = %v", err)
	}
	got, err := os.ReadFile(modOrderPath)
	if err != nil {
		t.Fatalf("read mod_order.txt: %v", err)
	}
	want := string(original) + modarchive.ModID + "\r\n"
	if string(got) != want {
		t.Fatalf("mod_order.txt = %q, want %q", got, want)
	}
}

func TestInstallIntoDocumentsLeavesExistingModOrderEntryByteStable(t *testing.T) {
	documents := t.TempDir()
	modsRoot := filepath.Join(documents, ModsDirectoryName)
	if err := os.MkdirAll(modsRoot, 0o755); err != nil {
		t.Fatalf("create mods root: %v", err)
	}
	modOrderPath := filepath.Join(modsRoot, ModOrderFilename)
	original := []byte("first_mod\n  " + modarchive.ModID + "  \nlast_mod")
	if err := os.WriteFile(modOrderPath, original, 0o644); err != nil {
		t.Fatalf("write mod_order.txt: %v", err)
	}

	if _, err := installIntoDocuments(documents, localization.Russian, []localization.DialogueRow{{ID: "id", Text: "text"}}); err != nil {
		t.Fatalf("installIntoDocuments() error = %v", err)
	}
	got, err := os.ReadFile(modOrderPath)
	if err != nil {
		t.Fatalf("read mod_order.txt: %v", err)
	}
	if string(got) != string(original) {
		t.Fatalf("existing mod_order entry was rewritten: got %q, want byte-stable %q", got, original)
	}
}

func TestEnsureModOrderContainsPreservesExistingLinesWithoutTrailingNewline(t *testing.T) {
	modsRoot := t.TempDir()
	path := filepath.Join(modsRoot, ModOrderFilename)
	if err := os.WriteFile(path, []byte("first_mod\nsecond_mod"), 0o600); err != nil {
		t.Fatalf("write load order: %v", err)
	}

	if err := ensureModOrderContains(modsRoot, modarchive.ModID); err != nil {
		t.Fatalf("ensureModOrderContains() error = %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read load order: %v", err)
	}
	want := "first_mod\nsecond_mod\n" + modarchive.ModID + "\n"
	if string(got) != want {
		t.Fatalf("load order = %q, want %q", got, want)
	}
}
