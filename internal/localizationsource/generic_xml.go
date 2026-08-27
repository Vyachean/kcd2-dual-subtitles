package localizationsource

import (
	"encoding/xml"
	"fmt"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
)

type genericLocalizationTable struct {
	XMLName xml.Name                 `xml:"Table"`
	Rows    []genericLocalizationRow `xml:"Row"`
}

type genericLocalizationRow struct {
	Cells []string `xml:"Cell"`
}

// parseLocalizationResource keeps the retail dialogue table strict while
// accepting Warhorse's generic mod-localization row shape, where the middle
// source-language cell is optional and is not used for displayed text.
func parseLocalizationResource(data []byte, explicitDialogue bool) ([]localization.DialogueRow, error) {
	if explicitDialogue {
		return localization.ParseDialogueXML(data)
	}
	return parseGenericLocalizationXML(data)
}

func parseGenericLocalizationXML(data []byte) ([]localization.DialogueRow, error) {
	var table genericLocalizationTable
	if err := xml.Unmarshal(data, &table); err != nil {
		return nil, fmt.Errorf("invalid localization XML: %w", err)
	}
	if table.XMLName.Local != "Table" {
		return nil, fmt.Errorf("invalid localization XML: root element is <%s>, want <Table>", table.XMLName.Local)
	}

	rows := make([]localization.DialogueRow, 0, len(table.Rows))
	for index, row := range table.Rows {
		if len(row.Cells) != 2 && len(row.Cells) != 3 {
			return nil, fmt.Errorf("invalid localization row at row %d: got %d cells, want 2 or 3", index, len(row.Cells))
		}
		if row.Cells[0] == "" {
			return nil, fmt.Errorf("invalid localization row at row %d: empty ID", index)
		}

		parsed := localization.DialogueRow{ID: row.Cells[0], Text: row.Cells[len(row.Cells)-1]}
		if len(row.Cells) == 3 {
			parsed.Source = row.Cells[1]
		}
		rows = append(rows, parsed)
	}
	return rows, nil
}
