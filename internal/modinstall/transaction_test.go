package modinstall

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/modarchive"
)

func TestRecoverInterruptedReplacementRestoresPreviousInstallationAndLoadOrder(t *testing.T) {
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
	orderPath := filepath.Join(modsRoot, ModOrderFilename)
	originalOrder := []byte("other_mod\r\n")
	if err := os.WriteFile(orderPath, originalOrder, 0o600); err != nil {
		t.Fatalf("write original load order: %v", err)
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
	changed, err := tx.updateModOrderIfPresent(modsRoot, modarchive.ModID)
	if err != nil || !changed {
		t.Fatalf("updateModOrderIfPresent() changed=%v err=%v", changed, err)
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
	gotOrder, err := os.ReadFile(orderPath)
	if err != nil || string(gotOrder) != string(originalOrder) {
		t.Fatalf("recovered load order = %q err=%v, want %q", gotOrder, err, originalOrder)
	}
	orderInfo, err := os.Stat(orderPath)
	if err != nil || orderInfo.Mode().Perm() != 0o600 {
		t.Fatalf("recovered load-order permissions = %v err=%v, want 0600", orderInfo, err)
	}
	if _, err := os.Stat(tx.root); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("transaction survived recovery: %v", err)
	}
}

func TestRecoverInterruptedFreshPublishRemovesPartialTargetAndRestoresLoadOrder(t *testing.T) {
	parent := t.TempDir()
	modsRoot := filepath.Join(parent, ModsDirectoryName)
	if err := os.MkdirAll(modsRoot, 0o755); err != nil {
		t.Fatalf("create mods root: %v", err)
	}
	orderPath := filepath.Join(modsRoot, ModOrderFilename)
	originalOrder := []byte("other_mod\n")
	if err := os.WriteFile(orderPath, originalOrder, 0o644); err != nil {
		t.Fatalf("write original load order: %v", err)
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
	changed, err := tx.updateModOrderIfPresent(modsRoot, modarchive.ModID)
	if err != nil || !changed {
		t.Fatalf("updateModOrderIfPresent() changed=%v err=%v", changed, err)
	}

	if err := recoverInstallTransactions(modsRoot); err != nil {
		t.Fatalf("recoverInstallTransactions() error = %v", err)
	}
	if _, err := os.Stat(target); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("uncommitted fresh target survived recovery: %v", err)
	}
	gotOrder, err := os.ReadFile(orderPath)
	if err != nil || string(gotOrder) != string(originalOrder) {
		t.Fatalf("recovered load order = %q err=%v, want %q", gotOrder, err, originalOrder)
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

func TestRecoverCommittedTransactionKeepsPublishedTargetAndLoadOrder(t *testing.T) {
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
	orderPath := filepath.Join(modsRoot, ModOrderFilename)
	if err := os.WriteFile(orderPath, []byte("other_mod\n"), 0o644); err != nil {
		t.Fatalf("write initial load order: %v", err)
	}
	changed, err := tx.updateModOrderIfPresent(modsRoot, modarchive.ModID)
	if err != nil || !changed {
		t.Fatalf("updateModOrderIfPresent() changed=%v err=%v", changed, err)
	}
	committedOrder, err := os.ReadFile(orderPath)
	if err != nil {
		t.Fatalf("read committed load order: %v", err)
	}
	if err := tx.setState(transactionStatePublishing); err != nil {
		t.Fatalf("set publishing state: %v", err)
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
	gotOrder, err := os.ReadFile(orderPath)
	if err != nil || string(gotOrder) != string(committedOrder) {
		t.Fatalf("committed load order changed during cleanup: data=%q err=%v", gotOrder, err)
	}
	if _, err := os.Stat(tx.root); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("committed transaction survived cleanup: %v", err)
	}
}

func TestRecoverKeepsTransactionWhenPreviousRestoreIsTemporarilyBlocked(t *testing.T) {
	originalRename := renamePath
	originalSleep := sleepRenameRetry
	defer func() {
		renamePath = originalRename
		sleepRenameRetry = originalSleep
	}()
	sleepRenameRetry = func(time.Duration) {}

	parent := t.TempDir()
	modsRoot := filepath.Join(parent, ModsDirectoryName)
	if err := os.MkdirAll(modsRoot, 0o755); err != nil {
		t.Fatalf("create mods root: %v", err)
	}
	target := filepath.Join(modsRoot, modarchive.ModID)
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatalf("create target: %v", err)
	}
	if err := os.WriteFile(filepath.Join(target, "sentinel.txt"), []byte("previous"), 0o644); err != nil {
		t.Fatalf("write sentinel: %v", err)
	}
	tx, err := beginInstallTransaction(modsRoot)
	if err != nil {
		t.Fatalf("beginInstallTransaction() error = %v", err)
	}
	if err := os.Rename(target, tx.previous); err != nil {
		t.Fatalf("move previous target: %v", err)
	}
	if err := tx.setState(transactionStatePublishing); err != nil {
		t.Fatalf("set publishing state: %v", err)
	}
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatalf("create interrupted target: %v", err)
	}

	renamePath = func(oldPath, newPath string) error {
		if oldPath == tx.previous && newPath == target {
			return os.ErrPermission
		}
		return os.Rename(oldPath, newPath)
	}
	if err := recoverInstallTransactions(modsRoot); err == nil {
		t.Fatal("recoverInstallTransactions() error = nil, want blocked restore")
	}
	if info, err := os.Stat(tx.previous); err != nil || !info.IsDir() {
		t.Fatalf("recoverable previous installation was discarded: info=%v err=%v", info, err)
	}
	if info, err := os.Stat(tx.root); err != nil || !info.IsDir() {
		t.Fatalf("transaction workspace was discarded after failed rollback: info=%v err=%v", info, err)
	}

	renamePath = originalRename
	if err := recoverInstallTransactions(modsRoot); err != nil {
		t.Fatalf("second recovery error = %v", err)
	}
	got, err := os.ReadFile(filepath.Join(target, "sentinel.txt"))
	if err != nil || string(got) != "previous" {
		t.Fatalf("second recovery did not restore previous install: data=%q err=%v", got, err)
	}
}
