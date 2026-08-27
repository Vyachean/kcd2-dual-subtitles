package generator

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
)

func TestGenerateUsesExplicitModsRootForLocalizationSources(t *testing.T) {
	gameRoot := createGameRoot(t, true, true)
	customMods := filepath.Join(t.TempDir(), "custom-mods")
	modDir := filepath.Join(customMods, "russian-fix")
	if err := os.MkdirAll(filepath.Join(modDir, "Localization"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "mod.manifest"), []byte(`<?xml version="1.0"?><kcd_mod><info><name>Custom Russian Fix</name><modid>custom_russian_fix</modid></info></kcd_mod>`), 0o600); err != nil {
		t.Fatal(err)
	}
	writeLanguagePAK(t, filepath.Join(modDir, "Localization", "Russian_xml.pak"), `<?xml version="1.0" encoding="utf-8"?>
<Table>
<Row><Cell>different</Cell><Cell>custom corrected source</Cell><Cell>Исправлено из custom Mods</Cell></Row>
</Table>`)

	output := filepath.Join(t.TempDir(), "dual-subtitles.zip")
	result, err := Generate(Request{
		GameRoot:          gameRoot,
		ModsRoot:          customMods,
		MainLanguage:      localization.Russian,
		SecondaryLanguage: localization.English,
		OutputPath:        output,
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if !reflect.DeepEqual(result.MainLocalizationOverrides, []string{"Custom Russian Fix"}) {
		t.Fatalf("MainLocalizationOverrides = %#v", result.MainLocalizationOverrides)
	}

	rows := readGeneratedPatchRows(t, output, "Russian_xml.pak")
	for _, row := range rows {
		if row.ID == "different" {
			if want := "[RU] Исправлено из custom Mods\\n[EN] Secondary"; row.Text != want {
				t.Fatalf("generated text = %q, want %q", row.Text, want)
			}
			return
		}
	}
	t.Fatal("generated patch does not contain custom-root corrected row")
}
