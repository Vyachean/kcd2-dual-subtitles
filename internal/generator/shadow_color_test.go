package generator

import (
	"errors"
	"testing"
)

func TestDefaultHUDPresentationUsesBlackShadowColor(t *testing.T) {
	presentation := DefaultHUDPresentationConfig()
	if presentation.ShadowColor != "#000000" {
		t.Fatalf("ShadowColor = %q, want #000000", presentation.ShadowColor)
	}
	if presentation.Shadow {
		t.Fatal("default presentation unexpectedly enables shadow")
	}
}

func TestNormalizeHUDPresentationDefaultsEmptyShadowColorToBlack(t *testing.T) {
	presentation := DefaultHUDPresentationConfig()
	presentation.ShadowColor = ""

	normalized, err := NormalizeHUDPresentationConfig(presentation)
	if err != nil {
		t.Fatalf("NormalizeHUDPresentationConfig() error = %v", err)
	}
	if normalized.ShadowColor != DefaultHUDShadowColor {
		t.Fatalf("ShadowColor = %q, want %q", normalized.ShadowColor, DefaultHUDShadowColor)
	}
}

func TestNormalizeHUDPresentationAcceptsAndConvertsShadowColor(t *testing.T) {
	presentation := DefaultHUDPresentationConfig()
	presentation.Shadow = true
	presentation.ShadowColor = " #12Ab34 "

	normalized, err := NormalizeHUDPresentationConfig(presentation)
	if err != nil {
		t.Fatalf("NormalizeHUDPresentationConfig() error = %v", err)
	}
	if normalized.ShadowColor != "#12Ab34" {
		t.Fatalf("ShadowColor = %q, want trimmed #12Ab34", normalized.ShadowColor)
	}
	value, err := hudColorValue(normalized.ShadowColor)
	if err != nil {
		t.Fatalf("hudColorValue() error = %v", err)
	}
	if value != 0x12AB34 {
		t.Fatalf("hudColorValue() = %#x, want 0x12AB34", value)
	}
}

func TestNormalizeHUDPresentationRejectsInvalidShadowColor(t *testing.T) {
	presentation := DefaultHUDPresentationConfig()
	presentation.Shadow = true
	presentation.ShadowColor = "blue"

	_, err := NormalizeHUDPresentationConfig(presentation)
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("error = %v, want ErrInvalidRequest", err)
	}
}
