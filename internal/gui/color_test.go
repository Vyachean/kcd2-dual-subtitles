package gui

import "testing"

func TestParseHexRGB(t *testing.T) {
	red, green, blue, ok := parseHexRGB("  #7fDBff  ")
	if !ok || red != 0x7f || green != 0xdb || blue != 0xff {
		t.Fatalf("parseHexRGB() = %02X %02X %02X, %v", red, green, blue, ok)
	}
}

func TestParseHexRGBRejectsInvalid(t *testing.T) {
	for _, value := range []string{"", "7FDBFF", "#FFF", "#12GG34", "#12345678"} {
		if _, _, _, ok := parseHexRGB(value); ok {
			t.Fatalf("parseHexRGB(%q) unexpectedly succeeded", value)
		}
	}
}

func TestFormatHexRGBUsesCanonicalUppercase(t *testing.T) {
	if got := formatHexRGB(0x0a, 0xb1, 0xff); got != "#0AB1FF" {
		t.Fatalf("formatHexRGB() = %q", got)
	}
}
