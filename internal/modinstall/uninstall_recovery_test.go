package modinstall

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/modarchive"
)

func TestUninstallRecoversInterruptedReplacementBeforeRemoval(t *testing.T) {
	parent := t.TempDir()
	modsRoot := filepath.Join(parent, ModsDirectoryName)
	if err := os.MkdirAll(modsRoot, 0o755); err != nil {
		t.Fatalf("create mods root: %v", err)
	}
	target := filepath.Join(modsRoot, modarchive.ModID)
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatalf("create previous target: %v", err)
	}
	if err := os.WriteFile(filepath.Join(target, "previous.txt"), []byte("previous"), 0o644); err != nil {
		t.Fatalf("write previous target: %v", err)
	}

	tx, err := beginInstallTransaction(modsRoot)
	if err != nil {
		t.Fatalf("beginInstallTransaction() error = %v", err)
	}
	if err := renamePathWithRetry(target, tx.previous); err != nil {
		t.Fatalf("park previous target: %v", err)
	}
	if err := tx.setState(transactionStatePublishing); err != nil {
		t.Fatalf("set publishing state: %v", err)
	}
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatalf("create partial replacement: %v", err)
	}
	if err := os.WriteFile(filepath.Join(target, "partial.txt"), []byte("partial"), 0o644); err != nil {
		t.Fatalf("write partial replacement: %v", err)
	}

	result, err := uninstallFromModsRoot(modsRoot)
	if err != nil {
		t.Fatalf("uninstallFromModsRoot() error = %v", err)
	}
	if !result.RemovedMod {
		t.Fatalf("result = %+v, want recovered previous mod to be removed", result)
	}
	if _, err := os.Stat(target); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("target survived uninstall: %v", err)
	}
	if _, err := os.Stat(tx.root); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("interrupted transaction survived uninstall recovery: %v", err)
	}
}

func TestUninstallCleansLegacyScannedStagingWithoutCanonicalMod(t *testing.T) {
	parent := t.TempDir()
	modsRoot := filepath.Join(parent, ModsDirectoryName)
	legacy := filepath.Join(modsRoot, "."+modarchive.ModID+".staging-orphan")
	if err := os.MkdirAll(legacy, 0o755); err != nil {
		t.Fatalf("create legacy staging: %v", err)
	}
	if err := os.WriteFile(filepath.Join(legacy, modarchive.ManifestFilename), []byte("legacy"), 0o644); err != nil {
		t.Fatalf("write legacy manifest: %v", err)
	}

	result, err := uninstallFromModsRoot(modsRoot)
	if err != nil {
		t.Fatalf("uninstallFromModsRoot() error = %v", err)
	}
	if result.RemovedMod {
		t.Fatalf("result = %+v, canonical mod was absent", result)
	}
	if _, err := os.Stat(legacy); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("legacy scanned staging survived uninstall: %v", err)
	}
}
