package gfxpatch

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestAppendTextFieldReadabilityEmitsConfiguredShadowColor(t *testing.T) {
	builder := newActionBuilder()
	appendTextFieldReadability(builder, func() {
		builder.pushString("testField")
	}, HUDReadabilityConfig{
		Shadow:      true,
		ShadowColor: 0x123456,
	})
	data, err := builder.finish()
	if err != nil {
		t.Fatalf("finish readability actions: %v", err)
	}

	payload := make([]byte, 5)
	payload[0] = 7 // AVM1 Push integer.
	binary.LittleEndian.PutUint32(payload[1:], 0x123456)
	want, err := longAction(actionPush, payload)
	if err != nil {
		t.Fatalf("build expected integer push: %v", err)
	}
	if got := bytes.Count(data, want); got != 1 {
		t.Fatalf("configured shadow color integer pushes = %d, want 1", got)
	}
}
