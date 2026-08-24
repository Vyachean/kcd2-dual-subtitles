//go:build windows

package gui

import (
	"unsafe"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/generator"
)

type shadowIntensityOption struct {
	label string
	value generator.HUDShadowIntensity
}

var shadowIntensityOptions = []shadowIntensityOption{
	{label: "Subtle", value: generator.HUDShadowSubtle},
	{label: "Normal", value: generator.HUDShadowNormal},
	{label: "Strong", value: generator.HUDShadowStrong},
}

func (w *nativeWindow) initializeShadowIntensityCombo() {
	selectedIndex := 0
	for index, option := range shadowIntensityOptions {
		text := mustUTF16(option.label)
		procSendMessageW.Call(w.shadowIntensityCombo, cbAddString, 0, uintptr(unsafe.Pointer(text)))
		if option.value == w.presentation.ShadowIntensity {
			selectedIndex = index
		}
	}
	procSendMessageW.Call(w.shadowIntensityCombo, cbSetCurSel, uintptr(selectedIndex), 0)
}

func (w *nativeWindow) selectedShadowIntensity() generator.HUDShadowIntensity {
	if w.shadowIntensityCombo == 0 {
		return generator.DefaultHUDShadowIntensity
	}
	selected, _, _ := procSendMessageW.Call(w.shadowIntensityCombo, cbGetCurSel, 0, 0)
	index := int(selected)
	if index < 0 || index >= len(shadowIntensityOptions) {
		return generator.DefaultHUDShadowIntensity
	}
	return shadowIntensityOptions[index].value
}
