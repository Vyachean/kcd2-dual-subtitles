package gamedetect

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDetectInInstallRootsFindsValidContentRootsDeterministically(t *testing.T) {
	base := t.TempDir()
	rootA := filepath.Join(base, "LibraryA")
	rootB := filepath.Join(base, "LibraryB")
	gameZ := createGameLayout(t, rootA, "Zeta")
	gameA := createGameLayout(t, rootB, "Alpha")
	if err := os.MkdirAll(filepath.Join(rootA, "NotKCD2", "Content", "Localization"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := detectInInstallRoots([]string{rootA, rootB, rootA})
	want := []string{gameZ, gameA}
	if !reflect.DeepEqual(got.Candidates, want) {
		t.Fatalf("Candidates = %#v, want %#v", got.Candidates, want)
	}
	if _, ok := got.Unique(); ok {
		t.Fatal("Unique() = true for multiple candidates")
	}
}

func TestDetectInInstallRootsUniqueAndMissing(t *testing.T) {
	root := filepath.Join(t.TempDir(), "Games")
	game := createGameLayout(t, root, "Kingdom Come")

	got := detectInInstallRoots([]string{root})
	if unique, ok := got.Unique(); !ok || unique != game {
		t.Fatalf("Unique() = %q, %v; want %q, true", unique, ok, game)
	}

	missing := detectInInstallRoots([]string{filepath.Join(t.TempDir(), "missing")})
	if len(missing.Candidates) != 0 {
		t.Fatalf("missing candidates = %#v, want none", missing.Candidates)
	}
	if _, ok := missing.Unique(); ok {
		t.Fatal("Unique() = true for no candidates")
	}
}

func TestNormalizeSelectionAcceptsContentOrParent(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "Kingdom Come")
	content := createGameLayoutAtContent(t, filepath.Join(parent, "Content"), "English_xml.pak", "Italian_xml.pak")

	for _, input := range []string{content, parent, `"` + parent + `"`} {
		got, err := NormalizeSelection(input)
		if err != nil {
			t.Fatalf("NormalizeSelection(%q) error = %v", input, err)
		}
		if got != content {
			t.Fatalf("NormalizeSelection(%q) = %q, want %q", input, got, content)
		}
	}
}

func TestIsGameRootDoesNotRequireEnglishOrRussian(t *testing.T) {
	content := createGameLayoutAtContent(t, filepath.Join(t.TempDir(), "Content"), "German_xml.pak", "Czech_xml.pak")
	if !IsGameRoot(content) {
		t.Fatal("German/Czech-only compatible layout was rejected")
	}
}

func TestIsGameRootRequiresTwoKnownInstalledLanguages(t *testing.T) {
	content := filepath.Join(t.TempDir(), "Content")
	writeCoreFiles(t, content)
	writeFixtureFile(t, filepath.Join(content, "Localization", "German_xml.pak"))
	writeFixtureFile(t, filepath.Join(content, "Localization", "FutureLanguage_xml.pak"))

	if IsGameRoot(content) {
		t.Fatal("layout with only one known supported language was accepted")
	}
}

func TestNormalizeSelectionRejectsInvalidLayout(t *testing.T) {
	_, err := NormalizeSelection(t.TempDir())
	if !errors.Is(err, ErrInvalidGameRoot) {
		t.Fatalf("error = %v, want ErrInvalidGameRoot", err)
	}
}

func createGameLayout(t *testing.T, installRoot, name string) string {
	t.Helper()
	return createGameLayoutAtContent(t, filepath.Join(installRoot, name, "Content"), "English_xml.pak", "Italian_xml.pak")
}

func createGameLayoutAtContent(t *testing.T, content string, languagePAKs ...string) string {
	t.Helper()
	writeCoreFiles(t, content)
	for _, filename := range languagePAKs {
		writeFixtureFile(t, filepath.Join(content, "Localization", filename))
	}
	absolute, err := filepath.Abs(content)
	if err != nil {
		t.Fatalf("Abs(%q): %v", content, err)
	}
	return filepath.Clean(absolute)
}

func writeCoreFiles(t *testing.T, content string) {
	t.Helper()
	for _, relative := range requiredCoreFiles {
		writeFixtureFile(t, filepath.Join(content, relative))
	}
}

func writeFixtureFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", path, err)
	}
	if err := os.WriteFile(path, []byte("fixture"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}
