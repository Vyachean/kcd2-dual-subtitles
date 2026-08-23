package modarchive

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
)

func TestLocalizationPakRawHeadersAreCryPakSafe(t *testing.T) {
	rows := []localization.DialogueRow{{ID: "id", Source: "source", Text: "Русский\\nEnglish"}}
	pak, err := buildLocalizationPAK(rows)
	if err != nil {
		t.Fatalf("buildLocalizationPAK() error = %v", err)
	}

	const (
		localSignature   = 0x04034b50
		centralSignature = 0x02014b50
	)
	if len(pak) < 46 {
		t.Fatalf("PAK too short: %d bytes", len(pak))
	}
	if got := binary.LittleEndian.Uint32(pak[0:4]); got != localSignature {
		t.Fatalf("local header signature = %#x, want %#x", got, localSignature)
	}

	localFlags := binary.LittleEndian.Uint16(pak[6:8])
	localMethod := binary.LittleEndian.Uint16(pak[8:10])
	localCRC := binary.LittleEndian.Uint32(pak[14:18])
	localCompressed := binary.LittleEndian.Uint32(pak[18:22])
	localUncompressed := binary.LittleEndian.Uint32(pak[22:26])
	localNameLen := int(binary.LittleEndian.Uint16(pak[26:28]))
	localExtraLen := int(binary.LittleEndian.Uint16(pak[28:30]))
	localNameEnd := 30 + localNameLen
	if localNameEnd+localExtraLen > len(pak) {
		t.Fatal("local header lengths exceed PAK size")
	}
	localName := string(pak[30:localNameEnd])

	centralOffset := bytes.Index(pak, []byte{'P', 'K', 0x01, 0x02})
	if centralOffset < 0 || centralOffset+46 > len(pak) {
		t.Fatal("central directory header not found")
	}
	central := pak[centralOffset:]
	if got := binary.LittleEndian.Uint32(central[0:4]); got != centralSignature {
		t.Fatalf("central signature = %#x, want %#x", got, centralSignature)
	}
	centralFlags := binary.LittleEndian.Uint16(central[8:10])
	centralMethod := binary.LittleEndian.Uint16(central[10:12])
	centralCRC := binary.LittleEndian.Uint32(central[16:20])
	centralCompressed := binary.LittleEndian.Uint32(central[20:24])
	centralUncompressed := binary.LittleEndian.Uint32(central[24:28])
	centralNameLen := int(binary.LittleEndian.Uint16(central[28:30]))
	centralExtraLen := int(binary.LittleEndian.Uint16(central[30:32]))
	centralNameEnd := 46 + centralNameLen
	if centralNameEnd+centralExtraLen > len(central) {
		t.Fatal("central header lengths exceed PAK size")
	}
	centralName := string(central[46:centralNameEnd])

	if localName != LocalizationPatchArchivePath || centralName != LocalizationPatchArchivePath {
		t.Fatalf("entry names local=%q central=%q, want %q", localName, centralName, LocalizationPatchArchivePath)
	}
	if localFlags&0x8 != 0 || centralFlags&0x8 != 0 {
		t.Fatalf("data-descriptor flag set: local=%#x central=%#x", localFlags, centralFlags)
	}
	if localMethod != 0 || centralMethod != 0 {
		t.Fatalf("compression method local=%d central=%d, want Store (0)", localMethod, centralMethod)
	}
	if localExtraLen != 0 || centralExtraLen != 0 {
		t.Fatalf("extra lengths local=%d central=%d, want 0/0", localExtraLen, centralExtraLen)
	}
	if localCRC == 0 || localCRC != centralCRC {
		t.Fatalf("CRC local=%#x central=%#x, want populated and equal", localCRC, centralCRC)
	}
	if localCompressed == 0 || localCompressed != centralCompressed || localCompressed != localUncompressed || centralCompressed != centralUncompressed {
		t.Fatalf("sizes local=%d/%d central=%d/%d, want populated Store sizes that agree", localCompressed, localUncompressed, centralCompressed, centralUncompressed)
	}

	dataStart := localNameEnd + localExtraLen
	dataEnd := dataStart + int(localCompressed)
	if dataEnd != centralOffset {
		t.Fatalf("local file data ends at %d, central directory starts at %d; unexpected descriptor/gap", dataEnd, centralOffset)
	}
	if got := crc32.ChecksumIEEE(pak[dataStart:dataEnd]); got != localCRC {
		t.Fatalf("raw entry CRC = %#x, header CRC = %#x", got, localCRC)
	}
}
