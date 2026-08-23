package gfxpatch

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/subtitlepayload"
)

const (
	maxHUDUncompressed = 64 << 20

	tagEnd       uint16 = 0
	tagShowFrame uint16 = 1
	tagDoAction  uint16 = 12

	actionEnd             byte = 0x00
	actionNot             byte = 0x12
	actionPop             byte = 0x17
	actionGetVariable     byte = 0x1c
	actionSetVariable     byte = 0x1d
	actionStringAdd       byte = 0x21
	actionCallFunction    byte = 0x3d
	actionReturn          byte = 0x3e
	actionEquals2         byte = 0x49
	actionGetMember       byte = 0x4e
	actionCallMethod      byte = 0x52
	actionStoreRegister   byte = 0x87
	actionDefineFunction2 byte = 0x8e
	actionTry             byte = 0x8f
	actionWith            byte = 0x94
	actionPush            byte = 0x96
	actionIf              byte = 0x9d
	actionDefineFunction  byte = 0x9b
)

const (
	originalSubtitleFunction = "__kcd2ds_original_fc_setSubtitles"
	wrapperVersionVariable   = "__kcd2ds_hud_wrapper_version"
	subtitleFunction          = "fc_setSubtitles"
)

var (
	ErrInvalidGFX       = errors.New("invalid GFX/SWF")
	ErrSubtitleTarget   = errors.New("subtitle function target not found")
	ErrAmbiguousTarget  = errors.New("ambiguous subtitle function target")
	ErrSemanticMismatch = errors.New("subtitle HUD semantic anchors do not match")
)

var requiredHUDAnchors = []string{
	"bc",
	"subtitles",
	"inside",
	"tField",
	"TextExtension",
	"setSubtitlesBackground",
}

type container struct {
	signature string
	version   byte
	body      []byte
}

type swfTag struct {
	code         uint16
	start        int
	end          int
	payloadStart int
	payloadEnd   int
}

type function2Info struct {
	name     string
	params   []string
	codeSize int
}

// PatchHUD derives a retail HUD override by inserting a narrow AVM1 wrapper
// immediately after the unique root DoAction that defines fc_setSubtitles.
// Existing project-patched input is returned unchanged so regeneration is
// idempotent. Unknown structures fail closed.
func PatchHUD(input []byte) ([]byte, error) {
	decoded, err := decodeContainer(input)
	if err != nil {
		return nil, err
	}
	if bytes.Contains(decoded.body, []byte(subtitlepayload.HUDWrapperMarker)) {
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

	wrapperActions, err := buildWrapperActions()
	if err != nil {
		return nil, fmt.Errorf("build subtitle wrapper: %w", err)
	}
	wrapperTag, err := encodeTag(tagDoAction, wrapperActions)
	if err != nil {
		return nil, fmt.Errorf("encode subtitle wrapper tag: %w", err)
	}

	patchedBody := make([]byte, 0, len(decoded.body)+len(wrapperTag))
	patchedBody = append(patchedBody, decoded.body[:target.end]...)
	patchedBody = append(patchedBody, wrapperTag...)
	patchedBody = append(patchedBody, decoded.body[target.end:]...)

	return encodeContainer(container{
		signature: decoded.signature,
		version:   decoded.version,
		body:      patchedBody,
	})
}

func decodeContainer(input []byte) (container, error) {
	if len(input) < 8 {
		return container{}, fmt.Errorf("%w: file is shorter than SWF header", ErrInvalidGFX)
	}
	signature := string(input[:3])
	if signature != "FWS" && signature != "GFX" && signature != "CWS" && signature != "CFX" {
		return container{}, fmt.Errorf("%w: unsupported signature %q", ErrInvalidGFX, signature)
	}
	declared := binary.LittleEndian.Uint32(input[4:8])
	if declared < 8 || declared > maxHUDUncompressed {
		return container{}, fmt.Errorf("%w: invalid declared size %d", ErrInvalidGFX, declared)
	}
	expectedBody := int(declared) - 8

	var body []byte
	if signature == "FWS" || signature == "GFX" {
		if len(input) != int(declared) {
			return container{}, fmt.Errorf("%w: uncompressed size=%d, declared=%d", ErrInvalidGFX, len(input), declared)
		}
		body = append([]byte(nil), input[8:]...)
	} else {
		reader, err := zlib.NewReader(bytes.NewReader(input[8:]))
		if err != nil {
			return container{}, fmt.Errorf("%w: open compressed body: %v", ErrInvalidGFX, err)
		}
		body, err = io.ReadAll(io.LimitReader(reader, int64(expectedBody)+1))
		closeErr := reader.Close()
		if err != nil {
			return container{}, fmt.Errorf("%w: read compressed body: %v", ErrInvalidGFX, err)
		}
		if closeErr != nil {
			return container{}, fmt.Errorf("%w: close compressed body: %v", ErrInvalidGFX, closeErr)
		}
		if len(body) != expectedBody {
			return container{}, fmt.Errorf("%w: inflated body=%d, want %d", ErrInvalidGFX, len(body), expectedBody)
		}
	}

	return container{signature: signature, version: input[3], body: body}, nil
}

func encodeContainer(value container) ([]byte, error) {
	declared := 8 + len(value.body)
	if declared > maxHUDUncompressed {
		return nil, fmt.Errorf("%w: patched HUD exceeds size limit", ErrInvalidGFX)
	}

	header := make([]byte, 8)
	copy(header[:3], value.signature)
	header[3] = value.version
	binary.LittleEndian.PutUint32(header[4:8], uint32(declared))

	if value.signature == "FWS" || value.signature == "GFX" {
		return append(header, value.body...), nil
	}

	var compressed bytes.Buffer
	writer := zlib.NewWriter(&compressed)
	if _, err := writer.Write(value.body); err != nil {
		_ = writer.Close()
		return nil, fmt.Errorf("compress patched GFX: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close patched GFX compressor: %w", err)
	}
	return append(header, compressed.Bytes()...), nil
}

func parseRootTags(body []byte) ([]swfTag, error) {
	if len(body) == 0 {
		return nil, fmt.Errorf("%w: missing RECT", ErrInvalidGFX)
	}
	nbits := int(body[0] >> 3)
	if nbits == 0 {
		return nil, fmt.Errorf("%w: RECT has zero coordinate bits", ErrInvalidGFX)
	}
	rectBytes := (5 + 4*nbits + 7) / 8
	tagsStart := rectBytes + 4 // FrameRate + FrameCount.
	if tagsStart > len(body) {
		return nil, fmt.Errorf("%w: truncated SWF frame header", ErrInvalidGFX)
	}

	var tags []swfTag
	pos := tagsStart
	foundEnd := false
	for pos < len(body) {
		start := pos
		if len(body)-pos < 2 {
			return nil, fmt.Errorf("%w: truncated tag header", ErrInvalidGFX)
		}
		header := binary.LittleEndian.Uint16(body[pos : pos+2])
		pos += 2
		code := header >> 6
		length := int(header & 0x3f)
		if length == 0x3f {
			if len(body)-pos < 4 {
				return nil, fmt.Errorf("%w: truncated long tag length", ErrInvalidGFX)
			}
			longLength := binary.LittleEndian.Uint32(body[pos : pos+4])
			pos += 4
			if longLength > uint32(len(body)-pos) {
				return nil, fmt.Errorf("%w: tag %d exceeds file boundary", ErrInvalidGFX, code)
			}
			length = int(longLength)
		}
		if length > len(body)-pos {
			return nil, fmt.Errorf("%w: tag %d payload is truncated", ErrInvalidGFX, code)
		}
		payloadStart := pos
		payloadEnd := pos + length
		pos = payloadEnd
		tags = append(tags, swfTag{
			code:         code,
			start:        start,
			end:          pos,
			payloadStart: payloadStart,
			payloadEnd:   payloadEnd,
		})
		if code == tagEnd {
			if length != 0 {
				return nil, fmt.Errorf("%w: End tag has payload", ErrInvalidGFX)
			}
			if pos != len(body) {
				return nil, fmt.Errorf("%w: trailing bytes after End tag", ErrInvalidGFX)
			}
			foundEnd = true
			break
		}
	}
	if !foundEnd {
		return nil, fmt.Errorf("%w: missing End tag", ErrInvalidGFX)
	}
	return tags, nil
}

func encodeTag(code uint16, payload []byte) ([]byte, error) {
	if code > 1023 {
		return nil, fmt.Errorf("tag code %d exceeds SWF limit", code)
	}
	var out []byte
	if len(payload) < 0x3f {
		header := (code << 6) | uint16(len(payload))
		out = make([]byte, 2, 2+len(payload))
		binary.LittleEndian.PutUint16(out, header)
	} else {
		if uint64(len(payload)) > uint64(^uint32(0)) {
			return nil, fmt.Errorf("tag payload is too large")
		}
		header := (code << 6) | 0x3f
		out = make([]byte, 6, 6+len(payload))
		binary.LittleEndian.PutUint16(out[:2], header)
		binary.LittleEndian.PutUint32(out[2:6], uint32(len(payload)))
	}
	return append(out, payload...), nil
}

func countSubtitleFunctions(actions []byte) (int, error) {
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
			if info.name == subtitleFunction && equalStrings(info.params, []string{"text", "speakerName", "isPlayer"}) {
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

func parseFunction2(data []byte) (function2Info, error) {
	pos := 0
	name, next, err := readCString(data, pos)
	if err != nil {
		return function2Info{}, fmt.Errorf("parse DefineFunction2 name: %w", err)
	}
	pos = next
	if len(data)-pos < 5 {
		return function2Info{}, fmt.Errorf("truncated DefineFunction2 header")
	}
	paramCount := int(binary.LittleEndian.Uint16(data[pos : pos+2]))
	pos += 2
	pos++    // RegisterCount.
	pos += 2 // Flags.

	params := make([]string, 0, paramCount)
	for range paramCount {
		if pos >= len(data) {
			return function2Info{}, fmt.Errorf("truncated DefineFunction2 parameter register")
		}
		pos++ // Register.
		param, next, err := readCString(data, pos)
		if err != nil {
			return function2Info{}, fmt.Errorf("parse DefineFunction2 parameter: %w", err)
		}
		pos = next
		params = append(params, param)
	}
	if len(data)-pos != 2 {
		return function2Info{}, fmt.Errorf("DefineFunction2 metadata has %d unexpected bytes", len(data)-pos)
	}
	codeSize := int(binary.LittleEndian.Uint16(data[pos : pos+2]))
	return function2Info{name: name, params: params, codeSize: codeSize}, nil
}

func defineFunctionCodeSize(data []byte) (int, error) {
	pos := 0
	_, next, err := readCString(data, pos)
	if err != nil {
		return 0, err
	}
	pos = next
	if len(data)-pos < 2 {
		return 0, fmt.Errorf("truncated DefineFunction parameter count")
	}
	paramCount := int(binary.LittleEndian.Uint16(data[pos : pos+2]))
	pos += 2
	for range paramCount {
		_, next, err := readCString(data, pos)
		if err != nil {
			return 0, err
		}
		pos = next
	}
	if len(data)-pos != 2 {
		return 0, fmt.Errorf("invalid DefineFunction metadata")
	}
	return int(binary.LittleEndian.Uint16(data[pos : pos+2])), nil
}

func tryCodeSize(data []byte) (int, error) {
	if len(data) < 7 {
		return 0, fmt.Errorf("truncated ActionTry metadata")
	}
	flags := data[0]
	trySize := int(binary.LittleEndian.Uint16(data[1:3]))
	catchSize := int(binary.LittleEndian.Uint16(data[3:5]))
	finallySize := int(binary.LittleEndian.Uint16(data[5:7]))
	pos := 7
	if flags&0x04 != 0 { // CatchInRegisterFlag.
		if pos >= len(data) {
			return 0, fmt.Errorf("truncated ActionTry catch register")
		}
		pos++
	} else {
		_, next, err := readCString(data, pos)
		if err != nil {
			return 0, fmt.Errorf("parse ActionTry catch name: %w", err)
		}
		pos = next
	}
	if pos != len(data) {
		return 0, fmt.Errorf("ActionTry metadata has %d unexpected bytes", len(data)-pos)
	}
	return trySize + catchSize + finallySize, nil
}

func readCString(data []byte, start int) (string, int, error) {
	if start < 0 || start > len(data) {
		return "", start, fmt.Errorf("invalid string offset")
	}
	index := bytes.IndexByte(data[start:], 0)
	if index < 0 {
		return "", start, fmt.Errorf("unterminated string")
	}
	return string(data[start : start+index]), start + index + 1, nil
}

func equalStrings(a, b []string) bool {
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

func buildWrapperActions() ([]byte, error) {
	var top []byte
	top = append(top, pushString(wrapperVersionVariable)...)
	top = append(top, pushString(subtitlepayload.HUDWrapperMarker)...)
	top = append(top, actionSetVariable)

	top = append(top, pushString(originalSubtitleFunction)...)
	top = append(top, pushString(subtitleFunction)...)
	top = append(top, actionGetVariable)
	top = append(top, actionSetVariable)

	body, err := buildWrapperBody()
	if err != nil {
		return nil, err
	}
	definition, err := defineFunction2(subtitleFunction, []functionParam{
		{register: 1, name: "text"},
		{register: 2, name: "speakerName"},
		{register: 3, name: "isPlayer"},
	}, 6, body)
	if err != nil {
		return nil, err
	}
	top = append(top, definition...)
	top = append(top, actionEnd)
	return top, nil
}

type functionParam struct {
	register byte
	name     string
}

func defineFunction2(name string, params []functionParam, registerCount byte, body []byte) ([]byte, error) {
	if len(params) > 0xffff || len(body) > 0xffff {
		return nil, fmt.Errorf("wrapper function exceeds AVM1 size limit")
	}
	var meta bytes.Buffer
	writeCString(&meta, name)
	_ = binary.Write(&meta, binary.LittleEndian, uint16(len(params)))
	_ = meta.WriteByte(registerCount)
	_ = binary.Write(&meta, binary.LittleEndian, uint16(0)) // DefineFunction2 flags.
	for _, param := range params {
		_ = meta.WriteByte(param.register)
		writeCString(&meta, param.name)
	}
	_ = binary.Write(&meta, binary.LittleEndian, uint16(len(body)))

	record, err := longAction(actionDefineFunction2, meta.Bytes())
	if err != nil {
		return nil, err
	}
	return append(record, body...), nil
}

func buildWrapperBody() ([]byte, error) {
	builder := newActionBuilder()

	// result = __kcd2ds_original_fc_setSubtitles(text, speakerName, isPlayer)
	builder.pushRegister(3)
	builder.pushRegister(2)
	builder.pushRegister(1)
	builder.pushInt(3)
	builder.pushString(originalSubtitleFunction)
	builder.simple(actionCallFunction)
	builder.storeRegister(4)
	builder.simple(actionPop)

	// if (text.indexOf(Prefix) != 0) goto returnResult
	builder.pushString(subtitlepayload.Prefix)
	builder.pushInt(1)
	builder.pushRegister(1)
	builder.pushString("indexOf")
	builder.simple(actionCallMethod)
	builder.pushInt(0)
	builder.simple(actionEquals2)
	builder.simple(actionNot)
	builder.ifTrue("returnResult")

	// end = text.indexOf(Suffix); malformed payloads keep vanilla output.
	builder.pushString(subtitlepayload.Suffix)
	builder.pushInt(1)
	builder.pushRegister(1)
	builder.pushString("indexOf")
	builder.simple(actionCallMethod)
	builder.storeRegister(5)
	builder.simple(actionPop)
	builder.pushRegister(5)
	builder.pushInt(-1)
	builder.simple(actionEquals2)
	builder.ifTrue("returnResult")

	// secondary = text.substring(len(Prefix), end)
	builder.pushRegister(5) // Right-most argument first.
	builder.pushInt(int32(len(subtitlepayload.Prefix)))
	builder.pushInt(2)
	builder.pushRegister(1)
	builder.pushString("substring")
	builder.simple(actionCallMethod)
	builder.storeRegister(6)
	builder.simple(actionPop)

	// bc.subtitles.inside.tField.appendHtml(styled secondary)
	builder.pushString("<br/><font color='" + subtitlepayload.SecondaryColor + "' size='" + strconv.Itoa(subtitlepayload.SecondarySize) + "'><i>")
	builder.pushRegister(6)
	builder.simple(actionStringAdd)
	builder.pushString("</i></font>")
	builder.simple(actionStringAdd)
	builder.pushInt(1)
	builder.pushString("bc")
	builder.simple(actionGetVariable)
	for _, member := range []string{"subtitles", "inside", "tField"} {
		builder.pushString(member)
		builder.simple(actionGetMember)
	}
	builder.pushString("appendHtml")
	builder.simple(actionCallMethod)
	builder.simple(actionPop)

	// Reuse the vanilla background measurement after the final htmlText change.
	builder.pushInt(0)
	builder.pushString("setSubtitlesBackground")
	builder.simple(actionCallFunction)
	builder.simple(actionPop)

	builder.label("returnResult")
	builder.pushRegister(4)
	builder.simple(actionReturn)
	builder.simple(actionEnd)
	return builder.finish()
}

func pushString(value string) []byte {
	payload := append([]byte{0}, []byte(value)...)
	payload = append(payload, 0)
	record, err := longAction(actionPush, payload)
	if err != nil {
		panic(err) // Constant project strings are bounded by construction.
	}
	return record
}

func longAction(code byte, payload []byte) ([]byte, error) {
	if len(payload) > 0xffff {
		return nil, fmt.Errorf("action 0x%02x payload exceeds AVM1 limit", code)
	}
	out := make([]byte, 3, 3+len(payload))
	out[0] = code
	binary.LittleEndian.PutUint16(out[1:3], uint16(len(payload)))
	return append(out, payload...), nil
}

func writeCString(buffer *bytes.Buffer, value string) {
	_, _ = buffer.WriteString(value)
	_ = buffer.WriteByte(0)
}

type branchFixup struct {
	offsetPos int
	base      int
	label     string
}

type actionBuilder struct {
	data   []byte
	labels map[string]int
	fixups []branchFixup
}

func newActionBuilder() *actionBuilder {
	return &actionBuilder{labels: make(map[string]int)}
}

func (b *actionBuilder) simple(code byte) {
	b.data = append(b.data, code)
}

func (b *actionBuilder) pushString(value string) {
	b.data = append(b.data, pushString(value)...)
}

func (b *actionBuilder) pushInt(value int32) {
	payload := make([]byte, 5)
	payload[0] = 7 // AVM1 Push integer.
	binary.LittleEndian.PutUint32(payload[1:], uint32(value))
	record, err := longAction(actionPush, payload)
	if err != nil {
		panic(err)
	}
	b.data = append(b.data, record...)
}

func (b *actionBuilder) pushRegister(register byte) {
	payload := []byte{4, register} // AVM1 Push register.
	record, err := longAction(actionPush, payload)
	if err != nil {
		panic(err)
	}
	b.data = append(b.data, record...)
}

func (b *actionBuilder) storeRegister(register byte) {
	record, err := longAction(actionStoreRegister, []byte{register})
	if err != nil {
		panic(err)
	}
	b.data = append(b.data, record...)
}

func (b *actionBuilder) ifTrue(label string) {
	start := len(b.data)
	b.data = append(b.data, actionIf, 2, 0, 0, 0)
	b.fixups = append(b.fixups, branchFixup{
		offsetPos: start + 3,
		base:      start + 5,
		label:     label,
	})
}

func (b *actionBuilder) label(name string) {
	if _, exists := b.labels[name]; exists {
		panic("duplicate action label " + name)
	}
	b.labels[name] = len(b.data)
}

func (b *actionBuilder) finish() ([]byte, error) {
	out := append([]byte(nil), b.data...)
	for _, fixup := range b.fixups {
		target, ok := b.labels[fixup.label]
		if !ok {
			return nil, fmt.Errorf("unknown action label %q", fixup.label)
		}
		delta := target - fixup.base
		if delta < -32768 || delta > 32767 {
			return nil, fmt.Errorf("action branch to %q exceeds SI16 range", fixup.label)
		}
		binary.LittleEndian.PutUint16(out[fixup.offsetPos:fixup.offsetPos+2], uint16(int16(delta)))
	}
	return out, nil
}
