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
