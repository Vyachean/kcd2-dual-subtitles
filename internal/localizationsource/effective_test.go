package localizationsource

import (
	"archive/zip"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/modarchive"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/modinstall"
)

func TestResolveFromModsRootStockOnly(t *testing.T) {
	stock := []localization.DialogueRow{{ID: "a", Source: "s", Text: "stock"}}
	result, err := resolveFromModsRoot(stock, filepath.Join(t.TempDir(), "missing"), "Chineses_xml.pak")
	if err != nil {
		t.Fatalf("resolveFromModsRoot() error = %v", err)
	}
	if !reflect.DeepEqual(result.Rows, stock) || len(result.Contributions) != 0 {
		t.Fatalf("result = %+v, want unchanged stock with no contributions", result)
	}
}

func TestResolveFromModsRootAppliesFullDialogueOverrideAndFallback(t *testing.T) {
	modsRoot := t.TempDir()
	writeLocalizationMod(t, modsRoot, "chinesesfixptf", "chinesesfixptf", "Chineses Fix", "Chineses_xml.pak", map[string]string{
		localization.DialogueXMLArchivePath: dialogueXML(
			localization.DialogueRow{ID: "a", Source: "mod", Text: "corrected"},
			localization.DialogueRow{ID: "new", Source: "mod", Text: "new dialogue"},
		),
	})
	stock := []localization.DialogueRow{
		{ID: "a", Source: "stock", Text: "old"},
		{ID: "b", Source: "stock", Text: "fallback"},
	}

	result, err := resolveFromModsRoot(stock, modsRoot, "Chineses_xml.pak")
	if err != nil {
		t.Fatalf("resolveFromModsRoot() error = %v", err)
	}
	want := []localization.DialogueRow{
		{ID: "a", Source: "mod", Text: "corrected"},
		{ID: "b", Source: "stock", Text: "fallback"},
		{ID: "new", Source: "mod", Text: "new dialogue"},
	}
	if !reflect.DeepEqual(result.Rows, want) {
		t.Fatalf("Rows = %#v, want %#v", result.Rows, want)
	}
	if len(result.Contributions) != 1 || result.Contributions[0].ModID != "chinesesfixptf" {
		t.Fatalf("Contributions = %+v", result.Contributions)
	}
}

func TestResolveFromModsRootPatchOverridesKnownDialogueOnly(t *testing.T) {
	modsRoot := t.TempDir()
	writeLocalizationMod(t, modsRoot, "fix", "fix", "Fix", "English_xml.pak", map[string]string{
		"text_ui__fix.xml": dialogueXML(
			localization.DialogueRow{ID: "dialogue", Source: "mod", Text: "fixed"},
			localization.DialogueRow{ID: "unrelated-ui", Source: "mod", Text: "ignore"},
		),
	})
	stock := []localization.DialogueRow{{ID: "dialogue", Source: "stock", Text: "old"}}

	result, err := resolveFromModsRoot(stock, modsRoot, "English_xml.pak")
	if err != nil {
		t.Fatalf("resolveFromModsRoot() error = %v", err)
	}
	want := []localization.DialogueRow{{ID: "dialogue", Source: "mod", Text: "fixed"}}
	if !reflect.DeepEqual(result.Rows, want) {
		t.Fatalf("Rows = %#v, want %#v", result.Rows, want)
	}
}

func TestResolveFromModsRootLaterModWins(t *testing.T) {
	modsRoot := t.TempDir()
	writeLocalizationMod(t, modsRoot, "a-first", "first", "First", "English_xml.pak", map[string]string{
		localization.DialogueXMLArchivePath: dialogueXML(localization.DialogueRow{ID: "a", Source: "first", Text: "first"}),
	})
	writeLocalizationMod(t, modsRoot, "z-last", "last", "Last", "English_xml.pak", map[string]string{
		localization.DialogueXMLArchivePath: dialogueXML(localization.DialogueRow{ID: "a", Source: "last", Text: "last"}),
	})

	result, err := resolveFromModsRoot([]localization.DialogueRow{{ID: "a", Source: "stock", Text: "stock"}}, modsRoot, "English_xml.pak")
	if err != nil {
		t.Fatalf("resolveFromModsRoot() error = %v", err)
	}
	if got := result.Rows[0].Text; got != "last" {
		t.Fatalf("final text = %q, want last", got)
	}
}

func TestResolveFromModsRootModOrderIsWhitelistAndOverridesAlphabeticalOrder(t *testing.T) {
	modsRoot := t.TempDir()
	writeLocalizationMod(t, modsRoot, "a-folder", "mod_a", "A", "English_xml.pak", map[string]string{
		localization.DialogueXMLArchivePath: dialogueXML(localization.DialogueRow{ID: "a", Source: "a", Text: "A"}),
	})
	writeLocalizationMod(t, modsRoot, "z-folder", "mod_z", "Z", "English_xml.pak", map[string]string{
		localization.DialogueXMLArchivePath: dialogueXML(localization.DialogueRow{ID: "a", Source: "z", Text: "Z"}),
	})
	writeLocalizationMod(t, modsRoot, "inactive", "inactive", "Inactive", "English_xml.pak", map[string]string{
		localization.DialogueXMLArchivePath: dialogueXML(localization.DialogueRow{ID: "a", Source: "inactive", Text: "INACTIVE"}),
	})
	if err := os.WriteFile(filepath.Join(modsRoot, modinstall.ModOrderFilename), []byte("mod_z\nmod_a\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := resolveFromModsRoot([]localization.DialogueRow{{ID: "a", Source: "stock", Text: "stock"}}, modsRoot, "English_xml.pak")
	if err != nil {
		t.Fatalf("resolveFromModsRoot() error = %v", err)
	}
	if got := result.Rows[0].Text; got != "A" {
		t.Fatalf("final text = %q, want A from explicit last mod", got)
	}
	if len(result.Contributions) != 2 || result.Contributions[0].ModID != "mod_z" || result.Contributions[1].ModID != "mod_a" {
		t.Fatalf("Contributions = %+v", result.Contributions)
	}
}

func TestResolveFromModsRootManifestIDCanDifferFromFolder(t *testing.T) {
	modsRoot := t.TempDir()
	writeLocalizationMod(t, modsRoot, "folder-name", "real_id", "Real", "English_xml.pak", map[string]string{
		localization.DialogueXMLArchivePath: dialogueXML(localization.DialogueRow{ID: "a", Source: "mod", Text: "fixed"}),
	})
	if err := os.WriteFile(filepath.Join(modsRoot, modinstall.ModOrderFilename), []byte("real_id\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := resolveFromModsRoot([]localization.DialogueRow{{ID: "a", Source: "stock", Text: "stock"}}, modsRoot, "English_xml.pak")
	if err != nil {
		t.Fatalf("resolveFromModsRoot() error = %v", err)
	}
	if got := result.Rows[0].Text; got != "fixed" {
		t.Fatalf("final text = %q, want fixed", got)
	}
}

func TestResolveFromModsRootIgnoresOtherLanguagesAndNoDialogueResources(t *testing.T) {
	modsRoot := t.TempDir()
	writeLocalizationMod(t, modsRoot, "other-language", "other_language", "Other", "Russian_xml.pak", map[string]string{
		localization.DialogueXMLArchivePath: dialogueXML(localization.DialogueRow{ID: "a", Source: "mod", Text: "wrong language"}),
	})
	writeLocalizationMod(t, modsRoot, "no-dialogue", "no_dialogue", "No dialogue", "English_xml.pak", map[string]string{
		"text_ui_items.xml": dialogueXML(localization.DialogueRow{ID: "a", Source: "mod", Text: "wrong table"}),
	})
	stock := []localization.DialogueRow{{ID: "a", Source: "stock", Text: "stock"}}

	result, err := resolveFromModsRoot(stock, modsRoot, "English_xml.pak")
	if err != nil {
		t.Fatalf("resolveFromModsRoot() error = %v", err)
	}
	if !reflect.DeepEqual(result.Rows, stock) || len(result.Contributions) != 0 {
		t.Fatalf("result = %+v, want stock only", result)
	}
}

func TestResolveFromModsRootIgnoresInvalidManifest(t *testing.T) {
	modsRoot := t.TempDir()
	writeLocalizationMod(t, modsRoot, "invalid", "invalid-id", "Invalid", "English_xml.pak", map[string]string{
		localization.DialogueXMLArchivePath: dialogueXML(localization.DialogueRow{ID: "a", Source: "mod", Text: "must not load"}),
	})
	stock := []localization.DialogueRow{{ID: "a", Source: "stock", Text: "stock"}}

	result, err := resolveFromModsRoot(stock, modsRoot, "English_xml.pak")
	if err != nil {
		t.Fatalf("resolveFromModsRoot() error = %v", err)
	}
	if !reflect.DeepEqual(result.Rows, stock) || len(result.Contributions) != 0 {
		t.Fatalf("result = %+v, want invalid mod ignored", result)
	}
}

func TestResolveFromModsRootRejectsMalformedAndDuplicateDialogueResources(t *testing.T) {
	t.Run("malformed", func(t *testing.T) {
		modsRoot := t.TempDir()
		writeLocalizationMod(t, modsRoot, "bad", "bad", "Bad", "English_xml.pak", map[string]string{
			localization.DialogueXMLArchivePath: `<Table><Row>`,
		})
		_, err := resolveFromModsRoot([]localization.DialogueRow{{ID: "a", Text: "stock"}}, modsRoot, "English_xml.pak")
		if err == nil || !strings.Contains(err.Error(), "bad") || !strings.Contains(err.Error(), localization.DialogueXMLArchivePath) {
			t.Fatalf("error = %v, want source-specific malformed error", err)
		}
	})

	t.Run("duplicate", func(t *testing.T) {
		modsRoot := t.TempDir()
		writeLocalizationMod(t, modsRoot, "dup", "dup", "Dup", "English_xml.pak", map[string]string{
			localization.DialogueXMLArchivePath: dialogueXML(
				localization.DialogueRow{ID: "a", Text: "one"},
				localization.DialogueRow{ID: "a", Text: "two"},
			),
		})
		_, err := resolveFromModsRoot([]localization.DialogueRow{{ID: "a", Text: "stock"}}, modsRoot, "English_xml.pak")
		if err == nil || !strings.Contains(err.Error(), "duplicate dialogue ID") {
			t.Fatalf("error = %v, want duplicate-ID error", err)
		}
	})
}

func TestResolveFromModsRootExcludesOwnCanonicalAndLegacyStaging(t *testing.T) {
	modsRoot := t.TempDir()
	for _, folder := range []string{modarchive.ModID, "." + modarchive.ModID + ".staging-123"} {
		writeLocalizationMod(t, modsRoot, folder, modarchive.ModID, "Dual Subtitles", "English_xml.pak", map[string]string{
			localization.DialogueXMLArchivePath: dialogueXML(localization.DialogueRow{ID: "a", Source: "self", Text: "self"}),
		})
	}
	stock := []localization.DialogueRow{{ID: "a", Source: "stock", Text: "stock"}}

	result, err := resolveFromModsRoot(stock, modsRoot, "English_xml.pak")
	if err != nil {
		t.Fatalf("resolveFromModsRoot() error = %v", err)
	}
	if !reflect.DeepEqual(result.Rows, stock) || len(result.Contributions) != 0 {
		t.Fatalf("result = %+v, want self-owned sources excluded", result)
	}
}

func writeLocalizationMod(t *testing.T, modsRoot, folder, modID, name, pakFilename string, resources map[string]string) {
	t.Helper()
	dir := filepath.Join(modsRoot, folder)
	if err := os.MkdirAll(filepath.Join(dir, "Localization"), 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `<?xml version="1.0"?><kcd_mod><info><name>` + name + `</name><modid>` + modID + `</modid></info></kcd_mod>`
	if err := os.WriteFile(filepath.Join(dir, "mod.manifest"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(filepath.Join(dir, "Localization", pakFilename))
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	for resource, contents := range resources {
		entry, err := writer.Create(resource)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(contents)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func dialogueXML(rows ...localization.DialogueRow) string {
	var builder strings.Builder
	builder.WriteString(`<?xml version="1.0" encoding="utf-8"?><Table>`)
	for _, row := range rows {
		builder.WriteString("<Row><Cell>")
		builder.WriteString(row.ID)
		builder.WriteString("</Cell><Cell>")
		builder.WriteString(row.Source)
		builder.WriteString("</Cell><Cell>")
		builder.WriteString(row.Text)
		builder.WriteString("</Cell></Row>")
	}
	builder.WriteString("</Table>")
	return builder.String()
}
