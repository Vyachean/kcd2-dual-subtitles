package gui

import (
	"errors"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/generator"
)

func TestDefaultPresentationInputUsesBlackShadowColor(t *testing.T) {
	input := defaultPresentationInput()
	if input.ShadowColor != generator.DefaultHUDShadowColor {
		t.Fatalf("ShadowColor = %q, want %q", input.ShadowColor, generator.DefaultHUDShadowColor)
	}
}

func TestPresentationInputMapsShadowColor(t *testing.T) {
	input := defaultPresentationInput()
	input.Styled = true
	input.Shadow = true
	input.ShadowColor = "#654321"

	presentation, err := input.hudPresentation()
	if err != nil {
		t.Fatalf("hudPresentation() error = %v", err)
	}
	if presentation == nil || !presentation.Shadow || presentation.ShadowColor != "#654321" {
		t.Fatalf("presentation = %+v", presentation)
	}
}

func TestPresentationInputRejectsInvalidShadowColor(t *testing.T) {
	input := defaultPresentationInput()
	input.Styled = true
	input.Shadow = true
	input.ShadowColor = "not-a-color"

	_, err := input.hudPresentation()
	if !errors.Is(err, generator.ErrInvalidRequest) {
		t.Fatalf("error = %v, want ErrInvalidRequest", err)
	}
}
