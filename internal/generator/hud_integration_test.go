package generator

import (
	"archive/zip"
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/modarchive"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/subtitlepayload"
)

func TestGenerateHUDPrototypeDerivesAndPackagesHUD(t *testing.T) {
	originalRead := readRetailHUD
	originalPatch := patchRetailHUD
	defer func() {
		readRetailHUD = originalRead
		patchRetailHUD = originalPatch
	}()

	readRetailHUD = func(gameRoot string) ([]byte, error) {
		if gameRoot == "" {
			t.Fatal("readRetailHUD received empty game root")
		}
		return []byte("retail-hud"), nil
	}
	patchRetailHUD = func(input []byte) ([]byte, error) {
		if string(input) != "retail-hud" {
			t.Fatalf("patchRetailHUD input = %q, want retail-hud", input)
		}
		return []byte("derived-hud"), nil
	}

	gameRoot := createGameRoot(t, true, true)
	output := filepath.Join(t.TempDir(), "hud-prototype.zip")
	result, err := Generate(Request{
		GameRoot:          gameRoot,
		MainLanguage:      localization.Russian,
		SecondaryLanguage: localization.English,
		SubtitleStyle:     SubtitleStyleHUD,
		OutputPath:        output,
		Version:           "v0.3.0-test",
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if !result.HUDOverride || result.SubtitleStyle != SubtitleStyleHUD {
		t.Fatalf("result = %+v, want HUD override/style", result)
	}

	outer, err := zip.OpenReader(output)
	if err != nil {
		t.Fatalf("open generated archive: %v", err)
	}
	defer outer.Close()
	dataPath := "kcd_dual_subtitles/Data/" + modarchive.DataPAKFilename
	dataPAK := readArchiveEntry(t, &outer.Reader, dataPath)
	nestedData, err := zip.NewReader(bytes.NewReader(dataPAK), int64(len(dataPAK)))
	if err != nil {
		t.Fatalf("open generated data PAK: %v", err)
	}
	if got := readArchiveEntry(t, nestedData, modarchive.HUDArchivePath); string(got) != "derived-hud" {
		t.Fatalf("derived HUD = %q, want derived-hud", got)
	}

	localizationPAK := readArchiveEntry(t, &outer.Reader, "kcd_dual_subtitles/Localization/Russian_xml.pak")
	nestedLocalization, err := zip.NewReader(bytes.NewReader(localizationPAK), int64(len(localizationPAK)))
	if err != nil {
		t.Fatalf("open generated localization PAK: %v", err)
	}
	patchXML := readArchiveEntry(t, nestedLocalization, modarchive.LocalizationPatchArchivePath)
	rows, err := localization.ParseDialogueXML(patchXML)
	if err != nil {
		t.Fatalf("parse generated HUD localization patch: %v", err)
	}
	if len(rows) != 1 || rows[0].ID != "different" {
		t.Fatalf("patch rows = %#v, want only different row", rows)
	}
	for _, want := range []string{
		"[RU] Основной",
		"<font color='" + subtitlepayload.SecondaryColor + "' size='24'><i>[EN] Secondary</i></font>",
	} {
		if !strings.Contains(rows[0].Text, want) {
			t.Fatalf("HUD HTML row = %q, missing %q", rows[0].Text, want)
		}
	}
	if strings.Contains(rows[0].Text, "<p") || strings.Contains(rows[0].Text, "align=") {
		t.Fatalf("HUD HTML row unexpectedly forces paragraph alignment: %q", rows[0].Text)
	}
	if strings.Contains(rows[0].Text, subtitlepayload.Prefix) || strings.Contains(rows[0].Text, subtitlepayload.Suffix) {
		t.Fatalf("HUD HTML row unexpectedly contains carrier sentinel: %q", rows[0].Text)
	}
}

func TestGenerateDefaultTaggedPathNeverReadsHUD(t *testing.T) {
	originalRead := readRetailHUD
	defer func() { readRetailHUD = originalRead }()
	readRetailHUD = func(string) ([]byte, error) {
		return nil, errors.New("HUD must not be read for tagged generation")
	}

	gameRoot := createGameRoot(t, true, true)
	result, err := Generate(Request{
		GameRoot:          gameRoot,
		MainLanguage:      localization.Russian,
		SecondaryLanguage: localization.English,
		OutputPath:        filepath.Join(t.TempDir(), "tagged.zip"),
	})
	if err != nil {
		t.Fatalf("Generate(tagged) error = %v", err)
	}
	if result.HUDOverride {
		t.Fatal("default tagged generation unexpectedly reports HUD override")
	}
}
