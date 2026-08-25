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
	installTransactionPrefix       = ".kcd2-dual-subtitles-install-"
	transactionStateMarkerPrefix   = "state-"
	transactionStagedDirname       = "staged"
	transactionPreviousName        = "previous"
	transactionModOrderNextName    = "mod_order.next"
	transactionModOrderPreviousName = "mod_order.previous"

	transactionStateBuilding   = "building"
	transactionStatePublishing = "publishing"
	transactionStateCommitted  = "committed"
)

type installTransaction struct {
	root             string
	staged           string
	previous         string
	modOrderNext     string
	modOrderPrevious string
}

func beginInstallTransaction(modsRoot string) (*installTransaction, error) {
	parent := filepath.Dir(filepath.Clean(modsRoot))
	root, err := os.MkdirTemp(parent, installTransactionPrefix+"*")
	if err != nil {
		return nil, fmt.Errorf("create install transaction beside %q: %w", modsRoot, err)
	}
	tx := &installTransaction{
		root:             root,
		staged:           filepath.Join(root, transactionStagedDirname),
		previous:         filepath.Join(root, transactionPreviousName),
		modOrderNext:     filepath.Join(root, transactionModOrderNextName),
		modOrderPrevious: filepath.Join(root, transactionModOrderPreviousName),
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

// setState records monotonic state markers instead of rewriting one state file.
// A process termination during a transition can therefore never erase the last
// durable state: committed wins over publishing, which wins over building.
func (tx *installTransaction) setState(state string) error {
	if tx == nil || tx.root == "" {
		return errors.New("install transaction is not initialized")
	}
	switch state {
	case transactionStateBuilding, transactionStatePublishing, transactionStateCommitted:
	default:
		return fmt.Errorf("unknown install transaction state %q", state)
	}

	path := filepath.Join(tx.root, transactionStateMarkerPrefix+state)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if errors.Is(err, os.ErrExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("create install transaction state marker %q: %w", path, err)
	}
	closed := false
	defer func() {
		if !closed {
			_ = file.Close()
		}
	}()
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync install transaction state marker %q: %w", path, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close install transaction state marker %q: %w", path, err)
	}
	closed = true
	return nil
}

func transactionState(root string) (string, error) {
	for _, state := range []string{transactionStateCommitted, transactionStatePublishing, transactionStateBuilding} {
		path := filepath.Join(root, transactionStateMarkerPrefix+state)
		info, err := os.Lstat(path)
		switch {
		case err == nil:
			if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
				return "", fmt.Errorf("invalid install transaction state marker %q", path)
			}
			return state, nil
		case errors.Is(err, os.ErrNotExist):
			continue
		default:
			return "", fmt.Errorf("inspect install transaction state marker %q: %w", path, err)
		}
	}
	return "", nil
}

// updateModOrderIfPresent makes the load-order change part of the same durable
// transaction as the mod directory. The original file is kept in tx until the
// install reaches committed, so recovery can restore both resources together.
func (tx *installTransaction) updateModOrderIfPresent(modsRoot, modID string) (bool, error) {
	path := filepath.Join(modsRoot, ModOrderFilename)
	original, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read %q: %w", path, err)
	}
	if modOrderContains(original, modID) {
		return false, nil
	}

	info, err := os.Lstat(path)
	if err != nil {
		return false, fmt.Errorf("inspect %q: %w", path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return false, fmt.Errorf("refusing to replace non-regular load order at %q", path)
	}

	updated := modOrderWithEntry(original, modID)
	if err := writeSyncedExclusiveFile(tx.modOrderNext, updated, info.Mode().Perm()); err != nil {
		return false, fmt.Errorf("prepare updated load order: %w", err)
	}
	if err := renamePathWithRetry(path, tx.modOrderPrevious); err != nil {
		_ = os.Remove(tx.modOrderNext)
		return false, fmt.Errorf("preserve previous load order: %w", err)
	}
	if err := renamePathWithRetry(tx.modOrderNext, path); err != nil {
		rollbackErr := renamePathWithRetry(tx.modOrderPrevious, path)
		if rollbackErr != nil {
			return false, errors.Join(
				fmt.Errorf("publish updated load order: %w", err),
				fmt.Errorf("rollback previous load order: %w", rollbackErr),
			)
		}
		return false, fmt.Errorf("publish updated load order: %w", err)
	}
	return true, nil
}

func writeSyncedExclusiveFile(path string, data []byte, mode os.FileMode) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	closed := false
	defer func() {
		if !closed {
			_ = file.Close()
		}
	}()
	if _, err := file.Write(data); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	closed = true
	return nil
}

func (tx *installTransaction) restoreModOrder(modsRoot string) error {
	if tx == nil {
		return nil
	}
	return restoreInterruptedModOrder(modsRoot, tx.modOrderPrevious)
}

func restoreInterruptedModOrder(modsRoot, previous string) error {
	previousInfo, err := os.Lstat(previous)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return nil
	case err != nil:
		return fmt.Errorf("inspect previous load order %q: %w", previous, err)
	case previousInfo.Mode()&os.ModeSymlink != 0 || !previousInfo.Mode().IsRegular():
		return fmt.Errorf("refusing to recover invalid previous load order %q", previous)
	}

	path := filepath.Join(modsRoot, ModOrderFilename)
	currentInfo, currentErr := os.Lstat(path)
	switch {
	case errors.Is(currentErr, os.ErrNotExist):
	case currentErr != nil:
		return fmt.Errorf("inspect interrupted load order %q: %w", path, currentErr)
	case currentInfo.Mode()&os.ModeSymlink != 0 || !currentInfo.Mode().IsRegular():
		return fmt.Errorf("refusing to replace invalid interrupted load order %q", path)
	default:
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("remove interrupted load order %q: %w", path, err)
		}
	}
	if err := renamePathWithRetry(previous, path); err != nil {
		return fmt.Errorf("restore previous load order from %q: %w", previous, err)
	}
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
	state, err := transactionState(root)
	if err != nil {
		return err
	}
	if state == transactionStateCommitted {
		if err := os.RemoveAll(root); err != nil {
			return fmt.Errorf("clean committed install transaction %q: %w", root, err)
		}
		return nil
	}

	target := filepath.Join(modsRoot, modarchive.ModID)
	previous := filepath.Join(root, transactionPreviousName)
	previousInfo, previousErr := os.Lstat(previous)
	hasPrevious := previousErr == nil
	if previousErr != nil && !errors.Is(previousErr, os.ErrNotExist) {
		return fmt.Errorf("inspect previous installation in transaction %q: %w", root, previousErr)
	}

	var modErr error
	if hasPrevious {
		if previousInfo.Mode()&os.ModeSymlink != 0 || !previousInfo.IsDir() {
			modErr = fmt.Errorf("refusing to recover invalid previous installation %q", previous)
		} else if err := removeRecoverableTarget(target); err != nil {
			modErr = err
		} else if err := renamePathWithRetry(previous, target); err != nil {
			modErr = fmt.Errorf("restore interrupted previous installation from %q: %w", previous, err)
		}
	} else if state == transactionStatePublishing {
		// A fresh install can be interrupted while the guarded copy fallback is
		// writing the final target. With no previous installation to restore,
		// remove that uncommitted target rather than leaving a partial mod behind.
		modErr = removeRecoverableTarget(target)
	}

	orderErr := restoreInterruptedModOrder(modsRoot, filepath.Join(root, transactionModOrderPreviousName))
	if modErr != nil || orderErr != nil {
		// Keep the transaction workspace. A subsequent run can retry any resource
		// whose rollback was blocked by a transient sharing/permission failure.
		return errors.Join(modErr, orderErr)
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
