package application

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSelectedModsLocationRevalidatesCustomRoot(t *testing.T) {
	gameRoot := filepath.Join(t.TempDir(), "game")
	createApplicationGameLayout(t, gameRoot)
	customMods := filepath.Join(t.TempDir(), "custom-mods")
	if err := os.MkdirAll(customMods, 0o755); err != nil {
		t.Fatal(err)
	}

	service := Service{state: &serviceState{}}
	if _, err := service.ValidateGameRoot(gameRoot); err != nil {
		t.Fatal(err)
	}
	if _, err := service.SetModsRootOverride(customMods); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(customMods); err != nil {
		t.Fatal(err)
	}

	_, err := service.SelectedModsLocation()
	if err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("SelectedModsLocation() error = %v, want missing custom-root error", err)
	}
}
