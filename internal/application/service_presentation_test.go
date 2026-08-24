package application

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/generator"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
)

func TestGenerateAndInstallWithPresentationSelectsHUDAndNormalizesConfig(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "Kingdom Come")
	content := filepath.Join(parent, "Content")
	createApplicationGameLayout(t, content)

	var got generator.Request
	service := Service{
		version: "v0.3.0-test",
		generate: func(request generator.Request) (generator.Result, error) {
			got = request
			return generator.Result{InstallPath: "installed"}, nil
		},
	}
	presentation := generator.DefaultHUDPresentationConfig()
	presentation.SecondaryColor = "  #12aBcD  "
	presentation.SecondarySize = 18
	presentation.SecondaryItalic = false
	presentation.ShowLanguageTags = false

	_, err := service.GenerateAndInstallWithPresentation(parent, localization.Russian, localization.English, &presentation)
	if err != nil {
		t.Fatalf("GenerateAndInstallWithPresentation() error = %v", err)
	}
	if got.GameRoot != content || got.SubtitleStyle != generator.SubtitleStyleHUD || got.HUDPresentation == nil {
		t.Fatalf("request = %+v, want normalized HUD request", got)
	}
	if got.HUDPresentation.SecondaryColor != "#12aBcD" || got.HUDPresentation.SecondarySize != 18 || got.HUDPresentation.SecondaryItalic || got.HUDPresentation.ShowLanguageTags {
		t.Fatalf("HUD presentation = %+v, want normalized explicit values", got.HUDPresentation)
	}
}

func TestGenerateAndInstallPreservesLegacyTaggedRequest(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "Kingdom Come")
	content := filepath.Join(parent, "Content")
	createApplicationGameLayout(t, content)

	var got generator.Request
	service := Service{generate: func(request generator.Request) (generator.Result, error) {
		got = request
		return generator.Result{}, nil
	}}
	_, err := service.GenerateAndInstall(parent, localization.Russian, localization.English)
	if err != nil {
		t.Fatalf("GenerateAndInstall() error = %v", err)
	}
	if got.SubtitleStyle != "" || got.HUDPresentation != nil {
		t.Fatalf("legacy request = %+v, want default tagged style without HUD presentation", got)
	}
}

func TestGenerateAndInstallWithPresentationRejectsInvalidConfigBeforeGameRoot(t *testing.T) {
	called := false
	service := Service{generate: func(generator.Request) (generator.Result, error) {
		called = true
		return generator.Result{}, nil
	}}
	presentation := generator.DefaultHUDPresentationConfig()
	presentation.SecondaryColor = "blue"

	_, err := service.GenerateAndInstallWithPresentation("not-a-game", localization.Russian, localization.English, &presentation)
	if !errors.Is(err, generator.ErrInvalidRequest) {
		t.Fatalf("error = %v, want generator.ErrInvalidRequest", err)
	}
	if called {
		t.Fatal("generator called for invalid presentation")
	}
}
