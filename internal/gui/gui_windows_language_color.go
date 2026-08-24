//go:build windows

package gui

import (
	"fmt"
	"syscall"
	"unsafe"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
)

const (
	idColorPickerButton = 1005
	cbResetContent      = 0x014B
	ccRGBInit           = 0x00000001
	ccFullOpen          = 0x00000002
)

var (
	guiComdlg32       = syscall.NewLazyDLL("comdlg32.dll")
	procChooseColorW  = guiComdlg32.NewProc("ChooseColorW")
)

type chooseColor struct {
	StructSize   uint32
	Owner        uintptr
	Instance     uintptr
	RGBResult    uint32
	CustomColors *uint32
	Flags        uint32
	CustomData   uintptr
	Hook         uintptr
	TemplateName *uint16
}

func (w *nativeWindow) refreshLanguageControls(gameRoot string) error {
	previousMain, _ := w.selectedLanguage(w.mainCombo)
	previousSecondary, _ := w.selectedLanguage(w.secondaryCombo)

	languages, err := localization.InstalledLanguages(gameRoot)
	if err != nil {
		return err
	}
	w.languages = languages

	procSendMessageW.Call(w.mainCombo, cbResetContent, 0, 0)
	procSendMessageW.Call(w.secondaryCombo, cbResetContent, 0, 0)
	for _, info := range languages {
		text := mustUTF16(string(info.Language))
		procSendMessageW.Call(w.mainCombo, cbAddString, 0, uintptr(unsafe.Pointer(text)))
		procSendMessageW.Call(w.secondaryCombo, cbAddString, 0, uintptr(unsafe.Pointer(text)))
	}

	mainIndex, secondaryIndex := preferredLanguageIndexes(languages, previousMain, previousSecondary)
	if mainIndex >= 0 {
		procSendMessageW.Call(w.mainCombo, cbSetCurSel, uintptr(mainIndex), 0)
	}
	if secondaryIndex >= 0 {
		procSendMessageW.Call(w.secondaryCombo, cbSetCurSel, uintptr(secondaryIndex), 0)
	}
	if w.generateButton != 0 {
		w.enable(w.generateButton, !w.busy && len(languages) >= 2)
	}
	return nil
}

func (w *nativeWindow) chooseSecondaryColor() {
	red, green, blue, ok := parseHexRGB(w.text(w.colorEdit))
	if !ok {
		red, green, blue, _ = parseHexRGB(defaultPresentationInput().SecondaryColor)
	}

	picker := chooseColor{
		StructSize:   uint32(unsafe.Sizeof(chooseColor{})),
		Owner:        w.hwnd,
		RGBResult:    rgbToColorRef(red, green, blue),
		CustomColors: &w.customColors[0],
		Flags:        ccRGBInit | ccFullOpen,
	}
	chosen, _, _ := procChooseColorW.Call(uintptr(unsafe.Pointer(&picker)))
	if chosen == 0 {
		return
	}

	red, green, blue = colorRefToRGB(picker.RGBResult)
	value := formatHexRGB(red, green, blue)
	w.setText(w.colorEdit, value)
	w.presentation.SecondaryColor = value
}

func (w *nativeWindow) requireAtLeastTwoInstalledLanguages() error {
	if len(w.languages) >= 2 {
		return nil
	}
	return fmt.Errorf("the selected KCD2 installation has only %d supported localization PAK(s); at least two are required", len(w.languages))
}
