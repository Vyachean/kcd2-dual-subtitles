package gfxpatch

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

const (
	bubbleFunction       = "fc_setBubbleText"
	bubbleHTMLMarker     = "KCD2DS_HUD_BUBBLE_HTML_V1"
	actionSubtractBubble = byte(0x0b)
)

var requiredBubbleHUDAnchors = []string{
	"cccursor",
	"bubbles",
	"BUBBLE_NAME_CORE",
	"inside",
	"tfText",
	"TextExtension",
	"setTextSize",
	"setBubblesBackground",
}

// PatchHUDDirectHTMLAll applies the accepted bottom-screen subtitle HTML patch
// first and then the equivalent post-vanilla HTML path for overhead NPC
// bubbles without changing TextField readability properties.
func PatchHUDDirectHTMLAll(input []byte) ([]byte, error) {
	return PatchHUDDirectHTMLAllWithReadability(input, HUDReadabilityConfig{})
}

// PatchHUDDirectHTMLAllWithReadability applies the direct-HTML path to standard
// subtitles and overhead bubbles with the same optional whole-TextField
// readability effects. Keeping the two transformations separate makes each one
// independently fail-closed and preserves their retail lifecycle/geometry.
func PatchHUDDirectHTMLAllWithReadability(input []byte, readability HUDReadabilityConfig) ([]byte, error) {
	patched, err := PatchHUDDirectHTMLWithReadability(input, readability)
	if err != nil {
		return nil, err
	}
	patched, err = PatchHUDBubbleDirectHTMLWithReadability(patched, readability)
	if err != nil {
		return nil, fmt.Errorf("patch overhead subtitle bubbles: %w", err)
	}
	return patched, nil
}

// PatchHUDBubbleDirectHTML rewrites the retail bubble function using the
// accepted direct-HTML path without changing TextField readability properties.
func PatchHUDBubbleDirectHTML(input []byte) ([]byte, error) {
	return PatchHUDBubbleDirectHTMLWithReadability(input, HUDReadabilityConfig{})
}

// PatchHUDBubbleDirectHTMLWithReadability rewrites the retail fc_setBubbleText
// function so the original localization HTML is restored after vanilla text
// processing and global sizing. The retail function remains authoritative for
// wrapping, depth/distance behavior and normal bubble lifecycle; only the final
// TextField content, optional readability properties, vertical offset and
// background measurement are refreshed.
func PatchHUDBubbleDirectHTMLWithReadability(input []byte, readability HUDReadabilityConfig) ([]byte, error) {
	decoded, err := decodeContainer(input)
	if err != nil {
		return nil, err
	}
	if bytes.Contains(decoded.body, []byte(bubbleHTMLMarker)) {
		return append([]byte(nil), input...), nil
	}
	for _, anchor := range requiredBubbleHUDAnchors {
		if !bytes.Contains(decoded.body, []byte(anchor)) {
			return nil, fmt.Errorf("%w: missing bubble anchor %q", ErrSemanticMismatch, anchor)
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
		count, err := countBubbleFunctions(decoded.body[tag.payloadStart:tag.payloadEnd])
		if err != nil {
			return nil, fmt.Errorf("%w: parse root DoAction for bubble target: %v", ErrInvalidGFX, err)
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
		return nil, fmt.Errorf("%w: %s", ErrSubtitleTarget, bubbleFunction)
	case matches != 1 || target == nil:
		return nil, fmt.Errorf("%w: found %d %s definitions", ErrAmbiguousTarget, matches, bubbleFunction)
	}

	actions := decoded.body[target.payloadStart:target.payloadEnd]
	rewritten, rewrittenCount, err := rewriteBubbleDirectHTMLActions(actions, readability)
	if err != nil {
		return nil, fmt.Errorf("rewrite %s direct HTML: %w", bubbleFunction, err)
	}
	if rewrittenCount != 1 {
		return nil, fmt.Errorf("%w: rewrote %d %s definitions", ErrAmbiguousTarget, rewrittenCount, bubbleFunction)
	}
	rewrittenTag, err := encodeTag(tagDoAction, rewritten)
	if err != nil {
		return nil, fmt.Errorf("encode bubble direct HTML tag: %w", err)
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

func countBubbleFunctions(actions []byte) (int, error) {
	count := 0
	pos := 0
	for pos < len(actions) {
		code := actions[pos]
		pos++
		if code == actionEnd {
			for _, trailing := range actions[pos:] {
				if trailing != 0 {
					return 0, fmt.Errorf("non-zero bytes after ActionEnd")
				}
			}
			return count, nil
		}
		if code < 0x80 {
			continue
		}
		if len(actions)-pos < 2 {
			return 0, fmt.Errorf("truncated action length for 0x%02x", code)
		}
		length := int(binary.LittleEndian.Uint16(actions[pos : pos+2]))
		pos += 2
		if length > len(actions)-pos {
			return 0, fmt.Errorf("action 0x%02x data is truncated", code)
		}
		data := actions[pos : pos+length]
		pos += length

		trailingCodeSize := 0
		switch code {
		case actionDefineFunction2:
			info, err := parseFunction2(data)
			if err != nil {
				return 0, err
			}
			trailingCodeSize = info.codeSize
			if info.name == bubbleFunction && equalStrings(info.params, []string{"bubbleId", "bubbleText", "speakerName", "playerDistance"}) {
				count++
			}
		case actionDefineFunction:
			size, err := defineFunctionCodeSize(data)
			if err != nil {
				return 0, err
			}
			trailingCodeSize = size
		case actionWith:
			if len(data) != 2 {
				return 0, fmt.Errorf("invalid ActionWith metadata size %d", len(data))
			}
			trailingCodeSize = int(binary.LittleEndian.Uint16(data))
		case actionTry:
			size, err := tryCodeSize(data)
			if err != nil {
				return 0, err
			}
			trailingCodeSize = size
		}
		if trailingCodeSize > len(actions)-pos {
			return 0, fmt.Errorf("action 0x%02x nested body is truncated", code)
		}
		pos += trailingCodeSize
	}
	return 0, fmt.Errorf("missing ActionEnd")
}

func rewriteBubbleDirectHTMLActions(actions []byte, readability HUDReadabilityConfig) ([]byte, int, error) {
	out := make([]byte, 0, len(actions)+320)
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
			if info.name == bubbleFunction && equalStrings(paramNames, []string{"bubbleId", "bubbleText", "speakerName", "playerDistance"}) {
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

		bubbleIDRegister, textRegister, savedRegister, err := chooseBubbleHTMLRegisters(*targetInfo, actions[bodyStart:bodyEnd])
		if err != nil {
			return nil, 0, err
		}
		prelude, err := buildBubbleHTMLPrelude(textRegister, savedRegister)
		if err != nil {
			return nil, 0, err
		}
		postlude, err := buildBubbleHTMLPostlude(bubbleIDRegister, savedRegister, readability)
		if err != nil {
			return nil, 0, err
		}

		newCodeSize := targetInfo.codeSize + len(prelude) + len(postlude)
		if newCodeSize > 0xffff {
			return nil, 0, fmt.Errorf("direct HTML %s body exceeds AVM1 size limit", bubbleFunction)
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

func chooseBubbleHTMLRegisters(info inlineFunction2Info, body []byte) (bubbleIDRegister, textRegister, savedRegister byte, err error) {
	if len(info.params) != 4 {
		return 0, 0, 0, fmt.Errorf("%w: %s parameter contract changed", ErrSemanticMismatch, bubbleFunction)
	}
	bubbleIDRegister = info.params[0].register
	textRegister = info.params[1].register
	if bubbleIDRegister == 0 || textRegister == 0 {
		return 0, 0, 0, fmt.Errorf("%w: %s ID/text parameters are not register-bound", ErrSemanticMismatch, bubbleFunction)
	}
	if info.registerCount == 0 {
		return 0, 0, 0, fmt.Errorf("%w: %s has no AVM1 registers", ErrSemanticMismatch, bubbleFunction)
	}

	parameterRegisters := make(map[byte]bool)
	for _, param := range info.params {
		if param.register != 0 {
			parameterRegisters[param.register] = true
		}
	}
	used, err := flatFunctionRegisters(body)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("%w: inspect %s registers: %v", ErrSemanticMismatch, bubbleFunction, err)
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
		return 0, 0, 0, fmt.Errorf("%w: no spare persistent register for original bubble HTML", ErrSemanticMismatch)
	}
	if err := validateDirectHTMLEarlyReturns(body); err != nil {
		return 0, 0, 0, fmt.Errorf("%w: inspect %s return flow: %v", ErrSemanticMismatch, bubbleFunction, err)
	}
	return bubbleIDRegister, textRegister, savedRegister, nil
}

func buildBubbleHTMLPrelude(textRegister, savedRegister byte) ([]byte, error) {
	builder := newActionBuilder()
	builder.pushString(bubbleHTMLMarker)
	builder.simple(actionPop)
	builder.pushRegister(textRegister)
	builder.storeRegister(savedRegister)
	builder.simple(actionPop)
	return builder.finish()
}

func buildBubbleHTMLPostlude(bubbleIDRegister, savedRegister byte, readability HUDReadabilityConfig) ([]byte, error) {
	builder := newActionBuilder()

	pushBubbleTextField(builder, bubbleIDRegister)
	builder.pushString("htmlText")
	builder.pushRegister(savedRegister)
	builder.simple(actionSetMember)

	appendTextFieldReadability(builder, func() {
		pushBubbleTextField(builder, bubbleIDRegister)
	}, readability)

	pushBubbleInside(builder, bubbleIDRegister)
	builder.pushString("_y")
	builder.pushInt(0)
	pushBubbleTextField(builder, bubbleIDRegister)
	builder.pushString("textHeight")
	builder.simple(actionGetMember)
	builder.simple(actionSubtractBubble)
	builder.simple(actionSetMember)

	builder.pushInt(0)
	builder.pushString("setBubblesBackground")
	builder.simple(actionCallFunction)
	builder.simple(actionPop)
	return builder.finish()
}

func pushBubbleTextField(builder *actionBuilder, bubbleIDRegister byte) {
	pushBubbleInside(builder, bubbleIDRegister)
	builder.pushString("tfText")
	builder.simple(actionGetMember)
}

func pushBubbleInside(builder *actionBuilder, bubbleIDRegister byte) {
	builder.pushString("cccursor")
	builder.simple(actionGetVariable)
	builder.pushString("bubbles")
	builder.simple(actionGetMember)
	builder.pushString("Enum")
	builder.simple(actionGetVariable)
	builder.pushString("BUBBLE_NAME_CORE")
	builder.simple(actionGetMember)
	builder.pushRegister(bubbleIDRegister)
	builder.simple(actionAdd2)
	builder.simple(actionGetMember)
	builder.pushString("inside")
	builder.simple(actionGetMember)
}
