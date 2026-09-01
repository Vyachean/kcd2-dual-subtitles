package localizationsource

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
)

func TestResolveFromModsRootAcceptsLocalizationPAKSymlink(t *testing.T) {
	modsRoot := t.TempDir()
	targetRoot := t.TempDir()
	writeLocalizationMod(t, targetRoot, "staged", "staged", "Staged", "English_xml.pak", map[string]string{
		localization.DialogueXMLArchivePath: dialogueXML(localization.DialogueRow{ID: "a", Source: "mod", Text: "fixed"}),
	})

	modDir := writeSymlinkLocalizationModManifest(t, modsRoot, "deployed", "deployed")
	target := filepath.Join(targetRoot, "staged", "Localization", "English_xml.pak")
	link := filepath.Join(modDir, "Localization", "English_xml.pak")
	createTestSymlinkOrSkip(t, target, link)

	result, err := resolveFromModsRoot(
		[]localization.DialogueRow{{ID: "a", Source: "stock", Text: "stock"}},
		modsRoot,
		"English_xml.pak",
	)
	if err != nil {
		t.Fatalf("resolveFromModsRoot() error = %v", err)
	}
	if got := result.Rows[0].Text; got != "fixed" {
		t.Fatalf("effective text = %q, want fixed from symlinked PAK", got)
	}
	if len(result.Contributions) != 1 || result.Contributions[0].ModID != "deployed" {
		t.Fatalf("Contributions = %+v, want deployed mod", result.Contributions)
	}
}

func TestResolveFromModsRootRejectsBrokenLocalizationPAKSymlink(t *testing.T) {
	modsRoot := t.TempDir()
	modDir := writeSymlinkLocalizationModManifest(t, modsRoot, "broken", "broken")
	link := filepath.Join(modDir, "Localization", "English_xml.pak")
	createTestSymlinkOrSkip(t, filepath.Join(t.TempDir(), "missing.pak"), link)

	_, err := resolveFromModsRoot(
		[]localization.DialogueRow{{ID: "a", Source: "stock", Text: "stock"}},
		modsRoot,
		"English_xml.pak",
	)
	if err == nil || !strings.Contains(err.Error(), "resolve active localization PAK") || !strings.Contains(err.Error(), "English_xml.pak") {
		t.Fatalf("error = %v, want broken-link localization PAK error", err)
	}
}

func TestResolveFromModsRootRejectsLocalizationPAKSymlinkToDirectory(t *testing.T) {
	modsRoot := t.TempDir()
	modDir := writeSymlinkLocalizationModManifest(t, modsRoot, "directory", "directory")
	target := t.TempDir()
	link := filepath.Join(modDir, "Localization", "English_xml.pak")
	createTestSymlinkOrSkip(t, target, link)

	_, err := resolveFromModsRoot(
		[]localization.DialogueRow{{ID: "a", Source: "stock", Text: "stock"}},
		modsRoot,
		"English_xml.pak",
	)
	if err == nil || !strings.Contains(err.Error(), "does not resolve to a regular file") {
		t.Fatalf("error = %v, want non-regular resolved PAK error", err)
	}
}

func writeSymlinkLocalizationModManifest(t *testing.T, modsRoot, folder, modID string) string {
	t.Helper()
	dir := filepath.Join(modsRoot, folder)
	if err := os.MkdirAll(filepath.Join(dir, "Localization"), 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `<?xml version="1.0"?><kcd_mod><info><name>Symlink deployment</name><modid>` + modID + `</modid></info></kcd_mod>`
	if err := os.WriteFile(filepath.Join(dir, "mod.manifest"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

func createTestSymlinkOrSkip(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink creation is unavailable on this runner: %v", err)
	}
}
