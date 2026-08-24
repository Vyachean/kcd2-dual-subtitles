package generator

import "github.com/Vyachean/kcd2-dual-subtitles/internal/gfxpatch"

type hudShadowGeometry struct {
	blur     int32
	distance int32
	strength int32
}

func shadowGeometry(intensity HUDShadowIntensity) hudShadowGeometry {
	switch intensity {
	case HUDShadowSubtle:
		return hudShadowGeometry{blur: 1, distance: 1, strength: 1}
	case HUDShadowStrong:
		return hudShadowGeometry{blur: 3, distance: 2, strength: 2}
	case HUDShadowNormal:
		fallthrough
	default:
		// Exact v0.3 accepted shadow values.
		return hudShadowGeometry{blur: 2, distance: 1, strength: 1}
	}
}

func readabilityConfig(presentation HUDPresentationConfig, shadowColor int32) gfxpatch.HUDReadabilityConfig {
	geometry := shadowGeometry(presentation.ShadowIntensity)
	return gfxpatch.HUDReadabilityConfig{
		Outline:        presentation.Outline,
		Shadow:         presentation.Shadow,
		ShadowColor:    shadowColor,
		ShadowBlur:     geometry.blur,
		ShadowDistance: geometry.distance,
		ShadowStrength: geometry.strength,
	}
}
