package generator

import (
	"fmt"
	"strings"
)

// SubtitleStyle controls how a generated bilingual row is presented by the
// game's existing subtitle TextField. Tagged is the accepted v0.1/v0.2 format;
// differentiated is an experimental vanilla-HUD HTML style used for v0.3
// research acceptance.
type SubtitleStyle string

const (
	SubtitleStyleTagged         SubtitleStyle = "tagged"
	SubtitleStyleDifferentiated SubtitleStyle = "differentiated"
)

// ParseSubtitleStyle resolves a user-facing style name case-insensitively.
func ParseSubtitleStyle(value string) (SubtitleStyle, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", string(SubtitleStyleTagged):
		return SubtitleStyleTagged, true
	case string(SubtitleStyleDifferentiated):
		return SubtitleStyleDifferentiated, true
	default:
		return "", false
	}
}

func normalizeSubtitleStyle(style SubtitleStyle) (SubtitleStyle, error) {
	parsed, ok := ParseSubtitleStyle(string(style))
	if !ok {
		return "", fmt.Errorf("%w: unsupported subtitle style %q", ErrInvalidRequest, style)
	}
	return parsed, nil
}
