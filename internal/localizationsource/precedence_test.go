package localizationsource

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/modinstall"
)

func TestOverlayLocalizationPAKAppliesDialogueTableBeforePatchResources(t *testing.T) {
	modsRoot := t.TempDir()
	writeLocalizationMod(t, modsRoot, "combined", "combined", "Combined", "English_xml.pak", map[string]string{
		localization.DialogueXMLArchivePath: dialogueXML(
			localization.DialogueRow{ID: "a", Source: "full", Text: "full table"},
		),
		"text_ui__combined.xml": dialogueXML(
			localization.DialogueRow{ID: "a", Source: "patch", Text: "patch wins"},
		),
	})

	result, err := resolveFromModsRoot(
		[]localization.DialogueRow{{ID: "a", Source: "stock", Text: "stock"}},
		modsRoot,
		"English_xml.pak",
	)
	if err != nil {
		t.Fatalf("resolveFromModsRoot() error = %v", err)
	}
	if got := result.Rows[0].Text; got != "patch wins" {
		t.Fatalf("effective text = %q, want patch wins", got)
	}
}

func TestResolveFromModsRootAllowsLocalizationModWithoutExplicitModIDWhenOrderFileIsAbsent(t *testing.T) {
	modsRoot := t.TempDir()
	writeLocalizationMod(t, modsRoot, "generated-id", "temporary", "Generated ID", "English_xml.pak", map[string]string{
		localization.DialogueXMLArchivePath: dialogueXML(localization.DialogueRow{ID: "a", Source: "mod", Text: "override"}),
	})
	manifestPath := filepath.Join(modsRoot, "generated-id", "mod.manifest")
	manifest := `<?xml version="1.0"?><kcd_mod><info><name>Generated ID</name></info></kcd_mod>`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := resolveFromModsRoot(
		[]localization.DialogueRow{{ID: "a", Source: "stock", Text: "stock"}},
		modsRoot,
		"English_xml.pak",
	)
	if err != nil {
		t.Fatalf("resolveFromModsRoot() error = %v", err)
	}
	if got := result.Rows[0].Text; got != "override" {
		t.Fatalf("effective text = %q, want override", got)
	}
	if len(result.Contributions) != 1 || result.Contributions[0].Name != "Generated ID" || result.Contributions[0].ModID != "" {
		t.Fatalf("Contributions = %+v", result.Contributions)
	}
}

func TestResolveFromModsRootFailsClosedForLocalizationModWithoutExplicitModIDWhenOrderFileExists(t *testing.T) {
	modsRoot := t.TempDir()
	writeLocalizationMod(t, modsRoot, "generated-id", "temporary", "Generated ID", "English_xml.pak", map[string]string{
		localization.DialogueXMLArchivePath: dialogueXML(localization.DialogueRow{ID: "a", Source: "mod", Text: "override"}),
	})
	manifestPath := filepath.Join(modsRoot, "generated-id", "mod.manifest")
	manifest := `<?xml version="1.0"?><kcd_mod><info><name>Generated ID</name></info></kcd_mod>`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modsRoot, modinstall.ModOrderFilename), []byte("generated_id\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := resolveFromModsRoot(
		[]localization.DialogueRow{{ID: "a", Source: "stock", Text: "stock"}},
		modsRoot,
		"English_xml.pak",
	)
	if err == nil {
		t.Fatal("resolveFromModsRoot() error = nil, want fail-closed missing-modid error")
	}
	for _, want := range []string{"generated-id", "omits <modid>", "mod_order.txt"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, want substring %q", err, want)
		}
	}
}
