package localizationsource

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/modinstall"
)

func TestResolveFromModsRootDoesNotTreatFolderNameAsModOrderID(t *testing.T) {
	modsRoot := t.TempDir()
	writeLocalizationMod(t, modsRoot, "folder_name", "real_id", "Real", "English_xml.pak", map[string]string{
		localization.DialogueXMLArchivePath: dialogueXML(localization.DialogueRow{ID: "a", Source: "mod", Text: "override"}),
	})
	if err := os.WriteFile(filepath.Join(modsRoot, modinstall.ModOrderFilename), []byte("folder_name\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	stock := []localization.DialogueRow{{ID: "a", Source: "stock", Text: "stock"}}

	result, err := resolveFromModsRoot(stock, modsRoot, "English_xml.pak")
	if err != nil {
		t.Fatalf("resolveFromModsRoot() error = %v", err)
	}
	if !reflect.DeepEqual(result.Rows, stock) || len(result.Contributions) != 0 {
		t.Fatalf("result = %+v, want mod inactive because mod_order lists folder rather than modid", result)
	}
}
