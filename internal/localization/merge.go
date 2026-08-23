package localization

import (
	"errors"
	"fmt"
)

const BilingualSeparator = `\n`

var ErrDuplicateDialogueID = errors.New("duplicate dialogue ID")

// MergeStats summarizes how main-language rows were handled during a merge.
type MergeStats struct {
	Processed        int
	Bilingual        int
	Identical        int
	MissingSecondary int
	SecondaryOnly    int
}

// MergeDialogueRows combines secondary-language text into main-language rows by ID.
// Main row order, IDs, and Source values are preserved.
func MergeDialogueRows(main, secondary []DialogueRow) ([]DialogueRow, MergeStats, error) {
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

		merged[i].Text = mainRow.Text + BilingualSeparator + secondaryRow.Text
		stats.Bilingual++
	}

	return merged, stats, nil
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
