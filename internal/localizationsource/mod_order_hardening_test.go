package localizationsource

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/modinstall"
)

func TestModOrderIDMatchingIsExact(t *testing.T) {
	modsRoot := t.TempDir()
	writeLocalizationMod(t, modsRoot, "folder", "real_id", "Real", "English_xml.pak", map[string]string{
		localization.DialogueXMLArchivePath: dialogueXML(localization.DialogueRow{ID: "a", Source: "mod", Text: "override"}),
	})
	if err := os.WriteFile(filepath.Join(modsRoot, modinstall.ModOrderFilename), []byte("REAL_ID\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	stock := []localization.DialogueRow{{ID: "a", Source: "stock", Text: "stock"}}

	result, err := resolveFromModsRoot(stock, modsRoot, "English_xml.pak")
	if err != nil {
		t.Fatalf("resolveFromModsRoot() error = %v", err)
	}
	if !reflect.DeepEqual(result.Rows, stock) || len(result.Contributions) != 0 {
		t.Fatalf("result = %+v, want uppercase order ID not to activate lowercase modid", result)
	}
}

func TestDuplicateActiveModOrderIDFailsClosed(t *testing.T) {
	modsRoot := t.TempDir()
	writeLocalizationMod(t, modsRoot, "folder", "real_id", "Real", "English_xml.pak", map[string]string{
		localization.DialogueXMLArchivePath: dialogueXML(localization.DialogueRow{ID: "a", Source: "mod", Text: "override"}),
	})
	if err := os.WriteFile(filepath.Join(modsRoot, modinstall.ModOrderFilename), []byte("real_id\nreal_id\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := resolveFromModsRoot([]localization.DialogueRow{{ID: "a", Source: "stock", Text: "stock"}}, modsRoot, "English_xml.pak")
	if err == nil || !strings.Contains(err.Error(), "duplicate active localization mod ID") {
		t.Fatalf("error = %v, want duplicate active mod-order ID failure", err)
	}
}

func TestModOrderUTF8BOMDoesNotChangeFirstID(t *testing.T) {
	modsRoot := t.TempDir()
	writeLocalizationMod(t, modsRoot, "folder", "real_id", "Real", "English_xml.pak", map[string]string{
		localization.DialogueXMLArchivePath: dialogueXML(localization.DialogueRow{ID: "a", Source: "mod", Text: "override"}),
	})
	if err := os.WriteFile(filepath.Join(modsRoot, modinstall.ModOrderFilename), append([]byte{0xEF, 0xBB, 0xBF}, []byte("real_id\n")...), 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := resolveFromModsRoot([]localization.DialogueRow{{ID: "a", Source: "stock", Text: "stock"}}, modsRoot, "English_xml.pak")
	if err != nil {
		t.Fatalf("resolveFromModsRoot() error = %v", err)
	}
	if len(result.Contributions) != 1 || result.Rows[0].Text != "override" {
		t.Fatalf("result = %+v, want BOM-prefixed first order ID to activate mod", result)
	}
}

func TestManifestWithoutNameOrModIDIsInactive(t *testing.T) {
	modsRoot := t.TempDir()
	writeLocalizationMod(t, modsRoot, "anonymous", "temporary", "Temporary", "English_xml.pak", map[string]string{
		localization.DialogueXMLArchivePath: dialogueXML(localization.DialogueRow{ID: "a", Source: "mod", Text: "override"}),
	})
	manifestPath := filepath.Join(modsRoot, "anonymous", "mod.manifest")
	if err := os.WriteFile(manifestPath, []byte(`<?xml version="1.0"?><kcd_mod><info></info></kcd_mod>`), 0o600); err != nil {
		t.Fatal(err)
	}
	stock := []localization.DialogueRow{{ID: "a", Source: "stock", Text: "stock"}}

	result, err := resolveFromModsRoot(stock, modsRoot, "English_xml.pak")
	if err != nil {
		t.Fatalf("resolveFromModsRoot() error = %v", err)
	}
	if !reflect.DeepEqual(result.Rows, stock) || len(result.Contributions) != 0 {
		t.Fatalf("result = %+v, want identity-less manifest inactive", result)
	}
}

func TestUnlistedModDoesNotValidateItsLocalizationPAK(t *testing.T) {
	modsRoot := t.TempDir()
	writeLocalizationMod(t, modsRoot, "unlisted", "unlisted", "Unlisted", "English_xml.pak", map[string]string{
		localization.DialogueXMLArchivePath: dialogueXML(localization.DialogueRow{ID: "a", Source: "mod", Text: "override"}),
	})
	pakPath := filepath.Join(modsRoot, "unlisted", "Localization", "English_xml.pak")
	if err := os.Remove(pakPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(pakPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modsRoot, modinstall.ModOrderFilename), []byte("some_other_mod\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	stock := []localization.DialogueRow{{ID: "a", Source: "stock", Text: "stock"}}

	result, err := resolveFromModsRoot(stock, modsRoot, "English_xml.pak")
	if err != nil {
		t.Fatalf("resolveFromModsRoot() error = %v, want unlisted mod ignored before PAK validation", err)
	}
	if !reflect.DeepEqual(result.Rows, stock) || len(result.Contributions) != 0 {
		t.Fatalf("result = %+v, want unlisted mod ignored", result)
	}
}
