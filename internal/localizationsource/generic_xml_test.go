package localizationsource

import (
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
)

func TestParseGenericLocalizationXMLAcceptsOptionalSourceCell(t *testing.T) {
	rows, err := parseGenericLocalizationXML([]byte(`<?xml version="1.0"?><Table>
<Row><Cell>two_cell</Cell><Cell>Translated two</Cell></Row>
<Row><Cell>three_cell</Cell><Cell>Source</Cell><Cell>Translated three</Cell></Row>
</Table>`))
	if err != nil {
		t.Fatalf("parseGenericLocalizationXML() error = %v", err)
	}
	want := []localization.DialogueRow{
		{ID: "two_cell", Text: "Translated two"},
		{ID: "three_cell", Source: "Source", Text: "Translated three"},
	}
	if len(rows) != len(want) {
		t.Fatalf("rows = %+v, want %+v", rows, want)
	}
	for i := range want {
		if rows[i] != want[i] {
			t.Fatalf("row %d = %+v, want %+v", i, rows[i], want[i])
		}
	}
}

func TestGenericTwoCellPatchOverlaysOnlyKnownDialogue(t *testing.T) {
	modsRoot := t.TempDir()
	writeLocalizationMod(t, modsRoot, "generic", "generic", "Generic", "English_xml.pak", map[string]string{
		"localization_generic.xml": `<?xml version="1.0"?><Table>
<Row><Cell>known</Cell><Cell>Corrected known</Cell></Row>
<Row><Cell>ui_unknown</Cell><Cell>Must remain outside dialogue</Cell></Row>
</Table>`,
	})

	result, err := resolveFromModsRoot(
		[]localization.DialogueRow{{ID: "known", Source: "stock", Text: "Stock"}},
		modsRoot,
		"English_xml.pak",
	)
	if err != nil {
		t.Fatalf("resolveFromModsRoot() error = %v", err)
	}
	if len(result.Rows) != 1 || result.Rows[0].ID != "known" || result.Rows[0].Text != "Corrected known" {
		t.Fatalf("rows = %+v, want only corrected known dialogue", result.Rows)
	}
	if len(result.Contributions) != 1 || result.Contributions[0].Name != "Generic" {
		t.Fatalf("contributions = %+v, want Generic", result.Contributions)
	}
}

func TestExplicitDialogueTableRemainsStrictThreeCell(t *testing.T) {
	_, err := parseLocalizationResource([]byte(`<?xml version="1.0"?><Table><Row><Cell>id</Cell><Cell>text</Cell></Row></Table>`), true)
	if err == nil {
		t.Fatal("two-cell explicit dialogue table unexpectedly accepted")
	}
}
