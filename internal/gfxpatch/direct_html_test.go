package gfxpatch

import (
	"bytes"
	"errors"
	"testing"
)

func TestPatchHUDDirectHTMLRewritesRetailFunctionInPlace(t *testing.T) {
	body := append(pushString("ORIGINAL_BODY_SENTINEL"), actionPop)
	input := syntheticRegisteredHUD(t, "CFX", 7, body)

	patched, err := PatchHUDDirectHTML(input)
	if err != nil {
		t.Fatalf("PatchHUDDirectHTML() error = %v", err)
	}
	if bytes.Equal(patched, input) {
		t.Fatal("PatchHUDDirectHTML() returned unchanged input")
	}
	decoded, err := decodeContainer(patched)
	if err != nil {
		t.Fatalf("decode patched HUD: %v", err)
	}
	for _, marker := range []string{
		directHTMLMarker,
		"ORIGINAL_BODY_SENTINEL",
		"htmlText",
		"updateSubtitlePosition",
		"setSubtitlesBackground",
	} {
		if !bytes.Contains(decoded.body, []byte(marker)) {
			t.Fatalf("patched HUD does not contain %q", marker)
		}
	}
	if bytes.Contains(decoded.body, []byte(HUDDiagnosticText)) {
		t.Fatal("direct HTML HUD unexpectedly contains rc.5 diagnostic marker")
	}

	idempotent, err := PatchHUDDirectHTML(patched)
	if err != nil {
		t.Fatalf("PatchHUDDirectHTML(already patched) error = %v", err)
	}
	if !bytes.Equal(idempotent, patched) {
		t.Fatal("already direct-HTML-patched HUD was modified again")
	}
}

func TestPatchHUDDirectHTMLKeepsRetailContainerSignature(t *testing.T) {
	for _, signature := range []string{"FWS", "GFX", "CWS", "CFX"} {
		t.Run(signature, func(t *testing.T) {
			input := syntheticRegisteredHUD(t, signature, 7, nil)
			patched, err := PatchHUDDirectHTML(input)
			if err != nil {
				t.Fatalf("PatchHUDDirectHTML() error = %v", err)
			}
			if string(patched[:3]) != signature {
				t.Fatalf("signature = %q, want %q", patched[:3], signature)
			}
		})
	}
}

func TestPatchHUDDirectHTMLFailsClosedWhenNoSpareRegisterExists(t *testing.T) {
	var body []byte
	for _, register := range []byte{2, 4, 5} {
		body = append(body, pushRegisterForTest(t, register)...)
		body = append(body, actionPop)
	}
	input := syntheticRegisteredHUD(t, "FWS", 6, body)

	_, err := PatchHUDDirectHTML(input)
	if !errors.Is(err, ErrSemanticMismatch) {
		t.Fatalf("PatchHUDDirectHTML() error = %v, want ErrSemanticMismatch", err)
	}
}

func TestPatchHUDDirectHTMLRejectsEarlyReturn(t *testing.T) {
	body := append(pushString("result"), actionReturn)
	input := syntheticRegisteredHUD(t, "FWS", 7, body)

	_, err := PatchHUDDirectHTML(input)
	if !errors.Is(err, ErrSemanticMismatch) {
		t.Fatalf("PatchHUDDirectHTML() error = %v, want ErrSemanticMismatch", err)
	}
}
