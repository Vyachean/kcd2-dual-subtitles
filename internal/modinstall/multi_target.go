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

	// KCD2 scans every direct child directory of the mod root as a candidate
	// mod, including dot-prefixed directories. A staging directory inside
	// modsRoot can therefore become an enabled duplicate of kcd_dual_subtitles
	// if cleanup is delayed or fails (for example in OneDrive-backed Documents).
	// Stage beside the scanned mod root instead. This remains on the same volume
	// for the normal <game-root>/Mods and GDK <Documents>/kingdomcome_mods layouts,
	// so the preferred rename publication path remains available.
	stagingParent := filepath.Dir(filepath.Clean(modsRoot))
	staging, err := os.MkdirTemp(stagingParent, "."+modarchive.ModID+".staging-*")
	if err != nil {
		return "", fmt.Errorf("create staged mod directory beside %q: %w", modsRoot, err)
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

// v0.3.2 and earlier created .kcd_dual_subtitles.staging-* directly inside
// the scanned mod root. A leaked directory has the same manifest modid as the
// published mod and can shadow it on the next game launch. The prefix is owned
// exclusively by this installer, so remove those legacy orphans before doing
// any conflict scan or publication.
func cleanupLegacyToolTempDirs(modsRoot string) error {
	entries, err := os.ReadDir(modsRoot)
	if err != nil {
		return fmt.Errorf("inspect KCD2 mod root %q for legacy staging directories: %w", modsRoot, err)
	}
	prefix := "." + modarchive.ModID + ".staging-"
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), prefix) {
			continue
		}
		path := filepath.Join(modsRoot, entry.Name())
		info, err := os.Lstat(path)
		if err != nil {
			return fmt.Errorf("inspect legacy staging path %q: %w", path, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing to remove symlink at legacy staging path %q", path)
		}
		if !info.IsDir() {
			return fmt.Errorf("refusing to remove non-directory legacy staging path %q", path)
		}
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("remove orphaned legacy staging directory %q: %w", path, err)
		}
	}
	return nil
}

func verifyPublishedGeneratedMod(target string, targetLanguages []localization.Language, rows []localization.DialogueRow, hud []byte, version string, withHUD bool) error {
	if withHUD {
		return modarchive.VerifyDirectoryVersionedWithHUDForLanguages(target, targetLanguages, rows, hud, version)
	}
	return modarchive.VerifyDirectoryVersionedForLanguages(target, targetLanguages, rows, version)
}
