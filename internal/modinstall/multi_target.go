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
	if withHUD {
		conflicts, err := findForeignHUDOverrides(modsRoot)
		if err != nil {
			return "", err
		}
		if len(conflicts) != 0 {
			return "", fmt.Errorf("%w: %s", ErrHUDConflict, strings.Join(conflicts, ", "))
		}
	}
	if err := os.MkdirAll(modsRoot, 0o755); err != nil {
		return "", fmt.Errorf("create KCD2 mod directory %q: %w", modsRoot, err)
	}

	staging, err := os.MkdirTemp(modsRoot, "."+modarchive.ModID+".staging-*")
	if err != nil {
		return "", fmt.Errorf("create staged mod directory in %q: %w", modsRoot, err)
	}
	defer func() { _ = os.RemoveAll(staging) }()

	if withHUD {
		err = modarchive.WriteDirectoryVersionedWithHUDForLanguages(staging, targetLanguages, rows, hud, version)
	} else {
		err = modarchive.WriteDirectoryVersionedForLanguages(staging, targetLanguages, rows, version)
	}
	if err != nil {
		return "", fmt.Errorf("build staged mod directory: %w", err)
	}

	target := filepath.Join(modsRoot, modarchive.ModID)
	backup := staging + ".previous"
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
		if err := renamePathWithRetry(target, backup); err != nil {
			return "", fmt.Errorf("preserve previous mod directory %q: %w", target, err)
		}
		hadPrevious = true
	case errors.Is(statErr, os.ErrNotExist):
	default:
		return "", fmt.Errorf("inspect existing mod path %q: %w", target, statErr)
	}

	if err := publishStagedDirectory(staging, target); err != nil {
		rollbackErr := rollbackInstalledMod(target, backup, hadPrevious)
		if rollbackErr != nil {
			return "", errors.Join(err, rollbackErr)
		}
		return "", err
	}

	if err := verifyPublishedGeneratedMod(target, targetLanguages, rows, hud, version, withHUD); err != nil {
		rollbackErr := rollbackInstalledMod(target, backup, hadPrevious)
		verificationErr := fmt.Errorf("verify published generated mod: %w", err)
		if rollbackErr != nil {
			return "", errors.Join(verificationErr, rollbackErr)
		}
		return "", verificationErr
	}

	if err := ensureModOrderContains(modsRoot, modarchive.ModID); err != nil {
		rollbackErr := rollbackInstalledMod(target, backup, hadPrevious)
		if rollbackErr != nil {
			return "", errors.Join(fmt.Errorf("update %s: %w", ModOrderFilename, err), rollbackErr)
		}
		return "", fmt.Errorf("update %s: %w", ModOrderFilename, err)
	}
	if hadPrevious {
		_ = os.RemoveAll(backup)
	}
	return target, nil
}

func verifyPublishedGeneratedMod(target string, targetLanguages []localization.Language, rows []localization.DialogueRow, hud []byte, version string, withHUD bool) error {
	if withHUD {
		return modarchive.VerifyDirectoryVersionedWithHUDForLanguages(target, targetLanguages, rows, hud, version)
	}
	return modarchive.VerifyDirectoryVersionedForLanguages(target, targetLanguages, rows, version)
}
