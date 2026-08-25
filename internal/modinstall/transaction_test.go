package modinstall

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/modarchive"
)

func TestRecoverInterruptedReplacementRestoresPreviousInstallation(t *testing.T) {
	parent := t.TempDir()
	modsRoot := filepath.Join(parent, ModsDirectoryName)
	if err := os.MkdirAll(modsRoot, 0o755); err != nil {
		t.Fatalf("create mods root: %v", err)
	}
	target := filepath.Join(modsRoot, modarchive.ModID)
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatalf("create previous target: %v", err)
	}
	if err := os.WriteFile(filepath.Join(target, "sentinel.txt"), []byte("previous"), 0o644); err != nil {
		t.Fatalf("write previous sentinel: %v", err)
	}

	tx, err := beginInstallTransaction(modsRoot)
	if err != nil {
		t.Fatalf("beginInstallTransaction() error = %v", err)
	}
	if err := renamePathWithRetry(target, tx.previous); err != nil {
		t.Fatalf("move previous installation: %v", err)
	}
	if err := tx.setState(transactionStatePublishing); err != nil {
		t.Fatalf("set publishing state: %v", err)
	}
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatalf("create interrupted replacement: %v", err)
	}
	if err := os.WriteFile(filepath.Join(target, "sentinel.txt"), []byte("partial"), 0o644); err != nil {
		t.Fatalf("write partial sentinel: %v", err)
	}

	if err := recoverInstallTransactions(modsRoot); err != nil {
		t.Fatalf("recoverInstallTransactions() error = %v", err)
	}
	got, err := os.ReadFile(filepath.Join(target, "sentinel.txt"))
	if err != nil {
		t.Fatalf("read recovered sentinel: %v", err)
	}
	if string(got) != "previous" {
		t.Fatalf("recovered target = %q, want previous", got)
	}
	if _, err := os.Stat(tx.root); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("transaction survived recovery: %v", err)
	}
}

func TestRecoverInterruptedFreshPublishRemovesPartialTarget(t *testing.T) {
	parent := t.TempDir()
	modsRoot := filepath.Join(parent, ModsDirectoryName)
	if err := os.MkdirAll(modsRoot, 0o755); err != nil {
		t.Fatalf("create mods root: %v", err)
	}
	tx, err := beginInstallTransaction(modsRoot)
	if err != nil {
		t.Fatalf("beginInstallTransaction() error = %v", err)
	}
	if err := tx.setState(transactionStatePublishing); err != nil {
		t.Fatalf("set publishing state: %v", err)
	}
	target := filepath.Join(modsRoot, modarchive.ModID)
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatalf("create partial target: %v", err)
	}
	if err := os.WriteFile(filepath.Join(target, "partial.txt"), []byte("partial"), 0o644); err != nil {
		t.Fatalf("write partial target: %v", err)
	}

	if err := recoverInstallTransactions(modsRoot); err != nil {
		t.Fatalf("recoverInstallTransactions() error = %v", err)
	}
	if _, err := os.Stat(target); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("uncommitted fresh target survived recovery: %v", err)
	}
}

func TestRecoverBuildingTransactionLeavesExistingTargetUntouched(t *testing.T) {
	parent := t.TempDir()
	modsRoot := filepath.Join(parent, ModsDirectoryName)
	if err := os.MkdirAll(modsRoot, 0o755); err != nil {
		t.Fatalf("create mods root: %v", err)
	}
	target := filepath.Join(modsRoot, modarchive.ModID)
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatalf("create target: %v", err)
	}
	if err := os.WriteFile(filepath.Join(target, "sentinel.txt"), []byte("current"), 0o644); err != nil {
		t.Fatalf("write target sentinel: %v", err)
	}
	tx, err := beginInstallTransaction(modsRoot)
	if err != nil {
		t.Fatalf("beginInstallTransaction() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(tx.staged, "partial-build.txt"), []byte("partial"), 0o644); err != nil {
		t.Fatalf("write partial build: %v", err)
	}

	if err := recoverInstallTransactions(modsRoot); err != nil {
		t.Fatalf("recoverInstallTransactions() error = %v", err)
	}
	got, err := os.ReadFile(filepath.Join(target, "sentinel.txt"))
	if err != nil || string(got) != "current" {
		t.Fatalf("existing target changed during building recovery: data=%q err=%v", got, err)
	}
}

func TestRecoverCommittedTransactionKeepsPublishedTarget(t *testing.T) {
	parent := t.TempDir()
	modsRoot := filepath.Join(parent, ModsDirectoryName)
	if err := os.MkdirAll(modsRoot, 0o755); err != nil {
		t.Fatalf("create mods root: %v", err)
	}
	tx, err := beginInstallTransaction(modsRoot)
	if err != nil {
		t.Fatalf("beginInstallTransaction() error = %v", err)
	}
	if err := os.Mkdir(tx.previous, 0o755); err != nil {
		t.Fatalf("create previous installation: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tx.previous, "sentinel.txt"), []byte("previous"), 0o644); err != nil {
		t.Fatalf("write previous sentinel: %v", err)
	}
	target := filepath.Join(modsRoot, modarchive.ModID)
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatalf("create committed target: %v", err)
	}
	if err := os.WriteFile(filepath.Join(target, "sentinel.txt"), []byte("committed"), 0o644); err != nil {
		t.Fatalf("write committed sentinel: %v", err)
	}
	if err := tx.setState(transactionStateCommitted); err != nil {
		t.Fatalf("set committed state: %v", err)
	}

	if err := recoverInstallTransactions(modsRoot); err != nil {
		t.Fatalf("recoverInstallTransactions() error = %v", err)
	}
	got, err := os.ReadFile(filepath.Join(target, "sentinel.txt"))
	if err != nil || string(got) != "committed" {
		t.Fatalf("committed target was not retained: data=%q err=%v", got, err)
	}
	if _, err := os.Stat(tx.root); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("committed transaction survived cleanup: %v", err)
	}
}
