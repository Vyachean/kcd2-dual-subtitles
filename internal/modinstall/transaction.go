package modinstall

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/modarchive"
)

const (
	installTransactionPrefix = ".kcd2-dual-subtitles-install-"
	transactionStateFilename = "state"
	transactionStagedDirname = "staged"
	transactionPreviousName  = "previous"

	transactionStateBuilding   = "building"
	transactionStatePublishing = "publishing"
	transactionStateCommitted  = "committed"
)

type installTransaction struct {
	root     string
	staged   string
	previous string
}

func beginInstallTransaction(modsRoot string) (*installTransaction, error) {
	parent := filepath.Dir(filepath.Clean(modsRoot))
	root, err := os.MkdirTemp(parent, installTransactionPrefix+"*")
	if err != nil {
		return nil, fmt.Errorf("create install transaction beside %q: %w", modsRoot, err)
	}
	tx := &installTransaction{
		root:     root,
		staged:   filepath.Join(root, transactionStagedDirname),
		previous: filepath.Join(root, transactionPreviousName),
	}
	if err := os.Mkdir(tx.staged, 0o755); err != nil {
		_ = os.RemoveAll(root)
		return nil, fmt.Errorf("create staged mod directory in transaction %q: %w", root, err)
	}
	if err := tx.setState(transactionStateBuilding); err != nil {
		_ = os.RemoveAll(root)
		return nil, err
	}
	return tx, nil
}

func (tx *installTransaction) setState(state string) error {
	if tx == nil || tx.root == "" {
		return errors.New("install transaction is not initialized")
	}
	path := filepath.Join(tx.root, transactionStateFilename)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("open install transaction state %q: %w", path, err)
	}
	closed := false
	defer func() {
		if !closed {
			_ = file.Close()
		}
	}()
	if _, err := file.WriteString(state + "\n"); err != nil {
		return fmt.Errorf("write install transaction state %q: %w", path, err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync install transaction state %q: %w", path, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close install transaction state %q: %w", path, err)
	}
	closed = true
	return nil
}

func recoverInstallTransactions(modsRoot string) error {
	modsRoot = filepath.Clean(modsRoot)
	parent := filepath.Dir(modsRoot)
	entries, err := os.ReadDir(parent)
	if err != nil {
		return fmt.Errorf("inspect install transaction parent %q: %w", parent, err)
	}
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), installTransactionPrefix) {
			continue
		}
		root := filepath.Join(parent, entry.Name())
		info, err := os.Lstat(root)
		if err != nil {
			return fmt.Errorf("inspect install transaction %q: %w", root, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing to recover symlinked install transaction %q", root)
		}
		if !info.IsDir() {
			return fmt.Errorf("refusing to recover non-directory install transaction %q", root)
		}
		if err := recoverInstallTransaction(modsRoot, root); err != nil {
			return err
		}
	}
	return nil
}

func recoverInstallTransaction(modsRoot, root string) error {
	stateData, stateErr := os.ReadFile(filepath.Join(root, transactionStateFilename))
	state := strings.TrimSpace(string(stateData))
	if stateErr != nil && !errors.Is(stateErr, os.ErrNotExist) {
		return fmt.Errorf("read install transaction state in %q: %w", root, stateErr)
	}

	target := filepath.Join(modsRoot, modarchive.ModID)
	previous := filepath.Join(root, transactionPreviousName)

	if state == transactionStateCommitted {
		if err := os.RemoveAll(root); err != nil {
			return fmt.Errorf("clean committed install transaction %q: %w", root, err)
		}
		return nil
	}

	previousInfo, previousErr := os.Lstat(previous)
	hasPrevious := previousErr == nil
	if previousErr != nil && !errors.Is(previousErr, os.ErrNotExist) {
		return fmt.Errorf("inspect previous installation in transaction %q: %w", root, previousErr)
	}
	if hasPrevious {
		if previousInfo.Mode()&os.ModeSymlink != 0 || !previousInfo.IsDir() {
			return fmt.Errorf("refusing to recover invalid previous installation %q", previous)
		}
		if err := removeRecoverableTarget(target); err != nil {
			return err
		}
		if err := renamePathWithRetry(previous, target); err != nil {
			return fmt.Errorf("restore interrupted previous installation from %q: %w", previous, err)
		}
	} else if state == transactionStatePublishing {
		// A fresh install can be interrupted while the guarded copy fallback is
		// writing the final target. With no previous installation to restore,
		// remove that uncommitted target rather than leaving a partial mod behind.
		if err := removeRecoverableTarget(target); err != nil {
			return err
		}
	}

	if err := os.RemoveAll(root); err != nil {
		return fmt.Errorf("clean interrupted install transaction %q: %w", root, err)
	}
	return nil
}

func removeRecoverableTarget(target string) error {
	info, err := os.Lstat(target)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return nil
	case err != nil:
		return fmt.Errorf("inspect interrupted mod target %q: %w", target, err)
	case info.Mode()&os.ModeSymlink != 0:
		return fmt.Errorf("refusing to replace symlink at interrupted mod target %q", target)
	case !info.IsDir():
		return fmt.Errorf("refusing to replace non-directory at interrupted mod target %q", target)
	default:
		if err := os.RemoveAll(target); err != nil {
			return fmt.Errorf("remove interrupted mod target %q: %w", target, err)
		}
		return nil
	}
}
