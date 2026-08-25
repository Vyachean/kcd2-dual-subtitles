package modinstall

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/modarchive"
)

func TestRecoverKeepsTargetWhenPreviousWasAlreadyRestored(t *testing.T) {
	parent := t.TempDir()
	modsRoot := filepath.Join(parent, ModsDirectoryName)
	if err := os.MkdirAll(modsRoot, 0o755); err != nil {
		t.Fatalf("create mods root: %v", err)
	}
	tx, err := beginInstallTransaction(modsRoot)
	if err != nil {
		t.Fatalf("beginInstallTransaction() error = %v", err)
	}
	if err := tx.markHadPrevious(); err != nil {
		t.Fatalf("markHadPrevious() error = %v", err)
	}
	if err := tx.setState(transactionStatePublishing); err != nil {
		t.Fatalf("set publishing state: %v", err)
	}

	// Simulate a successful rollback whose previous directory was renamed back
	// to the canonical target, followed by termination before tx cleanup.
	target := filepath.Join(modsRoot, modarchive.ModID)
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatalf("create restored target: %v", err)
	}
	if err := os.WriteFile(filepath.Join(target, "sentinel.txt"), []byte("restored"), 0o644); err != nil {
		t.Fatalf("write restored sentinel: %v", err)
	}

	if err := recoverInstallTransactions(modsRoot); err != nil {
		t.Fatalf("recoverInstallTransactions() error = %v", err)
	}
	got, err := os.ReadFile(filepath.Join(target, "sentinel.txt"))
	if err != nil || string(got) != "restored" {
		t.Fatalf("already-restored target changed: data=%q err=%v", got, err)
	}
	if _, err := os.Stat(tx.root); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("transaction survived cleanup: %v", err)
	}
}

func TestRecoverKeepsPublishedTargetFromMarkerlessCommittedCleanupResidue(t *testing.T) {
	parent := t.TempDir()
	modsRoot := filepath.Join(parent, ModsDirectoryName)
	if err := os.MkdirAll(modsRoot, 0o755); err != nil {
		t.Fatalf("create mods root: %v", err)
	}
	target := filepath.Join(modsRoot, modarchive.ModID)
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatalf("create published target: %v", err)
	}
	if err := os.WriteFile(filepath.Join(target, "sentinel.txt"), []byte("published"), 0o644); err != nil {
		t.Fatalf("write published sentinel: %v", err)
	}

	// Safe committed cleanup removes every transaction entry before removing the
	// committed marker. Termination after that final marker removal can leave an
	// empty transaction directory, which must never be interpreted as evidence
	// that the verified published target should be rolled back or deleted.
	root, err := os.MkdirTemp(parent, installTransactionPrefix+"*")
	if err != nil {
		t.Fatalf("create markerless transaction residue: %v", err)
	}

	if err := recoverInstallTransactions(modsRoot); err != nil {
		t.Fatalf("recoverInstallTransactions() error = %v", err)
	}
	got, err := os.ReadFile(filepath.Join(target, "sentinel.txt"))
	if err != nil || string(got) != "published" {
		t.Fatalf("published target changed: data=%q err=%v", got, err)
	}
	if _, err := os.Stat(root); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("markerless transaction residue survived cleanup: %v", err)
	}
}
