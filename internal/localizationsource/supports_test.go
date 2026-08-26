package localizationsource

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
)

func TestSupportsGameVersion(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		version  string
		want     bool
	}{
		{name: "exact", patterns: []string{"1.5.6"}, version: "1.5.6", want: true},
		{name: "prefix wildcard", patterns: []string{"1.5*"}, version: "1.5.6", want: true},
		{name: "documented major wildcard", patterns: []string{"1.5*"}, version: "1.5", want: true},
		{name: "dot wildcard does not match bare major", patterns: []string{"1.5.*"}, version: "1.5", want: false},
		{name: "one of several", patterns: []string{"1.4*", "1.5*"}, version: "1.5.6", want: true},
		{name: "unsupported", patterns: []string{"1.4*"}, version: "1.5.6", want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := supportsGameVersion(test.patterns, test.version); got != test.want {
				t.Fatalf("supportsGameVersion(%q, %q) = %v, want %v", test.patterns, test.version, got, test.want)
			}
		})
	}
}

func TestResolveFromModsRootWithVersionSkipsUnsupportedMod(t *testing.T) {
	modsRoot := t.TempDir()
	writeLocalizationMod(t, modsRoot, "versioned", "versioned", "Versioned", "English_xml.pak", map[string]string{
		localization.DialogueXMLArchivePath: dialogueXML(localization.DialogueRow{ID: "a", Source: "mod", Text: "override"}),
	})
	manifestPath := filepath.Join(modsRoot, "versioned", "mod.manifest")
	manifest := `<?xml version="1.0"?><kcd_mod><info><name>Versioned</name><modid>versioned</modid></info><supports><version>1.4*</version></supports></kcd_mod>`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	stock := []localization.DialogueRow{{ID: "a", Source: "stock", Text: "stock"}}

	result, err := resolveFromModsRootWithVersion(stock, modsRoot, "English_xml.pak", "1.5.6", nil)
	if err != nil {
		t.Fatalf("resolveFromModsRootWithVersion() error = %v", err)
	}
	if !reflect.DeepEqual(result.Rows, stock) || len(result.Contributions) != 0 {
		t.Fatalf("result = %+v, want unsupported mod ignored", result)
	}
}

func TestResolveFromModsRootWithVersionUsesSupportedMod(t *testing.T) {
	modsRoot := t.TempDir()
	writeLocalizationMod(t, modsRoot, "versioned", "versioned", "Versioned", "English_xml.pak", map[string]string{
		localization.DialogueXMLArchivePath: dialogueXML(localization.DialogueRow{ID: "a", Source: "mod", Text: "override"}),
	})
	manifestPath := filepath.Join(modsRoot, "versioned", "mod.manifest")
	manifest := `<?xml version="1.0"?><kcd_mod><info><name>Versioned</name><modid>versioned</modid></info><supports><version>1.5*</version></supports></kcd_mod>`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := resolveFromModsRootWithVersion([]localization.DialogueRow{{ID: "a", Source: "stock", Text: "stock"}}, modsRoot, "English_xml.pak", "1.5.6", nil)
	if err != nil {
		t.Fatalf("resolveFromModsRootWithVersion() error = %v", err)
	}
	if got := result.Rows[0].Text; got != "override" {
		t.Fatalf("effective text = %q, want override", got)
	}
}

func TestResolveFromModsRootWithVersionFailsClosedWhenSupportsCannotBeEvaluated(t *testing.T) {
	modsRoot := t.TempDir()
	writeLocalizationMod(t, modsRoot, "versioned", "versioned", "Versioned", "English_xml.pak", map[string]string{
		localization.DialogueXMLArchivePath: dialogueXML(localization.DialogueRow{ID: "a", Source: "mod", Text: "override"}),
	})
	manifestPath := filepath.Join(modsRoot, "versioned", "mod.manifest")
	manifest := `<?xml version="1.0"?><kcd_mod><info><name>Versioned</name><modid>versioned</modid></info><supports><version>1.5*</version></supports></kcd_mod>`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := resolveFromModsRootWithVersion(
		[]localization.DialogueRow{{ID: "a", Source: "stock", Text: "stock"}},
		modsRoot,
		"English_xml.pak",
		"",
		errors.New("system.cfg unavailable"),
	)
	if err == nil || !strings.Contains(err.Error(), "manifest <supports>") {
		t.Fatalf("error = %v, want fail-closed supports evaluation error", err)
	}
}

func TestReadGameVersion(t *testing.T) {
	gameRoot := t.TempDir()
	config := "sys_streaming_memory_budget = 20480\r\nwh_sys_version = \"1.5.6\"\r\nsys_flash_address_space = 65536\r\n"
	if err := os.WriteFile(filepath.Join(gameRoot, "system.cfg"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	version, err := readGameVersion(gameRoot)
	if err != nil {
		t.Fatalf("readGameVersion() error = %v", err)
	}
	if version != "1.5.6" {
		t.Fatalf("readGameVersion() = %q, want 1.5.6", version)
	}
}
