package gui

import (
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/generator"
)

func TestDefaultPresentationInputUsesNormalShadowIntensity(t *testing.T) {
	input := defaultPresentationInput()
	if input.ShadowIntensity != generator.HUDShadowNormal {
		t.Fatalf("ShadowIntensity = %q, want %q", input.ShadowIntensity, generator.HUDShadowNormal)
	}
}

func TestPresentationInputMapsShadowIntensity(t *testing.T) {
	input := defaultPresentationInput()
	input.Styled = true
	input.Shadow = true
	input.ShadowIntensity = generator.HUDShadowStrong

	presentation, err := input.hudPresentation()
	if err != nil {
		t.Fatalf("hudPresentation() error = %v", err)
	}
	if presentation == nil || presentation.ShadowIntensity != generator.HUDShadowStrong {
		t.Fatalf("presentation = %+v, want strong shadow intensity", presentation)
	}
}
