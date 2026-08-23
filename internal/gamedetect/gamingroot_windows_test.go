//go:build windows

package gamedetect

import (
	"encoding/binary"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
	"unicode/utf16"
)

func TestParseGamingRootResolvesRelativeAndAbsoluteLocations(t *testing.T) {
	data := gamingRootFixture(`\XboxGames`, `C:\CustomXbox`)
	got, err := parseGamingRoot(data, `C:\`)
	if err != nil {
		t.Fatalf("parseGamingRoot() error = %v", err)
	}
	want := []string{filepath.Clean(`C:\XboxGames`), filepath.Clean(`C:\CustomXbox`)}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("roots = %#v, want %#v", got, want)
	}
}

func TestParseGamingRootRejectsMalformedData(t *testing.T) {
	for _, data := range [][]byte{
		nil,
		[]byte("nope"),
		append([]byte("RGBX"), 0, 0, 0, 0, 0, 0),
	} {
		if _, err := parseGamingRoot(data, `C:\`); !errors.Is(err, errInvalidGamingRoot) {
			t.Fatalf("parseGamingRoot(%v) error = %v, want errInvalidGamingRoot", data, err)
		}
	}
}

func TestXboxRootsForDrivesKeepsDefaultWhenMarkerInvalid(t *testing.T) {
	drive := t.TempDir()
	got := xboxRootsForDrives([]string{drive})
	want := []string{filepath.Join(drive, "XboxGames")}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("roots = %#v, want %#v", got, want)
	}
}

func gamingRootFixture(locations ...string) []byte {
	data := []byte{'R', 'G', 'B', 'X', 0, 0, 0, 0}
	binary.LittleEndian.PutUint32(data[4:8], uint32(len(locations)))
	for _, location := range locations {
		for _, unit := range utf16.Encode([]rune(location)) {
			var encoded [2]byte
			binary.LittleEndian.PutUint16(encoded[:], unit)
			data = append(data, encoded[:]...)
		}
		data = append(data, 0, 0)
	}
	return data
}
