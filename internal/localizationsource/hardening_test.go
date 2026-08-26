package localizationsource

import (
	"archive/zip"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
)

func TestResolveFromModsRootDoesNotReportIdenticalLocalizationAsContribution(t *testing.T) {
	modsRoot := t.TempDir()
	row := localization.DialogueRow{ID: "a", Source: "same", Text: "same"}
	writeLocalizationMod(t, modsRoot, "same", "same", "Same", "English_xml.pak", map[string]string{
		localization.DialogueXMLArchivePath: dialogueXML(row),
	})

	result, err := resolveFromModsRoot([]localization.DialogueRow{row}, modsRoot, "English_xml.pak")
	if err != nil {
		t.Fatalf("resolveFromModsRoot() error = %v", err)
	}
	if !reflect.DeepEqual(result.Rows, []localization.DialogueRow{row}) || len(result.Contributions) != 0 {
		t.Fatalf("result = %+v, want unchanged rows and no contributions", result)
	}
}

func TestResolveFromModsRootAcceptsCaseInsensitiveDialogueResourceName(t *testing.T) {
	modsRoot := t.TempDir()
	writeLocalizationMod(t, modsRoot, "casefix", "casefix", "Case Fix", "English_xml.pak", map[string]string{
		"TEXT_UI_DIALOG.XML": dialogueXML(localization.DialogueRow{ID: "a", Source: "mod", Text: "fixed"}),
	})

	result, err := resolveFromModsRoot(
		[]localization.DialogueRow{{ID: "a", Source: "stock", Text: "stock"}},
		modsRoot,
		"English_xml.pak",
	)
	if err != nil {
		t.Fatalf("resolveFromModsRoot() error = %v", err)
	}
	if got := result.Rows[0].Text; got != "fixed" {
		t.Fatalf("effective text = %q, want fixed", got)
	}
}

func TestResolveFromModsRootRejectsCaseFoldDuplicateResources(t *testing.T) {
	modsRoot := t.TempDir()
	writeLocalizationMod(t, modsRoot, "duplicate", "duplicate", "Duplicate", "English_xml.pak", map[string]string{
		"text_ui_dialog.xml": dialogueXML(localization.DialogueRow{ID: "a", Source: "one", Text: "one"}),
		"TEXT_UI_DIALOG.XML": dialogueXML(localization.DialogueRow{ID: "a", Source: "two", Text: "two"}),
	})

	_, err := resolveFromModsRoot(
		[]localization.DialogueRow{{ID: "a", Source: "stock", Text: "stock"}},
		modsRoot,
		"English_xml.pak",
	)
	if err == nil || !strings.Contains(err.Error(), "duplicate localization resource") {
		t.Fatalf("error = %v, want duplicate-resource error", err)
	}
}

func TestReadZipEntryLimitedRejectsOversizedResource(t *testing.T) {
	pakPath := filepath.Join(t.TempDir(), "test.pak")
	file, err := os.Create(pakPath)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	entry, err := writer.Create("text_ui_dialog.xml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte("0123456789")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	reader, err := zip.OpenReader(pakPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	if len(reader.File) != 1 {
		t.Fatalf("archive entries = %d, want 1", len(reader.File))
	}
	if _, err := readZipEntryLimited(reader.File[0], 5); err == nil || !strings.Contains(err.Error(), "size limit") {
		t.Fatalf("readZipEntryLimited() error = %v, want size-limit error", err)
	}
}
