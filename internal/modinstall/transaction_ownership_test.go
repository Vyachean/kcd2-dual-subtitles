package modinstall

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRecoverInstallTransactionsIgnoresOwnedSiblingRoot(t *testing.T) {
	parent := t.TempDir()
	firstRoot := filepath.Join(parent, "ModsA")
	secondRoot := filepath.Join(parent, "ModsB")
	for _, root := range []string{firstRoot, secondRoot} {
		if err := os.MkdirAll(root, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	tx, err := beginInstallTransaction(firstRoot)
	if err != nil {
		t.Fatalf("beginInstallTransaction() error = %v", err)
	}
	if err := recoverInstallTransactionsWithLegacy(secondRoot, false); err != nil {
		t.Fatalf("recover second root error = %v", err)
	}
	if info, err := os.Stat(tx.root); err != nil || !info.IsDir() {
		t.Fatalf("first-root transaction was touched by second-root recovery: info=%v err=%v", info, err)
	}

	if err := recoverInstallTransactionsWithLegacy(firstRoot, false); err != nil {
		t.Fatalf("recover owner root error = %v", err)
	}
	if _, err := os.Stat(tx.root); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("owned transaction survived owner recovery: %v", err)
	}
}

func TestCustomRecoveryIgnoresLegacyUnownedTransaction(t *testing.T) {
	parent := t.TempDir()
	customRoot := filepath.Join(parent, "CustomMods")
	if err := os.MkdirAll(customRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	legacyRoot, err := os.MkdirTemp(parent, installTransactionPrefix+"*")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(legacyRoot, transactionStagedDirname), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacyRoot, transactionStateMarkerPrefix+transactionStateBuilding), []byte("building\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := recoverInstallTransactionsWithLegacy(customRoot, false); err != nil {
		t.Fatalf("custom recovery error = %v", err)
	}
	if info, err := os.Stat(legacyRoot); err != nil || !info.IsDir() {
		t.Fatalf("custom recovery touched legacy unowned transaction: info=%v err=%v", info, err)
	}

	if err := recoverInstallTransactionsWithLegacy(customRoot, true); err != nil {
		t.Fatalf("legacy-compatible recovery error = %v", err)
	}
	if _, err := os.Stat(legacyRoot); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("legacy-compatible recovery did not clean unowned transaction: %v", err)
	}
}
