package generator

import (
	"archive/zip"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
)

const russianFixture = `<?xml version="1.0" encoding="utf-8"?>
<Table>
<Row><Cell>different</Cell><Cell>source</Cell><Cell>Основной</Cell></Row>
<Row><Cell>identical</Cell><Cell>source</Cell><Cell>[pause]</Cell></Row>
<Row><Cell>missing</Cell><Cell>source</Cell><Cell>Только основной</Cell></Row>
</Table>`

const englishFixture = `<?xml version="1.0" encoding="utf-8"?>
<Table>
<Row><Cell>different</Cell><Cell>source</Cell><Cell>Secondary</Cell></Row>
<Row><Cell>identical</Cell><Cell>source</Cell><Cell>[pause]</Cell></Row>
</Table>`

func TestGenerateXboxStyleRootEndToEnd(t *testing.T) {
	gameRoot := createGameRoot(t, true, true)
	if _, err := os.Stat(filepath.Join(gameRoot, "KingdomCome.exe")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("test root unexpectedly contains KingdomCome.exe: %v", err)
	}

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

	wantStats := localization.MergeStats{
		Processed:        3,
		Bilingual:        1,
		Identical:        1,
		MissingSecondary: 1,
		SecondaryOnly:    0,
	}
	if result.OutputPath != output {
		t.Fatalf("Result.OutputPath = %q, want %q", result.OutputPath, output)
	}
	if result.Stats != wantStats {
		t.Fatalf("Result.Stats = %+v, want %+v", result.Stats, wantStats)
	}
	if info, err := os.Stat(output); err != nil || info.IsDir() {
		t.Fatalf("generated output is not a file: info=%v err=%v", info, err)
	}
}

func TestGenerateRejectsInvalidRoot(t *testing.T) {
	output := filepath.Join(t.TempDir(), "out.zip")
	_, err := Generate(Request{
		GameRoot:          t.TempDir(),
		MainLanguage:      localization.Russian,
		SecondaryLanguage: localization.English,
		OutputPath:        output,
	})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("Generate() error = %v, want errors.Is(..., ErrInvalidRequest)", err)
	}
}

func TestGenerateRejectsMissingLanguagePak(t *testing.T) {
	gameRoot := createGameRoot(t, true, false)
	output := filepath.Join(t.TempDir(), "out.zip")

	_, err := Generate(Request{
		GameRoot:          gameRoot,
		MainLanguage:      localization.Russian,
		SecondaryLanguage: localization.English,
		OutputPath:        output,
	})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("Generate() error = %v, want errors.Is(..., ErrInvalidRequest)", err)
	}
}

func TestGenerateRejectsSameLanguage(t *testing.T) {
	_, err := Generate(Request{
		GameRoot:          "unused",
		MainLanguage:      localization.Russian,
		SecondaryLanguage: localization.Russian,
		OutputPath:        "unused.zip",
	})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("Generate() error = %v, want errors.Is(..., ErrInvalidRequest)", err)
	}
}

func TestGenerateReportsMalformedPakAsRuntimeFailure(t *testing.T) {
	gameRoot := t.TempDir()
	localizationDir := filepath.Join(gameRoot, "Localization")
	if err := os.Mkdir(localizationDir, 0o755); err != nil {
		t.Fatalf("create Localization directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(localizationDir, "Russian_xml.pak"), []byte("not a zip"), 0o600); err != nil {
		t.Fatalf("write malformed Russian PAK: %v", err)
	}
	writeLanguagePAK(t, filepath.Join(localizationDir, "English_xml.pak"), englishFixture)

	_, err := Generate(Request{
		GameRoot:          gameRoot,
		MainLanguage:      localization.Russian,
		SecondaryLanguage: localization.English,
		OutputPath:        filepath.Join(t.TempDir(), "out.zip"),
	})
	if err == nil {
		t.Fatal("Generate() error = nil, want runtime failure")
	}
	if errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("malformed PAK was classified as invalid request: %v", err)
	}
}

func createGameRoot(t *testing.T, includeRussian, includeEnglish bool) string {
	t.Helper()

	gameRoot := t.TempDir()
	localizationDir := filepath.Join(gameRoot, "Localization")
	if err := os.Mkdir(localizationDir, 0o755); err != nil {
		t.Fatalf("create Localization directory: %v", err)
	}
	if includeRussian {
		writeLanguagePAK(t, filepath.Join(localizationDir, "Russian_xml.pak"), russianFixture)
	}
	if includeEnglish {
		writeLanguagePAK(t, filepath.Join(localizationDir, "English_xml.pak"), englishFixture)
	}
	return gameRoot
}

func writeLanguagePAK(t *testing.T, path, dialogueXML string) {
	t.Helper()

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create language PAK %q: %v", path, err)
	}
	writer := zip.NewWriter(file)
	entry, err := writer.Create(localization.DialogueXMLArchivePath)
	if err != nil {
		_ = writer.Close()
		_ = file.Close()
		t.Fatalf("create dialogue XML entry: %v", err)
	}
	if _, err := entry.Write([]byte(dialogueXML)); err != nil {
		_ = writer.Close()
		_ = file.Close()
		t.Fatalf("write dialogue XML entry: %v", err)
	}
	if err := writer.Close(); err != nil {
		_ = file.Close()
		t.Fatalf("close language PAK writer: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close language PAK: %v", err)
	}
}
