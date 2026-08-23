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

func TestGenerateWritesDifferentiatedLocalizationPatch(t *testing.T) {
	gameRoot := createGameRoot(t, true, true)
	output := filepath.Join(t.TempDir(), "dual-subtitles.zip")

	result, err := Generate(Request{
		GameRoot:          gameRoot,
		MainLanguage:      localization.Russian,
		SecondaryLanguage: localization.English,
		SubtitleStyle:     SubtitleStyleDifferentiated,
		OutputPath:        output,
		Version:           "v0.3.0-style-test",
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if result.SubtitleStyle != SubtitleStyleDifferentiated {
		t.Fatalf("SubtitleStyle = %q, want %q", result.SubtitleStyle, SubtitleStyleDifferentiated)
	}

	outer, err := zip.OpenReader(output)
	if err != nil {
		t.Fatalf("open generated archive: %v", err)
	}
	defer outer.Close()

	pakData := readArchiveEntry(t, &outer.Reader, "kcd_dual_subtitles/Localization/Russian_xml.pak")
	pak, err := zip.NewReader(bytes.NewReader(pakData), int64(len(pakData)))
	if err != nil {
		t.Fatalf("open generated localization PAK: %v", err)
	}
	patchData := readArchiveEntry(t, pak, modarchive.LocalizationPatchArchivePath)

	if !bytes.Contains(patchData, []byte("&lt;font color=&#39;#A8A8A8&#39;&gt;&lt;i&gt;")) {
		t.Fatalf("raw patch XML does not contain XML-escaped Scaleform markup: %s", patchData)
	}
	if bytes.Contains(patchData, []byte("<font color=")) {
		t.Fatalf("raw patch XML contains unescaped Scaleform element markup: %s", patchData)
	}

	rows, err := localization.ParseDialogueXML(patchData)
	if err != nil {
		t.Fatalf("parse generated localization patch: %v", err)
	}

	want := `[RU] Основной\n<font color='#A8A8A8'><i>[EN] Secondary</i></font>`
	for _, row := range rows {
		if row.ID != "different" {
			continue
		}
		if row.Text != want {
			t.Fatalf("generated differentiated text = %q, want %q", row.Text, want)
		}
		if strings.ContainsRune(row.Text, '\n') {
			t.Fatalf("generated differentiated text contains real newline: %q", row.Text)
		}
		return
	}
	t.Fatal("generated localization patch does not contain row different")
}

func TestGenerateDefaultsToAcceptedTaggedStyle(t *testing.T) {
	gameRoot := createGameRoot(t, true, true)
	output := filepath.Join(t.TempDir(), "dual-subtitles.zip")

	result, err := Generate(Request{
		GameRoot:          gameRoot,
		MainLanguage:      localization.Russian,
		SecondaryLanguage: localization.English,
		OutputPath:        output,
		Version:           "v0.3.0-style-test",
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if result.SubtitleStyle != SubtitleStyleTagged {
		t.Fatalf("SubtitleStyle = %q, want %q", result.SubtitleStyle, SubtitleStyleTagged)
	}
}

func TestGenerateRejectsUnsupportedSubtitleStyle(t *testing.T) {
	_, err := Generate(Request{
		GameRoot:          createGameRoot(t, true, true),
		MainLanguage:      localization.Russian,
		SecondaryLanguage: localization.English,
		SubtitleStyle:     SubtitleStyle("unknown"),
		OutputPath:        filepath.Join(t.TempDir(), "dual-subtitles.zip"),
	})
	if err == nil {
		t.Fatal("Generate() error = nil, want unsupported subtitle style error")
	}
	if !strings.Contains(err.Error(), "unsupported subtitle style") {
		t.Fatalf("Generate() error = %v, want unsupported subtitle style", err)
	}
}
