package gui

import (
	"fmt"
	"strconv"
	"strings"
)

func parseHexRGB(value string) (red, green, blue byte, ok bool) {
	value = strings.TrimSpace(value)
	if len(value) != 7 || value[0] != '#' {
		return 0, 0, 0, false
	}
	parsed, err := strconv.ParseUint(value[1:], 16, 24)
	if err != nil {
		return 0, 0, 0, false
	}
	return byte(parsed >> 16), byte(parsed >> 8), byte(parsed), true
}

func formatHexRGB(red, green, blue byte) string {
	return fmt.Sprintf("#%02X%02X%02X", red, green, blue)
}

// Win32 COLORREF stores bytes as 0x00BBGGRR rather than RGB order.
func rgbToColorRef(red, green, blue byte) uint32 {
	return uint32(red) | uint32(green)<<8 | uint32(blue)<<16
}

func colorRefToRGB(value uint32) (red, green, blue byte) {
	return byte(value), byte(value >> 8), byte(value >> 16)
}
