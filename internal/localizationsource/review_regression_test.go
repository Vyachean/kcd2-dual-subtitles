package localizationsource

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/modinstall"
)

func TestIdenticalDialogueWriterIsTrackedForPrecedenceButNotReportedAsChange(t *testing.T) {
	modsRoot := t.TempDir()
	writeLocalizationMod(t, modsRoot, "z_same", "same", "Same text writer", "English_xml.pak", map[string]string{
		"text_ui__same.xml": dialogueXML(localization.DialogueRow{ID: "a", Source: "mod", Text: "same"}),
	})

	result, err := resolveFromModsRoot([]localization.DialogueRow{{ID: "a", Source: "stock", Text: "same"}}, modsRoot, "English_xml.pak")
	if err != nil {
		t.Fatalf("resolveFromModsRoot() error = %v", err)
	}
	if len(result.Contributions) != 0 {
		t.Fatalf("Contributions = %+v, want no user-facing effective change", result.Contributions)
	}
	if len(result.DialogueWriters) != 1 || result.DialogueWriters[0].ModID != "same" {
		t.Fatalf("DialogueWriters = %+v, want same-text writer tracked for precedence", result.DialogueWriters)
	}
}

func TestSupportsIsNotEvaluatedWithoutSelectedLanguagePAK(t *testing.T) {
	modsRoot := t.TempDir()
	writeLocalizationMod(t, modsRoot, "versioned", "versioned", "Versioned", "Russian_xml.pak", map[string]string{
		"text_ui__versioned.xml": dialogueXML(localization.DialogueRow{ID: "a", Source: "mod", Text: "Russian only"}),
	})
	manifestPath := filepath.Join(modsRoot, "versioned", "mod.manifest")
	manifest := `<?xml version="1.0"?><kcd_mod><info><name>Versioned</name><modid>versioned</modid></info><supports><version>1.5*</version></supports></kcd_mod>`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modsRoot, modinstall.ModOrderFilename), []byte("versioned\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := resolveFromModsRootWithVersion(
		[]localization.DialogueRow{{ID: "a", Source: "stock", Text: "English"}},
		modsRoot,
		"English_xml.pak",
		"",
		errors.New("system.cfg unavailable"),
	)
	if err != nil {
		t.Fatalf("resolveFromModsRootWithVersion() error = %v, want unrelated language mod ignored", err)
	}
	if len(result.Contributions) != 0 || len(result.DialogueWriters) != 0 {
		t.Fatalf("result = %+v, want stock-only English source", result)
	}
}

func TestMissingExplicitModIDIsDeferredUntilSelectedLanguagePAKExists(t *testing.T) {
	modsRoot := t.TempDir()
	writeLocalizationMod(t, modsRoot, "generated", "temporary", "Generated Name", "Russian_xml.pak", map[string]string{
		"text_ui__generated_name.xml": dialogueXML(localization.DialogueRow{ID: "a", Source: "mod", Text: "Russian only"}),
	})
	manifestPath := filepath.Join(modsRoot, "generated", "mod.manifest")
	manifest := `<?xml version="1.0"?><kcd_mod><info><name>Generated Name</name></info><supports><version>1.5*</version></supports></kcd_mod>`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modsRoot, modinstall.ModOrderFilename), []byte("generated_name\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := resolveFromModsRootWithVersion(
		[]localization.DialogueRow{{ID: "a", Source: "stock", Text: "English"}},
		modsRoot,
		"English_xml.pak",
		"",
		errors.New("system.cfg unavailable"),
	); err != nil {
		t.Fatalf("irrelevant missing-modid mod error = %v, want selected-language relevance checked first", err)
	}
}
