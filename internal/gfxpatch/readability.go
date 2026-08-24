package gfxpatch

// HUDReadabilityConfig controls optional whole-TextField readability effects.
// The effects intentionally apply to the complete bilingual subtitle block,
// because the retail HUD renders both generated lines through one TextField.
// Zero values preserve the retail/default TextField presentation.
type HUDReadabilityConfig struct {
	Outline bool
	Shadow  bool
}

const (
	hudOutlineWidth  = 1
	hudShadowColor   = 0x000000
	hudShadowAlpha   = 1
	hudShadowBlur    = 2
	hudShadowAngle   = 45
	hudShadowDistance = 1
	hudShadowQuality = 1
	hudShadowStrength = 1
)

func appendTextFieldReadability(builder *actionBuilder, pushTextField func(), config HUDReadabilityConfig) {
	if config.Outline {
		setTextFieldIntProperty(builder, pushTextField, "outline", hudOutlineWidth)
	}
	if !config.Shadow {
		return
	}

	for _, property := range []struct {
		name  string
		value int32
	}{
		{name: "shadowColor", value: hudShadowColor},
		{name: "shadowAlpha", value: hudShadowAlpha},
		{name: "shadowBlurX", value: hudShadowBlur},
		{name: "shadowBlurY", value: hudShadowBlur},
		{name: "shadowAngle", value: hudShadowAngle},
		{name: "shadowDistance", value: hudShadowDistance},
		{name: "shadowQuality", value: hudShadowQuality},
		{name: "shadowStrength", value: hudShadowStrength},
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
