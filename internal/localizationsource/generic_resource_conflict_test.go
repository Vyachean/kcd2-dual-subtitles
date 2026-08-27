package localizationsource

import (
	"strings"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
)

func TestResolveFromModsRootFailsClosedWhenGenericResourcesConflictOnDialogueID(t *testing.T) {
	modsRoot := t.TempDir()
	writeLocalizationMod(t, modsRoot, "multi", "multi", "Multi", "English_xml.pak", map[string]string{
		"first_multi.xml": dialogueXML(
			localization.DialogueRow{ID: "dialogue", Source: "first", Text: "first value"},
		),
		"second_multi.xml": dialogueXML(
			localization.DialogueRow{ID: "dialogue", Source: "second", Text: "second value"},
		),
	})

	_, err := resolveFromModsRoot(
		[]localization.DialogueRow{{ID: "dialogue", Source: "stock", Text: "stock"}},
		modsRoot,
		"English_xml.pak",
	)
	if err == nil {
		t.Fatal("resolveFromModsRoot() error = nil, want ambiguous cross-resource dialogue error")
	}
	for _, want := range []string{"dialogue", "first_multi.xml", "second_multi.xml", "order"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, want substring %q", err, want)
		}
	}
}

func TestResolveFromModsRootAllowsSameDialogueTextAcrossGenericResources(t *testing.T) {
	modsRoot := t.TempDir()
	writeLocalizationMod(t, modsRoot, "multi", "multi", "Multi", "English_xml.pak", map[string]string{
		"first_multi.xml": dialogueXML(
			localization.DialogueRow{ID: "dialogue", Source: "first", Text: "same value"},
		),
		"second_multi.xml": dialogueXML(
			localization.DialogueRow{ID: "dialogue", Source: "second", Text: "same value"},
		),
	})

	result, err := resolveFromModsRoot(
		[]localization.DialogueRow{{ID: "dialogue", Source: "stock", Text: "stock"}},
		modsRoot,
		"English_xml.pak",
	)
	if err != nil {
		t.Fatalf("resolveFromModsRoot() error = %v", err)
	}
	if len(result.Rows) != 1 || result.Rows[0].Text != "same value" {
		t.Fatalf("rows = %+v, want same dialogue text", result.Rows)
	}
}
