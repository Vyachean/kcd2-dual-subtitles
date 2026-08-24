//go:build windows

package gamedetect

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseSteamLibraryFoldersSupportsModernAndLegacyEntries(t *testing.T) {
	data := []byte(`
"libraryfolders"
{
    "0"
    {
        "path" "C:\\Program Files (x86)\\Steam"
    }
    "1"
    {
        "path" "D:\\SteamLibrary"
    }
    "2" "E:\\LegacySteamLibrary"
}
`)
	got := parseSteamLibraryFolders(data)
	want := uniquePaths([]string{
		`C:\Program Files (x86)\Steam`,
		`D:\SteamLibrary`,
		`E:\LegacySteamLibrary`,
	})
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseSteamLibraryFolders() = %#v, want %#v", got, want)
	}
}

func TestSteamLibraryDiscoveryFindsStructurallyValidGame(t *testing.T) {
	base := t.TempDir()
	steamRoot := filepath.Join(base, "Steam")
	customLibrary := filepath.Join(base, "CustomLibrary")
	gameRoot := createGameLayoutAtContent(t, filepath.Join(customLibrary, "steamapps", "common", "KCD2"), "English_xml.pak", "German_xml.pak")

	if err := os.MkdirAll(filepath.Join(steamRoot, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	vdfPath := strings.ReplaceAll(customLibrary, `\`, `\\`)
	vdf := `"libraryfolders" { "1" { "path" "` + vdfPath + `" } }`
	if err := os.WriteFile(filepath.Join(steamRoot, "config", "libraryfolders.vdf"), []byte(vdf), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(steamRoot, "steamapps", "common", "Unrelated"), 0o755); err != nil {
		t.Fatal(err)
	}

	libraries := steamLibraryRoots([]string{steamRoot})
	result := detectCandidatePaths(steamCommonCandidates(libraries))
	if unique, ok := result.Unique(); !ok || unique != gameRoot {
		t.Fatalf("Unique() = %q, %v; want %q, true (libraries=%#v candidates=%#v)", unique, ok, gameRoot, libraries, result.Candidates)
	}
}

func TestEpicManifestDiscoveryIgnoresMalformedAndUnrelatedEntries(t *testing.T) {
	manifestDir := filepath.Join(t.TempDir(), "Manifests")
	if err := os.MkdirAll(manifestDir, 0o755); err != nil {
		t.Fatal(err)
	}
	gameRoot := createGameLayoutAtContent(t, filepath.Join(t.TempDir(), "EpicKCD2"), "English_xml.pak", "French_xml.pak")
	unrelated := filepath.Join(t.TempDir(), "OtherGame")
	if err := os.MkdirAll(unrelated, 0o755); err != nil {
		t.Fatal(err)
	}

	writeEpicManifest(t, filepath.Join(manifestDir, "kcd.item"), gameRoot)
	writeEpicManifest(t, filepath.Join(manifestDir, "other.item"), unrelated)
	if err := os.WriteFile(filepath.Join(manifestDir, "broken.item"), []byte(`{"InstallLocation":`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(manifestDir, "ignore.txt"), []byte(`{"InstallLocation":"ignored"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	locations := epicInstallLocationsFromManifestDir(manifestDir)
	result := detectCandidatePaths(locations)
	if unique, ok := result.Unique(); !ok || unique != gameRoot {
		t.Fatalf("Unique() = %q, %v; want %q, true (locations=%#v)", unique, ok, gameRoot, locations)
	}
}

func TestGOGLibraryConfigDiscoveryUsesLibraryChildrenAndStructuralValidation(t *testing.T) {
	base := t.TempDir()
	library := filepath.Join(base, "GOG Games")
	gameRoot := createGameLayoutAtContent(t, filepath.Join(library, "Kingdom Come Deliverance II"), "Czech_xml.pak", "German_xml.pak")
	if err := os.MkdirAll(filepath.Join(library, "Other Game"), 0o755); err != nil {
		t.Fatal(err)
	}

	configPath := filepath.Join(base, "config.json")
	config, err := json.Marshal(map[string]string{"libraryPath": library})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, config, 0o644); err != nil {
		t.Fatal(err)
	}
	root, ok := gogLibraryRootFromConfig(configPath)
	if !ok || root != filepath.Clean(library) {
		t.Fatalf("gogLibraryRootFromConfig() = %q, %v; want %q, true", root, ok, filepath.Clean(library))
	}
	result := detectCandidatePaths(childDirectories([]string{root}))
	if unique, ok := result.Unique(); !ok || unique != gameRoot {
		t.Fatalf("Unique() = %q, %v; want %q, true", unique, ok, gameRoot)
	}
}

func TestMergeDetectionResultsDeduplicatesSameInstallationAcrossLaunchers(t *testing.T) {
	gameRoot := createGameLayoutAtContent(t, filepath.Join(t.TempDir(), "KCD2"), "English_xml.pak", "Italian_xml.pak")
	result := mergeDetectionResults(
		Result{Candidates: []string{gameRoot}},
		Result{Candidates: []string{strings.ToUpper(gameRoot)}},
		Result{Candidates: []string{gameRoot}},
	)
	if len(result.Candidates) != 1 || !strings.EqualFold(result.Candidates[0], gameRoot) {
		t.Fatalf("Candidates = %#v, want one %q candidate", result.Candidates, gameRoot)
	}
}

func TestMalformedLauncherMetadataFailsSoft(t *testing.T) {
	if got := parseSteamLibraryFolders([]byte(`"libraryfolders" { "1" { "path" "unterminated }`)); len(got) != 0 {
		t.Fatalf("malformed Steam VDF roots = %#v, want none", got)
	}
	configPath := filepath.Join(t.TempDir(), "broken-config.json")
	if err := os.WriteFile(configPath, []byte(`{"libraryPath":`), 0o644); err != nil {
		t.Fatal(err)
	}
	if root, ok := gogLibraryRootFromConfig(configPath); ok || root != "" {
		t.Fatalf("malformed GOG config = %q, %v; want empty, false", root, ok)
	}
	if got := epicInstallLocationsFromManifestDir(filepath.Join(t.TempDir(), "missing")); len(got) != 0 {
		t.Fatalf("missing Epic manifest locations = %#v, want none", got)
	}
}

func writeEpicManifest(t *testing.T, path, installLocation string) {
	t.Helper()
	data, err := json.Marshal(map[string]string{"InstallLocation": installLocation})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
