package gui

import (
	"errors"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/generator"
)

func TestDefaultPresentationInputPreservesGameAppearance(t *testing.T) {
	input := defaultPresentationInput()
	if input.Styled || input.ShowLanguageTags || input.PrimaryItalic || input.SecondaryItalic || input.Outline || input.Shadow {
		t.Fatalf("default presentation unexpectedly enables an appearance option: %+v", input)
	}
	if input.PrimaryColor != "" || input.PrimarySize != "" || input.SecondaryColor != "" || input.SecondarySize != "" {
		t.Fatalf("default presentation unexpectedly overrides color or size: %+v", input)
	}

	presentation, err := input.presentationConfig()
	if err != nil {
		t.Fatalf("presentationConfig() error = %v", err)
	}
	if presentation != nil {
		t.Fatalf("presentation = %+v, want nil plain path", presentation)
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
	presentation, err := input.presentationConfig()
	if err != nil {
		t.Fatalf("presentationConfig() error = %v", err)
	}
	if presentation == nil {
		t.Fatal("presentation is nil for styled mode")
	}
	if presentation.SecondaryColor != "#12aBcD" || presentation.SecondarySize != 18 || presentation.SecondaryItalic || presentation.ShowLanguageTags {
		t.Fatalf("presentation = %+v, want normalized explicit values", presentation)
	}
	if !generator.PresentationRequiresHUD(*presentation) {
		t.Fatalf("presentation = %+v, want HUD override requirement", presentation)
	}
}

func TestPresentationInputAllowsLanguageTagsWithoutStyledMode(t *testing.T) {
	input := defaultPresentationInput()
	input.ShowLanguageTags = true

	presentation, err := input.presentationConfig()
	if err != nil {
		t.Fatalf("presentationConfig() error = %v", err)
	}
	if presentation == nil || !presentation.ShowLanguageTags {
		t.Fatalf("presentation = %+v, want language-tags-only config", presentation)
	}
	if generator.PresentationRequiresHUD(*presentation) {
		t.Fatalf("presentation = %+v, tags alone must not require HUD", presentation)
	}
	if got := generator.PresentationSubtitleStyle(*presentation); got != generator.SubtitleStyleTagged {
		t.Fatalf("style = %q, want %q", got, generator.SubtitleStyleTagged)
	}
}

func TestPresentationInputIgnoresDisabledStyleFieldsWhenStyledModeIsOff(t *testing.T) {
	presentation, err := (presentationInput{
		Styled:         false,
		SecondaryColor: "not-a-color",
		SecondarySize:  "not-a-number",
	}).presentationConfig()
	if err != nil {
		t.Fatalf("presentationConfig() error = %v, want disabled fields ignored", err)
	}
	if presentation != nil {
		t.Fatalf("presentation = %+v, want nil plain path", presentation)
	}
}

func TestPresentationInputAcceptsBlankStyledOverrides(t *testing.T) {
	input := defaultPresentationInput()
	input.Styled = true

	presentation, err := input.presentationConfig()
	if err != nil {
		t.Fatalf("presentationConfig() error = %v", err)
	}
	if presentation == nil {
		t.Fatal("presentation is nil for enabled customization")
	}
	if generator.PresentationRequiresHUD(*presentation) {
		t.Fatalf("blank optional overrides unexpectedly require HUD: %+v", presentation)
	}
	if got := generator.PresentationSubtitleStyle(*presentation); got != generator.SubtitleStylePlain {
		t.Fatalf("style = %q, want %q", got, generator.SubtitleStylePlain)
	}
}

func TestPresentationInputRejectsNonNumericSize(t *testing.T) {
	input := defaultPresentationInput()
	input.Styled = true
	input.SecondarySize = "large"
	if _, err := input.presentationConfig(); err == nil {
		t.Fatal("presentationConfig() error = nil, want actionable size error")
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
			_, err := input.presentationConfig()
			if !errors.Is(err, generator.ErrInvalidRequest) {
				t.Fatalf("error = %v, want generator.ErrInvalidRequest", err)
			}
		})
	}
}
