package generator

import (
	"archive/zip"
	"bytes"
	"io"
	"path/filepath"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/modarchive"
)

func TestGenerateWritesTaggedLocalizationPatchInSelectedLanguageOrder(t *testing.T) {
	tests := []struct {
		name      string
		main      localization.Language
		secondary localization.Language
		pakName   string
		wantText  string
	}{
		{
			name:      "Russian main",
			main:      localization.Russian,
			secondary: localization.English,
			pakName:   "Russian_xml.pak",
			wantText:  `[RU] Основной\n[EN] Secondary`,
		},
		{
			name:      "English main",
			main:      localization.English,
			secondary: localization.Russian,
			pakName:   "English_xml.pak",
			wantText:  `[EN] Secondary\n[RU] Основной`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gameRoot := createGameRoot(t, true, true)
			output := filepath.Join(t.TempDir(), "dual-subtitles.zip")
			_, err := Generate(Request{
				GameRoot:          gameRoot,
				MainLanguage:      tt.main,
				SecondaryLanguage: tt.secondary,
				OutputPath:        output,
				Version:           "v0.1.0-test",
			})
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}

			outer, err := zip.OpenReader(output)
			if err != nil {
				t.Fatalf("open generated archive: %v", err)
			}
			defer outer.Close()

			pakData := readArchiveEntry(t, &outer.Reader, "kcd_dual_subtitles/Localization/"+tt.pakName)
			pak, err := zip.NewReader(bytes.NewReader(pakData), int64(len(pakData)))
			if err != nil {
				t.Fatalf("open generated localization PAK: %v", err)
			}
			patchData := readArchiveEntry(t, pak, modarchive.LocalizationPatchArchivePath)
			rows, err := localization.ParseDialogueXML(patchData)
			if err != nil {
				t.Fatalf("parse generated localization patch: %v", err)
			}

			var got string
			for _, row := range rows {
				if row.ID == "different" {
					got = row.Text
					break
				}
			}
			if got != tt.wantText {
				t.Fatalf("generated bilingual text = %q, want %q", got, tt.wantText)
			}
		})
	}
}

func readArchiveEntry(t *testing.T, reader *zip.Reader, name string) []byte {
	t.Helper()
	for _, file := range reader.File {
		if file.Name != name {
			continue
		}
		entry, err := file.Open()
		if err != nil {
			t.Fatalf("open archive entry %q: %v", name, err)
		}
		data, readErr := io.ReadAll(entry)
		closeErr := entry.Close()
		if readErr != nil {
			t.Fatalf("read archive entry %q: %v", name, readErr)
		}
		if closeErr != nil {
			t.Fatalf("close archive entry %q: %v", name, closeErr)
		}
		return data
	}
	t.Fatalf("archive entry %q not found", name)
	return nil
}
