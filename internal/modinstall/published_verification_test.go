package modinstall

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/modarchive"
)

func TestPublishedArtifactVerificationRollsBackCorruptReplacement(t *testing.T) {
	originalRename := renamePath
	originalCopy := copyPublishPath
	originalSleep := sleepRenameRetry
	defer func() {
		renamePath = originalRename
		copyPublishPath = originalCopy
		sleepRenameRetry = originalSleep
	}()
	sleepRenameRetry = func(_ time.Duration) {}

	modsRoot := t.TempDir()
	target := filepath.Join(modsRoot, modarchive.ModID)
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatalf("create previous mod: %v", err)
	}
	previousSentinel := filepath.Join(target, "previous.txt")
	if err := os.WriteFile(previousSentinel, []byte("previous"), 0o644); err != nil {
		t.Fatalf("write previous sentinel: %v", err)
	}

	renamePath = func(oldPath, newPath string) error {
		base := filepath.Base(oldPath)
		if strings.Contains(base, ".staging-") && !strings.HasSuffix(base, ".previous") {
			return os.ErrPermission
		}
		return os.Rename(oldPath, newPath)
	}
	copyPublishPath = func(source, destination string) error {
		if err := copyDirectoryNoReplace(source, destination); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(destination, "Data", modarchive.DataPAKFilename), []byte("corrupt"), 0o644)
	}

	rows := []localization.DialogueRow{{ID: "line", Text: "styled"}}
	targets := []localization.Language{localization.English, localization.Czech}
	_, err := installIntoModsRootVersionedForLanguages(modsRoot, targets, rows, []byte("derived-hud"), "v-test", true)
	if !errors.Is(err, modarchive.ErrArtifactVerification) {
		t.Fatalf("install error = %v, want ErrArtifactVerification", err)
	}
	if got, readErr := os.ReadFile(previousSentinel); readErr != nil || string(got) != "previous" {
		t.Fatalf("previous installation was not restored: data=%q err=%v", got, readErr)
	}
	if _, statErr := os.Stat(filepath.Join(target, "Data", modarchive.DataPAKFilename)); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("corrupt replacement survived rollback: %v", statErr)
	}
}
