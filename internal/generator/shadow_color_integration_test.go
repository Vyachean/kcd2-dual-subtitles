package generator

import (
	"path/filepath"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/gfxpatch"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
)

func TestGenerateHUDPassesConfiguredShadowColorToReadabilityPatcher(t *testing.T) {
	originalRead := readRetailHUD
	originalPatch := patchRetailHUD
	originalReadabilityPatch := patchRetailHUDReadability
	defer func() {
		readRetailHUD = originalRead
		patchRetailHUD = originalPatch
		patchRetailHUDReadability = originalReadabilityPatch
	}()

	readRetailHUD = func(string) ([]byte, error) { return []byte("retail-hud"), nil }
	patchRetailHUD = func([]byte) ([]byte, error) {
		t.Fatal("legacy HUD patcher called while shadow is enabled")
		return nil, nil
	}
	patchRetailHUDReadability = func(input []byte, config gfxpatch.HUDReadabilityConfig) ([]byte, error) {
		if string(input) != "retail-hud" {
			t.Fatalf("readability patch input = %q, want retail-hud", input)
		}
		if !config.Shadow || config.Outline {
			t.Fatalf("readability config = %+v, want shadow-only", config)
		}
		if config.ShadowColor != 0x123456 {
			t.Fatalf("ShadowColor = %#x, want 0x123456", config.ShadowColor)
		}
		return []byte("shadow-colored-hud"), nil
	}

	presentation := DefaultHUDPresentationConfig()
	presentation.Shadow = true
	presentation.ShadowColor = "#123456"

	gameRoot := createGameRoot(t, true, true)
	_, err := Generate(Request{
		GameRoot:          gameRoot,
		MainLanguage:      localization.Russian,
		SecondaryLanguage: localization.English,
		SubtitleStyle:     SubtitleStyleHUD,
		HUDPresentation:   &presentation,
		OutputPath:        filepath.Join(t.TempDir(), "shadow-color.zip"),
		Version:           "shadow-color-test",
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
}
