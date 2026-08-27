package generator

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
)

func TestGenerateRejectsLaterSameTextDialogueWriterWithoutModOrder(t *testing.T) {
	gameRoot := createGameRoot(t, true, true)
	modDir := filepath.Join(gameRoot, "Mods", "z_same")
	if err := os.MkdirAll(filepath.Join(modDir, "Localization"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "mod.manifest"), []byte(`<?xml version="1.0"?><kcd_mod><info><name>Same Text Writer</name><modid>same</modid></info></kcd_mod>`), 0o600); err != nil {
		t.Fatal(err)
	}
	writeLanguagePAK(t, filepath.Join(modDir, "Localization", "Russian_xml.pak"), `<?xml version="1.0" encoding="utf-8"?>
<Table>
<Row><Cell>different</Cell><Cell>same source</Cell><Cell>Основной</Cell></Row>
</Table>`)

	_, err := Generate(Request{
		GameRoot:          gameRoot,
		MainLanguage:      localization.Russian,
		SecondaryLanguage: localization.English,
	})
	if !errors.Is(err, ErrUnsafeLocalizationLoadOrder) {
		t.Fatalf("Generate() error = %v, want ErrUnsafeLocalizationLoadOrder for later same-text dialogue writer", err)
	}
}
