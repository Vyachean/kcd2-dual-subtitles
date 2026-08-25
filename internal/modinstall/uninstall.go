package modinstall

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/modarchive"
)

// UninstallResult describes what the uninstall operation changed.
type UninstallResult struct {
	Path            string
	RemovedMod      bool
	UpdatedModOrder bool
}

// UninstallForGameRoot resolves the same target used by automatic installation
// and removes only this tool's mod and load-order entry there.
func UninstallForGameRoot(gameRoot string) (UninstallResult, error) {
	location, err := ResolveInstallLocation(gameRoot)
	if err != nil {
		return UninstallResult{}, err
	}
	return uninstallFromModsRoot(location.ModsRoot)
}

// Documents-specific uninstall remains only for focused GDK filesystem tests.
func uninstallFromDocuments(documents string) (UninstallResult, error) {
	if documents == "" {
		return UninstallResult{}, errors.New("Documents path is empty")
	}
	return uninstallFromModsRoot(filepath.Join(documents, ModsDirectoryName))
}

func uninstallFromModsRoot(modsRoot string) (UninstallResult, error) {
	modsRoot = strings.TrimSpace(modsRoot)
	if modsRoot == "" {
		return UninstallResult{}, errors.New("KCD2 mod root is empty")
	}

	releaseLock, err := acquireInstallLock(modsRoot)
	if err != nil {
		return UninstallResult{}, err
	}
	defer releaseLock()

	// Resolve an interrupted Generate/Regenerate transaction before uninstalling.
	// Otherwise an old installation parked in the transaction workspace could be
	// restored by a later run after the user had explicitly uninstalled the mod.
	if err := recoverInstallTransactions(modsRoot); err != nil {
		return UninstallResult{}, err
	}
	if rootInfo, err := os.Stat(modsRoot); err == nil {
		if !rootInfo.IsDir() {
			return UninstallResult{}, fmt.Errorf("KCD2 mod root is not a directory: %q", modsRoot)
		}
		if err := cleanupLegacyToolTempDirs(modsRoot); err != nil {
			return UninstallResult{}, err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return UninstallResult{}, fmt.Errorf("inspect KCD2 mod root %q: %w", modsRoot, err)
	}

	target := filepath.Join(modsRoot, modarchive.ModID)
	result := UninstallResult{Path: target}

	info, targetErr := os.Lstat(target)
	switch {
	case targetErr == nil:
		if info.Mode()&os.ModeSymlink != 0 {
			return result, fmt.Errorf("refusing to uninstall symlink at mod path %q", target)
		}
		if !info.IsDir() {
			return result, fmt.Errorf("refusing to uninstall non-directory at mod path %q", target)
		}
	case errors.Is(targetErr, os.ErrNotExist):
		// Already absent; still clean a stale load-order entry below.
	default:
		return result, fmt.Errorf("inspect mod path %q: %w", target, targetErr)
	}

	orderPath := filepath.Join(modsRoot, ModOrderFilename)
	originalOrder, orderErr := os.ReadFile(orderPath)
	orderExists := orderErr == nil
	if orderErr != nil && !errors.Is(orderErr, os.ErrNotExist) {
		return result, fmt.Errorf("read %q: %w", orderPath, orderErr)
	}
	updatedOrder, orderChanged := removeModOrderEntries(originalOrder, modarchive.ModID)

	var orderTemp string
	if orderExists && orderChanged {
		orderInfo, err := os.Stat(orderPath)
		if err != nil {
			return result, fmt.Errorf("inspect %q: %w", orderPath, err)
		}
		temporary, err := os.CreateTemp(modsRoot, ".mod_order.txt.uninstall-*")
		if err != nil {
			return result, fmt.Errorf("create temporary load order: %w", err)
		}
		orderTemp = temporary.Name()
		closed := false
		defer func() {
			if !closed {
				_ = temporary.Close()
			}
			_ = os.Remove(orderTemp)
		}()
		if err := temporary.Chmod(orderInfo.Mode().Perm()); err != nil {
			return result, fmt.Errorf("set temporary load-order permissions: %w", err)
		}
		if _, err := temporary.Write(updatedOrder); err != nil {
			return result, fmt.Errorf("write temporary load order: %w", err)
		}
		if err := temporary.Sync(); err != nil {
			return result, fmt.Errorf("sync temporary load order: %w", err)
		}
		if err := temporary.Close(); err != nil {
			return result, fmt.Errorf("close temporary load order: %w", err)
		}
		closed = true
	}

	var modBackup string
	if targetErr == nil {
		// Directory-shaped work must never live under modsRoot: KCD2 scans every
		// direct child directory there as a candidate mod. Keep the uninstall
		// backup on the same volume but beside the scanned root, so an interrupted
		// uninstall cannot leave another loadable copy of this mod behind.
		backupParent := filepath.Dir(filepath.Clean(modsRoot))
		backupPath, err := reserveSiblingPath(backupParent, ".kcd2-dual-subtitles-uninstall-*")
		if err != nil {
			return result, err
		}
		modBackup = backupPath
		if err := renamePath(target, modBackup); err != nil {
			return result, fmt.Errorf("stage installed mod for removal: %w", err)
		}
	}

	rollbackMod := func() error {
		if modBackup == "" {
			return nil
		}
		if err := renamePath(modBackup, target); err != nil {
			return fmt.Errorf("restore installed mod from %q: %w", modBackup, err)
		}
		return nil
	}

	var orderBackup string
	if orderExists && orderChanged {
		backupPath, err := reserveSiblingPath(modsRoot, ".mod_order.txt.uninstall-backup-*")
		if err != nil {
			if rollbackErr := rollbackMod(); rollbackErr != nil {
				return result, errors.Join(err, rollbackErr)
			}
			return result, err
		}
		orderBackup = backupPath
		if err := renamePath(orderPath, orderBackup); err != nil {
			if rollbackErr := rollbackMod(); rollbackErr != nil {
				return result, errors.Join(fmt.Errorf("preserve previous load order: %w", err), rollbackErr)
			}
			return result, fmt.Errorf("preserve previous load order: %w", err)
		}
		if err := renamePath(orderTemp, orderPath); err != nil {
			orderRollbackErr := renamePath(orderBackup, orderPath)
			modRollbackErr := rollbackMod()
			return result, errors.Join(
				fmt.Errorf("publish updated load order: %w", err),
				wrapRollback("restore previous load order", orderRollbackErr),
				modRollbackErr,
			)
		}
		result.UpdatedModOrder = true
	}

	if modBackup != "" {
		if err := os.RemoveAll(modBackup); err != nil {
			return result, fmt.Errorf("remove staged mod directory %q: %w", modBackup, err)
		}
		result.RemovedMod = true
	}
	if orderBackup != "" {
		_ = os.Remove(orderBackup)
	}
	return result, nil
}

func removeModOrderEntries(data []byte, modID string) ([]byte, bool) {
	if len(data) == 0 {
		return append([]byte(nil), data...), false
	}

	var output bytes.Buffer
	changed := false
	for start := 0; start < len(data); {
		lineEnd := bytes.IndexByte(data[start:], '\n')
		end := len(data)
		if lineEnd >= 0 {
			end = start + lineEnd + 1
		}
		segment := data[start:end]
		content := segment
		if len(content) > 0 && content[len(content)-1] == '\n' {
			content = content[:len(content)-1]
		}
		if len(content) > 0 && content[len(content)-1] == '\r' {
			content = content[:len(content)-1]
		}
		if string(bytes.TrimSpace(content)) == modID {
			changed = true
		} else {
			_, _ = output.Write(segment)
		}
		start = end
	}
	return output.Bytes(), changed
}

func reserveSiblingPath(directory, pattern string) (string, error) {
	placeholder, err := os.CreateTemp(directory, pattern)
	if err != nil {
		return "", fmt.Errorf("reserve backup path: %w", err)
	}
	path := placeholder.Name()
	if err := placeholder.Close(); err != nil {
		return "", fmt.Errorf("close backup placeholder: %w", err)
	}
	if err := os.Remove(path); err != nil {
		return "", fmt.Errorf("prepare backup path: %w", err)
	}
	return path, nil
}

func wrapRollback(label string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", label, err)
}
