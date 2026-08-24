package gfxpatch

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/subtitlepayload"
)

func TestPatchHUDDiagnosticInjectsDirectMarkerAndWrapper(t *testing.T) {
	input := syntheticHUD(t, "FWS", true, 1, nil)

	patched, err := PatchHUDDiagnostic(input)
	if err != nil {
		t.Fatalf("PatchHUDDiagnostic() error = %v", err)
	}
	decoded, err := decodeContainer(patched)
	if err != nil {
		t.Fatalf("decode patched HUD: %v", err)
	}
	if !bytes.Contains(decoded.body, []byte(HUDDiagnosticText)) {
		t.Fatal("patched HUD does not contain direct diagnostic text")
	}
	if !bytes.Contains(decoded.body, []byte(subtitlepayload.HUDWrapperMarker)) {
		t.Fatal("patched HUD does not retain experimental wrapper")
	}
}

func TestRewriteSubtitleFunctionActionsExpandsFunctionBody(t *testing.T) {
	actions := syntheticTargetActions(t, true)
	before := firstSubtitleFunctionInfo(t, actions)

	rewritten, matches, err := rewriteSubtitleFunctionActions(actions)
	if err != nil {
		t.Fatalf("rewriteSubtitleFunctionActions() error = %v", err)
	}
	if matches != 1 {
		t.Fatalf("rewriteSubtitleFunctionActions() matches = %d, want 1", matches)
	}
	after := firstSubtitleFunctionInfo(t, rewritten)
	if after.codeSize <= before.codeSize {
		t.Fatalf("codeSize = %d after rewrite, want > %d", after.codeSize, before.codeSize)
	}
	if !bytes.Contains(rewritten, []byte(HUDDiagnosticText)) {
		t.Fatal("rewritten function does not contain diagnostic text")
	}
	if count, err := countSubtitleFunctions(rewritten); err != nil || count != 1 {
		t.Fatalf("countSubtitleFunctions(rewritten) = %d, %v; want 1, nil", count, err)
	}
}

func TestDiagnosticAssignmentUsesParameterRegisterWhenPresent(t *testing.T) {
	body := []byte{actionEnd}
	definition, err := defineFunction2(subtitleFunction, []functionParam{
		{register: 3, name: "text"},
		{register: 0, name: "speakerName"},
		{register: 0, name: "isPlayer"},
	}, 3, body)
	if err != nil {
		t.Fatalf("defineFunction2() error = %v", err)
	}
	actions := append(definition, actionEnd)

	rewritten, matches, err := rewriteSubtitleFunctionActions(actions)
	if err != nil {
		t.Fatalf("rewriteSubtitleFunctionActions() error = %v", err)
	}
	if matches != 1 {
		t.Fatalf("matches = %d, want 1", matches)
	}
	storeRegister3 := []byte{actionStoreRegister, 1, 0, 3}
	if !bytes.Contains(rewritten, storeRegister3) {
		t.Fatalf("rewritten function does not store diagnostic text in register 3: %x", rewritten)
	}
}

func firstSubtitleFunctionInfo(t *testing.T, actions []byte) diagnosticFunction2Info {
	t.Helper()
	pos := 0
	for pos < len(actions) {
		code := actions[pos]
		pos++
		if code == actionEnd {
			break
		}
		if code < 0x80 {
			continue
		}
		if len(actions)-pos < 2 {
			t.Fatal("truncated action length")
		}
		length := int(binary.LittleEndian.Uint16(actions[pos : pos+2]))
		pos += 2
		if length > len(actions)-pos {
			t.Fatal("truncated action payload")
		}
		data := actions[pos : pos+length]
		pos += length

		trailing := 0
		switch code {
		case actionDefineFunction2:
			info, err := parseDiagnosticFunction2(data)
			if err != nil {
				t.Fatalf("parseDiagnosticFunction2() error = %v", err)
			}
			trailing = info.codeSize
			paramNames := make([]string, 0, len(info.params))
			for _, param := range info.params {
				paramNames = append(paramNames, param.name)
			}
			if info.name == subtitleFunction && equalStrings(paramNames, []string{"text", "speakerName", "isPlayer"}) {
				return info
			}
		case actionDefineFunction:
			size, err := defineFunctionCodeSize(data)
			if err != nil {
				t.Fatalf("defineFunctionCodeSize() error = %v", err)
			}
			trailing = size
		case actionWith:
			trailing = int(binary.LittleEndian.Uint16(data))
		case actionTry:
			size, err := tryCodeSize(data)
			if err != nil {
				t.Fatalf("tryCodeSize() error = %v", err)
			}
			trailing = size
		}
		if trailing > len(actions)-pos {
			t.Fatal("truncated nested body")
		}
		pos += trailing
	}
	t.Fatal("fc_setSubtitles not found")
	return diagnosticFunction2Info{}
}
