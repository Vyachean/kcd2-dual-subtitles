package modinstall

import (
	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
)

// InstallVersionedForLanguagesInModsRoot publishes directly into an
// already-selected Mods root. It is used when the application has an explicit
// user override and must not resolve a different location from gameRoot.
func InstallVersionedForLanguagesInModsRoot(modsRoot string, targetLanguages []localization.Language, rows []localization.DialogueRow, version string) (string, error) {
	return installIntoModsRootVersionedForLanguages(modsRoot, targetLanguages, rows, nil, version, false)
}

// InstallVersionedWithHUDForLanguagesInModsRoot is the HUD equivalent of
// InstallVersionedForLanguagesInModsRoot and retains the same conflict and
// transaction guards as the normal installer.
func InstallVersionedWithHUDForLanguagesInModsRoot(modsRoot string, targetLanguages []localization.Language, rows []localization.DialogueRow, hud []byte, version string) (string, error) {
	return installIntoModsRootVersionedForLanguages(modsRoot, targetLanguages, rows, hud, version, true)
}

// UninstallFromModsRoot removes only this project from an already-selected Mods
// root. This keeps custom-root uninstall aligned with generation and status.
func UninstallFromModsRoot(modsRoot string) (UninstallResult, error) {
	return uninstallFromModsRoot(modsRoot)
}
