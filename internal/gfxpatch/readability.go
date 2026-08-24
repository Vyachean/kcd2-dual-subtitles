package gfxpatch

// HUDReadabilityConfig controls optional whole-TextField readability effects.
// The effects intentionally apply to the complete bilingual subtitle block,
// because the retail HUD renders both generated lines through one TextField.
// Zero ShadowColor is black. Zero geometry values preserve the accepted Normal
// shadow defaults so existing callers remain behavior-compatible.
type HUDReadabilityConfig struct {
	Outline        bool
	Shadow         bool
	ShadowColor    int32
	ShadowBlur     int32
	ShadowDistance int32
	ShadowStrength int32
}

const (
	hudOutlineWidth         = 1
	hudShadowAlpha          = 1
	hudShadowNormalBlur     = 2
	hudShadowAngle          = 45
	hudShadowNormalDistance = 1
	hudShadowQuality        = 1
	hudShadowNormalStrength = 1
)

func appendTextFieldReadability(builder *actionBuilder, pushTextField func(), config HUDReadabilityConfig) {
	if config.Outline {
		setTextFieldIntProperty(builder, pushTextField, "outline", hudOutlineWidth)
	}
	if !config.Shadow {
		return
	}

	blur := config.ShadowBlur
	if blur == 0 {
		blur = hudShadowNormalBlur
	}
	distance := config.ShadowDistance
	if distance == 0 {
		distance = hudShadowNormalDistance
	}
	strength := config.ShadowStrength
	if strength == 0 {
		strength = hudShadowNormalStrength
	}

	for _, property := range []struct {
		name  string
		value int32
	}{
		{name: "shadowColor", value: config.ShadowColor},
		{name: "shadowAlpha", value: hudShadowAlpha},
		{name: "shadowBlurX", value: blur},
		{name: "shadowBlurY", value: blur},
		{name: "shadowAngle", value: hudShadowAngle},
		{name: "shadowDistance", value: distance},
		{name: "shadowQuality", value: hudShadowQuality},
		{name: "shadowStrength", value: strength},
	} {
		setTextFieldIntProperty(builder, pushTextField, property.name, property.value)
	}
}

func setTextFieldIntProperty(builder *actionBuilder, pushTextField func(), name string, value int32) {
	pushTextField()
	builder.pushString(name)
	builder.pushInt(value)
	builder.simple(actionSetMember)
}
