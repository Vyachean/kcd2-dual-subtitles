package modinstall

import (
	"archive/zip"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/modarchive"
)

func TestFindForeignHUDOverridesDetectsLooseAndPAKOverrides(t *testing.T) {
	modsRoot := filepath.Join(t.TempDir(), ModsDirectoryName)
	if err := os.MkdirAll(modsRoot, 0o755); err != nil {
		t.Fatalf("create mods root: %v", err)
	}

	loose := filepath.Join(modsRoot, "a_loose", "Libs", "UI")
	if err := os.MkdirAll(loose, 0o755); err != nil {
		t.Fatalf("create loose HUD dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(loose, "hud.gfx"), []byte("foreign"), 0o644); err != nil {
		t.Fatalf("write loose HUD: %v", err)
	}

	dataDir := filepath.Join(modsRoot, "b_pak", "Data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("create Data dir: %v", err)
	}
	writeHUDConflictPAK(t, filepath.Join(dataDir, "ui.pak"), true)

	ownData := filepath.Join(modsRoot, modarchive.ModID, "Data")
	if err := os.MkdirAll(ownData, 0o755); err != nil {
		t.Fatalf("create own Data dir: %v", err)
	}
	writeHUDConflictPAK(t, filepath.Join(ownData, "own.pak"), true)

	conflicts, err := findForeignHUDOverrides(modsRoot)
	if err != nil {
		t.Fatalf("findForeignHUDOverrides() error = %v", err)
	}
	want := []string{"a_loose", "b_pak"}
	if !reflect.DeepEqual(conflicts, want) {
		t.Fatalf("conflicts = %#v, want %#v", conflicts, want)
	}
}

func TestFindForeignHUDOverridesIgnoresOrdinaryDataPAKs(t *testing.T) {
	modsRoot := filepath.Join(t.TempDir(), ModsDirectoryName)
	dataDir := filepath.Join(modsRoot, "ordinary", "Data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("create Data dir: %v", err)
	}
	writeHUDConflictPAK(t, filepath.Join(dataDir, "ordinary.pak"), false)
	conflicts, err := findForeignHUDOverrides(modsRoot)
	if err != nil {
		t.Fatalf("findForeignHUDOverrides() error = %v", err)
	}
	if len(conflicts) != 0 {
		t.Fatalf("conflicts = %#v, want none", conflicts)
	}
}

func TestInstallIntoDocumentsVersionedWithHUDRejectsForeignConflictBeforeReplacement(t *testing.T) {
	documents := t.TempDir()
	modsRoot := filepath.Join(documents, ModsDirectoryName)
	foreignData := filepath.Join(modsRoot, "foreign", "Data")
	if err := os.MkdirAll(foreignData, 0o755); err != nil {
		t.Fatalf("create foreign Data dir: %v", err)
	}
	writeHUDConflictPAK(t, filepath.Join(foreignData, "hud.pak"), true)

	_, err := installIntoDocumentsVersionedWithHUD(
		documents,
		localization.Russian,
		[]localization.DialogueRow{{ID: "id", Text: "text"}},
		[]byte("hud"),
		"dev",
	)
	if !errors.Is(err, ErrHUDConflict) {
		t.Fatalf("install error = %v, want ErrHUDConflict", err)
	}
	if _, statErr := os.Stat(filepath.Join(modsRoot, modarchive.ModID)); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("own mod exists after conflict: %v", statErr)
	}
}

func TestInstallIntoDocumentsVersionedWithHUDWritesOwnDataPAK(t *testing.T) {
	documents := t.TempDir()
	installed, err := installIntoDocumentsVersionedWithHUD(
		documents,
		localization.English,
		[]localization.DialogueRow{{ID: "id", Text: "text"}},
		[]byte("patched-hud"),
		"v0.3.0-test",
	)
	if err != nil {
		t.Fatalf("installIntoDocumentsVersionedWithHUD() error = %v", err)
	}
	dataPAKPath := filepath.Join(installed, "Data", modarchive.DataPAKFilename)
	reader, err := zip.OpenReader(dataPAKPath)
	if err != nil {
		t.Fatalf("open installed data PAK: %v", err)
	}
	defer reader.Close()
	if len(reader.File) != 1 || reader.File[0].Name != modarchive.HUDArchivePath {
		t.Fatalf("installed HUD PAK entries = %#v, want only %q", reader.File, modarchive.HUDArchivePath)
	}
}

func writeHUDConflictPAK(t *testing.T, path string, withHUD bool) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create test PAK: %v", err)
	}
	writer := zip.NewWriter(file)
	name := "Scripts/example.lua"
	if withHUD {
		name = modarchive.HUDArchivePath
	}
	entry, err := writer.Create(name)
	if err != nil {
		t.Fatalf("create PAK entry: %v", err)
	}
	if _, err := entry.Write([]byte("data")); err != nil {
		t.Fatalf("write PAK entry: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close PAK writer: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close PAK file: %v", err)
	}
}
