package localizationsource

import (
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
)

func TestResolveFromModsRootAppliesGenericTextPatchWithLastDuplicateWinning(t *testing.T) {
	modsRoot := t.TempDir()
	writeLocalizationMod(t, modsRoot, "chinesesfixptf", "chinesesfixptf", "Chineses Fix", "Chineses_xml.pak", map[string]string{
		"text__chinesesfixptf.xml": dialogueXML(
			localization.DialogueRow{ID: "ui_unrelated", Source: "UI", Text: "ignored first"},
			localization.DialogueRow{ID: "dialogue", Source: "first", Text: "first correction"},
			localization.DialogueRow{ID: "ui_unrelated", Source: "UI", Text: "ignored last"},
			localization.DialogueRow{ID: "dialogue", Source: "last", Text: "final correction"},
		),
	})

	result, err := resolveFromModsRoot(
		[]localization.DialogueRow{{ID: "dialogue", Source: "stock", Text: "stock"}},
		modsRoot,
		"Chineses_xml.pak",
	)
	if err != nil {
		t.Fatalf("resolveFromModsRoot() error = %v", err)
	}
	if len(result.Rows) != 1 {
		t.Fatalf("rows = %+v, want only the known dialogue row", result.Rows)
	}
	if got := result.Rows[0]; got.Source != "last" || got.Text != "final correction" {
		t.Fatalf("effective dialogue = %+v, want last duplicate row", got)
	}
	if len(result.Contributions) != 1 || result.Contributions[0].Name != "Chineses Fix" {
		t.Fatalf("contributions = %+v, want Chineses Fix", result.Contributions)
	}
}

func TestGenericTextPatchDoesNotIntroduceUnknownDialogueIDs(t *testing.T) {
	modsRoot := t.TempDir()
	writeLocalizationMod(t, modsRoot, "generic", "generic", "Generic", "English_xml.pak", map[string]string{
		"TEXT__GENERIC.XML": dialogueXML(
			localization.DialogueRow{ID: "unknown", Source: "source", Text: "must stay outside dialogue"},
		),
	})

	stock := []localization.DialogueRow{{ID: "known", Source: "stock", Text: "stock"}}
	result, err := resolveFromModsRoot(stock, modsRoot, "English_xml.pak")
	if err != nil {
		t.Fatalf("resolveFromModsRoot() error = %v", err)
	}
	if len(result.Rows) != 1 || result.Rows[0] != stock[0] || len(result.Contributions) != 0 {
		t.Fatalf("result = %+v, want unchanged stock and no contribution", result)
	}
}
