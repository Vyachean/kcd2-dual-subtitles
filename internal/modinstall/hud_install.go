package modinstall

import (
	"archive/zip"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/modarchive"
)

var ErrHUDConflict = errors.New("another installed mod overrides Libs/UI/hud.gfx")

// Documents-specific single-language HUD installation remains only as a
// focused legacy test helper. Product code uses the layout-aware multi-language
// installer in multi_target.go.
func installIntoDocumentsVersionedWithHUD(documents string, mainLanguage localization.Language, rows []localization.DialogueRow, hud []byte, version string) (string, error) {
	if documents == "" {
		return "", errors.New("Documents path is empty")
	}
	modsRoot := filepath.Join(documents, ModsDirectoryName)
	conflicts, err := findForeignHUDOverrides(modsRoot)
	if err != nil {
		return "", err
	}
	if len(conflicts) != 0 {
		return "", fmt.Errorf("%w: %s", ErrHUDConflict, strings.Join(conflicts, ", "))
	}
	if err := os.MkdirAll(modsRoot, 0o755); err != nil {
		return "", fmt.Errorf("create KCD2 mod directory %q: %w", modsRoot, err)
	}

	staging, err := os.MkdirTemp(modsRoot, "."+modarchive.ModID+".staging-*")
	if err != nil {
		return "", fmt.Errorf("create staged mod directory in %q: %w", modsRoot, err)
	}
	stagingActive := true
	defer func() {
		if stagingActive {
			_ = os.RemoveAll(staging)
		}
	}()

	if err := modarchive.WriteDirectoryVersionedWithHUD(staging, mainLanguage, rows, hud, version); err != nil {
		return "", fmt.Errorf("build staged HUD mod directory: %w", err)
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
		if err := renamePath(target, backup); err != nil {
			return "", fmt.Errorf("preserve previous mod directory %q: %w", target, err)
		}
		hadPrevious = true
	case errors.Is(statErr, os.ErrNotExist):
	default:
		return "", fmt.Errorf("inspect existing mod path %q: %w", target, statErr)
	}

	if err := renamePath(staging, target); err != nil {
		if hadPrevious {
			if rollbackErr := renamePath(backup, target); rollbackErr != nil {
				return "", errors.Join(
					fmt.Errorf("publish staged HUD mod to %q: %w", target, err),
					fmt.Errorf("rollback previous mod from %q: %w", backup, rollbackErr),
				)
			}
		}
		return "", fmt.Errorf("publish staged HUD mod to %q: %w", target, err)
	}
	stagingActive = false

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

func findForeignHUDOverrides(modsRoot string) ([]string, error) {
	entries, err := os.ReadDir(modsRoot)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("inspect KCD2 mods directory %q for HUD conflicts: %w", modsRoot, err)
	}

	var conflicts []string
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == modarchive.ModID || strings.HasPrefix(entry.Name(), "."+modarchive.ModID+".") {
			continue
		}
		modRoot := filepath.Join(modsRoot, entry.Name())
		if hasLooseHUD(modRoot) {
			conflicts = append(conflicts, entry.Name())
			continue
		}
		hasHUD, err := dataPAKsContainHUD(modRoot)
		if err != nil {
			return nil, fmt.Errorf("inspect mod %q for HUD conflict: %w", entry.Name(), err)
		}
		if hasHUD {
			conflicts = append(conflicts, entry.Name())
		}
	}
	sort.Strings(conflicts)
	return conflicts, nil
}

func hasLooseHUD(modRoot string) bool {
	candidates := []string{
		filepath.Join(modRoot, filepath.FromSlash(modarchive.HUDArchivePath)),
		filepath.Join(modRoot, "Data", filepath.FromSlash(modarchive.HUDArchivePath)),
	}
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			return true
		}
	}
	return false
}

func dataPAKsContainHUD(modRoot string) (bool, error) {
	dataDir := filepath.Join(modRoot, "Data")
	entries, err := os.ReadDir(dataDir)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".pak") {
			continue
		}
		path := filepath.Join(dataDir, entry.Name())
		reader, err := zip.OpenReader(path)
		if err != nil {
			return false, fmt.Errorf("open %q: %w", path, err)
		}
		contains := false
		for _, file := range reader.File {
			name := strings.ReplaceAll(file.Name, "\\", "/")
			if strings.EqualFold(name, modarchive.HUDArchivePath) {
				contains = true
				break
			}
		}
		closeErr := reader.Close()
		if closeErr != nil {
			return false, fmt.Errorf("close %q: %w", path, closeErr)
		}
		if contains {
			return true, nil
		}
	}
	return false, nil
}
