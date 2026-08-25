package modinstall

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/modarchive"
)

func TestLegacyRecoveryRestoresOnlyPreviousInstallOverPartialTarget(t *testing.T) {
	modsRoot := t.TempDir()
	target := filepath.Join(modsRoot, modarchive.ModID)
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatalf("create partial target: %v", err)
	}
	if err := os.WriteFile(filepath.Join(target, "sentinel.txt"), []byte("partial"), 0o644); err != nil {
		t.Fatalf("write partial target: %v", err)
	}

	staging := filepath.Join(modsRoot, legacyStagingPrefix+"123")
	previous := staging + legacyPreviousSuffix
	if err := os.Mkdir(staging, 0o755); err != nil {
		t.Fatalf("create legacy staging: %v", err)
	}
	if err := os.Mkdir(previous, 0o755); err != nil {
		t.Fatalf("create legacy previous: %v", err)
	}
	if err := os.WriteFile(filepath.Join(previous, "sentinel.txt"), []byte("previous"), 0o644); err != nil {
		t.Fatalf("write previous sentinel: %v", err)
	}

	if err := cleanupLegacyToolTempDirs(modsRoot); err != nil {
		t.Fatalf("cleanupLegacyToolTempDirs() error = %v", err)
	}
	got, err := os.ReadFile(filepath.Join(target, "sentinel.txt"))
	if err != nil || string(got) != "previous" {
		t.Fatalf("recovered target = %q err=%v, want previous", got, err)
	}
	if _, err := os.Stat(staging); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("legacy staging survived recovery: %v", err)
	}
	if _, err := os.Stat(previous); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("legacy previous path survived recovery: %v", err)
	}
}

func TestLegacyRecoveryRestoresPreviousInstallWhenCanonicalTargetIsMissing(t *testing.T) {
	modsRoot := t.TempDir()
	previous := filepath.Join(modsRoot, legacyStagingPrefix+"123"+legacyPreviousSuffix)
	if err := os.Mkdir(previous, 0o755); err != nil {
		t.Fatalf("create legacy previous: %v", err)
	}
	if err := os.WriteFile(filepath.Join(previous, "sentinel.txt"), []byte("previous"), 0o644); err != nil {
		t.Fatalf("write previous sentinel: %v", err)
	}

	if err := cleanupLegacyToolTempDirs(modsRoot); err != nil {
		t.Fatalf("cleanupLegacyToolTempDirs() error = %v", err)
	}
	target := filepath.Join(modsRoot, modarchive.ModID)
	got, err := os.ReadFile(filepath.Join(target, "sentinel.txt"))
	if err != nil || string(got) != "previous" {
		t.Fatalf("recovered target = %q err=%v, want previous", got, err)
	}
}

func TestLegacyRecoveryFailsClosedWithMultiplePreviousInstalls(t *testing.T) {
	modsRoot := t.TempDir()
	for _, suffix := range []string{"123", "456"} {
		path := filepath.Join(modsRoot, legacyStagingPrefix+suffix+legacyPreviousSuffix)
		if err := os.Mkdir(path, 0o755); err != nil {
			t.Fatalf("create legacy previous %q: %v", path, err)
		}
	}

	err := cleanupLegacyToolTempDirs(modsRoot)
	if err == nil || !strings.Contains(err.Error(), "ambiguous legacy install recovery") {
		t.Fatalf("cleanupLegacyToolTempDirs() error = %v, want ambiguous recovery", err)
	}
	for _, suffix := range []string{"123", "456"} {
		path := filepath.Join(modsRoot, legacyStagingPrefix+suffix+legacyPreviousSuffix)
		if info, statErr := os.Stat(path); statErr != nil || !info.IsDir() {
			t.Fatalf("ambiguous backup was modified: path=%q info=%v err=%v", path, info, statErr)
		}
	}
}

func TestLegacyRecoveryRestoresMissingLoadOrderAndRemovesTemporaryFile(t *testing.T) {
	modsRoot := t.TempDir()
	previous := filepath.Join(modsRoot, legacyModOrderPrevPrefix+"123")
	original := []byte("first_mod\r\nsecond_mod\r\n")
	if err := os.WriteFile(previous, original, 0o600); err != nil {
		t.Fatalf("write legacy previous load order: %v", err)
	}
	previousInfo, err := os.Stat(previous)
	if err != nil {
		t.Fatalf("inspect legacy previous load order: %v", err)
	}
	previousMode := previousInfo.Mode().Perm()
	temporary := filepath.Join(modsRoot, legacyModOrderTempPrefix+"456")
	if err := os.WriteFile(temporary, []byte("partial"), 0o644); err != nil {
		t.Fatalf("write legacy temporary load order: %v", err)
	}

	if err := cleanupLegacyToolTempDirs(modsRoot); err != nil {
		t.Fatalf("cleanupLegacyToolTempDirs() error = %v", err)
	}
	orderPath := filepath.Join(modsRoot, ModOrderFilename)
	got, err := os.ReadFile(orderPath)
	if err != nil || string(got) != string(original) {
		t.Fatalf("restored load order = %q err=%v, want %q", got, err, original)
	}
	info, err := os.Stat(orderPath)
	if err != nil || info.Mode().Perm() != previousMode {
		t.Fatalf("restored load-order mode = %v err=%v, want %v", info, err, previousMode)
	}
	if _, err := os.Stat(temporary); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("legacy temporary load order survived: %v", err)
	}
}

func TestLegacyRecoveryKeepsCurrentLoadOrderAndCleansResidue(t *testing.T) {
	modsRoot := t.TempDir()
	orderPath := filepath.Join(modsRoot, ModOrderFilename)
	current := []byte("current\n")
	if err := os.WriteFile(orderPath, current, 0o644); err != nil {
		t.Fatalf("write current load order: %v", err)
	}
	previous := filepath.Join(modsRoot, legacyModOrderPrevPrefix+"123")
	if err := os.WriteFile(previous, []byte("old\n"), 0o644); err != nil {
		t.Fatalf("write stale previous load order: %v", err)
	}
	temporary := filepath.Join(modsRoot, legacyModOrderTempPrefix+"456")
	if err := os.WriteFile(temporary, []byte("temporary\n"), 0o644); err != nil {
		t.Fatalf("write stale temporary load order: %v", err)
	}

	if err := cleanupLegacyToolTempDirs(modsRoot); err != nil {
		t.Fatalf("cleanupLegacyToolTempDirs() error = %v", err)
	}
	got, err := os.ReadFile(orderPath)
	if err != nil || string(got) != string(current) {
		t.Fatalf("current load order changed: data=%q err=%v", got, err)
	}
	for _, path := range []string{previous, temporary} {
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("legacy load-order residue survived at %q: %v", path, err)
		}
	}
}
