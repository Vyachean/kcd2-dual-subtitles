package gui

import "testing"

func TestDefaultPresentationInputLeavesReadabilityDisabled(t *testing.T) {
	input := defaultPresentationInput()
	if input.Outline || input.Shadow {
		t.Fatalf("default readability = outline:%v shadow:%v, want both disabled", input.Outline, input.Shadow)
	}
}

func TestPresentationInputMapsReadabilityFlags(t *testing.T) {
	input := defaultPresentationInput()
	input.Styled = true
	input.Outline = true
	input.Shadow = true

	presentation, err := input.hudPresentation()
	if err != nil {
		t.Fatalf("hudPresentation() error = %v", err)
	}
	if presentation == nil || !presentation.Outline || !presentation.Shadow {
		t.Fatalf("presentation = %+v, want outline+shadow", presentation)
	}
}

func TestPresentationInputIgnoresReadabilityWhenStyledModeIsOff(t *testing.T) {
	input := defaultPresentationInput()
	input.Outline = true
	input.Shadow = true

	presentation, err := input.hudPresentation()
	if err != nil {
		t.Fatalf("hudPresentation() error = %v", err)
	}
	if presentation != nil {
		t.Fatalf("presentation = %+v, want nil tagged path", presentation)
	}
}
