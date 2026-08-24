package gfxpatch

import (
	"bytes"
	"errors"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/subtitlepayload"
)

func TestPatchHUDInlineRewritesExistingSubtitleFunctionWithoutWrapper(t *testing.T) {
	input := syntheticRegisteredHUD(t, "CFX", 7, nil)

	patched, err := PatchHUDInline(input)
	if err != nil {
		t.Fatalf("PatchHUDInline() error = %v", err)
	}
	if bytes.Equal(patched, input) {
		t.Fatal("PatchHUDInline() returned unchanged input")
	}
	decoded, err := decodeContainer(patched)
	if err != nil {
		t.Fatalf("decode patched HUD: %v", err)
	}
	for _, marker := range []string{
		inlineHUDMarker,
		subtitlepayload.Prefix,
		subtitlepayload.Suffix,
		subtitlepayload.SecondaryColor,
		"htmlText",
		"updateSubtitlePosition",
		"setSubtitlesBackground",
	} {
		if !bytes.Contains(decoded.body, []byte(marker)) {
			t.Fatalf("patched HUD does not contain %q", marker)
		}
	}
	if bytes.Contains(decoded.body, []byte(HUDDiagnosticText)) {
		t.Fatal("inline HUD unexpectedly contains the rc.5 diagnostic marker")
	}
	if bytes.Contains(decoded.body, []byte(originalSubtitleFunction)) {
		t.Fatal("inline HUD unexpectedly contains the failed late-wrapper function")
	}

	tags, err := parseRootTags(decoded.body)
	if err != nil {
		t.Fatalf("parse patched tags: %v", err)
	}
	var codes []uint16
	for _, tag := range tags {
		codes = append(codes, tag.code)
	}
	wantCodes := []uint16{tagDoAction, tagShowFrame, tagEnd}
	if !equalTagCodes(codes, wantCodes) {
		t.Fatalf("patched tag codes = %v, want %v", codes, wantCodes)
	}

	idempotent, err := PatchHUDInline(patched)
	if err != nil {
		t.Fatalf("PatchHUDInline(already patched) error = %v", err)
	}
	if !bytes.Equal(idempotent, patched) {
		t.Fatal("already-inline-patched HUD was modified again")
	}
}

func TestPatchHUDInlineKeepsRetailContainerSignature(t *testing.T) {
	for _, signature := range []string{"FWS", "GFX", "CWS", "CFX"} {
		t.Run(signature, func(t *testing.T) {
			input := syntheticRegisteredHUD(t, signature, 7, nil)
			patched, err := PatchHUDInline(input)
			if err != nil {
				t.Fatalf("PatchHUDInline() error = %v", err)
			}
			if string(patched[:3]) != signature {
				t.Fatalf("signature = %q, want %q", patched[:3], signature)
			}
		})
	}
}

func TestPatchHUDInlineFailsClosedWhenNoSparePersistentRegisterExists(t *testing.T) {
	var body []byte
	for _, register := range []byte{2, 4, 5} {
		body = append(body, pushRegisterForTest(t, register)...)
		body = append(body, actionPop)
	}
	input := syntheticRegisteredHUD(t, "FWS", 6, body)

	_, err := PatchHUDInline(input)
	if !errors.Is(err, ErrSemanticMismatch) {
		t.Fatalf("PatchHUDInline() error = %v, want ErrSemanticMismatch", err)
	}
}

func syntheticRegisteredHUD(t *testing.T, signature string, registerCount byte, functionBody []byte) []byte {
	t.Helper()

	var rootBody []byte
	rootBody = append(rootBody, 0x08, 0x00) // RECT: Nbits=1, all coordinates zero.
	rootBody = append(rootBody, 0x00, 0x0c) // FrameRate 12.0.
	rootBody = append(rootBody, 0x01, 0x00) // FrameCount=1.

	actions := syntheticAnchorActions()
	actions = actions[:len(actions)-1] // Remove ActionEnd before function definition.
	definition, err := defineFunction2(subtitleFunction, []functionParam{
		{register: 1, name: "text"},
		{register: 3, name: "speakerName"},
		{register: 6, name: "isPlayer"},
	}, registerCount, functionBody)
	if err != nil {
		t.Fatalf("define synthetic registered fc_setSubtitles: %v", err)
	}
	actions = append(actions, definition...)
	actions = append(actions, actionEnd)
	doAction, err := encodeTag(tagDoAction, actions)
	if err != nil {
		t.Fatalf("encode DoAction: %v", err)
	}
	rootBody = append(rootBody, doAction...)

	showFrame, err := encodeTag(tagShowFrame, nil)
	if err != nil {
		t.Fatalf("encode ShowFrame: %v", err)
	}
	rootBody = append(rootBody, showFrame...)
	end, err := encodeTag(tagEnd, nil)
	if err != nil {
		t.Fatalf("encode End: %v", err)
	}
	rootBody = append(rootBody, end...)

	encoded, err := encodeContainer(container{signature: signature, version: 8, body: rootBody})
	if err != nil {
		t.Fatalf("encode synthetic HUD: %v", err)
	}
	return encoded
}

func pushRegisterForTest(t *testing.T, register byte) []byte {
	t.Helper()
	record, err := longAction(actionPush, []byte{4, register})
	if err != nil {
		t.Fatalf("push register %d: %v", register, err)
	}
	return record
}
