package localizationsource

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
)

func TestResolveSkipsSupportsEvaluationUntilSelectedLanguagePAKExists(t *testing.T) {
	modsRoot := t.TempDir()
	writeLocalizationMod(t, modsRoot, "russian_only", "russian_only", "Russian only", "Russian_xml.pak", map[string]string{
		localization.DialogueXMLArchivePath: dialogueXML(localization.DialogueRow{ID: "a", Source: "mod", Text: "override"}),
	})
	manifestPath := filepath.Join(modsRoot, "russian_only", "mod.manifest")
	manifest := `<?xml version="1.0"?><kcd_mod><info><name>Russian only</name><modid>russian_only</modid></info><supports><version>1.5*</version></supports></kcd_mod>`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}

	stock := []localization.DialogueRow{{ID: "a", Source: "stock", Text: "stock"}}
	result, err := resolveFromModsRootWithVersion(stock, modsRoot, "English_xml.pak", "", errors.New("system.cfg unavailable"))
	if err != nil {
		t.Fatalf("resolveFromModsRootWithVersion() error = %v, want unrelated language mod ignored", err)
	}
	if !reflect.DeepEqual(result.Rows, stock) || len(result.Contributions) != 0 || len(result.DialogueWriters) != 0 {
		t.Fatalf("result = %+v, want unchanged stock with no relevant mod", result)
	}
}
