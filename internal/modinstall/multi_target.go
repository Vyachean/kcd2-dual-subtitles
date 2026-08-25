package modinstall

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/modarchive"
)

// InstallVersionedForLanguages installs one generated subtitle payload under
// every supplied localization PAK name in the mod root resolved from gameRoot.
func InstallVersionedForLanguages(gameRoot string, targetLanguages []localization.Language, rows []localization.DialogueRow, version string) (string, error) {
	location, err := ResolveInstallLocation(gameRoot)
	if err != nil {
		return "", err
	}
	return installIntoModsRootVersionedForLanguages(location.ModsRoot, targetLanguages, rows, nil, version, false)
}

// InstallVersionedWithHUDForLanguages installs the HUD variant under every
// supplied localization PAK name while retaining the foreign-HUD conflict
// guard in the same resolved mod root.
func InstallVersionedWithHUDForLanguages(gameRoot string, targetLanguages []localization.Language, rows []localization.DialogueRow, hud []byte, version string) (string, error) {
	location, err := ResolveInstallLocation(gameRoot)
	if err != nil {
		return "", err
	}
	return installIntoModsRootVersionedForLanguages(location.ModsRoot, targetLanguages, rows, hud, version, true)
}

// Kept as a focused filesystem helper for the GDK/Documents tests and legacy
// single-root acceptance coverage.
func installIntoDocumentsVersionedForLanguages(documents string, targetLanguages []localization.Language, rows []localization.DialogueRow, hud []byte, version string, withHUD bool) (string, error) {
	if documents == "" {
		return "", errors.New("Documents path is empty")
	}
	return installIntoModsRootVersionedForLanguages(filepath.Join(documents, ModsDirectoryName), targetLanguages, rows, hud, version, withHUD)
}

func installIntoModsRootVersionedForLanguages(modsRoot string, targetLanguages []localization.Language, rows []localization.DialogueRow, hud []byte, version string, withHUD bool) (string, error) {
	modsRoot = strings.TrimSpace(modsRoot)
	if modsRoot == "" {
		return "", errors.New("KCD2 mod root is empty")
	}
	if err := os.MkdirAll(modsRoot, 0o755); err != nil {
		return "", fmt.Errorf("create KCD2 mod directory %q: %w", modsRoot, err)
	}

	releaseLock, err := acquireInstallLock(modsRoot)
	if err != nil {
		return "", err
	}
	defer releaseLock()

	// Repair any interrupted transaction before inspecting conflicts or building
	// another replacement. Transaction workspaces live beside modsRoot and are
	// therefore never visible to KCD2's direct-child mod scan.
	if err := recoverInstallTransactions(modsRoot); err != nil {
		return "", err
	}
	if err := cleanupLegacyToolTempDirs(modsRoot); err != nil {
		return "", err
	}
	if withHUD {
		conflicts, err := findForeignHUDOverrides(modsRoot)
		if err != nil {
			return "", err
		}
		if len(conflicts) != 0 {
			return "", fmt.Errorf("%w: %s", ErrHUDConflict, strings.Join(conflicts, ", "))
		}
	}

	tx, err := beginInstallTransaction(modsRoot)
	if err != nil {
		return "", err
	}
	cleanupTransaction := true
	defer func() {
		if cleanupTransaction {
			_ = os.RemoveAll(tx.root)
		}
	}()

	if withHUD {
		err = modarchive.WriteDirectoryVersionedWithHUDForLanguages(tx.staged, targetLanguages, rows, hud, version)
	} else {
		err = modarchive.WriteDirectoryVersionedForLanguages(tx.staged, targetLanguages, rows, version)
	}
	if err != nil {
		return "", fmt.Errorf("build staged mod directory: %w", err)
	}

	target := filepath.Join(modsRoot, modarchive.ModID)
	hadPrevious := false
	info, statErr := os.Lstat(target)
	switch {
	case statErr == nil:
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("refusing to replace symlink at mod path %q", target)
		}
		if !info.IsDir() {
			return "", fmt.Errorf("refusing to replace non-directory at mod path %q", target)
		}
		if err := tx.markHadPrevious(); err != nil {
			return "", fmt.Errorf("record previous installation in transaction: %w", err)
		}
		if err := renamePathWithRetry(target, tx.previous); err != nil {
			return "", fmt.Errorf("preserve previous mod directory %q: %w", target, err)
		}
		hadPrevious = true
		// tx now owns the only previous installation. Do not delete it unless a
		// rollback succeeds or the replacement reaches committed.
		cleanupTransaction = false
	case errors.Is(statErr, os.ErrNotExist):
	default:
		return "", fmt.Errorf("inspect existing mod path %q: %w", target, statErr)
	}

	// Persist publishing before the first write to the final target. A killed
	// copy fallback can then be distinguished from an untouched installation.
	if err := tx.setState(transactionStatePublishing); err != nil {
		rollbackErr := rollbackInstallTransaction(tx, modsRoot, target, hadPrevious)
		if rollbackErr == nil {
			cleanupTransaction = true
		}
		if rollbackErr != nil {
			return "", errors.Join(err, rollbackErr)
		}
		return "", err
	}
	cleanupTransaction = false

	if err := publishStagedDirectory(tx.staged, target); err != nil {
		rollbackErr := rollbackInstallTransaction(tx, modsRoot, target, hadPrevious)
		if rollbackErr == nil {
			cleanupTransaction = true
		}
		if rollbackErr != nil {
			return "", errors.Join(err, rollbackErr)
		}
		return "", err
	}

	if err := verifyPublishedGeneratedMod(target, targetLanguages, rows, hud, version, withHUD); err != nil {
		verificationErr := fmt.Errorf("verify published generated mod: %w", err)
		rollbackErr := rollbackInstallTransaction(tx, modsRoot, target, hadPrevious)
		if rollbackErr == nil {
			cleanupTransaction = true
		}
		if rollbackErr != nil {
			return "", errors.Join(verificationErr, rollbackErr)
		}
		return "", verificationErr
	}

	if _, err := tx.updateModOrderIfPresent(modsRoot, modarchive.ModID); err != nil {
		updateErr := fmt.Errorf("update %s: %w", ModOrderFilename, err)
		rollbackErr := rollbackInstallTransaction(tx, modsRoot, target, hadPrevious)
		if rollbackErr == nil {
			cleanupTransaction = true
		}
		if rollbackErr != nil {
			return "", errors.Join(updateErr, rollbackErr)
		}
		return "", updateErr
	}

	if err := tx.setState(transactionStateCommitted); err != nil {
		commitErr := fmt.Errorf("commit install transaction: %w", err)
		rollbackErr := rollbackInstallTransaction(tx, modsRoot, target, hadPrevious)
		if rollbackErr == nil {
			cleanupTransaction = true
		}
		if rollbackErr != nil {
			return "", errors.Join(commitErr, rollbackErr)
		}
		return "", commitErr
	}

	// The committed marker is now the source of truth. Disable the generic defer
	// and remove that marker last so termination during cleanup cannot make a
	// committed replacement look like an interrupted publication.
	cleanupTransaction = false
	_ = cleanupCommittedInstallTransaction(tx.root)
	return target, nil
}

func rollbackInstallTransaction(tx *installTransaction, modsRoot, target string, hadPrevious bool) error {
	modErr := rollbackInstalledMod(target, tx.previous, hadPrevious)
	orderErr := tx.restoreModOrder(modsRoot)
	return errors.Join(modErr, orderErr)
}

func verifyPublishedGeneratedMod(target string, targetLanguages []localization.Language, rows []localization.DialogueRow, hud []byte, version string, withHUD bool) error {
	if withHUD {
		return modarchive.VerifyDirectoryVersionedWithHUDForLanguages(target, targetLanguages, rows, hud, version)
	}
	return modarchive.VerifyDirectoryVersionedForLanguages(target, targetLanguages, rows, version)
}
