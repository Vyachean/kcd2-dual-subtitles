package gfxpatch

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

const directHTMLMarker = "KCD2DS_HUD_DIRECT_HTML_V1"

// PatchHUDDirectHTML rewrites the retail subtitle function using the accepted
// direct-HTML path without changing TextField readability properties.
func PatchHUDDirectHTML(input []byte) ([]byte, error) {
	return PatchHUDDirectHTMLWithReadability(input, HUDReadabilityConfig{})
}

// PatchHUDDirectHTMLWithReadability rewrites the existing retail
// fc_setSubtitles body with the smallest post-vanilla HTML path possible. The
// original text argument is saved before the retail body runs, then assigned
// directly to the standard subtitle TextField htmlText property after vanilla
// sizing/layout work. Optional readability effects are applied to the complete
// TextField after the final htmlText assignment.
func PatchHUDDirectHTMLWithReadability(input []byte, readability HUDReadabilityConfig) ([]byte, error) {
	decoded, err := decodeContainer(input)
	if err != nil {
		return nil, err
	}
	if bytes.Contains(decoded.body, []byte(directHTMLMarker)) {
		return append([]byte(nil), input...), nil
	}

	for _, anchor := range requiredHUDAnchors {
		if !bytes.Contains(decoded.body, []byte(anchor)) {
			return nil, fmt.Errorf("%w: missing %q", ErrSemanticMismatch, anchor)
		}
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
	rewritten, rewrittenCount, err := rewriteSubtitleDirectHTMLActions(actions, readability)
	if err != nil {
		return nil, fmt.Errorf("rewrite fc_setSubtitles direct HTML: %w", err)
	}
	if rewrittenCount != 1 {
		return nil, fmt.Errorf("%w: rewrote %d fc_setSubtitles definitions", ErrAmbiguousTarget, rewrittenCount)
	}
	rewrittenTag, err := encodeTag(tagDoAction, rewritten)
	if err != nil {
		return nil, fmt.Errorf("encode direct HTML subtitle tag: %w", err)
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

func rewriteSubtitleDirectHTMLActions(actions []byte, readability HUDReadabilityConfig) ([]byte, int, error) {
	out := make([]byte, 0, len(actions)+256)
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
		var targetInfo *inlineFunction2Info
		switch code {
		case actionDefineFunction2:
			info, err := parseInlineFunction2(data)
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

		textRegister, savedRegister, err := chooseDirectHTMLRegisters(*targetInfo, actions[bodyStart:bodyEnd])
		if err != nil {
			return nil, 0, err
		}
		prelude, err := buildDirectHTMLPrelude(textRegister, savedRegister)
		if err != nil {
			return nil, 0, err
		}
		postlude, err := buildDirectHTMLPostlude(savedRegister, readability)
		if err != nil {
			return nil, 0, err
		}

		newCodeSize := targetInfo.codeSize + len(prelude) + len(postlude)
		if newCodeSize > 0xffff {
			return nil, 0, fmt.Errorf("direct HTML fc_setSubtitles body exceeds AVM1 size limit")
		}
		modifiedData := append([]byte(nil), data...)
		binary.LittleEndian.PutUint16(modifiedData[targetInfo.codeSizeOffset:targetInfo.codeSizeOffset+2], uint16(newCodeSize))

		out = append(out, actions[actionStart:dataStart]...)
		out = append(out, modifiedData...)
		out = append(out, prelude...)
		out = append(out, actions[bodyStart:bodyEnd]...)
		out = append(out, postlude...)
	}

	return nil, 0, fmt.Errorf("missing ActionEnd")
}

func chooseDirectHTMLRegisters(info inlineFunction2Info, body []byte) (textRegister, savedRegister byte, err error) {
	if len(info.params) != 3 {
		return 0, 0, fmt.Errorf("%w: fc_setSubtitles parameter contract changed", ErrSemanticMismatch)
	}
	textRegister = info.params[0].register
	if textRegister == 0 {
		return 0, 0, fmt.Errorf("%w: fc_setSubtitles text parameter is not register-bound", ErrSemanticMismatch)
	}
	if info.registerCount == 0 {
		return 0, 0, fmt.Errorf("%w: fc_setSubtitles has no AVM1 registers", ErrSemanticMismatch)
	}

	parameterRegisters := make(map[byte]bool)
	for _, param := range info.params {
		if param.register != 0 {
			parameterRegisters[param.register] = true
		}
	}
	used, err := flatFunctionRegisters(body)
	if err != nil {
		return 0, 0, fmt.Errorf("%w: inspect fc_setSubtitles registers: %v", ErrSemanticMismatch, err)
	}

	for candidate := info.registerCount; candidate >= 1; candidate-- {
		if !parameterRegisters[candidate] && !used[candidate] {
			savedRegister = candidate
			break
		}
		if candidate == 1 {
			break
		}
	}
	if savedRegister == 0 {
		return 0, 0, fmt.Errorf("%w: no spare persistent register for original subtitle HTML", ErrSemanticMismatch)
	}
	if err := validateDirectHTMLEarlyReturns(body); err != nil {
		return 0, 0, fmt.Errorf("%w: inspect fc_setSubtitles return flow: %v", ErrSemanticMismatch, err)
	}
	return textRegister, savedRegister, nil
}

func validateDirectHTMLEarlyReturns(actions []byte) error {
	bypassTargets := make(map[int]bool)
	var returns []int
	pos := 0

	for pos < len(actions) {
		start := pos
		code := actions[pos]
		pos++
		if code == actionReturn {
			returns = append(returns, start)
			continue
		}
		if code < 0x80 {
			continue
		}
		if len(actions)-pos < 2 {
			return fmt.Errorf("truncated action length for 0x%02x", code)
		}
		length := int(binary.LittleEndian.Uint16(actions[pos : pos+2]))
		pos += 2
		if length > len(actions)-pos {
			return fmt.Errorf("action 0x%02x data is truncated", code)
		}
		data := actions[pos : pos+length]
		pos += length

		if code != actionIf {
			continue
		}
		if len(data) != 2 {
			return fmt.Errorf("invalid ActionIf payload size %d", len(data))
		}
		target := pos + int(int16(binary.LittleEndian.Uint16(data)))
		if target < 0 || target > len(actions) {
			return fmt.Errorf("ActionIf target %d is outside function body", target)
		}
		bypassTargets[target] = true
	}

	for _, returnPos := range returns {
		afterReturn := returnPos + 1
		if afterReturn >= len(actions) || !bypassTargets[afterReturn] {
			return fmt.Errorf("unguarded ActionReturn at byte offset %d", returnPos)
		}
	}
	return nil
}

func buildDirectHTMLPrelude(textRegister, savedRegister byte) ([]byte, error) {
	builder := newActionBuilder()
	builder.pushString(directHTMLMarker)
	builder.simple(actionPop)
	builder.pushRegister(textRegister)
	builder.storeRegister(savedRegister)
	builder.simple(actionPop)
	return builder.finish()
}

func buildDirectHTMLPostlude(savedRegister byte, readability HUDReadabilityConfig) ([]byte, error) {
	builder := newActionBuilder()
	builder.pushTextField()
	builder.pushString("htmlText")
	builder.pushRegister(savedRegister)
	builder.simple(actionSetMember)

	appendTextFieldReadability(builder, builder.pushTextField, readability)

	for _, functionName := range []string{"updateSubtitlePosition", "setSubtitlesBackground"} {
		builder.pushInt(0)
		builder.pushString(functionName)
		builder.simple(actionCallFunction)
		builder.simple(actionPop)
	}
	return builder.finish()
}
