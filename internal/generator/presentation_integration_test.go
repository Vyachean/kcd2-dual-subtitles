package generator

import (
	"archive/zip"
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/modarchive"
)

func TestGenerateHUDUsesExplicitPresentationConfig(t *testing.T) {
	originalRead := readRetailHUD
	originalPatch := patchRetailHUD
	defer func() {
		readRetailHUD = originalRead
		patchRetailHUD = originalPatch
	}()

	readRetailHUD = func(string) ([]byte, error) { return []byte("retail-hud"), nil }
	patchRetailHUD = func([]byte) ([]byte, error) { return []byte("derived-hud"), nil }

	presentation := DefaultHUDPresentationConfig()
	presentation.SecondaryColor = "#123ABC"
	presentation.SecondarySize = 18
	presentation.SecondaryItalic = false
	presentation.ShowLanguageTags = false

	gameRoot := createGameRoot(t, true, true)
	output := filepath.Join(t.TempDir(), "hud-custom-presentation.zip")
	_, err := Generate(Request{
		GameRoot:          gameRoot,
		MainLanguage:      localization.Russian,
		SecondaryLanguage: localization.English,
		SubtitleStyle:     SubtitleStyleHUD,
		HUDPresentation:   &presentation,
		OutputPath:        output,
		Version:           "v0.3.0-test",
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	outer, err := zip.OpenReader(output)
	if err != nil {
		t.Fatalf("open generated archive: %v", err)
	}
	defer outer.Close()
	localizationPAK := readArchiveEntry(t, &outer.Reader, "kcd_dual_subtitles/Localization/Russian_xml.pak")
	nestedLocalization, err := zip.NewReader(bytes.NewReader(localizationPAK), int64(len(localizationPAK)))
	if err != nil {
		t.Fatalf("open generated localization PAK: %v", err)
	}
	patchXML := readArchiveEntry(t, nestedLocalization, modarchive.LocalizationPatchArchivePath)
	rows, err := localization.ParseDialogueXML(patchXML)
	if err != nil {
		t.Fatalf("parse generated localization patch: %v", err)
	}
	if len(rows) != 1 || rows[0].ID != "different" {
		t.Fatalf("patch rows = %#v, want only different row", rows)
	}

	got := rows[0].Text
	want := "Основной<br/><font color='#123ABC' size='18'>Secondary</font>"
	if got != want {
		t.Fatalf("HUD HTML row = %q, want %q", got, want)
	}
	if strings.Contains(got, "[RU]") || strings.Contains(got, "[EN]") || strings.Contains(got, "<i>") || strings.Contains(got, "</i>") {
		t.Fatalf("HUD HTML row unexpectedly contains disabled presentation markup: %q", got)
	}
}
