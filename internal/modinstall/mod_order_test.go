package modinstall

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
)

func TestInstallIntoDocumentsLeavesModOrderUntouched(t *testing.T) {
	documents := t.TempDir()
	modsRoot := filepath.Join(documents, ModsDirectoryName)
	if err := os.MkdirAll(modsRoot, 0o755); err != nil {
		t.Fatalf("create mods root: %v", err)
	}
	modOrderPath := filepath.Join(modsRoot, "mod_order.txt")
	original := []byte("another_mod\n")
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
		t.Fatalf("mod_order.txt changed: got %q, want %q", got, original)
	}
}
