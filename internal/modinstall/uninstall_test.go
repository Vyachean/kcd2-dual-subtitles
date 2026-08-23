package modinstall

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/modarchive"
)

func TestUninstallFromDocumentsRemovesOnlyOwnModAndOrderLines(t *testing.T) {
	documents := t.TempDir()
	modsRoot := filepath.Join(documents, ModsDirectoryName)
	ownMod := filepath.Join(modsRoot, modarchive.ModID)
	otherMod := filepath.Join(modsRoot, "other_mod")
	if err := os.MkdirAll(ownMod, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(otherMod, 0o755); err != nil {
		t.Fatal(err)
	}
	order := []byte("first_mod\r\n" + modarchive.ModID + "\r\nother_mod\r\n" + modarchive.ModID + "\r\n")
	orderPath := filepath.Join(modsRoot, ModOrderFilename)
	if err := os.WriteFile(orderPath, order, 0o640); err != nil {
		t.Fatal(err)
	}

	result, err := uninstallFromDocuments(documents)
	if err != nil {
		t.Fatalf("uninstallFromDocuments() error = %v", err)
	}
	if !result.RemovedMod || !result.UpdatedModOrder || result.Path != ownMod {
		t.Fatalf("result = %+v", result)
	}
	if _, err := os.Stat(ownMod); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("own mod still exists or unexpected error: %v", err)
	}
	if info, err := os.Stat(otherMod); err != nil || !info.IsDir() {
		t.Fatalf("other mod changed: info=%v err=%v", info, err)
	}
	gotOrder, err := os.ReadFile(orderPath)
	if err != nil {
		t.Fatal(err)
	}
	wantOrder := "first_mod\r\nother_mod\r\n"
	if string(gotOrder) != wantOrder {
		t.Fatalf("mod_order = %q, want %q", gotOrder, wantOrder)
	}
	info, err := os.Stat(orderPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("mod_order permissions = %o, want 640", info.Mode().Perm())
	}
}

func TestUninstallFromDocumentsLeavesAbsentOrderAbsent(t *testing.T) {
	documents := t.TempDir()
	ownMod := filepath.Join(documents, ModsDirectoryName, modarchive.ModID)
	if err := os.MkdirAll(ownMod, 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := uninstallFromDocuments(documents)
	if err != nil {
		t.Fatalf("uninstallFromDocuments() error = %v", err)
	}
	if !result.RemovedMod || result.UpdatedModOrder {
		t.Fatalf("result = %+v", result)
	}
	if _, err := os.Stat(filepath.Join(documents, ModsDirectoryName, ModOrderFilename)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("mod_order unexpectedly exists: %v", err)
	}
}

func TestUninstallFromDocumentsCleansStaleOrderWithoutMod(t *testing.T) {
	documents := t.TempDir()
	modsRoot := filepath.Join(documents, ModsDirectoryName)
	if err := os.MkdirAll(modsRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	orderPath := filepath.Join(modsRoot, ModOrderFilename)
	if err := os.WriteFile(orderPath, []byte("other\n  "+modarchive.ModID+"  \nlast"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := uninstallFromDocuments(documents)
	if err != nil {
		t.Fatalf("uninstallFromDocuments() error = %v", err)
	}
	if result.RemovedMod || !result.UpdatedModOrder {
		t.Fatalf("result = %+v", result)
	}
	got, err := os.ReadFile(orderPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "other\nlast" {
		t.Fatalf("mod_order = %q, want %q", got, "other\\nlast")
	}
}

func TestUninstallFromDocumentsRollsBackModWhenOrderPublishFails(t *testing.T) {
	documents := t.TempDir()
	modsRoot := filepath.Join(documents, ModsDirectoryName)
	ownMod := filepath.Join(modsRoot, modarchive.ModID)
	if err := os.MkdirAll(ownMod, 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(ownMod, "marker.txt")
	if err := os.WriteFile(marker, []byte("previous"), 0o644); err != nil {
		t.Fatal(err)
	}
	orderPath := filepath.Join(modsRoot, ModOrderFilename)
	originalOrder := []byte("other\n" + modarchive.ModID + "\n")
	if err := os.WriteFile(orderPath, originalOrder, 0o644); err != nil {
		t.Fatal(err)
	}

	originalRename := renamePath
	defer func() { renamePath = originalRename }()
	publishFailures := 0
	renamePath = func(oldPath, newPath string) error {
		if strings.Contains(filepath.Base(oldPath), ".mod_order.txt.uninstall-") && newPath == orderPath {
			publishFailures++
			return errors.New("injected publish failure")
		}
		return os.Rename(oldPath, newPath)
	}

	_, err := uninstallFromDocuments(documents)
	if err == nil || publishFailures != 1 {
		t.Fatalf("error = %v, publishFailures=%d", err, publishFailures)
	}
	if got, err := os.ReadFile(marker); err != nil || string(got) != "previous" {
		t.Fatalf("previous mod not restored: %q err=%v", got, err)
	}
	gotOrder, err := os.ReadFile(orderPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotOrder) != string(originalOrder) {
		t.Fatalf("mod_order after rollback = %q, want %q", gotOrder, originalOrder)
	}
}

func TestRemoveModOrderEntriesPreservesUnrelatedBytes(t *testing.T) {
	input := []byte("alpha\r\n# comment\n  " + modarchive.ModID + " \r\nomega")
	got, changed := removeModOrderEntries(input, modarchive.ModID)
	if !changed {
		t.Fatal("changed = false, want true")
	}
	want := []byte("alpha\r\n# comment\nomega")
	if string(got) != string(want) {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestInspectInDocuments(t *testing.T) {
	documents := t.TempDir()
	status, err := inspectInDocuments(documents)
	if err != nil || status.Installed {
		t.Fatalf("initial status = %+v, err=%v", status, err)
	}
	if err := os.MkdirAll(status.Path, 0o755); err != nil {
		t.Fatal(err)
	}
	status, err = inspectInDocuments(documents)
	if err != nil || !status.Installed {
		t.Fatalf("installed status = %+v, err=%v", status, err)
	}
}
