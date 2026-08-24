package gfxpatch

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

const HUDDiagnosticText = "[KCD2DS HUD ACTIVE]"

// PatchHUDDiagnostic first injects a direct assignment into the existing
// retail fc_setSubtitles body, then applies the experimental wrapper. The
// direct assignment is intentionally temporary acceptance instrumentation: if
// the derived hud.gfx is actually selected by retail KCD2, every subtitle
// routed through fc_setSubtitles must visibly become HUDDiagnosticText even if
// the wrapper itself is broken.
func PatchHUDDiagnostic(input []byte) ([]byte, error) {
	diagnostic, err := injectSubtitleDiagnostic(input)
	if err != nil {
		return nil, fmt.Errorf("inject direct subtitle diagnostic: %w", err)
	}
	return PatchHUD(diagnostic)
}

func injectSubtitleDiagnostic(input []byte) ([]byte, error) {
	decoded, err := decodeContainer(input)
	if err != nil {
		return nil, err
	}
	if bytes.Contains(decoded.body, []byte(HUDDiagnosticText)) {
		return append([]byte(nil), input...), nil
	}

	tags, err := parseRootTags(decoded.body)
	if err != nil {
		return nil, err
	}

	var target *swfTag
	matches := 0
	for i := range tags {
		tag := tags[i]
		if tag.code != tagDoAction {
			continue
		}
		count, err := countSubtitleFunctions(decoded.body[tag.payloadStart:tag.payloadEnd])
		if err != nil {
			return nil, fmt.Errorf("%w: parse root DoAction: %v", ErrInvalidGFX, err)
		}
		if count == 0 {
			continue
		}
		matches += count
		candidate := tag
		target = &candidate
	}

	switch {
	case matches == 0:
		return nil, ErrSubtitleTarget
	case matches != 1 || target == nil:
		return nil, fmt.Errorf("%w: found %d fc_setSubtitles definitions", ErrAmbiguousTarget, matches)
	}

	actions := decoded.body[target.payloadStart:target.payloadEnd]
	rewritten, rewrittenCount, err := rewriteSubtitleFunctionActions(actions)
	if err != nil {
		return nil, fmt.Errorf("rewrite fc_setSubtitles: %w", err)
	}
	if rewrittenCount != 1 {
		return nil, fmt.Errorf("%w: rewrote %d fc_setSubtitles definitions", ErrAmbiguousTarget, rewrittenCount)
	}
	rewrittenTag, err := encodeTag(tagDoAction, rewritten)
	if err != nil {
		return nil, fmt.Errorf("encode diagnostic subtitle tag: %w", err)
	}

	patchedBody := make([]byte, 0, len(decoded.body)+len(rewrittenTag)-(target.end-target.start))
	patchedBody = append(patchedBody, decoded.body[:target.start]...)
	patchedBody = append(patchedBody, rewrittenTag...)
	patchedBody = append(patchedBody, decoded.body[target.end:]...)

	return encodeContainer(container{
		signature: decoded.signature,
		version:   decoded.version,
		body:      patchedBody,
	})
}

type diagnosticFunction2Info struct {
	name           string
	params         []functionParam
	codeSize       int
	codeSizeOffset int
}

func parseDiagnosticFunction2(data []byte) (diagnosticFunction2Info, error) {
	pos := 0
	name, next, err := readCString(data, pos)
	if err != nil {
		return diagnosticFunction2Info{}, fmt.Errorf("parse DefineFunction2 name: %w", err)
	}
	pos = next
	if len(data)-pos < 5 {
		return diagnosticFunction2Info{}, fmt.Errorf("truncated DefineFunction2 header")
	}
	paramCount := int(binary.LittleEndian.Uint16(data[pos : pos+2]))
	pos += 2
	pos++    // RegisterCount.
	pos += 2 // Flags.

	params := make([]functionParam, 0, paramCount)
	for range paramCount {
		if pos >= len(data) {
			return diagnosticFunction2Info{}, fmt.Errorf("truncated DefineFunction2 parameter register")
		}
		register := data[pos]
		pos++
		param, next, err := readCString(data, pos)
		if err != nil {
			return diagnosticFunction2Info{}, fmt.Errorf("parse DefineFunction2 parameter: %w", err)
		}
		pos = next
		params = append(params, functionParam{register: register, name: param})
	}
	if len(data)-pos != 2 {
		return diagnosticFunction2Info{}, fmt.Errorf("DefineFunction2 metadata has %d unexpected bytes", len(data)-pos)
	}
	return diagnosticFunction2Info{
		name:           name,
		params:         params,
		codeSize:       int(binary.LittleEndian.Uint16(data[pos : pos+2])),
		codeSizeOffset: pos,
	}, nil
}

func rewriteSubtitleFunctionActions(actions []byte) ([]byte, int, error) {
	out := make([]byte, 0, len(actions)+64)
	matches := 0
	pos := 0

	for pos < len(actions) {
		actionStart := pos
		code := actions[pos]
		pos++
		if code == actionEnd {
			for _, trailing := range actions[pos:] {
				if trailing != 0 {
					return nil, 0, fmt.Errorf("non-zero bytes after ActionEnd")
				}
			}
			out = append(out, actions[actionStart:]...)
			return out, matches, nil
		}
		if code < 0x80 {
			out = append(out, actions[actionStart:pos]...)
			continue
		}
		if len(actions)-pos < 2 {
			return nil, 0, fmt.Errorf("truncated action length for 0x%02x", code)
		}
		length := int(binary.LittleEndian.Uint16(actions[pos : pos+2]))
		pos += 2
		dataStart := pos
		if length > len(actions)-pos {
			return nil, 0, fmt.Errorf("action 0x%02x data is truncated", code)
		}
		dataEnd := pos + length
		data := actions[pos:dataEnd]
		pos = dataEnd

		trailingCodeSize := 0
		var targetInfo *diagnosticFunction2Info
		switch code {
		case actionDefineFunction2:
			info, err := parseDiagnosticFunction2(data)
			if err != nil {
				return nil, 0, err
			}
			trailingCodeSize = info.codeSize
			paramNames := make([]string, 0, len(info.params))
			for _, param := range info.params {
				paramNames = append(paramNames, param.name)
			}
			if info.name == subtitleFunction && equalStrings(paramNames, []string{"text", "speakerName", "isPlayer"}) {
				matches++
				copyInfo := info
				targetInfo = &copyInfo
			}
		case actionDefineFunction:
			size, err := defineFunctionCodeSize(data)
			if err != nil {
				return nil, 0, err
			}
			trailingCodeSize = size
		case actionWith:
			if len(data) != 2 {
				return nil, 0, fmt.Errorf("invalid ActionWith metadata size %d", len(data))
			}
			trailingCodeSize = int(binary.LittleEndian.Uint16(data))
		case actionTry:
			size, err := tryCodeSize(data)
			if err != nil {
				return nil, 0, err
			}
			trailingCodeSize = size
		}
		if trailingCodeSize > len(actions)-pos {
			return nil, 0, fmt.Errorf("action 0x%02x nested body is truncated", code)
		}
		bodyStart := pos
		bodyEnd := pos + trailingCodeSize
		pos = bodyEnd

		if targetInfo == nil {
			out = append(out, actions[actionStart:pos]...)
			continue
		}
		if len(targetInfo.params) == 0 {
			return nil, 0, fmt.Errorf("fc_setSubtitles has no text parameter")
		}

		assignment, err := diagnosticTextAssignment(targetInfo.params[0].register)
		if err != nil {
			return nil, 0, err
		}
		newCodeSize := targetInfo.codeSize + len(assignment)
		if newCodeSize > 0xffff {
			return nil, 0, fmt.Errorf("diagnostic fc_setSubtitles body exceeds AVM1 size limit")
		}
		modifiedData := append([]byte(nil), data...)
		binary.LittleEndian.PutUint16(modifiedData[targetInfo.codeSizeOffset:targetInfo.codeSizeOffset+2], uint16(newCodeSize))

		out = append(out, actions[actionStart:dataStart]...)
		out = append(out, modifiedData...)
		out = append(out, assignment...)
		out = append(out, actions[bodyStart:bodyEnd]...)
	}

	return nil, 0, fmt.Errorf("missing ActionEnd")
}

func diagnosticTextAssignment(register byte) ([]byte, error) {
	var actions []byte
	if register == 0 {
		actions = append(actions, pushString("text")...)
		actions = append(actions, pushString(HUDDiagnosticText)...)
		actions = append(actions, actionSetVariable)
		return actions, nil
	}

	actions = append(actions, pushString(HUDDiagnosticText)...)
	store, err := longAction(actionStoreRegister, []byte{register})
	if err != nil {
		return nil, err
	}
	actions = append(actions, store...)
	actions = append(actions, actionPop)
	return actions, nil
}
