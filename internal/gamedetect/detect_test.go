package gamedetect

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDetectInXboxRootsFindsValidContentRootsDeterministically(t *testing.T) {
	base := t.TempDir()
	rootA := filepath.Join(base, "XboxA")
	rootB := filepath.Join(base, "XboxB")
	gameZ := createGameLayout(t, rootA, "Zeta")
	gameA := createGameLayout(t, rootB, "Alpha")
	if err := os.MkdirAll(filepath.Join(rootA, "NotKCD2", "Content", "Localization"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := detectInXboxRoots([]string{rootA, rootB, rootA})
	want := []string{gameA, gameZ}
	if !reflect.DeepEqual(got.Candidates, want) {
		t.Fatalf("Candidates = %#v, want %#v", got.Candidates, want)
	}
	if _, ok := got.Unique(); ok {
		t.Fatal("Unique() = true for multiple candidates")
	}
}

func TestDetectInXboxRootsUniqueAndMissing(t *testing.T) {
	root := filepath.Join(t.TempDir(), "XboxGames")
	game := createGameLayout(t, root, "Kingdom Come")

	got := detectInXboxRoots([]string{root})
	if unique, ok := got.Unique(); !ok || unique != game {
		t.Fatalf("Unique() = %q, %v; want %q, true", unique, ok, game)
	}

	missing := detectInXboxRoots([]string{filepath.Join(t.TempDir(), "missing")})
	if len(missing.Candidates) != 0 {
		t.Fatalf("missing candidates = %#v, want none", missing.Candidates)
	}
	if _, ok := missing.Unique(); ok {
		t.Fatal("Unique() = true for no candidates")
	}
}

func TestNormalizeSelectionAcceptsContentOrParent(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "Kingdom Come")
	content := createGameLayoutAtContent(t, filepath.Join(parent, "Content"))

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

func TestNormalizeSelectionRejectsInvalidLayout(t *testing.T) {
	_, err := NormalizeSelection(t.TempDir())
	if !errors.Is(err, ErrInvalidGameRoot) {
		t.Fatalf("error = %v, want ErrInvalidGameRoot", err)
	}
}

func createGameLayout(t *testing.T, xboxRoot, name string) string {
	t.Helper()
	return createGameLayoutAtContent(t, filepath.Join(xboxRoot, name, "Content"))
}

func createGameLayoutAtContent(t *testing.T, content string) string {
	t.Helper()
	for _, relative := range requiredRelativeFiles {
		path := filepath.Join(content, relative)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("MkdirAll(%q): %v", path, err)
		}
		if err := os.WriteFile(path, []byte("fixture"), 0o644); err != nil {
			t.Fatalf("WriteFile(%q): %v", path, err)
		}
	}
	absolute, err := filepath.Abs(content)
	if err != nil {
		t.Fatalf("Abs(%q): %v", content, err)
	}
	return filepath.Clean(absolute)
}
