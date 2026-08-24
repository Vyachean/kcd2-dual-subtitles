//go:build windows

package modinstall

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/modarchive"
)

func TestStandardGameRootInstallInspectUninstallLifecycle(t *testing.T) {
	gameRoot := filepath.Join(t.TempDir(), "KingdomComeDeliverance2")
	if err := os.MkdirAll(gameRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	rows := []localization.DialogueRow{{ID: "line", Text: "bilingual"}}
	languages := []localization.Language{localization.English, localization.German}

	installed, err := InstallVersionedForLanguages(gameRoot, languages, rows, "v0.3.2-test")
	if err != nil {
		t.Fatalf("InstallVersionedForLanguages() error = %v", err)
	}
	want := filepath.Join(gameRoot, "Mods", modarchive.ModID)
	if installed != want {
		t.Fatalf("installed path = %q, want %q", installed, want)
	}

	status, err := InspectForGameRoot(gameRoot)
	if err != nil {
		t.Fatalf("InspectForGameRoot() error = %v", err)
	}
	if !status.Installed || status.Path != want {
		t.Fatalf("status = %+v, want installed at %q", status, want)
	}

	result, err := UninstallForGameRoot(gameRoot)
	if err != nil {
		t.Fatalf("UninstallForGameRoot() error = %v", err)
	}
	if !result.RemovedMod || result.Path != want {
		t.Fatalf("uninstall result = %+v, want removed %q", result, want)
	}
	if _, err := os.Stat(want); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("mod still exists after uninstall: %v", err)
	}
}
