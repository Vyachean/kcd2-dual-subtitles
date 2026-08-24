package gfxpatch

import (
	"bytes"
	"testing"
)

func TestPatchHUDDirectHTMLAllDefaultHasNoReadabilityOverrides(t *testing.T) {
	input := syntheticDualSubtitleHUD(t, "CFX")
	patched, err := PatchHUDDirectHTMLAll(input)
	if err != nil {
		t.Fatalf("PatchHUDDirectHTMLAll() error = %v", err)
	}
	decoded, err := decodeContainer(patched)
	if err != nil {
		t.Fatalf("decode patched HUD: %v", err)
	}
	for _, property := range []string{"outline", "shadowColor", "shadowBlurX", "shadowDistance"} {
		if bytes.Contains(decoded.body, []byte(property)) {
			t.Fatalf("default HUD unexpectedly contains readability property %q", property)
		}
	}
}

func TestPatchHUDDirectHTMLAllWithOutlineAppliesToStandardAndBubbleFields(t *testing.T) {
	input := syntheticDualSubtitleHUD(t, "CFX")
	patched, err := PatchHUDDirectHTMLAllWithReadability(input, HUDReadabilityConfig{Outline: true})
	if err != nil {
		t.Fatalf("PatchHUDDirectHTMLAllWithReadability(outline) error = %v", err)
	}
	decoded, err := decodeContainer(patched)
	if err != nil {
		t.Fatalf("decode patched HUD: %v", err)
	}
	if got := bytes.Count(decoded.body, []byte("outline")); got != 2 {
		t.Fatalf("outline property occurrences = %d, want 2 (standard + bubble)", got)
	}
	if bytes.Contains(decoded.body, []byte("shadowColor")) {
		t.Fatal("outline-only HUD unexpectedly contains shadow properties")
	}
}

func TestPatchHUDDirectHTMLAllWithShadowAppliesToStandardAndBubbleFields(t *testing.T) {
	input := syntheticDualSubtitleHUD(t, "CFX")
	patched, err := PatchHUDDirectHTMLAllWithReadability(input, HUDReadabilityConfig{Shadow: true})
	if err != nil {
		t.Fatalf("PatchHUDDirectHTMLAllWithReadability(shadow) error = %v", err)
	}
	decoded, err := decodeContainer(patched)
	if err != nil {
		t.Fatalf("decode patched HUD: %v", err)
	}
	for _, property := range []string{
		"shadowColor",
		"shadowAlpha",
		"shadowBlurX",
		"shadowBlurY",
		"shadowAngle",
		"shadowDistance",
		"shadowQuality",
		"shadowStrength",
	} {
		if got := bytes.Count(decoded.body, []byte(property)); got != 2 {
			t.Fatalf("%s occurrences = %d, want 2 (standard + bubble)", property, got)
		}
	}
	if bytes.Contains(decoded.body, []byte("outline")) {
		t.Fatal("shadow-only HUD unexpectedly contains outline property")
	}
}
