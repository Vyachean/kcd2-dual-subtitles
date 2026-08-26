package generator

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
)

func TestGenerateComposesIndependentMainAndSecondaryLocalizationMods(t *testing.T) {
	gameRoot := createGameRoot(t, true, true)

	englishMod := filepath.Join(gameRoot, "Mods", "a_english_fix")
	if err := os.MkdirAll(filepath.Join(englishMod, "Localization"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(englishMod, "mod.manifest"), []byte(`<?xml version="1.0"?><kcd_mod><info><name>English Fix</name><modid>english_fix</modid></info></kcd_mod>`), 0o600); err != nil {
		t.Fatal(err)
	}
	writeLanguagePAK(t, filepath.Join(englishMod, "Localization", "English_xml.pak"), `<?xml version="1.0" encoding="utf-8"?>
<Table>
<Row><Cell>different</Cell><Cell>source</Cell><Cell>Corrected English</Cell></Row>
</Table>`)

	russianMod := filepath.Join(gameRoot, "Mods", "b_russian_fix")
	if err := os.MkdirAll(filepath.Join(russianMod, "Localization"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(russianMod, "mod.manifest"), []byte(`<?xml version="1.0"?><kcd_mod><info><name>Russian Fix</name><modid>russian_fix</modid></info></kcd_mod>`), 0o600); err != nil {
		t.Fatal(err)
	}
	writeLanguagePAK(t, filepath.Join(russianMod, "Localization", "Russian_xml.pak"), `<?xml version="1.0" encoding="utf-8"?>
<Table>
<Row><Cell>different</Cell><Cell>source</Cell><Cell>Исправленный русский</Cell></Row>
</Table>`)

	output := filepath.Join(t.TempDir(), "dual-subtitles.zip")
	result, err := Generate(Request{
		GameRoot:          gameRoot,
		MainLanguage:      localization.Russian,
		SecondaryLanguage: localization.English,
		OutputPath:        output,
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if !reflect.DeepEqual(result.MainLocalizationOverrides, []string{"Russian Fix"}) {
		t.Fatalf("MainLocalizationOverrides = %#v", result.MainLocalizationOverrides)
	}
	if !reflect.DeepEqual(result.SecondaryLocalizationOverrides, []string{"English Fix"}) {
		t.Fatalf("SecondaryLocalizationOverrides = %#v", result.SecondaryLocalizationOverrides)
	}

	rows := readGeneratedPatchRows(t, output, "Russian_xml.pak")
	for _, row := range rows {
		if row.ID != "different" {
			continue
		}
		want := "[RU] Исправленный русский\\n[EN] Corrected English"
		if row.Text != want {
			t.Fatalf("generated text = %q, want %q", row.Text, want)
		}
		return
	}
	t.Fatal("generated patch does not contain independently corrected dialogue row")
}
