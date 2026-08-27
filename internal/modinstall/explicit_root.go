package modinstall

import (
	"runtime"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
)

// InstallVersionedForLanguagesInModsRoot publishes directly into an
// already-selected custom Mods root. Custom-root operations deliberately ignore
// legacy unowned transaction workspaces because older public releases could
// only have created those for their layout-resolved automatic Mods root.
func InstallVersionedForLanguagesInModsRoot(modsRoot string, targetLanguages []localization.Language, rows []localization.DialogueRow, version string) (string, error) {
	if runtime.GOOS != "windows" {
		return "", ErrAutomaticInstallUnsupported
	}
	return installIntoModsRootVersionedForLanguagesWithLegacyRecovery(modsRoot, targetLanguages, rows, nil, version, false, false)
}

// InstallVersionedWithHUDForLanguagesInModsRoot is the HUD equivalent of
// InstallVersionedForLanguagesInModsRoot and retains the same conflict and
// transaction guards as the normal installer.
func InstallVersionedWithHUDForLanguagesInModsRoot(modsRoot string, targetLanguages []localization.Language, rows []localization.DialogueRow, hud []byte, version string) (string, error) {
	if runtime.GOOS != "windows" {
		return "", ErrAutomaticInstallUnsupported
	}
	return installIntoModsRootVersionedForLanguagesWithLegacyRecovery(modsRoot, targetLanguages, rows, hud, version, true, false)
}

// UninstallFromModsRoot removes only this project from an already-selected
// custom Mods root using the same transaction ownership policy as generation.
func UninstallFromModsRoot(modsRoot string) (UninstallResult, error) {
	if runtime.GOOS != "windows" {
		return UninstallResult{}, ErrAutomaticInstallUnsupported
	}
	return uninstallFromModsRootWithLegacyRecovery(modsRoot, false)
}
