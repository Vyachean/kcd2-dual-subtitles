package generator

import (
	"errors"
	"testing"
)

func TestDefaultHUDPresentationUsesNormalShadowIntensity(t *testing.T) {
	presentation := DefaultHUDPresentationConfig()
	if presentation.ShadowIntensity != HUDShadowNormal {
		t.Fatalf("ShadowIntensity = %q, want %q", presentation.ShadowIntensity, HUDShadowNormal)
	}
}

func TestNormalizeHUDPresentationDefaultsEmptyShadowIntensityToNormal(t *testing.T) {
	presentation := DefaultHUDPresentationConfig()
	presentation.ShadowIntensity = ""

	normalized, err := NormalizeHUDPresentationConfig(presentation)
	if err != nil {
		t.Fatalf("NormalizeHUDPresentationConfig() error = %v", err)
	}
	if normalized.ShadowIntensity != HUDShadowNormal {
		t.Fatalf("ShadowIntensity = %q, want %q", normalized.ShadowIntensity, HUDShadowNormal)
	}
}

func TestNormalizeHUDPresentationAcceptsShadowIntensityCaseInsensitively(t *testing.T) {
	presentation := DefaultHUDPresentationConfig()
	presentation.ShadowIntensity = " STRONG "

	normalized, err := NormalizeHUDPresentationConfig(presentation)
	if err != nil {
		t.Fatalf("NormalizeHUDPresentationConfig() error = %v", err)
	}
	if normalized.ShadowIntensity != HUDShadowStrong {
		t.Fatalf("ShadowIntensity = %q, want %q", normalized.ShadowIntensity, HUDShadowStrong)
	}
}

func TestNormalizeHUDPresentationRejectsUnknownShadowIntensity(t *testing.T) {
	presentation := DefaultHUDPresentationConfig()
	presentation.ShadowIntensity = "extreme"

	_, err := NormalizeHUDPresentationConfig(presentation)
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("error = %v, want ErrInvalidRequest", err)
	}
}

func TestShadowGeometryPresets(t *testing.T) {
	tests := []struct {
		name      string
		intensity HUDShadowIntensity
		want      hudShadowGeometry
	}{
		{name: "subtle", intensity: HUDShadowSubtle, want: hudShadowGeometry{blur: 1, distance: 1, strength: 1}},
		{name: "normal accepted v0.3", intensity: HUDShadowNormal, want: hudShadowGeometry{blur: 2, distance: 1, strength: 1}},
		{name: "strong", intensity: HUDShadowStrong, want: hudShadowGeometry{blur: 3, distance: 2, strength: 2}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := shadowGeometry(test.intensity); got != test.want {
				t.Fatalf("shadowGeometry(%q) = %+v, want %+v", test.intensity, got, test.want)
			}
		})
	}
}

func TestReadabilityConfigRoutesPresetGeometry(t *testing.T) {
	presentation := DefaultHUDPresentationConfig()
	presentation.Outline = true
	presentation.Shadow = true
	presentation.ShadowIntensity = HUDShadowStrong

	got := readabilityConfig(presentation, 0x123456)
	if !got.Outline || !got.Shadow || got.ShadowColor != 0x123456 {
		t.Fatalf("readabilityConfig() identity = %+v", got)
	}
	if got.ShadowBlur != 3 || got.ShadowDistance != 2 || got.ShadowStrength != 2 {
		t.Fatalf("readabilityConfig() geometry = %+v, want blur=3 distance=2 strength=2", got)
	}
}
