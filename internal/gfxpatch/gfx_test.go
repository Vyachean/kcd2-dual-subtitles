package gfxpatch

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/subtitlepayload"
)

func TestPatchHUDInjectsWrapperAfterUniqueSubtitleDefinition(t *testing.T) {
	input := syntheticHUD(t, "FWS", true, 1, []swfTagFixture{
		{code: 77, payload: []byte("unrelated-tag")},
	})

	patched, err := PatchHUD(input)
	if err != nil {
		t.Fatalf("PatchHUD() error = %v", err)
	}
	if bytes.Equal(patched, input) {
		t.Fatal("PatchHUD() returned unchanged input")
	}
	decoded, err := decodeContainer(patched)
	if err != nil {
		t.Fatalf("decode patched HUD: %v", err)
	}
	if !bytes.Contains(decoded.body, []byte(subtitlepayload.HUDWrapperMarker)) {
		t.Fatal("patched HUD does not contain wrapper marker")
	}
	if !bytes.Contains(decoded.body, []byte("htmlText")) || !bytes.Contains(decoded.body, []byte(subtitlepayload.SecondaryColor)) {
		t.Fatal("patched HUD does not contain direct htmlText member contract")
	}
	if bytes.Contains(decoded.body, []byte("appendHtml")) {
		t.Fatal("patched HUD unexpectedly depends on non-standard appendHtml")
	}

	tags, err := parseRootTags(decoded.body)
	if err != nil {
		t.Fatalf("parse patched tags: %v", err)
	}
	var codes []uint16
	var unrelated []byte
	for _, tag := range tags {
		codes = append(codes, tag.code)
		if tag.code == 77 {
			unrelated = append([]byte(nil), decoded.body[tag.payloadStart:tag.payloadEnd]...)
		}
	}
	wantCodes := []uint16{12, 12, 77, 1, 0}
	if !equalTagCodes(codes, wantCodes) {
		t.Fatalf("patched tag codes = %v, want %v", codes, wantCodes)
	}
	if string(unrelated) != "unrelated-tag" {
		t.Fatalf("unrelated payload = %q, want unchanged", unrelated)
	}
}

func TestPatchHUDSupportsRetailCompressedCFXDeterministically(t *testing.T) {
	input := syntheticHUD(t, "CFX", true, 1, nil)
	first, err := PatchHUD(input)
	if err != nil {
		t.Fatalf("PatchHUD(first) error = %v", err)
	}
	second, err := PatchHUD(input)
	if err != nil {
		t.Fatalf("PatchHUD(second) error = %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("PatchHUD() output is not deterministic")
	}
	if string(first[:3]) != "CFX" {
		t.Fatalf("patched signature = %q, want CFX", first[:3])
	}
	if _, err := decodeContainer(first); err != nil {
		t.Fatalf("patched CFX does not parse: %v", err)
	}

	idempotent, err := PatchHUD(first)
	if err != nil {
		t.Fatalf("PatchHUD(already patched) error = %v", err)
	}
	if !bytes.Equal(idempotent, first) {
		t.Fatal("already-patched HUD was modified again")
	}
}

func TestPatchHUDAcceptsAllSupportedContainerSignatures(t *testing.T) {
	for _, signature := range []string{"FWS", "GFX", "CWS", "CFX"} {
		t.Run(signature, func(t *testing.T) {
			input := syntheticHUD(t, signature, true, 1, nil)
			patched, err := PatchHUD(input)
			if err != nil {
				t.Fatalf("PatchHUD() error = %v", err)
			}
			if string(patched[:3]) != signature {
				t.Fatalf("signature = %q, want %q", patched[:3], signature)
			}
		})
	}
}

func TestPatchHUDFailsClosedForTargetProblems(t *testing.T) {
	t.Run("missing target", func(t *testing.T) {
		input := syntheticHUD(t, "FWS", true, 0, nil)
		_, err := PatchHUD(input)
		if !errors.Is(err, ErrSubtitleTarget) {
			t.Fatalf("PatchHUD() error = %v, want ErrSubtitleTarget", err)
		}
	})

	t.Run("duplicate target", func(t *testing.T) {
		input := syntheticHUD(t, "FWS", true, 2, nil)
		_, err := PatchHUD(input)
		if !errors.Is(err, ErrAmbiguousTarget) {
			t.Fatalf("PatchHUD() error = %v, want ErrAmbiguousTarget", err)
		}
	})

	t.Run("semantic anchor mismatch", func(t *testing.T) {
		input := syntheticHUD(t, "FWS", false, 1, nil)
		_, err := PatchHUD(input)
		if !errors.Is(err, ErrSemanticMismatch) {
			t.Fatalf("PatchHUD() error = %v, want ErrSemanticMismatch", err)
		}
	})
}

func TestPatchHUDRejectsMalformedContainers(t *testing.T) {
	cases := map[string][]byte{
		"short":          []byte("CFX"),
		"signature":      append([]byte("XYZ\x08\x08\x00\x00\x00"), 0),
		"bad compressed": append([]byte("CFX\x08\x10\x00\x00\x00"), []byte("not-zlib")...),
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := PatchHUD(input)
			if !errors.Is(err, ErrInvalidGFX) {
				t.Fatalf("PatchHUD() error = %v, want ErrInvalidGFX", err)
			}
		})
	}
}

func TestCountSubtitleFunctionsSkipsFunctionBodies(t *testing.T) {
	nested, err := defineFunction2(subtitleFunction, []functionParam{
		{register: 0, name: "text"},
		{register: 0, name: "speakerName"},
		{register: 0, name: "isPlayer"},
	}, 0, []byte{actionEnd})
	if err != nil {
		t.Fatalf("define nested function: %v", err)
	}
	outer, err := defineFunction2("outer", nil, 0, append(nested, actionEnd))
	if err != nil {
		t.Fatalf("define outer function: %v", err)
	}
	actions := append(outer, actionEnd)
	count, err := countSubtitleFunctions(actions)
	if err != nil {
		t.Fatalf("countSubtitleFunctions() error = %v", err)
	}
	if count != 0 {
		t.Fatalf("countSubtitleFunctions() = %d, want 0 for nested definition", count)
	}
}

type swfTagFixture struct {
	code    uint16
	payload []byte
}

func syntheticHUD(t *testing.T, signature string, includeAnchors bool, targetCount int, extras []swfTagFixture) []byte {
	t.Helper()
	var body []byte
	body = append(body, 0x08, 0x00) // RECT: Nbits=1, all coordinates zero.
	body = append(body, 0x00, 0x0c) // FrameRate 12.0 in SWF fixed8 byte order.
	body = append(body, 0x01, 0x00) // FrameCount=1.

	for i := 0; i < targetCount; i++ {
		actions := syntheticTargetActions(t, includeAnchors)
		tag, err := encodeTag(tagDoAction, actions)
		if err != nil {
			t.Fatalf("encode target DoAction: %v", err)
		}
		body = append(body, tag...)
	}
	if targetCount == 0 && includeAnchors {
		actions := syntheticAnchorActions()
		tag, err := encodeTag(tagDoAction, actions)
		if err != nil {
			t.Fatalf("encode anchor DoAction: %v", err)
		}
		body = append(body, tag...)
	}
	for _, extra := range extras {
		tag, err := encodeTag(extra.code, extra.payload)
		if err != nil {
			t.Fatalf("encode extra tag: %v", err)
		}
		body = append(body, tag...)
	}
	show, err := encodeTag(tagShowFrame, nil)
	if err != nil {
		t.Fatalf("encode ShowFrame: %v", err)
	}
	body = append(body, show...)
	end, err := encodeTag(tagEnd, nil)
	if err != nil {
		t.Fatalf("encode End: %v", err)
	}
	body = append(body, end...)

	data, err := encodeContainer(container{signature: signature, version: 8, body: body})
	if err != nil {
		t.Fatalf("encode synthetic HUD: %v", err)
	}
	return data
}

func syntheticTargetActions(t *testing.T, includeAnchors bool) []byte {
	t.Helper()
	var actions []byte
	if includeAnchors {
		actions = append(actions, syntheticAnchorActions()...)
		// Remove the anchor stream's ActionEnd before adding the function.
		actions = actions[:len(actions)-1]
	}
	definition, err := defineFunction2(subtitleFunction, []functionParam{
		{register: 0, name: "text"},
		{register: 0, name: "speakerName"},
		{register: 0, name: "isPlayer"},
	}, 0, []byte{actionEnd})
	if err != nil {
		t.Fatalf("define synthetic fc_setSubtitles: %v", err)
	}
	actions = append(actions, definition...)
	actions = append(actions, actionEnd)
	return actions
}

func syntheticAnchorActions() []byte {
	var actions []byte
	for _, anchor := range requiredHUDAnchors {
		actions = append(actions, pushString(anchor)...)
		actions = append(actions, actionPop)
	}
	return append(actions, actionEnd)
}

func equalTagCodes(a, b []uint16) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestParseRootTagsRejectsTruncatedLongTag(t *testing.T) {
	body := []byte{0x08, 0x00, 0, 0, 1, 0}
	header := make([]byte, 2)
	binary.LittleEndian.PutUint16(header, (tagDoAction<<6)|0x3f)
	body = append(body, header...)
	body = append(body, 1, 2) // Missing two bytes of UI32 length.
	_, err := parseRootTags(body)
	if !errors.Is(err, ErrInvalidGFX) {
		t.Fatalf("parseRootTags() error = %v, want ErrInvalidGFX", err)
	}
}
