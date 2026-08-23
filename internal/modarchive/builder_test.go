package modarchive

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
)

func TestBuildArchiveBytesStructure(t *testing.T) {
	rows := []localization.DialogueRow{
		{ID: "dialog_one", Source: "source & <meta>", Text: "Русский\\nEnglish"},
		{ID: "dialog_multiline", Source: "source", Text: "Первая строка.\nВторая строка.\\nFirst line.\nSecond line."},
	}

	tests := []struct {
		name           string
		language       localization.Language
		wantPAKArchive string
	}{
		{name: "Russian", language: localization.Russian, wantPAKArchive: modArchivePath("Localization/Russian_xml.pak")},
		{name: "English", language: localization.English, wantPAKArchive: modArchivePath("Localization/English_xml.pak")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			archiveData, err := buildArchiveBytes(tt.language, rows)
			if err != nil {
				t.Fatalf("buildArchiveBytes() error = %v", err)
			}

			outer := openZipBytes(t, archiveData)
			manifestPath := modArchivePath(ManifestFilename)
			wantOuterNames := []string{manifestPath, tt.wantPAKArchive}
			if got := zipNames(outer); !reflect.DeepEqual(got, wantOuterNames) {
				t.Fatalf("outer ZIP entries = %#v, want %#v", got, wantOuterNames)
			}

			manifestData := readZipEntry(t, outer, manifestPath)
			manifestText := string(manifestData)
			for _, want := range []string{
				"<name>KCD2 Dual Subtitles</name>",
				"<modid>kcd_dual_subtitles</modid>",
				"<author>Vyachean</author>",
				"<version>0.1.0</version>",
				"<created_on>2026-08-23</created_on>",
			} {
				if !strings.Contains(manifestText, want) {
					t.Fatalf("manifest does not contain %q:\n%s", want, manifestText)
				}
			}

			pakData := readZipEntry(t, outer, tt.wantPAKArchive)
			nested := openZipBytes(t, pakData)
			if got := zipNames(nested); !reflect.DeepEqual(got, []string{localization.DialogueXMLArchivePath}) {
				t.Fatalf("nested PAK entries = %#v, want [%q]", got, localization.DialogueXMLArchivePath)
			}
			if len(nested.File) != 1 {
				t.Fatalf("nested PAK entries = %d, want 1", len(nested.File))
			}
			entry := nested.File[0]
			if entry.Method != zip.Store {
				t.Fatalf("nested PAK compression = %d, want zip.Store (%d)", entry.Method, zip.Store)
			}
			if len(entry.Extra) != 0 {
				t.Fatalf("nested PAK entry has ZIP extra fields: %x", entry.Extra)
			}
			if entry.ModifiedDate != deterministicDOSDate || entry.ModifiedTime != deterministicDOSTime {
				t.Fatalf("nested PAK DOS timestamp = date:%d time:%d, want date:%d time:%d", entry.ModifiedDate, entry.ModifiedTime, deterministicDOSDate, deterministicDOSTime)
			}

			xmlData := readZipEntry(t, nested, localization.DialogueXMLArchivePath)
			parsed, err := localization.ParseDialogueXML(xmlData)
			if err != nil {
				t.Fatalf("parse generated dialogue XML: %v", err)
			}
			if !reflect.DeepEqual(parsed, rows) {
				t.Fatalf("generated dialogue rows = %#v, want %#v", parsed, rows)
			}
		})
	}
}

func TestGeneratedPathsMatchOverrideContract(t *testing.T) {
	if ModID != "kcd_dual_subtitles" {
		t.Fatalf("ModID = %q, want kcd_dual_subtitles", ModID)
	}
	if strings.ContainsAny(ModID, "0123456789-") {
		t.Fatalf("ModID contains characters excluded by the official mod-id contract: %q", ModID)
	}
	if localization.DialogueXMLArchivePath != "text_ui_dialog.xml" {
		t.Fatalf("DialogueXMLArchivePath = %q, want text_ui_dialog.xml", localization.DialogueXMLArchivePath)
	}
}

func TestBuildArchiveBytesDeterministic(t *testing.T) {
	rows := []localization.DialogueRow{{ID: "id", Source: "source", Text: "Основной\\nSecondary"}}

	first, err := buildArchiveBytes(localization.Russian, rows)
	if err != nil {
		t.Fatalf("first buildArchiveBytes() error = %v", err)
	}
	second, err := buildArchiveBytes(localization.Russian, rows)
	if err != nil {
		t.Fatalf("second buildArchiveBytes() error = %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("buildArchiveBytes() is not byte-deterministic for identical input")
	}
}

func TestBuildRejectsUnsupportedLanguage(t *testing.T) {
	output := filepath.Join(t.TempDir(), "mod.zip")

	err := Build(output, localization.Language("German"), []localization.DialogueRow{{ID: "id"}})
	if !errors.Is(err, ErrUnsupportedLanguage) {
		t.Fatalf("Build() error = %v, want errors.Is(..., ErrUnsupportedLanguage)", err)
	}
	if _, statErr := os.Stat(output); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("output exists after unsupported-language failure: %v", statErr)
	}
}

func TestBuildRejectsInvalidDialogueRowWithoutResidue(t *testing.T) {
	directory := t.TempDir()
	output := filepath.Join(directory, "mod.zip")

	err := Build(output, localization.Russian, []localization.DialogueRow{{ID: "", Text: "invalid"}})
	if !errors.Is(err, localization.ErrInvalidDialogueRow) {
		t.Fatalf("Build() error = %v, want errors.Is(..., ErrInvalidDialogueRow)", err)
	}

	entries, readErr := os.ReadDir(directory)
	if readErr != nil {
		t.Fatalf("ReadDir() error = %v", readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("failed generation left files behind: %#v", entries)
	}
}

func TestBuildDoesNotOverwriteExistingOutput(t *testing.T) {
	directory := t.TempDir()
	output := filepath.Join(directory, "mod.zip")
	original := []byte("existing output")
	if err := os.WriteFile(output, original, 0o600); err != nil {
		t.Fatalf("write existing output: %v", err)
	}

	err := Build(output, localization.Russian, []localization.DialogueRow{{ID: "id", Text: "text"}})
	if !errors.Is(err, ErrOutputExists) {
		t.Fatalf("Build() error = %v, want errors.Is(..., ErrOutputExists)", err)
	}

	got, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read existing output: %v", err)
	}
	if !bytes.Equal(got, original) {
		t.Fatalf("existing output changed: got %q, want %q", got, original)
	}
}

func TestBuildPublishesCompleteArchiveAndRemovesTemp(t *testing.T) {
	directory := t.TempDir()
	output := filepath.Join(directory, "dual-subtitles.zip")
	rows := []localization.DialogueRow{{ID: "id", Source: "source", Text: "Русский\\nEnglish"}}

	expected, err := buildArchiveBytes(localization.Russian, rows)
	if err != nil {
		t.Fatalf("buildArchiveBytes() error = %v", err)
	}
	if err := Build(output, localization.Russian, rows); err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	got, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read built output: %v", err)
	}
	if !bytes.Equal(got, expected) {
		t.Fatal("published archive bytes differ from fully built archive")
	}

	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(output) {
		t.Fatalf("output directory contains unexpected residue: %#v", entries)
	}
}

func openZipBytes(t *testing.T, data []byte) *zip.Reader {
	t.Helper()

	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("open ZIP bytes: %v", err)
	}
	return reader
}

func zipNames(reader *zip.Reader) []string {
	names := make([]string, len(reader.File))
	for i, file := range reader.File {
		names[i] = file.Name
	}
	return names
}

func readZipEntry(t *testing.T, reader *zip.Reader, name string) []byte {
	t.Helper()

	for _, file := range reader.File {
		if file.Name != name {
			continue
		}
		entry, err := file.Open()
		if err != nil {
			t.Fatalf("open ZIP entry %q: %v", name, err)
		}
		data, readErr := io.ReadAll(entry)
		closeErr := entry.Close()
		if readErr != nil {
			t.Fatalf("read ZIP entry %q: %v", name, readErr)
		}
		if closeErr != nil {
			t.Fatalf("close ZIP entry %q: %v", name, closeErr)
		}
		return data
	}

	t.Fatalf("ZIP entry %q not found", name)
	return nil
}
