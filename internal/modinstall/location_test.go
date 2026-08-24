package modinstall

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveInstallLocationUsesGameRootModsForStandardLayout(t *testing.T) {
	gameRoot := filepath.Join(t.TempDir(), "KingdomComeDeliverance2")
	if err := os.MkdirAll(gameRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	calledDocuments := false
	got, err := resolveInstallLocation(gameRoot, func() (string, error) {
		calledDocuments = true
		return "", errors.New("must not be called")
	})
	if err != nil {
		t.Fatalf("resolveInstallLocation() error = %v", err)
	}
	if calledDocuments {
		t.Fatal("standard layout unexpectedly resolved Documents")
	}
	want := filepath.Join(gameRoot, "Mods")
	if got.Layout != InstallLayoutGameRoot || got.ModsRoot != want {
		t.Fatalf("location = %+v, want layout=%q root=%q", got, InstallLayoutGameRoot, want)
	}
}

func TestResolveInstallLocationUsesDocumentsForGDKMarkers(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "Kingdom Come- Deliverance II")
	gameRoot := filepath.Join(parent, "Content")
	if err := os.MkdirAll(gameRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gameRoot, "gamelaunchhelper.exe"), []byte("fixture"), 0o644); err != nil {
		t.Fatal(err)
	}
	documents := filepath.Join(t.TempDir(), "Documents")

	got, err := resolveInstallLocation(gameRoot, func() (string, error) { return documents, nil })
	if err != nil {
		t.Fatalf("resolveInstallLocation() error = %v", err)
	}
	want := filepath.Join(documents, ModsDirectoryName)
	if got.Layout != InstallLayoutGDKDocuments || got.ModsRoot != want {
		t.Fatalf("location = %+v, want layout=%q root=%q", got, InstallLayoutGDKDocuments, want)
	}
}

func TestResolveInstallLocationRecognizesGDKMetadataAtParent(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "KCD2")
	gameRoot := filepath.Join(parent, "Content")
	if err := os.MkdirAll(gameRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parent, "MicrosoftGame.config"), []byte("fixture"), 0o644); err != nil {
		t.Fatal(err)
	}
	documents := filepath.Join(t.TempDir(), "Documents")

	got, err := resolveInstallLocation(gameRoot, func() (string, error) { return documents, nil })
	if err != nil {
		t.Fatalf("resolveInstallLocation() error = %v", err)
	}
	if got.Layout != InstallLayoutGDKDocuments {
		t.Fatalf("layout = %q, want %q", got.Layout, InstallLayoutGDKDocuments)
	}
}

func TestResolveInstallLocationDoesNotInferGDKFromContentFolderName(t *testing.T) {
	gameRoot := filepath.Join(t.TempDir(), "Content")
	if err := os.MkdirAll(gameRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := resolveInstallLocation(gameRoot, func() (string, error) {
		return "", errors.New("must not be called")
	})
	if err != nil {
		t.Fatalf("resolveInstallLocation() error = %v", err)
	}
	if got.Layout != InstallLayoutGameRoot {
		t.Fatalf("layout = %q, want %q", got.Layout, InstallLayoutGameRoot)
	}
}
