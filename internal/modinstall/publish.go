package modinstall

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

const renameRetryAttempts = 6

var (
	sleepRenameRetry = time.Sleep
	copyPublishPath  = copyDirectoryNoReplace
)

func renamePathWithRetry(oldPath, newPath string) error {
	var err error
	for attempt := 0; attempt < renameRetryAttempts; attempt++ {
		err = renamePath(oldPath, newPath)
		if err == nil || !retryableRenameError(err) {
			return err
		}
		if attempt+1 < renameRetryAttempts {
			sleepRenameRetry(time.Duration(attempt+1) * 100 * time.Millisecond)
		}
	}
	return err
}

func retryableRenameError(err error) bool {
	return errors.Is(err, os.ErrPermission) || platformRetryableRenameError(err)
}

// publishStagedDirectory prefers the existing same-volume directory rename.
// Windows cloud-backed Documents folders (notably OneDrive) can transiently
// deny that rename even after allowing the staging tree to be created and
// populated. After bounded retries, only retryable permission/sharing failures
// fall back to a guarded copy into an absent target. The caller keeps any
// previous installation backup until load-order publication also succeeds.
func publishStagedDirectory(staging, target string) error {
	renameErr := renamePathWithRetry(staging, target)
	if renameErr == nil {
		return nil
	}
	if !retryableRenameError(renameErr) {
		return fmt.Errorf("publish staged mod to %q: %w", target, renameErr)
	}

	if copyErr := copyPublishPath(staging, target); copyErr != nil {
		return errors.Join(
			fmt.Errorf("publish staged mod to %q after retrying rename: %w", target, renameErr),
			fmt.Errorf("copy staged mod into %q: %w", target, copyErr),
		)
	}
	return nil
}

func copyDirectoryNoReplace(source, target string) (err error) {
	if info, statErr := os.Lstat(target); statErr == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing to copy over symlink at %q", target)
		}
		return fmt.Errorf("refusing to copy over existing path %q", target)
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return fmt.Errorf("inspect copy target %q: %w", target, statErr)
	}

	rootInfo, err := os.Stat(source)
	if err != nil {
		return fmt.Errorf("inspect staged source %q: %w", source, err)
	}
	if !rootInfo.IsDir() {
		return fmt.Errorf("staged source is not a directory: %q", source)
	}
	if err := os.Mkdir(target, rootInfo.Mode().Perm()); err != nil {
		return fmt.Errorf("create copy target %q: %w", target, err)
	}
	published := false
	defer func() {
		if !published {
			if cleanupErr := os.RemoveAll(target); cleanupErr != nil && err == nil {
				err = fmt.Errorf("clean failed copy target %q: %w", target, cleanupErr)
			}
		}
	}()

	err = filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == source {
			return nil
		}
		relative, relErr := filepath.Rel(source, path)
		if relErr != nil {
			return relErr
		}
		destination := filepath.Join(target, relative)
		info, infoErr := entry.Info()
		if infoErr != nil {
			return infoErr
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing symlink in staged tree at %q", path)
		}
		if entry.IsDir() {
			return os.Mkdir(destination, info.Mode().Perm())
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported staged file type at %q", path)
		}
		return copyFileNoReplace(path, destination, info.Mode().Perm())
	})
	if err != nil {
		return fmt.Errorf("copy staged tree: %w", err)
	}
	published = true
	return nil
}

func copyFileNoReplace(source, target string, mode fs.FileMode) (err error) {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()

	output, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	closed := false
	defer func() {
		if !closed {
			_ = output.Close()
		}
	}()
	if _, err := io.Copy(output, input); err != nil {
		return err
	}
	if err := output.Sync(); err != nil {
		return err
	}
	if err := output.Close(); err != nil {
		return err
	}
	closed = true
	return nil
}
