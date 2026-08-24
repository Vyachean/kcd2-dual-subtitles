package modinstall

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/modarchive"
)

const (
	ModsDirectoryName = "kingdomcome_mods"
	ModOrderFilename  = "mod_order.txt"
)

var ErrAutomaticInstallUnsupported = errors.New("automatic installation is supported only on Windows; use --output to create a portable ZIP")

var renamePath = os.Rename

// Documents-specific single-language helpers remain only for focused legacy
// filesystem tests. Product code must enter through the layout-aware
// multi-language installer in multi_target.go.
func installIntoDocuments(documents string, mainLanguage localization.Language, rows []localization.DialogueRow) (string, error) {
	return installIntoDocumentsVersioned(documents, mainLanguage, rows, "dev")
}

func installIntoDocumentsVersioned(documents string, mainLanguage localization.Language, rows []localization.DialogueRow, version string) (string, error) {
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
	defer func() { _ = os.RemoveAll(staging) }()

	if err := modarchive.WriteDirectoryVersioned(staging, mainLanguage, rows, version); err != nil {
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
		// No previous installation.
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

	if err := ensureModOrderContains(modsRoot, modarchive.ModID); err != nil {
		rollbackErr := rollbackInstalledMod(target, backup, hadPrevious)
		if rollbackErr != nil {
			return "", errors.Join(
				fmt.Errorf("update %s: %w", ModOrderFilename, err),
				rollbackErr,
			)
		}
		return "", fmt.Errorf("update %s: %w", ModOrderFilename, err)
	}

	if hadPrevious {
		_ = os.RemoveAll(backup)
	}

	return target, nil
}

func rollbackInstalledMod(target, backup string, hadPrevious bool) error {
	if err := os.RemoveAll(target); err != nil {
		return fmt.Errorf("remove failed replacement at %q: %w", target, err)
	}
	if hadPrevious {
		if err := renamePathWithRetry(backup, target); err != nil {
			return fmt.Errorf("restore previous mod from %q: %w", backup, err)
		}
	}
	return nil
}

func ensureModOrderContains(modsRoot, modID string) error {
	path := filepath.Join(modsRoot, ModOrderFilename)
	original, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %q: %w", path, err)
	}
	if modOrderContains(original, modID) {
		return nil
	}

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("inspect %q: %w", path, err)
	}
	newline := []byte("\n")
	if bytes.Contains(original, []byte("\r\n")) {
		newline = []byte("\r\n")
	}

	updated := append([]byte(nil), original...)
	if len(updated) > 0 && !bytes.HasSuffix(updated, []byte("\n")) && !bytes.HasSuffix(updated, []byte("\r")) {
		updated = append(updated, newline...)
	}
	updated = append(updated, modID...)
	updated = append(updated, newline...)

	temporary, err := os.CreateTemp(modsRoot, ".mod_order.txt.tmp-*")
	if err != nil {
		return fmt.Errorf("create temporary load order: %w", err)
	}
	temporaryPath := temporary.Name()
	temporaryClosed := false
	defer func() {
		if !temporaryClosed {
			_ = temporary.Close()
		}
		_ = os.Remove(temporaryPath)
	}()

	if err := temporary.Chmod(info.Mode().Perm()); err != nil {
		return fmt.Errorf("set temporary load-order permissions: %w", err)
	}
	if _, err := temporary.Write(updated); err != nil {
		return fmt.Errorf("write temporary load order: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		return fmt.Errorf("sync temporary load order: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary load order: %w", err)
	}
	temporaryClosed = true

	backupPlaceholder, err := os.CreateTemp(modsRoot, ".mod_order.txt.previous-*")
	if err != nil {
		return fmt.Errorf("reserve load-order backup path: %w", err)
	}
	backupPath := backupPlaceholder.Name()
	if err := backupPlaceholder.Close(); err != nil {
		return fmt.Errorf("close load-order backup placeholder: %w", err)
	}
	if err := os.Remove(backupPath); err != nil {
		return fmt.Errorf("prepare load-order backup path: %w", err)
	}

	if err := renamePathWithRetry(path, backupPath); err != nil {
		return fmt.Errorf("preserve previous load order: %w", err)
	}
	if err := renamePathWithRetry(temporaryPath, path); err != nil {
		if rollbackErr := renamePathWithRetry(backupPath, path); rollbackErr != nil {
			return errors.Join(
				fmt.Errorf("publish updated load order: %w", err),
				fmt.Errorf("rollback previous load order: %w", rollbackErr),
			)
		}
		return fmt.Errorf("publish updated load order: %w", err)
	}
	_ = os.Remove(backupPath)
	return nil
}

func modOrderContains(data []byte, modID string) bool {
	for _, line := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		if strings.TrimSpace(line) == modID {
			return true
		}
	}
	return false
}
