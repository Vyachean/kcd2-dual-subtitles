package gfxpatch

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/subtitlepayload"
)

const inlineHUDMarker = "KCD2DS_HUD_INLINE_V1"

// PatchHUDInline rewrites the already-existing retail fc_setSubtitles body
// instead of replacing the global function with a later wrapper. Retail
// acceptance proved that KCD2 executes the derived function body but does not
// route subtitle calls through the separately redefined global function.
//
// The injected prelude extracts the secondary payload into a spare AVM1
// register and restores the primary text before vanilla rendering. The
// postlude runs after vanilla setTextSize/background work, appends the styled
// secondary line through htmlText, then remeasures position/background. This
// preserves the retail subtitle flow and allows the secondary HTML size to
// survive the vanilla global sizing pass.
func PatchHUDInline(input []byte) ([]byte, error) {
	decoded, err := decodeContainer(input)
	if err != nil {
		return nil, err
	}
	if bytes.Contains(decoded.body, []byte(inlineHUDMarker)) {
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
	rewritten, rewrittenCount, err := rewriteSubtitleInlineActions(actions)
	if err != nil {
		return nil, fmt.Errorf("rewrite fc_setSubtitles inline: %w", err)
	}
	if rewrittenCount != 1 {
		return nil, fmt.Errorf("%w: rewrote %d fc_setSubtitles definitions", ErrAmbiguousTarget, rewrittenCount)
	}
	rewrittenTag, err := encodeTag(tagDoAction, rewritten)
	if err != nil {
		return nil, fmt.Errorf("encode inline subtitle tag: %w", err)
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

type inlineFunction2Info struct {
	name           string
	params         []functionParam
	registerCount  byte
	codeSize       int
	codeSizeOffset int
}

func parseInlineFunction2(data []byte) (inlineFunction2Info, error) {
	pos := 0
	name, next, err := readCString(data, pos)
	if err != nil {
		return inlineFunction2Info{}, fmt.Errorf("parse DefineFunction2 name: %w", err)
	}
	pos = next
	if len(data)-pos < 5 {
		return inlineFunction2Info{}, fmt.Errorf("truncated DefineFunction2 header")
	}
	paramCount := int(binary.LittleEndian.Uint16(data[pos : pos+2]))
	pos += 2
	registerCount := data[pos]
	pos++
	pos += 2 // Flags.

	params := make([]functionParam, 0, paramCount)
	for range paramCount {
		if pos >= len(data) {
			return inlineFunction2Info{}, fmt.Errorf("truncated DefineFunction2 parameter register")
		}
		register := data[pos]
		pos++
		param, next, err := readCString(data, pos)
		if err != nil {
			return inlineFunction2Info{}, fmt.Errorf("parse DefineFunction2 parameter: %w", err)
		}
		pos = next
		params = append(params, functionParam{register: register, name: param})
	}
	if len(data)-pos != 2 {
		return inlineFunction2Info{}, fmt.Errorf("DefineFunction2 metadata has %d unexpected bytes", len(data)-pos)
	}
	return inlineFunction2Info{
		name:           name,
		params:         params,
		registerCount:  registerCount,
		codeSize:       int(binary.LittleEndian.Uint16(data[pos : pos+2])),
		codeSizeOffset: pos,
	}, nil
}

func rewriteSubtitleInlineActions(actions []byte) ([]byte, int, error) {
	out := make([]byte, 0, len(actions)+512)
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

		textRegister, tempRegister, secondaryRegister, err := chooseInlineRegisters(*targetInfo, actions[bodyStart:bodyEnd])
		if err != nil {
			return nil, 0, err
		}
		prelude, err := buildInlinePrelude(textRegister, tempRegister, secondaryRegister)
		if err != nil {
			return nil, 0, err
		}
		postlude, err := buildInlinePostlude(secondaryRegister)
		if err != nil {
			return nil, 0, err
		}

		newCodeSize := targetInfo.codeSize + len(prelude) + len(postlude)
		if newCodeSize > 0xffff {
			return nil, 0, fmt.Errorf("inline fc_setSubtitles body exceeds AVM1 size limit")
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

func chooseInlineRegisters(info inlineFunction2Info, body []byte) (textRegister, tempRegister, secondaryRegister byte, err error) {
	if len(info.params) != 3 {
		return 0, 0, 0, fmt.Errorf("%w: fc_setSubtitles parameter contract changed", ErrSemanticMismatch)
	}
	textRegister = info.params[0].register
	if textRegister == 0 {
		return 0, 0, 0, fmt.Errorf("%w: fc_setSubtitles text parameter is not register-bound", ErrSemanticMismatch)
	}
	if info.registerCount == 0 {
		return 0, 0, 0, fmt.Errorf("%w: fc_setSubtitles has no AVM1 registers", ErrSemanticMismatch)
	}

	parameterRegisters := make(map[byte]bool)
	for _, param := range info.params {
		if param.register != 0 {
			parameterRegisters[param.register] = true
		}
	}
	used, err := flatFunctionRegisters(body)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("%w: inspect fc_setSubtitles registers: %v", ErrSemanticMismatch, err)
	}

	for candidate := info.registerCount; candidate >= 1; candidate-- {
		if !parameterRegisters[candidate] && !used[candidate] {
			secondaryRegister = candidate
			break
		}
		if candidate == 1 {
			break
		}
	}
	if secondaryRegister == 0 {
		return 0, 0, 0, fmt.Errorf("%w: no spare persistent register for secondary subtitle", ErrSemanticMismatch)
	}

	for candidate := byte(1); candidate <= info.registerCount; candidate++ {
		if candidate != secondaryRegister && !parameterRegisters[candidate] {
			tempRegister = candidate
			break
		}
		if candidate == info.registerCount {
			break
		}
	}
	if tempRegister == 0 {
		return 0, 0, 0, fmt.Errorf("%w: no temporary register for subtitle carrier parsing", ErrSemanticMismatch)
	}
	return textRegister, tempRegister, secondaryRegister, nil
}

func flatFunctionRegisters(actions []byte) (map[byte]bool, error) {
	used := make(map[byte]bool)
	pos := 0
	for pos < len(actions) {
		code := actions[pos]
		pos++
		if code == actionEnd {
			return used, nil
		}
		if code < 0x80 {
			continue
		}
		if len(actions)-pos < 2 {
			return nil, fmt.Errorf("truncated action length for 0x%02x", code)
		}
		length := int(binary.LittleEndian.Uint16(actions[pos : pos+2]))
		pos += 2
		if length > len(actions)-pos {
			return nil, fmt.Errorf("action 0x%02x data is truncated", code)
		}
		data := actions[pos : pos+length]
		pos += length

		switch code {
		case actionStoreRegister:
			if len(data) != 1 {
				return nil, fmt.Errorf("invalid StoreRegister payload size %d", len(data))
			}
			used[data[0]] = true
		case actionPush:
			registers, err := pushRegisters(data)
			if err != nil {
				return nil, err
			}
			for _, register := range registers {
				used[register] = true
			}
		case actionDefineFunction2, actionDefineFunction, actionWith, actionTry:
			return nil, fmt.Errorf("unsupported nested AVM1 scope action 0x%02x in subtitle function", code)
		}
	}
	return used, nil
}

func pushRegisters(data []byte) ([]byte, error) {
	var registers []byte
	pos := 0
	for pos < len(data) {
		typeCode := data[pos]
		pos++
		switch typeCode {
		case 0: // String.
			_, next, err := readCString(data, pos)
			if err != nil {
				return nil, err
			}
			pos = next
		case 1, 7: // Float / integer.
			pos += 4
		case 2, 3: // Null / undefined.
		case 4: // Register.
			if pos >= len(data) {
				return nil, fmt.Errorf("truncated register push")
			}
			registers = append(registers, data[pos])
			pos++
		case 5, 8: // Boolean / Constant8.
			pos++
		case 6: // Double.
			pos += 8
		case 9: // Constant16.
			pos += 2
		default:
			return nil, fmt.Errorf("unsupported Push value type %d", typeCode)
		}
		if pos > len(data) {
			return nil, fmt.Errorf("truncated Push payload")
		}
	}
	return registers, nil
}

func buildInlinePrelude(textRegister, tempRegister, secondaryRegister byte) ([]byte, error) {
	builder := newActionBuilder()

	// Idempotence marker embedded in the executed body without changing state.
	builder.pushString(inlineHUDMarker)
	builder.simple(actionPop)

	// secondary = ""
	builder.pushString("")
	builder.storeRegister(secondaryRegister)
	builder.simple(actionPop)

	// if (text.indexOf(Prefix) != 0) goto done
	builder.pushString(subtitlepayload.Prefix)
	builder.pushInt(1)
	builder.pushRegister(textRegister)
	builder.pushString("indexOf")
	builder.simple(actionCallMethod)
	builder.pushInt(0)
	builder.simple(actionEquals2)
	builder.simple(actionNot)
	builder.ifTrue("done")

	// end = text.indexOf(Suffix, len(Prefix)); malformed carriers remain vanilla.
	builder.pushInt(int32(len(subtitlepayload.Prefix)))
	builder.pushString(subtitlepayload.Suffix)
	builder.pushInt(2)
	builder.pushRegister(textRegister)
	builder.pushString("indexOf")
	builder.simple(actionCallMethod)
	builder.storeRegister(tempRegister)
	builder.simple(actionPop)
	builder.pushRegister(tempRegister)
	builder.pushInt(-1)
	builder.simple(actionEquals2)
	builder.ifTrue("done")

	// secondary = text.substring(len(Prefix), end)
	builder.pushRegister(tempRegister)
	builder.pushInt(int32(len(subtitlepayload.Prefix)))
	builder.pushInt(2)
	builder.pushRegister(textRegister)
	builder.pushString("substring")
	builder.simple(actionCallMethod)
	builder.storeRegister(secondaryRegister)
	builder.simple(actionPop)

	// text = text.substring(end + len(Suffix))
	builder.pushRegister(tempRegister)
	builder.pushInt(int32(len(subtitlepayload.Suffix)))
	builder.simple(actionAdd2)
	builder.pushInt(1)
	builder.pushRegister(textRegister)
	builder.pushString("substring")
	builder.simple(actionCallMethod)
	builder.storeRegister(textRegister)
	builder.simple(actionPop)

	builder.label("done")
	return builder.finish()
}

func buildInlinePostlude(secondaryRegister byte) ([]byte, error) {
	builder := newActionBuilder()

	// Non-carrier/fallback rows preserve the retail function byte-for-byte at
	// runtime apart from the no-op prelude marker.
	builder.pushRegister(secondaryRegister)
	builder.pushString("")
	builder.simple(actionEquals2)
	builder.ifTrue("done")

	// tField.htmlText = tField.htmlText + styled secondary. This runs after the
	// vanilla setTextSize call, so the secondary size is not overwritten by the
	// global subtitle-size pass.
	builder.pushTextField()
	builder.pushString("htmlText")
	builder.pushTextField()
	builder.pushString("htmlText")
	builder.simple(actionGetMember)
	builder.pushString("<br/><font color='" + subtitlepayload.SecondaryColor + "' size='" + strconv.Itoa(subtitlepayload.SecondarySize) + "'><i>")
	builder.simple(actionStringAdd)
	builder.pushRegister(secondaryRegister)
	builder.simple(actionStringAdd)
	builder.pushString("</i></font>")
	builder.simple(actionStringAdd)
	builder.simple(actionSetMember)

	// The retail function measured layout before our final line existed.
	for _, functionName := range []string{"updateSubtitlePosition", "setSubtitlesBackground"} {
		builder.pushInt(0)
		builder.pushString(functionName)
		builder.simple(actionCallFunction)
		builder.simple(actionPop)
	}

	builder.label("done")
	return builder.finish()
}
