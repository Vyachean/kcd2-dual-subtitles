package application

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/gamedetect"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/generator"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/modinstall"
)

func TestGenerateAndInstallNormalizesParentAndPreservesExplicitLanguages(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "Kingdom Come")
	content := filepath.Join(parent, "Content")
	createApplicationGameLayout(t, content)

	var got generator.Request
	service := Service{
		version: "v0.2.0-test",
		state:   &serviceState{},
		generate: func(request generator.Request) (generator.Result, error) {
			got = request
			return generator.Result{InstallPath: "installed"}, nil
		},
	}

	result, err := service.GenerateAndInstall(parent, localization.English, localization.Russian)
	if err != nil {
		t.Fatalf("GenerateAndInstall() error = %v", err)
	}
	if result.InstallPath != "installed" {
		t.Fatalf("InstallPath = %q", result.InstallPath)
	}
	if got.GameRoot != content || got.MainLanguage != localization.English || got.SecondaryLanguage != localization.Russian || got.Version != "v0.2.0-test" {
		t.Fatalf("request = %+v", got)
	}
	if got.OutputPath != "" || got.CanaryID != "" {
		t.Fatalf("GUI generation unexpectedly enabled archive/canary options: %+v", got)
	}
	if service.currentGameRoot() != content {
		t.Fatalf("selected root = %q, want %q", service.currentGameRoot(), content)
	}
}

func TestGenerateAndInstallRejectsSameLanguageWithoutCallingGenerator(t *testing.T) {
	called := false
	service := Service{generate: func(generator.Request) (generator.Result, error) {
		called = true
		return generator.Result{}, nil
	}}
	_, err := service.GenerateAndInstall("unused", localization.Russian, localization.Russian)
	if !errors.Is(err, ErrSameLanguage) {
		t.Fatalf("error = %v, want ErrSameLanguage", err)
	}
	if called {
		t.Fatal("generator called for equal languages")
	}
}

func TestServiceDelegatesDetectionInspectionAndUninstallForSameRoot(t *testing.T) {
	var inspectedRoot string
	var uninstalledRoot string
	service := Service{
		state: &serviceState{},
		detect: func() (gamedetect.Result, error) {
			return gamedetect.Result{Candidates: []string{"game"}}, nil
		},
		inspect: func(root string) (modinstall.Status, error) {
			inspectedRoot = root
			return modinstall.Status{Installed: true, Path: "mod"}, nil
		},
		uninstall: func(root string) (modinstall.UninstallResult, error) {
			uninstalledRoot = root
			return modinstall.UninstallResult{Path: "mod", RemovedMod: true}, nil
		},
	}

	detected, err := service.DetectGame()
	if err != nil || len(detected.Candidates) != 1 || detected.Candidates[0] != "game" {
		t.Fatalf("DetectGame() = %+v, err=%v", detected, err)
	}
	status, err := service.InspectInstallation()
	if err != nil || !status.Installed || status.Path != "mod" {
		t.Fatalf("InspectInstallation() = %+v, err=%v", status, err)
	}
	uninstalled, err := service.Uninstall()
	if err != nil || !uninstalled.RemovedMod || uninstalled.Path != "mod" {
		t.Fatalf("Uninstall() = %+v, err=%v", uninstalled, err)
	}
	if inspectedRoot != "game" || uninstalledRoot != "game" {
		t.Fatalf("roots: inspect=%q uninstall=%q", inspectedRoot, uninstalledRoot)
	}
}

func TestServiceValueCopiesShareSelectedGameRoot(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "Kingdom Come")
	content := filepath.Join(parent, "Content")
	createApplicationGameLayout(t, content)

	service := Service{state: &serviceState{}}
	copyOfService := service
	if _, err := service.ValidateGameRoot(parent); err != nil {
		t.Fatal(err)
	}
	if got := copyOfService.currentGameRoot(); got != content {
		t.Fatalf("copied service root = %q, want %q", got, content)
	}
}

func TestServiceRequiresSelectedRootForStatusAndUninstall(t *testing.T) {
	service := Service{
		state: &serviceState{},
		inspect: func(string) (modinstall.Status, error) {
			t.Fatal("inspect called without a selected root")
			return modinstall.Status{}, nil
		},
		uninstall: func(string) (modinstall.UninstallResult, error) {
			t.Fatal("uninstall called without a selected root")
			return modinstall.UninstallResult{}, nil
		},
	}
	if _, err := service.InspectInstallation(); !errors.Is(err, ErrGameRootNotSelected) {
		t.Fatalf("InspectInstallation() error = %v", err)
	}
	if _, err := service.Uninstall(); !errors.Is(err, ErrGameRootNotSelected) {
		t.Fatalf("Uninstall() error = %v", err)
	}
}

func createApplicationGameLayout(t *testing.T, content string) {
	t.Helper()
	for _, relative := range []string{
		filepath.Join("Localization", "English_xml.pak"),
		filepath.Join("Localization", "Russian_xml.pak"),
		filepath.Join("Data", "Scripts.pak"),
		filepath.Join("Data", "Tables.pak"),
	} {
		path := filepath.Join(content, relative)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("fixture"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
