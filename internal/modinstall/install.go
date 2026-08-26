package modinstall

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
)

const (
	ModsDirectoryName = "kingdomcome_mods"
	ModOrderFilename  = "mod_order.txt"
)

var ErrAutomaticInstallUnsupported = errors.New("automatic installation is supported only on Windows; use --output to create a portable ZIP")

var renamePath = os.Rename

// Documents-specific single-language helpers remain only for focused legacy
// filesystem tests. Route them through the same transaction-safe multi-target
// installer so no code path can create a mod-shaped staging directory inside
// the scanned mod root.
func installIntoDocuments(documents string, mainLanguage localization.Language, rows []localization.DialogueRow) (string, error) {
	return installIntoDocumentsVersioned(documents, mainLanguage, rows, "dev")
}

func installIntoDocumentsVersioned(documents string, mainLanguage localization.Language, rows []localization.DialogueRow, version string) (string, error) {
	if documents == "" {
		return "", errors.New("Documents path is empty")
	}
	return installIntoDocumentsVersionedForLanguages(
		documents,
		[]localization.Language{mainLanguage},
		rows,
		nil,
		version,
		false,
	)
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
	updated := modOrderWithEntry(original, modID)

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

// modOrderWithEntry writes exactly one project entry as the final active
// mod_order.txt entry. Unrelated lines retain their original byte order.
func modOrderWithEntry(original []byte, modID string) []byte {
	newline := []byte("\n")
	if bytes.Contains(original, []byte("\r\n")) {
		newline = []byte("\r\n")
	}

	updated, _ := removeModOrderEntries(original, modID)
	if len(updated) > 0 && !bytes.HasSuffix(updated, []byte("\n")) && !bytes.HasSuffix(updated, []byte("\r")) {
		updated = append(updated, newline...)
	}
	updated = append(updated, modID...)
	updated = append(updated, newline...)
	return updated
}

// modOrderContains returns true only when modID occurs exactly once and is the
// final active entry. Comments and blank lines after it do not affect priority.
// This stronger invariant is required because the generated bilingual patch
// must load after every localization source it composed.
func modOrderContains(data []byte, modID string) bool {
	count := 0
	lastActive := ""
	for _, line := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if line == modID {
			count++
		}
		lastActive = line
	}
	return count == 1 && lastActive == modID
}
