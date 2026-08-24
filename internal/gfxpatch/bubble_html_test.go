package gfxpatch

import (
	"bytes"
	"errors"
	"testing"
)

func TestPatchHUDBubbleDirectHTMLRewritesRetailFunctionInPlace(t *testing.T) {
	body := append(pushString("BUBBLE_BODY_SENTINEL"), actionPop)
	input := syntheticBubbleHUD(t, "CFX", 13, body)

	patched, err := PatchHUDBubbleDirectHTML(input)
	if err != nil {
		t.Fatalf("PatchHUDBubbleDirectHTML() error = %v", err)
	}
	if bytes.Equal(patched, input) {
		t.Fatal("PatchHUDBubbleDirectHTML() returned unchanged input")
	}
	decoded, err := decodeContainer(patched)
	if err != nil {
		t.Fatalf("decode patched bubble HUD: %v", err)
	}
	for _, marker := range []string{
		bubbleHTMLMarker,
		"BUBBLE_BODY_SENTINEL",
		"cccursor",
		"bubbles",
		"BUBBLE_NAME_CORE",
		"tfText",
		"htmlText",
		"textHeight",
		"setBubblesBackground",
	} {
		if !bytes.Contains(decoded.body, []byte(marker)) {
			t.Fatalf("patched bubble HUD does not contain %q", marker)
		}
	}

	idempotent, err := PatchHUDBubbleDirectHTML(patched)
	if err != nil {
		t.Fatalf("PatchHUDBubbleDirectHTML(already patched) error = %v", err)
	}
	if !bytes.Equal(idempotent, patched) {
		t.Fatal("already bubble-patched HUD was modified again")
	}
}

func TestPatchHUDBubbleDirectHTMLFailsClosedWhenNoSpareRegisterExists(t *testing.T) {
	var body []byte
	for _, register := range []byte{1, 3, 4, 5, 6, 8, 9, 10} {
		body = append(body, pushRegisterForTest(t, register)...)
		body = append(body, actionPop)
	}
	input := syntheticBubbleHUD(t, "FWS", 12, body)

	_, err := PatchHUDBubbleDirectHTML(input)
	if !errors.Is(err, ErrSemanticMismatch) {
		t.Fatalf("PatchHUDBubbleDirectHTML() error = %v, want ErrSemanticMismatch", err)
	}
}

func TestPatchHUDBubbleDirectHTMLAllowsGuardedEarlyReturn(t *testing.T) {
	builder := newActionBuilder()
	builder.pushInt(1)
	builder.ifTrue("continue")
	builder.pushString("early")
	builder.simple(actionReturn)
	builder.label("continue")
	builder.pushString("BUBBLE_NORMAL_PATH_SENTINEL")
	builder.simple(actionPop)
	body, err := builder.finish()
	if err != nil {
		t.Fatalf("build bubble guarded return body: %v", err)
	}
	input := syntheticBubbleHUD(t, "FWS", 13, body)

	patched, err := PatchHUDBubbleDirectHTML(input)
	if err != nil {
		t.Fatalf("PatchHUDBubbleDirectHTML() guarded-return error = %v", err)
	}
	decoded, err := decodeContainer(patched)
	if err != nil {
		t.Fatalf("decode guarded-return bubble HUD: %v", err)
	}
	if !bytes.Contains(decoded.body, []byte("BUBBLE_NORMAL_PATH_SENTINEL")) || !bytes.Contains(decoded.body, []byte(bubbleHTMLMarker)) {
		t.Fatal("guarded-return bubble patch did not preserve normal path and HTML postlude")
	}
}

func TestPatchHUDDirectHTMLAllPreservesBottomPatchAndAddsBubblePatch(t *testing.T) {
	input := syntheticDualSubtitleHUD(t, "CFX")

	patched, err := PatchHUDDirectHTMLAll(input)
	if err != nil {
		t.Fatalf("PatchHUDDirectHTMLAll() error = %v", err)
	}
	decoded, err := decodeContainer(patched)
	if err != nil {
		t.Fatalf("decode combined patched HUD: %v", err)
	}
	for _, marker := range []string{directHTMLMarker, bubbleHTMLMarker, "tField", "tfText"} {
		if !bytes.Contains(decoded.body, []byte(marker)) {
			t.Fatalf("combined patched HUD does not contain %q", marker)
		}
	}

	idempotent, err := PatchHUDDirectHTMLAll(patched)
	if err != nil {
		t.Fatalf("PatchHUDDirectHTMLAll(already patched) error = %v", err)
	}
	if !bytes.Equal(idempotent, patched) {
		t.Fatal("combined patch is not idempotent")
	}
}

func syntheticBubbleHUD(t *testing.T, signature string, registerCount byte, functionBody []byte) []byte {
	t.Helper()
	return syntheticHUDWithFunctions(t, signature, nil, registerCount, functionBody)
}

func syntheticDualSubtitleHUD(t *testing.T, signature string) []byte {
	t.Helper()
	return syntheticHUDWithFunctions(t, signature, []byte{}, 13, []byte{})
}

func syntheticHUDWithFunctions(t *testing.T, signature string, subtitleBody []byte, bubbleRegisterCount byte, bubbleBody []byte) []byte {
	t.Helper()

	var rootBody []byte
	rootBody = append(rootBody, 0x08, 0x00)
	rootBody = append(rootBody, 0x00, 0x0c)
	rootBody = append(rootBody, 0x01, 0x00)

	actions := syntheticAnchorActions()
	actions = actions[:len(actions)-1]
	for _, anchor := range requiredBubbleHUDAnchors {
		actions = append(actions, pushString(anchor)...)
		actions = append(actions, actionPop)
	}

	if subtitleBody != nil {
		definition, err := defineFunction2(subtitleFunction, []functionParam{
			{register: 1, name: "text"},
			{register: 3, name: "speakerName"},
			{register: 6, name: "isPlayer"},
		}, 7, subtitleBody)
		if err != nil {
			t.Fatalf("define synthetic fc_setSubtitles: %v", err)
		}
		actions = append(actions, definition...)
	}

	bubbleDefinition, err := defineFunction2(bubbleFunction, []functionParam{
		{register: 11, name: "bubbleId"},
		{register: 2, name: "bubbleText"},
		{register: 7, name: "speakerName"},
		{register: 12, name: "playerDistance"},
	}, bubbleRegisterCount, bubbleBody)
	if err != nil {
		t.Fatalf("define synthetic fc_setBubbleText: %v", err)
	}
	actions = append(actions, bubbleDefinition...)
	actions = append(actions, actionEnd)

	doAction, err := encodeTag(tagDoAction, actions)
	if err != nil {
		t.Fatalf("encode bubble DoAction: %v", err)
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
		t.Fatalf("encode synthetic bubble HUD: %v", err)
	}
	return encoded
}
