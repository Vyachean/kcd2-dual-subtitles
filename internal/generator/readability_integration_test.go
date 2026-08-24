package generator

import (
	"path/filepath"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/gfxpatch"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
)

func TestGenerateHUDRoutesExplicitReadabilityToConfigurablePatcher(t *testing.T) {
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
		t.Fatal("legacy HUD patcher called while readability is enabled")
		return nil, nil
	}
	patchRetailHUDReadability = func(input []byte, config gfxpatch.HUDReadabilityConfig) ([]byte, error) {
		if string(input) != "retail-hud" {
			t.Fatalf("readability patch input = %q, want retail-hud", input)
		}
		if !config.Outline || !config.Shadow {
			t.Fatalf("readability config = %+v, want outline+shadow", config)
		}
		return []byte("readable-hud"), nil
	}

	presentation := DefaultHUDPresentationConfig()
	presentation.Outline = true
	presentation.Shadow = true

	gameRoot := createGameRoot(t, true, true)
	_, err := Generate(Request{
		GameRoot:          gameRoot,
		MainLanguage:      localization.Russian,
		SecondaryLanguage: localization.English,
		SubtitleStyle:     SubtitleStyleHUD,
		HUDPresentation:   &presentation,
		OutputPath:        filepath.Join(t.TempDir(), "readability.zip"),
		Version:           "v0.3.0-test",
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
}

func TestGenerateHUDWithoutReadabilityKeepsLegacyPatcher(t *testing.T) {
	originalRead := readRetailHUD
	originalPatch := patchRetailHUD
	originalReadabilityPatch := patchRetailHUDReadability
	defer func() {
		readRetailHUD = originalRead
		patchRetailHUD = originalPatch
		patchRetailHUDReadability = originalReadabilityPatch
	}()

	readRetailHUD = func(string) ([]byte, error) { return []byte("retail-hud"), nil }
	patchRetailHUD = func(input []byte) ([]byte, error) {
		if string(input) != "retail-hud" {
			t.Fatalf("legacy patch input = %q, want retail-hud", input)
		}
		return []byte("legacy-hud"), nil
	}
	patchRetailHUDReadability = func([]byte, gfxpatch.HUDReadabilityConfig) ([]byte, error) {
		t.Fatal("readability HUD patcher called for default presentation")
		return nil, nil
	}

	gameRoot := createGameRoot(t, true, true)
	_, err := Generate(Request{
		GameRoot:          gameRoot,
		MainLanguage:      localization.Russian,
		SecondaryLanguage: localization.English,
		SubtitleStyle:     SubtitleStyleHUD,
		OutputPath:        filepath.Join(t.TempDir(), "legacy.zip"),
		Version:           "v0.3.0-test",
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
}
