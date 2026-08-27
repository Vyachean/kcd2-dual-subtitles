package gui

import (
	"fmt"
	"strings"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/generator"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
)

type generationLogContext struct {
	GameRoot  string
	ModsRoot  string
	Main      localization.Language
	Secondary localization.Language
	Styled    bool
}

func formatGenerationStarted(context generationLogContext) string {
	lines := generationContextLines("Generation started", context)
	lines = append(lines,
		"→ Reading stock localization and active localization mods...",
		"Note: the generated patch is installed only for the selected Main and Secondary languages.",
	)
	return strings.Join(lines, "\r\n")
}

func formatGenerationSucceeded(context generationLogContext, result generator.Result) string {
	lines := generationContextLines("Generation completed", context)
	lines = append(lines, "✓ Main source built from stock "+string(context.Main)+" localization.")
	lines = appendLocalizationOverrides(lines, result.MainLocalizationOverrides)
	lines = append(lines, "✓ Secondary source built from stock "+string(context.Secondary)+" localization.")
	lines = appendLocalizationOverrides(lines, result.SecondaryLocalizationOverrides)
	lines = append(lines,
		fmt.Sprintf("✓ Merged %d main dialogue rows (%d bilingual).", result.Stats.Processed, result.Stats.Bilingual),
		fmt.Sprintf("✓ Generated %d changed dialogue rows for %d selected localization targets.", result.PatchRows, result.LocalizationTargets),
	)
	if result.HUDOverride {
		lines = append(lines, "✓ Custom subtitle appearance prepared from the installed game HUD.")
	} else {
		lines = append(lines, "✓ Game HUD left unchanged.")
	}
	lines = append(lines, "✓ Localization load order verified.")
	if strings.TrimSpace(result.InstallPath) != "" {
		lines = append(lines, "✓ Installed: "+result.InstallPath)
	}
	lines = append(lines,
		"Use one of the selected languages as KCD2's text language; regenerate after switching the game to another language.",
		"Restart KCD2 before testing.",
	)
	return strings.Join(lines, "\r\n")
}

func formatGenerationFailed(context generationLogContext, err error) string {
	lines := generationContextLines("Generation failed", context)
	if err != nil {
		lines = append(lines, "✗ "+err.Error())
	}
	lines = append(lines, "Generation did not complete successfully. Any interrupted replacement is recovered by the next install or uninstall operation.")
	return strings.Join(lines, "\r\n")
}

func generationContextLines(title string, context generationLogContext) []string {
	gameRoot := strings.TrimSpace(context.GameRoot)
	if gameRoot == "" {
		gameRoot = "not selected"
	}
	modsRoot := strings.TrimSpace(context.ModsRoot)
	if modsRoot == "" {
		modsRoot = "automatic"
	}
	mode := "game-default appearance"
	if context.Styled {
		mode = "custom subtitle appearance"
	}
	return []string{
		title,
		"Game folder: " + gameRoot,
		"Mods folder: " + modsRoot,
		"Main: " + string(context.Main),
		"Secondary: " + string(context.Secondary),
		"Mode: " + mode,
	}
}

func appendLocalizationOverrides(lines []string, overrides []string) []string {
	if len(overrides) == 0 {
		return append(lines, "  No installed localization mod changed dialogue.")
	}
	return append(lines, "  Applied localization mods: "+strings.Join(overrides, ", "))
}
