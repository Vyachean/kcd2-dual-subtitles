package generator

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/modarchive"
)

const czechFixture = `<?xml version="1.0" encoding="utf-8"?>
<Table>
<Row><Cell>different</Cell><Cell>source</Cell><Cell>Hlavní</Cell></Row>
<Row><Cell>identical</Cell><Cell>source</Cell><Cell>[pause]</Cell></Row>
</Table>`

const germanFixture = `<?xml version="1.0" encoding="utf-8"?>
<Table>
<Row><Cell>different</Cell><Cell>source</Cell><Cell>Sekundär</Cell></Row>
<Row><Cell>identical</Cell><Cell>source</Cell><Cell>[pause]</Cell></Row>
</Table>`

func TestGeneratePublishesSelectedPairOnly(t *testing.T) {
	gameRoot := t.TempDir()
	localizationDir := filepath.Join(gameRoot, "Localization")
	if err := os.Mkdir(localizationDir, 0o755); err != nil {
		t.Fatalf("create Localization directory: %v", err)
	}
	writeLanguagePAK(t, filepath.Join(localizationDir, "English_xml.pak"), englishFixture)
	writeLanguagePAK(t, filepath.Join(localizationDir, "Czech_xml.pak"), czechFixture)
	writeLanguagePAK(t, filepath.Join(localizationDir, "German_xml.pak"), germanFixture)

	output := filepath.Join(t.TempDir(), "czech-german.zip")
	result, err := Generate(Request{
		GameRoot:          gameRoot,
		MainLanguage:      localization.Czech,
		SecondaryLanguage: localization.German,
		OutputPath:        output,
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if result.LocalizationTargets != 2 {
		t.Fatalf("LocalizationTargets = %d, want 2", result.LocalizationTargets)
	}

	archive, err := zip.OpenReader(output)
	if err != nil {
		t.Fatalf("open generated archive: %v", err)
	}
	defer archive.Close()

	found := make(map[string]bool)
	for _, file := range archive.File {
		found[file.Name] = true
	}
	for _, pak := range []string{"Czech_xml.pak", "German_xml.pak"} {
		want := filepath.ToSlash(filepath.Join(modarchive.ModID, "Localization", pak))
		if !found[want] {
			t.Fatalf("generated archive missing %q; entries=%v", want, found)
		}
	}
	unexpected := filepath.ToSlash(filepath.Join(modarchive.ModID, "Localization", "English_xml.pak"))
	if found[unexpected] {
		t.Fatalf("generated archive unexpectedly contains unselected localization target %q", unexpected)
	}
}
