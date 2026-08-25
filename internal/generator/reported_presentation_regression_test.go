package generator

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/gfxpatch"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/modarchive"
)

func TestGenerateReportedEnglishCzechPresentationWithShadowPackagesStyleAndHUD(t *testing.T) {
	originalRead := readRetailHUD
	originalPatch := patchRetailHUD
	originalReadabilityPatch := patchRetailHUDReadability
	defer func() {
		readRetailHUD = originalRead
		patchRetailHUD = originalPatch
		patchRetailHUDReadability = originalReadabilityPatch
	}()

	readRetailHUD = func(string) ([]byte, error) { return []byte("retail-hud"), nil }
	patchRetailHUD = func([]byte) ([]byte, error) {
		t.Fatal("legacy HUD patcher called while Shadow is enabled")
		return nil, nil
	}
	patchRetailHUDReadability = func(input []byte, config gfxpatch.HUDReadabilityConfig) ([]byte, error) {
		if string(input) != "retail-hud" {
			t.Fatalf("readability patch input = %q, want retail-hud", input)
		}
		if config.Outline || !config.Shadow {
			t.Fatalf("readability config = %+v, want shadow only", config)
		}
		return []byte("derived-hud-with-shadow"), nil
	}

	const english = `<?xml version="1.0" encoding="utf-8"?>
<Table><Row><Cell>different</Cell><Cell>source</Cell><Cell>Primary</Cell></Row></Table>`
	const czech = `<?xml version="1.0" encoding="utf-8"?>
<Table><Row><Cell>different</Cell><Cell>source</Cell><Cell>Vedlejší</Cell></Row></Table>`

	gameRoot := t.TempDir()
	localizationDir := filepath.Join(gameRoot, "Localization")
	if err := os.Mkdir(localizationDir, 0o755); err != nil {
		t.Fatalf("create Localization directory: %v", err)
	}
	writeLanguagePAK(t, filepath.Join(localizationDir, "English_xml.pak"), english)
	writeLanguagePAK(t, filepath.Join(localizationDir, "Czech_xml.pak"), czech)

	presentation := DefaultHUDPresentationConfig()
	presentation.SecondaryColor = "#FF8080"
	presentation.SecondarySize = 24
	presentation.SecondaryItalic = false
	presentation.ShowLanguageTags = true
	presentation.Outline = false
	presentation.Shadow = true

	output := filepath.Join(t.TempDir(), "reported-presentation.zip")
	result, err := Generate(Request{
		GameRoot:          gameRoot,
		MainLanguage:      localization.English,
		SecondaryLanguage: localization.Czech,
		SubtitleStyle:     SubtitleStyleHUD,
		HUDPresentation:   &presentation,
		OutputPath:        output,
		Version:           "v0.3.2-regression",
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if !result.HUDOverride || result.LocalizationTargets != 2 {
		t.Fatalf("result = %+v, want verified HUD override for two target slots", result)
	}

	outer, err := zip.OpenReader(output)
	if err != nil {
		t.Fatalf("open generated archive: %v", err)
	}
	defer outer.Close()

	localizationPAK := readArchiveEntry(t, &outer.Reader, "kcd_dual_subtitles/Localization/English_xml.pak")
	nestedLocalization, err := zip.NewReader(bytes.NewReader(localizationPAK), int64(len(localizationPAK)))
	if err != nil {
		t.Fatalf("open generated localization PAK: %v", err)
	}
	patchXML := readArchiveEntry(t, nestedLocalization, modarchive.LocalizationPatchArchivePath)
	rows, err := localization.ParseDialogueXML(patchXML)
	if err != nil {
		t.Fatalf("parse generated localization patch: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("patch rows = %#v, want one bilingual row", rows)
	}
	wantText := "[EN] Primary<br/><font color='#FF8080' size='24'>[CS] Vedlejší</font>"
	if rows[0].Text != wantText {
		t.Fatalf("generated styled row = %q, want %q", rows[0].Text, wantText)
	}

	dataPAK := readArchiveEntry(t, &outer.Reader, "kcd_dual_subtitles/Data/"+modarchive.DataPAKFilename)
	nestedData, err := zip.NewReader(bytes.NewReader(dataPAK), int64(len(dataPAK)))
	if err != nil {
		t.Fatalf("open generated Data PAK: %v", err)
	}
	if got := readArchiveEntry(t, nestedData, modarchive.HUDArchivePath); string(got) != "derived-hud-with-shadow" {
		t.Fatalf("derived HUD = %q, want derived-hud-with-shadow", got)
	}
}
