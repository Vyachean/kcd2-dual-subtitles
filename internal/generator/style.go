package generator

import (
	"fmt"
	"strings"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/subtitlepayload"
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

	DefaultHUDSecondaryColor = subtitlepayload.SecondaryColor
	DefaultHUDSecondarySize  = subtitlepayload.SecondarySize
	MinHUDSecondarySize      = 12
	MaxHUDSecondarySize      = 48
	MinHUDPrimarySize        = MinHUDSecondarySize
	MaxHUDPrimarySize        = MaxHUDSecondarySize
)

// HUDPresentationConfig controls generation-time presentation for the proven
// direct-HTML HUD path. Empty PrimaryColor and zero PrimarySize mean that the
// corresponding primary-line property remains controlled by the retail game.
// Callers that want to customize secondary boolean fields should start from
// DefaultHUDPresentationConfig so false values remain explicit.
type HUDPresentationConfig struct {
	PrimaryColor     string
	PrimarySize      int
	PrimaryItalic    bool
	SecondaryColor   string
	SecondarySize    int
	SecondaryItalic  bool
	ShowLanguageTags bool
}

// DefaultHUDPresentationConfig returns the live-proven rc.10 presentation and
// leaves the primary line entirely under vanilla styling.
func DefaultHUDPresentationConfig() HUDPresentationConfig {
	return HUDPresentationConfig{
		SecondaryColor:   DefaultHUDSecondaryColor,
		SecondarySize:    DefaultHUDSecondarySize,
		SecondaryItalic:  true,
		ShowLanguageTags: true,
	}
}

// NormalizeHUDPresentationConfig validates and normalizes an explicit HUD
// presentation configuration without performing generation. UI callers use
// this to fail before touching game localization or HUD assets.
func NormalizeHUDPresentationConfig(config HUDPresentationConfig) (HUDPresentationConfig, error) {
	return normalizeHUDPresentation(SubtitleStyleHUD, &config)
}

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

func normalizeHUDPresentation(style SubtitleStyle, configured *HUDPresentationConfig) (HUDPresentationConfig, error) {
	if style != SubtitleStyleHUD {
		if configured != nil {
			return HUDPresentationConfig{}, fmt.Errorf("%w: HUD presentation options require subtitle style %q", ErrInvalidRequest, SubtitleStyleHUD)
		}
		return HUDPresentationConfig{}, nil
	}

	if configured == nil {
		return DefaultHUDPresentationConfig(), nil
	}

	presentation := *configured
	presentation.PrimaryColor = strings.TrimSpace(presentation.PrimaryColor)
	if presentation.PrimaryColor != "" && !validHUDColor(presentation.PrimaryColor) {
		return HUDPresentationConfig{}, fmt.Errorf("%w: primary color must be empty or use #RRGGBB format", ErrInvalidRequest)
	}
	if presentation.PrimarySize != 0 && (presentation.PrimarySize < MinHUDPrimarySize || presentation.PrimarySize > MaxHUDPrimarySize) {
		return HUDPresentationConfig{}, fmt.Errorf("%w: primary size must be 0 (vanilla) or between %d and %d", ErrInvalidRequest, MinHUDPrimarySize, MaxHUDPrimarySize)
	}

	presentation.SecondaryColor = strings.TrimSpace(presentation.SecondaryColor)
	if !validHUDColor(presentation.SecondaryColor) {
		return HUDPresentationConfig{}, fmt.Errorf("%w: secondary color must use #RRGGBB format", ErrInvalidRequest)
	}
	if presentation.SecondarySize < MinHUDSecondarySize || presentation.SecondarySize > MaxHUDSecondarySize {
		return HUDPresentationConfig{}, fmt.Errorf("%w: secondary size must be between %d and %d", ErrInvalidRequest, MinHUDSecondarySize, MaxHUDSecondarySize)
	}
	return presentation, nil
}

func validHUDColor(value string) bool {
	if len(value) != 7 || value[0] != '#' {
		return false
	}
	for _, r := range value[1:] {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}
