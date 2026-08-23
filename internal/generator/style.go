package generator

import (
	"fmt"
	"strings"
)

// SubtitleStyle controls the generated subtitle presentation contract. Tagged
// remains the accepted default. Differentiated is the failed localization-only
// Stage A experiment retained for reproducibility; HUD is the explicit direct-
// HUD Stage C1 prototype.
type SubtitleStyle string

const (
	SubtitleStyleTagged         SubtitleStyle = "tagged"
	SubtitleStyleDifferentiated SubtitleStyle = "differentiated"
	SubtitleStyleHUD            SubtitleStyle = "hud"
)

// ParseSubtitleStyle resolves a user-facing style name case-insensitively.
func ParseSubtitleStyle(value string) (SubtitleStyle, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", string(SubtitleStyleTagged):
		return SubtitleStyleTagged, true
	case string(SubtitleStyleDifferentiated):
		return SubtitleStyleDifferentiated, true
	case string(SubtitleStyleHUD):
		return SubtitleStyleHUD, true
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
