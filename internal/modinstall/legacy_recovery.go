package modinstall

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/modarchive"
)

const (
	legacyStagingPrefix      = "." + modarchive.ModID + ".staging-"
	legacyPreviousSuffix     = ".previous"
	legacyModOrderTempPrefix = ".mod_order.txt.tmp-"
	legacyModOrderPrevPrefix = ".mod_order.txt.previous-"
)

// cleanupLegacyToolTempDirs migrates crash residue produced by v0.3.2 and
// earlier before a new transaction starts. Legacy backups are recovered rather
// than deleted blindly: they may contain the only complete previous install.
func cleanupLegacyToolTempDirs(modsRoot string) error {
	if err := recoverLegacyInstallDirectories(modsRoot); err != nil {
		return err
	}
	return recoverLegacyModOrderFiles(modsRoot)
}

func recoverLegacyInstallDirectories(modsRoot string) error {
	entries, err := os.ReadDir(modsRoot)
	if err != nil {
		return fmt.Errorf("inspect KCD2 mod root %q for legacy install state: %w", modsRoot, err)
	}

	var legacyPaths []string
	var previousPaths []string
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), legacyStagingPrefix) {
			continue
		}
		path := filepath.Join(modsRoot, entry.Name())
		info, err := os.Lstat(path)
		if err != nil {
			return fmt.Errorf("inspect legacy staging path %q: %w", path, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing to recover symlink at legacy staging path %q", path)
		}
		if !info.IsDir() {
			return fmt.Errorf("refusing to recover non-directory legacy staging path %q", path)
		}
		legacyPaths = append(legacyPaths, path)
		if strings.HasSuffix(entry.Name(), legacyPreviousSuffix) {
			previousPaths = append(previousPaths, path)
		}
	}

	if len(previousPaths) > 1 {
		return fmt.Errorf("ambiguous legacy install recovery: found %d previous-install backups under %q", len(previousPaths), modsRoot)
	}
	if len(previousPaths) == 1 {
		target := filepath.Join(modsRoot, modarchive.ModID)
		if err := removeRecoverableTarget(target); err != nil {
			return fmt.Errorf("prepare legacy previous-install recovery: %w", err)
		}
		if err := renamePathWithRetry(previousPaths[0], target); err != nil {
			return fmt.Errorf("restore legacy previous installation from %q: %w", previousPaths[0], err)
		}
	}

	for _, path := range legacyPaths {
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("remove legacy staging residue %q: %w", path, err)
		}
	}
	return nil
}

func recoverLegacyModOrderFiles(modsRoot string) error {
	entries, err := os.ReadDir(modsRoot)
	if err != nil {
		return fmt.Errorf("inspect KCD2 mod root %q for legacy load-order state: %w", modsRoot, err)
	}

	var tempPaths []string
	var previousPaths []string
	for _, entry := range entries {
		isTemp := strings.HasPrefix(entry.Name(), legacyModOrderTempPrefix)
		isPrevious := strings.HasPrefix(entry.Name(), legacyModOrderPrevPrefix)
		if !isTemp && !isPrevious {
			continue
		}
		path := filepath.Join(modsRoot, entry.Name())
		info, err := os.Lstat(path)
		if err != nil {
			return fmt.Errorf("inspect legacy load-order path %q: %w", path, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return fmt.Errorf("refusing to recover non-regular legacy load-order path %q", path)
		}
		if isPrevious {
			previousPaths = append(previousPaths, path)
		} else {
			tempPaths = append(tempPaths, path)
		}
	}

	orderPath := filepath.Join(modsRoot, ModOrderFilename)
	orderInfo, orderErr := os.Lstat(orderPath)
	switch {
	case orderErr == nil:
		if orderInfo.Mode()&os.ModeSymlink != 0 || !orderInfo.Mode().IsRegular() {
			return fmt.Errorf("refusing to recover with invalid load order at %q", orderPath)
		}
	case errors.Is(orderErr, os.ErrNotExist):
		if len(previousPaths) > 1 {
			return fmt.Errorf("ambiguous legacy load-order recovery: found %d previous files under %q", len(previousPaths), modsRoot)
		}
		if len(previousPaths) == 1 {
			if err := renamePathWithRetry(previousPaths[0], orderPath); err != nil {
				return fmt.Errorf("restore legacy previous load order from %q: %w", previousPaths[0], err)
			}
		}
	default:
		return fmt.Errorf("inspect load order %q during legacy recovery: %w", orderPath, orderErr)
	}

	for _, path := range append(tempPaths, previousPaths...) {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove legacy load-order residue %q: %w", path, err)
		}
	}
	return nil
}
