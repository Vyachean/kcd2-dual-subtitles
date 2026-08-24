package gui

import (
	"errors"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/generator"
)

func TestDefaultPresentationInputLeavesPrimaryVanilla(t *testing.T) {
	input := defaultPresentationInput()
	if input.PrimaryColor != "" || input.PrimarySize != "" || input.PrimaryItalic {
		t.Fatalf("default primary input = %+v, want empty/vanilla", input)
	}
}

func TestPresentationInputMapsOptionalPrimaryOverrides(t *testing.T) {
	input := defaultPresentationInput()
	input.Styled = true
	input.PrimaryColor = " #ABCDEF "
	input.PrimarySize = " 30 "
	input.PrimaryItalic = true

	presentation, err := input.hudPresentation()
	if err != nil {
		t.Fatalf("hudPresentation() error = %v", err)
	}
	if presentation.PrimaryColor != "#ABCDEF" || presentation.PrimarySize != 30 || !presentation.PrimaryItalic {
		t.Fatalf("primary presentation = %+v", presentation)
	}
}

func TestPresentationInputEmptyPrimarySizeKeepsVanillaSize(t *testing.T) {
	input := defaultPresentationInput()
	input.Styled = true
	input.PrimaryColor = "#FFFFFF"
	input.PrimarySize = "   "

	presentation, err := input.hudPresentation()
	if err != nil {
		t.Fatalf("hudPresentation() error = %v", err)
	}
	if presentation.PrimarySize != 0 {
		t.Fatalf("PrimarySize = %d, want vanilla sentinel 0", presentation.PrimarySize)
	}
}

func TestPresentationInputRejectsInvalidPrimarySize(t *testing.T) {
	input := defaultPresentationInput()
	input.Styled = true
	input.PrimarySize = "large"
	if _, err := input.hudPresentation(); err == nil {
		t.Fatal("hudPresentation() error = nil, want actionable primary size error")
	}

	input.PrimarySize = "11"
	_, err := input.hudPresentation()
	if !errors.Is(err, generator.ErrInvalidRequest) {
		t.Fatalf("error = %v, want generator.ErrInvalidRequest", err)
	}
}
