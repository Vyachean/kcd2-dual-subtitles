package localization

import (
	"encoding/xml"
	"errors"
	"fmt"
)

var (
	ErrInvalidDialogueXML = errors.New("invalid dialogue XML")
	ErrInvalidDialogueRow = errors.New("invalid dialogue row")
)

// DialogueRow is one localized dialogue row from text_ui_dialog.xml.
type DialogueRow struct {
	ID     string
	Source string
	Text   string
}

type dialogueXMLTable struct {
	XMLName xml.Name         `xml:"Table"`
	Rows    []dialogueXMLRow `xml:"Row"`
}

type dialogueXMLRow struct {
	Cells []string `xml:"Cell"`
}

// ParseDialogueXML parses a KCD2 dialogue table into ordered rows.
func ParseDialogueXML(data []byte) ([]DialogueRow, error) {
	var table dialogueXMLTable
	if err := xml.Unmarshal(data, &table); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidDialogueXML, err)
	}
	if table.XMLName.Local != "Table" {
		return nil, fmt.Errorf("%w: root element is <%s>, want <Table>", ErrInvalidDialogueXML, table.XMLName.Local)
	}

	rows := make([]DialogueRow, 0, len(table.Rows))
	for index, row := range table.Rows {
		parsed, err := dialogueRowFromXML(index, row)
		if err != nil {
			return nil, err
		}
		rows = append(rows, parsed)
	}

	return rows, nil
}

// MarshalDialogueXML serializes ordered dialogue rows as deterministic UTF-8 XML.
func MarshalDialogueXML(rows []DialogueRow) ([]byte, error) {
	table := dialogueXMLTable{Rows: make([]dialogueXMLRow, 0, len(rows))}
	for index, row := range rows {
		if row.ID == "" {
			return nil, fmt.Errorf("%w at row %d: empty ID", ErrInvalidDialogueRow, index)
		}
		table.Rows = append(table.Rows, dialogueXMLRow{
			Cells: []string{row.ID, row.Source, row.Text},
		})
	}

	body, err := xml.Marshal(table)
	if err != nil {
		return nil, fmt.Errorf("marshal dialogue XML: %w", err)
	}

	output := make([]byte, 0, len(xml.Header)+len(body)+1)
	output = append(output, xml.Header...)
	output = append(output, body...)
	output = append(output, '\n')
	return output, nil
}

func dialogueRowFromXML(index int, row dialogueXMLRow) (DialogueRow, error) {
	if len(row.Cells) != 3 {
		return DialogueRow{}, fmt.Errorf("%w at row %d: got %d cells, want 3", ErrInvalidDialogueRow, index, len(row.Cells))
	}
	if row.Cells[0] == "" {
		return DialogueRow{}, fmt.Errorf("%w at row %d: empty ID", ErrInvalidDialogueRow, index)
	}

	return DialogueRow{
		ID:     row.Cells[0],
		Source: row.Cells[1],
		Text:   row.Cells[2],
	}, nil
}
