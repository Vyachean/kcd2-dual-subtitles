package localization

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/subtitlepayload"
)

const (
	BilingualSeparator           = `\n`
	DifferentiatedSecondaryColor = "#A8A8A8"
)

var ErrDuplicateDialogueID = errors.New("duplicate dialogue ID")

// MergeStats summarizes how main-language rows were handled during a merge.
type MergeStats struct {
	Processed              int
	Bilingual              int
	Identical              int
	MissingSecondary       int
	MainEmptyFallback      int
	SecondaryEmptyFallback int
	SecondaryOnly          int
}

// HUDPresentationOptions is the normalized formatting input supplied by the
// generator for truly bilingual rows in direct-HTML HUD mode. Empty/zero
// primary properties leave that property under vanilla control.
type HUDPresentationOptions struct {
	PrimaryColor     string
	PrimarySize      int
	PrimaryItalic    bool
	SecondaryColor   string
	SecondarySize    int
	SecondaryItalic  bool
	ShowLanguageTags bool
}

type bilingualFormatter func(mainText, secondaryText, mainTag, secondaryTag string) string

// MergeDialogueRows combines secondary-language text into main-language rows by ID.
// Main row order, IDs, and Source values are preserved.
func MergeDialogueRows(main, secondary []DialogueRow) ([]DialogueRow, MergeStats, error) {
	return mergeDialogueRows(main, secondary, "", "", formatPlainBilingual)
}

// MergeDialogueRowsTagged combines secondary-language text into main-language rows
// and prefixes only truly bilingual rows with compact language tags.
func MergeDialogueRowsTagged(main, secondary []DialogueRow, mainTag, secondaryTag string) ([]DialogueRow, MergeStats, error) {
	return mergeDialogueRows(main, secondary, mainTag, secondaryTag, formatTaggedBilingual)
}

// MergeDialogueRowsDifferentiated is the failed Stage A localization-only
// experiment retained for reproducibility of v0.3.0-rc.1. New experiments
// should use MergeDialogueRowsHUDPrototype instead.
func MergeDialogueRowsDifferentiated(main, secondary []DialogueRow, mainTag, secondaryTag string) ([]DialogueRow, MergeStats, error) {
	return mergeDialogueRows(main, secondary, mainTag, secondaryTag, formatDifferentiatedBilingual)
}

// MergeDialogueRowsHUDPrototype preserves the live-proven rc.10 defaults for
// callers that do not supply presentation options explicitly.
func MergeDialogueRowsHUDPrototype(main, secondary []DialogueRow, mainTag, secondaryTag string) ([]DialogueRow, MergeStats, error) {
	return MergeDialogueRowsHUD(main, secondary, mainTag, secondaryTag, HUDPresentationOptions{
		SecondaryColor:   subtitlepayload.SecondaryColor,
		SecondarySize:    subtitlepayload.SecondarySize,
		SecondaryItalic:  true,
		ShowLanguageTags: true,
	})
}

// MergeDialogueRowsHUD emits the complete two-line Scaleform HTML for the
// direct post-vanilla HUD path using generator-normalized presentation options.
// The derived HUD stores this original argument before vanilla processing and
// assigns it to htmlText only after the retail global subtitle-size pass.
func MergeDialogueRowsHUD(main, secondary []DialogueRow, mainTag, secondaryTag string, presentation HUDPresentationOptions) ([]DialogueRow, MergeStats, error) {
	format := func(mainText, secondaryText, mainTag, secondaryTag string) string {
		return formatHUDBilingual(mainText, secondaryText, mainTag, secondaryTag, presentation)
	}
	return mergeDialogueRows(main, secondary, mainTag, secondaryTag, format)
}

func mergeDialogueRows(main, secondary []DialogueRow, mainTag, secondaryTag string, format bilingualFormatter) ([]DialogueRow, MergeStats, error) {
	mainIDs, err := indexDialogueIDs(main, "main")
	if err != nil {
		return nil, MergeStats{}, err
	}
	secondaryByID, err := indexDialogueRows(secondary, "secondary")
	if err != nil {
		return nil, MergeStats{}, err
	}

	stats := MergeStats{Processed: len(main)}
	for id := range secondaryByID {
		if _, exists := mainIDs[id]; !exists {
			stats.SecondaryOnly++
		}
	}

	merged := make([]DialogueRow, len(main))
	for i, mainRow := range main {
		merged[i] = mainRow

		secondaryRow, exists := secondaryByID[mainRow.ID]
		if !exists {
			stats.MissingSecondary++
			continue
		}
		if mainRow.Text == secondaryRow.Text {
			stats.Identical++
			continue
		}
		if mainRow.Text == "" {
			merged[i].Text = secondaryRow.Text
			stats.MainEmptyFallback++
			continue
		}
		if secondaryRow.Text == "" {
			stats.SecondaryEmptyFallback++
			continue
		}

		merged[i].Text = format(mainRow.Text, secondaryRow.Text, mainTag, secondaryTag)
		stats.Bilingual++
	}

	return merged, stats, nil
}

func formatPlainBilingual(mainText, secondaryText, _, _ string) string {
	return mainText + BilingualSeparator + secondaryText
}

func formatTaggedBilingual(mainText, secondaryText, mainTag, secondaryTag string) string {
	if mainTag != "" && secondaryTag != "" {
		mainText = "[" + mainTag + "] " + mainText
		secondaryText = "[" + secondaryTag + "] " + secondaryText
	}
	return mainText + BilingualSeparator + secondaryText
}

func formatDifferentiatedBilingual(mainText, secondaryText, mainTag, secondaryTag string) string {
	if mainTag != "" && secondaryTag != "" {
		mainText = "[" + mainTag + "] " + mainText
		secondaryText = "[" + secondaryTag + "] " + secondaryText
	}
	return mainText + BilingualSeparator + "<font color='" + DifferentiatedSecondaryColor + "'><i>" + secondaryText + "</i></font>"
}

func formatHUDBilingual(mainText, secondaryText, mainTag, secondaryTag string, presentation HUDPresentationOptions) string {
	if presentation.ShowLanguageTags && mainTag != "" && secondaryTag != "" {
		mainText = "[" + mainTag + "] " + mainText
		secondaryText = "[" + secondaryTag + "] " + secondaryText
	}

	mainHTML := formatHUDLine(
		subtitlepayload.EncodeSecondaryHTML(mainText),
		presentation.PrimaryColor,
		presentation.PrimarySize,
		presentation.PrimaryItalic,
	)
	secondaryHTML := formatHUDLine(
		subtitlepayload.EncodeSecondaryHTML(secondaryText),
		presentation.SecondaryColor,
		presentation.SecondarySize,
		presentation.SecondaryItalic,
	)
	return mainHTML + "<br/>" + secondaryHTML
}

func formatHUDLine(text, color string, size int, italic bool) string {
	var prefix strings.Builder
	suffix := ""
	if color != "" || size != 0 {
		prefix.WriteString("<font")
		if color != "" {
			prefix.WriteString(" color='")
			prefix.WriteString(color)
			prefix.WriteString("'")
		}
		if size != 0 {
			prefix.WriteString(" size='")
			prefix.WriteString(strconv.Itoa(size))
			prefix.WriteString("'")
		}
		prefix.WriteString(">")
		suffix = "</font>"
	}
	if italic {
		prefix.WriteString("<i>")
		suffix = "</i>" + suffix
	}
	return prefix.String() + text + suffix
}

func indexDialogueIDs(rows []DialogueRow, side string) (map[string]struct{}, error) {
	ids := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		if _, exists := ids[row.ID]; exists {
			return nil, fmt.Errorf("%w in %s rows: %q", ErrDuplicateDialogueID, side, row.ID)
		}
		ids[row.ID] = struct{}{}
	}
	return ids, nil
}

func indexDialogueRows(rows []DialogueRow, side string) (map[string]DialogueRow, error) {
	indexed := make(map[string]DialogueRow, len(rows))
	for _, row := range rows {
		if _, exists := indexed[row.ID]; exists {
			return nil, fmt.Errorf("%w in %s rows: %q", ErrDuplicateDialogueID, side, row.ID)
		}
		indexed[row.ID] = row
	}
	return indexed, nil
}
