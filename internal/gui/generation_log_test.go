package gui

import (
	"errors"
	"strings"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/generator"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
)

func TestFormatGenerationStartedExplainsInputs(t *testing.T) {
	log := formatGenerationStarted(generationLogContext{
		GameRoot:  `C:\Games\KCD2\Content`,
		ModsRoot:  `C:\Users\Player\Documents\kingdomcome_mods`,
		Main:      localization.Russian,
		Secondary: localization.ChineseSimplified,
		Styled:    false,
	})

	for _, want := range []string{
		"Generation started",
		"Main: Russian",
		"Secondary: Chinese (Simplified)",
		`Mods folder: C:\Users\Player\Documents\kingdomcome_mods`,
		"Mode: game-default appearance",
		"Reading stock localization and active localization mods",
		"installed only for the selected Main and Secondary languages",
	} {
		if !strings.Contains(log, want) {
			t.Fatalf("log = %q, want %q", log, want)
		}
	}
}

func TestFormatGenerationSucceededShowsCompositionAndInstall(t *testing.T) {
	context := generationLogContext{
		ModsRoot:  `D:\KCD2Mods`,
		Main:      localization.Russian,
		Secondary: localization.ChineseSimplified,
		Styled:    true,
	}
	result := generator.Result{
		Stats: localization.MergeStats{
			Processed: 177930,
			Bilingual: 170000,
		},
		PatchRows:                      170000,
		HUDOverride:                    true,
		LocalizationTargets:            2,
		MainLocalizationOverrides:      []string{"Russian Fix"},
		SecondaryLocalizationOverrides: []string{"Chineses Fix"},
		InstallPath:                    `D:\KCD2Mods\kcd_dual_subtitles`,
	}

	log := formatGenerationSucceeded(context, result)
	for _, want := range []string{
		"Generation completed",
		"stock Russian localization",
		"Applied localization mods: Russian Fix",
		"stock Chinese (Simplified) localization",
		"Applied localization mods: Chineses Fix",
		"Merged 177930 main dialogue rows (170000 bilingual)",
		"Generated 170000 changed dialogue rows for 2 selected localization targets",
		"Custom subtitle appearance prepared",
		"Localization load order verified",
		`Installed: D:\KCD2Mods\kcd_dual_subtitles`,
		"Use one of the selected languages as KCD2's text language",
		"regenerate after switching the game to another language",
		"Restart KCD2 before testing",
	} {
		if !strings.Contains(log, want) {
			t.Fatalf("log = %q, want %q", log, want)
		}
	}
}

func TestFormatGenerationSucceededExplainsStockOnlySource(t *testing.T) {
	log := formatGenerationSucceeded(generationLogContext{
		Main:      localization.Russian,
		Secondary: localization.English,
	}, generator.Result{})

	if got := strings.Count(log, "No installed localization mod changed dialogue."); got != 2 {
		t.Fatalf("stock-only message count = %d, want 2; log = %q", got, log)
	}
	if !strings.Contains(log, "Game HUD left unchanged.") {
		t.Fatalf("log = %q, want game HUD unchanged message", log)
	}
}

func TestFormatGenerationFailedKeepsInputsAndErrorWithoutSuccessClaims(t *testing.T) {
	log := formatGenerationFailed(generationLogContext{
		ModsRoot:  `D:\KCD2Mods`,
		Main:      localization.Russian,
		Secondary: localization.English,
	}, errors.New("another installed mod overrides Libs/UI/hud.gfx"))

	for _, want := range []string{
		"Generation failed",
		"Main: Russian",
		"Secondary: English",
		"another installed mod overrides Libs/UI/hud.gfx",
		"Generation did not complete successfully",
		"recovered by the next install or uninstall operation",
	} {
		if !strings.Contains(log, want) {
			t.Fatalf("log = %q, want %q", log, want)
		}
	}
	if strings.Contains(log, "Localization load order verified") || strings.Contains(log, "Installed:") || strings.Contains(log, "No successful replacement was published") {
		t.Fatalf("failure log contains stale or successful publication claim: %q", log)
	}
}
