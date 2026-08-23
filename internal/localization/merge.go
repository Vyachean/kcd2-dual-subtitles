package localization

import (
	"errors"
	"fmt"
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

// MergeDialogueRowsDifferentiated keeps the main subtitle line in the vanilla
// style while wrapping the secondary line in the subset of Scaleform HTML that
// retail KCD2's TextExtension.setText already consumes through htmlText. The
// vanilla HUD applies one global subtitle font size after setText, so this
// experimental formatter intentionally distinguishes the secondary line only
// with color and italic styling; independent sizes require the later HUD patch.
func MergeDialogueRowsDifferentiated(main, secondary []DialogueRow, mainTag, secondaryTag string) ([]DialogueRow, MergeStats, error) {
	return mergeDialogueRows(main, secondary, mainTag, secondaryTag, formatDifferentiatedBilingual)
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
