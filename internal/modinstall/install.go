package modinstall

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/modarchive"
)

const ModsDirectoryName = "kingdomcome_mods"

var ErrAutomaticInstallUnsupported = errors.New("automatic installation is supported only on Windows; use --output to create a portable ZIP")

var renamePath = os.Rename

// Install writes the generated mod into the current Windows user's KCD2 mod
// directory and returns the installed mod directory.
func Install(mainLanguage localization.Language, rows []localization.DialogueRow) (string, error) {
	documents, err := documentsPath()
	if err != nil {
		return "", err
	}
	return installIntoDocuments(documents, mainLanguage, rows)
}

func installIntoDocuments(documents string, mainLanguage localization.Language, rows []localization.DialogueRow) (string, error) {
	if documents == "" {
		return "", errors.New("Documents path is empty")
	}

	modsRoot := filepath.Join(documents, ModsDirectoryName)
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

	if err := modarchive.WriteDirectory(staging, mainLanguage, rows); err != nil {
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
		if err := renamePath(target, backup); err != nil {
			return "", fmt.Errorf("preserve previous mod directory %q: %w", target, err)
		}
		hadPrevious = true
	case errors.Is(statErr, os.ErrNotExist):
		// No previous installation.
	default:
		return "", fmt.Errorf("inspect existing mod path %q: %w", target, statErr)
	}

	if err := renamePath(staging, target); err != nil {
		if hadPrevious {
			if rollbackErr := renamePath(backup, target); rollbackErr != nil {
				return "", errors.Join(
					fmt.Errorf("publish staged mod to %q: %w", target, err),
					fmt.Errorf("rollback previous mod from %q: %w", backup, rollbackErr),
				)
			}
		}
		return "", fmt.Errorf("publish staged mod to %q: %w", target, err)
	}
	stagingActive = false

	if hadPrevious {
		_ = os.RemoveAll(backup)
	}

	return target, nil
}
