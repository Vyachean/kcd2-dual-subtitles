package generator

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/modarchive"
)

func TestGenerateUsesActiveLocalizationModAsMainSource(t *testing.T) {
	gameRoot := createGameRoot(t, true, true)
	modDir := filepath.Join(gameRoot, "Mods", "russian-fix")
	if err := os.MkdirAll(filepath.Join(modDir, "Localization"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "mod.manifest"), []byte(`<?xml version="1.0"?><kcd_mod><info><name>Russian Fix</name><modid>russian-fix</modid></info></kcd_mod>`), 0o600); err != nil {
		t.Fatal(err)
	}
	writeLanguagePAK(t, filepath.Join(modDir, "Localization", "Russian_xml.pak"), `<?xml version="1.0" encoding="utf-8"?>
<Table>
<Row><Cell>different</Cell><Cell>corrected source</Cell><Cell>Исправленный</Cell></Row>
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
	if len(result.SecondaryLocalizationOverrides) != 0 {
		t.Fatalf("SecondaryLocalizationOverrides = %#v, want none", result.SecondaryLocalizationOverrides)
	}

	rows := readGeneratedPatchRows(t, output, "Russian_xml.pak")
	var corrected *localization.DialogueRow
	for i := range rows {
		if rows[i].ID == "different" {
			corrected = &rows[i]
			break
		}
	}
	if corrected == nil {
		t.Fatal("generated patch does not contain different row")
	}
	if want := "[RU] Исправленный\\n[EN] Secondary"; corrected.Text != want {
		t.Fatalf("generated text = %q, want %q", corrected.Text, want)
	}
}

func readGeneratedPatchRows(t *testing.T, output, languagePAK string) []localization.DialogueRow {
	t.Helper()
	outer, err := zip.OpenReader(output)
	if err != nil {
		t.Fatal(err)
	}
	defer outer.Close()

	wantPAK := filepath.ToSlash(filepath.Join(modarchive.ModID, "Localization", languagePAK))
	var pakBytes []byte
	for _, file := range outer.File {
		if file.Name != wantPAK {
			continue
		}
		entry, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		pakBytes, err = io.ReadAll(entry)
		_ = entry.Close()
		if err != nil {
			t.Fatal(err)
		}
		break
	}
	if len(pakBytes) == 0 {
		t.Fatalf("generated archive missing %s", wantPAK)
	}

	inner, err := zip.NewReader(bytes.NewReader(pakBytes), int64(len(pakBytes)))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range inner.File {
		if file.Name != modarchive.LocalizationPatchArchivePath {
			continue
		}
		entry, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(entry)
		_ = entry.Close()
		if err != nil {
			t.Fatal(err)
		}
		rows, err := localization.ParseDialogueXML(data)
		if err != nil {
			t.Fatal(err)
		}
		return rows
	}
	t.Fatalf("generated localization PAK missing %s", modarchive.LocalizationPatchArchivePath)
	return nil
}
