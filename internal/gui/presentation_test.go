package gui

import (
	"errors"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/generator"
)

func TestDefaultPresentationInputPreservesLegacyGUIPath(t *testing.T) {
	input := defaultPresentationInput()
	if input.Styled {
		t.Fatal("default GUI presentation unexpectedly enables styled mode")
	}
	if input.SecondaryColor != generator.DefaultHUDSecondaryColor || input.SecondarySize != "24" || !input.SecondaryItalic || !input.ShowLanguageTags {
		t.Fatalf("default presentation input = %+v", input)
	}

	presentation, err := input.hudPresentation()
	if err != nil {
		t.Fatalf("hudPresentation() error = %v", err)
	}
	if presentation != nil {
		t.Fatalf("presentation = %+v, want nil tagged path", presentation)
	}
}

func TestPresentationInputMapsStyledValuesToGeneratorConfig(t *testing.T) {
	input := presentationInput{
		Styled:           true,
		ShowLanguageTags: false,
		SecondaryColor:   "  #12aBcD  ",
		SecondarySize:    " 18 ",
		SecondaryItalic:  false,
	}
	presentation, err := input.hudPresentation()
	if err != nil {
		t.Fatalf("hudPresentation() error = %v", err)
	}
	if presentation == nil {
		t.Fatal("presentation is nil for styled mode")
	}
	if presentation.SecondaryColor != "#12aBcD" || presentation.SecondarySize != 18 || presentation.SecondaryItalic || presentation.ShowLanguageTags {
		t.Fatalf("presentation = %+v, want normalized explicit values", presentation)
	}
}

func TestPresentationInputIgnoresDisabledFieldsWhenStyledModeIsOff(t *testing.T) {
	presentation, err := (presentationInput{
		Styled:         false,
		SecondaryColor: "not-a-color",
		SecondarySize:  "not-a-number",
	}).hudPresentation()
	if err != nil {
		t.Fatalf("hudPresentation() error = %v, want disabled fields ignored", err)
	}
	if presentation != nil {
		t.Fatalf("presentation = %+v, want nil tagged path", presentation)
	}
}

func TestPresentationInputRejectsNonNumericSize(t *testing.T) {
	input := defaultPresentationInput()
	input.Styled = true
	input.SecondarySize = "large"
	if _, err := input.hudPresentation(); err == nil {
		t.Fatal("hudPresentation() error = nil, want actionable size error")
	}
}

func TestPresentationInputUsesGeneratorValidation(t *testing.T) {
	tests := []struct {
		name  string
		color string
		size  string
	}{
		{name: "invalid color", color: "#XYZXYZ", size: "24"},
		{name: "size below minimum", color: "#112233", size: "11"},
		{name: "size above maximum", color: "#112233", size: "49"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := defaultPresentationInput()
			input.Styled = true
			input.SecondaryColor = tt.color
			input.SecondarySize = tt.size
			_, err := input.hudPresentation()
			if !errors.Is(err, generator.ErrInvalidRequest) {
				t.Fatalf("error = %v, want generator.ErrInvalidRequest", err)
			}
		})
	}
}
