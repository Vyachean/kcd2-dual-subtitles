//go:build windows

package gui

import (
	"fmt"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/application"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
)

const (
	windowClassName = "KCD2DualSubtitlesWindow"

	wsCaption       = 0x00C00000
	wsSysMenu       = 0x00080000
	wsMinimizeBox   = 0x00020000
	wsChild         = 0x40000000
	wsVisible       = 0x10000000
	wsTabStop       = 0x00010000
	wsBorder        = 0x00800000
	wsVScroll       = 0x00200000
	esAutoHScroll   = 0x0080
	cbsDropdownList = 0x0003
	bsAutoCheckbox  = 0x0003
	bsGroupBox      = 0x0007

	wmCreate  = 0x0001
	wmDestroy = 0x0002
	wmCommand = 0x0111
	wmSetFont = 0x0030

	cbAddString = 0x0143
	cbGetCurSel = 0x0147
	cbSetCurSel = 0x014E
	bmGetCheck  = 0x00F0
	bmSetCheck  = 0x00F1

	bstUnchecked = 0
	bstChecked   = 1

	swShow = 5

	colorWindow    = 5
	idcArrow       = 32512
	defaultGUIFont = 17

	mbOK           = 0x00000000
	mbYesNo        = 0x00000004
	mbIconError    = 0x00000010
	mbIconQuestion = 0x00000020
	idYes          = 6

	bifReturnOnlyFSDirs = 0x0001
	bifEditBox          = 0x0010
	bifNewDialogStyle   = 0x0040

	coinitApartmentThreaded = 0x2

	idBrowseButton    = 1001
	idGenerateButton  = 1002
	idUninstallButton = 1003
	idStyledCheckbox  = 1004
)

var (
	guiUser32   = syscall.NewLazyDLL("user32.dll")
	guiKernel32 = syscall.NewLazyDLL("kernel32.dll")
	guiGDI32    = syscall.NewLazyDLL("gdi32.dll")
	guiShell32  = syscall.NewLazyDLL("shell32.dll")
	guiOle32    = syscall.NewLazyDLL("ole32.dll")

	procRegisterClassW       = guiUser32.NewProc("RegisterClassW")
	procCreateWindowExW      = guiUser32.NewProc("CreateWindowExW")
	procDefWindowProcW       = guiUser32.NewProc("DefWindowProcW")
	procShowWindowGUI        = guiUser32.NewProc("ShowWindow")
	procUpdateWindowGUI      = guiUser32.NewProc("UpdateWindow")
	procGetMessageW          = guiUser32.NewProc("GetMessageW")
	procTranslateMessage     = guiUser32.NewProc("TranslateMessage")
	procDispatchMessageW     = guiUser32.NewProc("DispatchMessageW")
	procPostQuitMessage      = guiUser32.NewProc("PostQuitMessage")
	procSendMessageW         = guiUser32.NewProc("SendMessageW")
	procSetWindowTextW       = guiUser32.NewProc("SetWindowTextW")
	procGetWindowTextLengthW = guiUser32.NewProc("GetWindowTextLengthW")
	procGetWindowTextW       = guiUser32.NewProc("GetWindowTextW")
	procEnableWindow         = guiUser32.NewProc("EnableWindow")
	procMessageBoxW          = guiUser32.NewProc("MessageBoxW")
	procLoadCursorW          = guiUser32.NewProc("LoadCursorW")

	procGetModuleHandleW = guiKernel32.NewProc("GetModuleHandleW")
	procGetStockObject   = guiGDI32.NewProc("GetStockObject")

	procSHBrowseForFolderW   = guiShell32.NewProc("SHBrowseForFolderW")
	procSHGetPathFromIDListW = guiShell32.NewProc("SHGetPathFromIDListW")
	procCoInitializeEx       = guiOle32.NewProc("CoInitializeEx")
	procCoUninitialize       = guiOle32.NewProc("CoUninitialize")
	procCoTaskMemFree        = guiOle32.NewProc("CoTaskMemFree")
)

type wndClass struct {
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
}

type point struct {
	X int32
	Y int32
}

type message struct {
	HWnd     uintptr
	Message  uint32
	WParam   uintptr
	LParam   uintptr
	Time     uint32
	Pt       point
	LPrivate uint32
}

type browseInfo struct {
	Owner       uintptr
	Root        uintptr
	DisplayName *uint16
	Title       *uint16
	Flags       uint32
	Callback    uintptr
	LParam      uintptr
	Image       int32
}

type nativeWindow struct {
	service      application.Service
	model        Model
	version      string
	presentation presentationInput
	busy         bool

	hwnd                     uintptr
	gameEdit                 uintptr
	mainCombo                uintptr
	secondaryCombo           uintptr
	styledCheckbox           uintptr
	tagsCheckbox             uintptr
	outlineCheckbox          uintptr
	shadowCheckbox           uintptr
	shadowColorEdit          uintptr
	shadowColorPickerButton  uintptr
	primaryColorEdit         uintptr
	primaryColorPickerButton uintptr
	primarySizeEdit          uintptr
	primaryItalicCheckbox    uintptr
	colorEdit                uintptr
	colorPickerButton        uintptr
	sizeEdit                 uintptr
	italicCheckbox           uintptr
	generateButton           uintptr
	uninstallButton          uintptr
	statusLabel              uintptr
	font                     uintptr
	startupErr               error
	languages                []localization.LanguageInfo
	customColors             [16]uint32
}

var activeWindow *nativeWindow

// Run starts the single-window native UI. The application core remains shared
// with the CLI; this function is only a thin Win32 presentation layer.
func Run(version string) int {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	service := application.New(version)
	detection, detectionErr := service.DetectGame()
	installation, installationErr := service.InspectInstallation()
	model := NewModel(detection, detectionErr, installation, installationErr)

	window := &nativeWindow{
		service:      service,
		model:        model,
		version:      version,
		presentation: defaultPresentationInput(),
	}
	activeWindow = window
	defer func() { activeWindow = nil }()

	hr, _, _ := procCoInitializeEx.Call(0, coinitApartmentThreaded)
	comInitialized := int32(hr) >= 0
	if comInitialized {
		defer procCoUninitialize.Call()
	}

	if err := window.create(); err != nil {
		showMessage(0, "KCD2 Dual Subtitles", err.Error(), mbOK|mbIconError)
		return 1
	}
	return window.messageLoop()
}

func (w *nativeWindow) create() error {
	instance, _, _ := procGetModuleHandleW.Call(0)
	if instance == 0 {
		return fmt.Errorf("GetModuleHandleW failed")
	}
	className := mustUTF16(windowClassName)
	cursor, _, _ := procLoadCursorW.Call(0, idcArrow)
	class := wndClass{
		WndProc:    syscall.NewCallback(windowProc),
		Instance:   instance,
		Cursor:     cursor,
		Background: colorWindow + 1,
		ClassName:  className,
	}
	atom, _, registerErr := procRegisterClassW.Call(uintptr(unsafe.Pointer(&class)))
	if atom == 0 {
		return fmt.Errorf("RegisterClassW failed: %v", registerErr)
	}

	title := mustUTF16(fmt.Sprintf("KCD2 Dual Subtitles %s", w.version))
	hwnd, _, createErr := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(title)),
		wsCaption|wsSysMenu|wsMinimizeBox,
		140,
		70,
		780,
		640,
		0,
		0,
		instance,
		0,
	)
	if hwnd == 0 {
		if w.startupErr != nil {
			return w.startupErr
		}
		return fmt.Errorf("CreateWindowExW failed: %v", createErr)
	}
	w.hwnd = hwnd
	procShowWindowGUI.Call(hwnd, swShow)
	procUpdateWindowGUI.Call(hwnd)
	return nil
}

func (w *nativeWindow) createControls(hwnd uintptr) error {
	w.hwnd = hwnd
	w.font, _, _ = procGetStockObject.Call(defaultGUIFont)

	if _, err := w.createControl("BUTTON", "Game installation", wsChild|wsVisible|bsGroupBox, 16, 12, 728, 82, 0); err != nil {
		return err
	}
	gameEdit, err := w.createControl("EDIT", w.model.GameRoot, wsChild|wsVisible|wsTabStop|wsBorder|esAutoHScroll, 32, 43, 568, 26, 0)
	if err != nil {
		return err
	}
	w.gameEdit = gameEdit
	if _, err := w.createControl("BUTTON", "Browse...", wsChild|wsVisible|wsTabStop, 612, 42, 112, 28, idBrowseButton); err != nil {
		return err
	}

	if _, err := w.createControl("BUTTON", "Subtitle languages", wsChild|wsVisible|bsGroupBox, 16, 104, 728, 104, 0); err != nil {
		return err
	}
	if _, err := w.createControl("STATIC", "Main", wsChild|wsVisible, 32, 134, 90, 22, 0); err != nil {
		return err
	}
	mainCombo, err := w.createControl("COMBOBOX", "", wsChild|wsVisible|wsTabStop|wsVScroll|cbsDropdownList, 132, 129, 200, 220, 0)
	if err != nil {
		return err
	}
	w.mainCombo = mainCombo
	if _, err := w.createControl("STATIC", "Secondary", wsChild|wsVisible, 370, 134, 120, 22, 0); err != nil {
		return err
	}
	secondaryCombo, err := w.createControl("COMBOBOX", "", wsChild|wsVisible|wsTabStop|wsVScroll|cbsDropdownList, 500, 129, 210, 220, 0)
	if err != nil {
		return err
	}
	w.secondaryCombo = secondaryCombo
	if _, err := w.createControl("STATIC", "These are subtitle text sources. The generated mod works with any installed in-game language.", wsChild|wsVisible, 32, 172, 680, 22, 0); err != nil {
		return err
	}

	if err := w.refreshLanguageControls(w.model.GameRoot); err != nil {
		return fmt.Errorf("discover installed languages: %w", err)
	}

	if _, err := w.createControl("BUTTON", "Appearance", wsChild|wsVisible|bsGroupBox, 16, 218, 728, 264, 0); err != nil {
		return err
	}
	styled, err := w.createControl("BUTTON", "Customize subtitle appearance", wsChild|wsVisible|wsTabStop|bsAutoCheckbox, 32, 245, 220, 24, idStyledCheckbox)
	if err != nil {
		return err
	}
	w.styledCheckbox = styled
	w.setChecked(w.styledCheckbox, w.presentation.Styled)

	tags, err := w.createControl("BUTTON", "Language tags", wsChild|wsVisible|wsTabStop|bsAutoCheckbox, 270, 245, 130, 24, 0)
	if err != nil {
		return err
	}
	w.tagsCheckbox = tags
	w.setChecked(w.tagsCheckbox, w.presentation.ShowLanguageTags)

	outline, err := w.createControl("BUTTON", "Outline", wsChild|wsVisible|wsTabStop|bsAutoCheckbox, 410, 245, 100, 24, 0)
	if err != nil {
		return err
	}
	w.outlineCheckbox = outline
	w.setChecked(w.outlineCheckbox, w.presentation.Outline)

	shadow, err := w.createControl("BUTTON", "Shadow", wsChild|wsVisible|wsTabStop|bsAutoCheckbox, 520, 245, 100, 24, idShadowCheckbox)
	if err != nil {
		return err
	}
	w.shadowCheckbox = shadow
	w.setChecked(w.shadowCheckbox, w.presentation.Shadow)

	if _, err := w.createControl("BUTTON", "Primary line", wsChild|wsVisible|bsGroupBox, 32, 278, 334, 170, 0); err != nil {
		return err
	}
	if _, err := w.createControl("STATIC", "Color", wsChild|wsVisible, 50, 310, 48, 22, 0); err != nil {
		return err
	}
	primaryColorEdit, err := w.createControl("EDIT", w.presentation.PrimaryColor, wsChild|wsVisible|wsTabStop|wsBorder|esAutoHScroll, 105, 306, 105, 26, 0)
	if err != nil {
		return err
	}
	w.primaryColorEdit = primaryColorEdit
	primaryColorPicker, err := w.createControl("BUTTON", "Color...", wsChild|wsVisible|wsTabStop, 220, 305, 90, 28, idPrimaryColorPickerButton)
	if err != nil {
		return err
	}
	w.primaryColorPickerButton = primaryColorPicker
	if _, err := w.createControl("STATIC", "Size", wsChild|wsVisible, 50, 347, 48, 22, 0); err != nil {
		return err
	}
	primarySizeEdit, err := w.createControl("EDIT", w.presentation.PrimarySize, wsChild|wsVisible|wsTabStop|wsBorder|esAutoHScroll, 105, 343, 70, 26, 0)
	if err != nil {
		return err
	}
	w.primarySizeEdit = primarySizeEdit
	primaryItalic, err := w.createControl("BUTTON", "Italic", wsChild|wsVisible|wsTabStop|bsAutoCheckbox, 190, 343, 110, 24, 0)
	if err != nil {
		return err
	}
	w.primaryItalicCheckbox = primaryItalic
	w.setChecked(w.primaryItalicCheckbox, w.presentation.PrimaryItalic)
	if _, err := w.createControl("STATIC", "Leave color and size blank to keep the game's default primary style.", wsChild|wsVisible, 50, 382, 290, 44, 0); err != nil {
		return err
	}

	if _, err := w.createControl("BUTTON", "Secondary line", wsChild|wsVisible|bsGroupBox, 382, 278, 334, 170, 0); err != nil {
		return err
	}
	if _, err := w.createControl("STATIC", "Color", wsChild|wsVisible, 400, 310, 48, 22, 0); err != nil {
		return err
	}
	colorEdit, err := w.createControl("EDIT", w.presentation.SecondaryColor, wsChild|wsVisible|wsTabStop|wsBorder|esAutoHScroll, 455, 306, 105, 26, 0)
	if err != nil {
		return err
	}
	w.colorEdit = colorEdit
	colorPicker, err := w.createControl("BUTTON", "Color...", wsChild|wsVisible|wsTabStop, 570, 305, 90, 28, idColorPickerButton)
	if err != nil {
		return err
	}
	w.colorPickerButton = colorPicker
	if _, err := w.createControl("STATIC", "Size", wsChild|wsVisible, 400, 347, 48, 22, 0); err != nil {
		return err
	}
	sizeEdit, err := w.createControl("EDIT", w.presentation.SecondarySize, wsChild|wsVisible|wsTabStop|wsBorder|esAutoHScroll, 455, 343, 70, 26, 0)
	if err != nil {
		return err
	}
	w.sizeEdit = sizeEdit
	italic, err := w.createControl("BUTTON", "Italic", wsChild|wsVisible|wsTabStop|bsAutoCheckbox, 540, 343, 110, 24, 0)
	if err != nil {
		return err
	}
	w.italicCheckbox = italic
	w.setChecked(w.italicCheckbox, w.presentation.SecondaryItalic)
	if _, err := w.createControl("STATIC", "The secondary line uses explicit presentation settings.", wsChild|wsVisible, 400, 382, 285, 44, 0); err != nil {
		return err
	}

	if _, err := w.createControl("STATIC", "Shadow color", wsChild|wsVisible, 50, 453, 90, 22, 0); err != nil {
		return err
	}
	shadowColorEdit, err := w.createControl("EDIT", w.presentation.ShadowColor, wsChild|wsVisible|wsTabStop|wsBorder|esAutoHScroll, 145, 449, 105, 26, 0)
	if err != nil {
		return err
	}
	w.shadowColorEdit = shadowColorEdit
	shadowColorPicker, err := w.createControl("BUTTON", "Color...", wsChild|wsVisible|wsTabStop, 260, 448, 90, 28, idShadowColorPickerButton)
	if err != nil {
		return err
	}
	w.shadowColorPickerButton = shadowColorPicker
	w.updatePresentationControls()

	generate, err := w.createControl("BUTTON", w.model.GenerateButtonLabel(), wsChild|wsVisible|wsTabStop, 16, 500, 210, 36, idGenerateButton)
	if err != nil {
		return err
	}
	w.generateButton = generate
	w.enable(w.generateButton, len(w.languages) >= 2)
	uninstall, err := w.createControl("BUTTON", "Uninstall", wsChild|wsVisible|wsTabStop, 238, 500, 120, 36, idUninstallButton)
	if err != nil {
		return err
	}
	w.uninstallButton = uninstall
	w.enable(w.uninstallButton, w.model.InstallationKnown && w.model.Installed)

	status, err := w.createControl("STATIC", w.model.Status, wsChild|wsVisible, 16, 552, 728, 48, 0)
	if err != nil {
		return err
	}
	w.statusLabel = status
	return nil
}

func (w *nativeWindow) createControl(className, text string, style uint32, x, y, width, height int32, id uintptr) (uintptr, error) {
	classPtr := mustUTF16(className)
	textPtr := mustUTF16(text)
	hwnd, _, callErr := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(classPtr)),
		uintptr(unsafe.Pointer(textPtr)),
		uintptr(style),
		uintptr(x),
		uintptr(y),
		uintptr(width),
		uintptr(height),
		w.hwnd,
		id,
		0,
		0,
	)
	if hwnd == 0 {
		return 0, fmt.Errorf("create %s control: %v", className, callErr)
	}
	if w.font != 0 {
		procSendMessageW.Call(hwnd, wmSetFont, w.font, 1)
	}
	return hwnd, nil
}

func (w *nativeWindow) messageLoop() int {
	var msg message
	for {
		result, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(result) == -1 {
			showMessage(w.hwnd, "KCD2 Dual Subtitles", "Windows message loop failed.", mbOK|mbIconError)
			return 1
		}
		if result == 0 {
			return 0
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

func windowProc(hwnd uintptr, message uint32, wParam, lParam uintptr) uintptr {
	window := activeWindow
	if window != nil {
		switch message {
		case wmCreate:
			if err := window.createControls(hwnd); err != nil {
				window.startupErr = err
				return ^uintptr(0)
			}
			return 0
		case wmCommand:
			window.handleCommand(uint16(wParam & 0xffff))
			return 0
		case wmDestroy:
			procPostQuitMessage.Call(0)
			return 0
		}
	}
	result, _, _ := procDefWindowProcW.Call(hwnd, uintptr(message), wParam, lParam)
	return result
}

func (w *nativeWindow) handleCommand(id uint16) {
	switch id {
	case idBrowseButton:
		w.browse()
	case idGenerateButton:
		w.generateAndInstall()
	case idUninstallButton:
		w.uninstall()
	case idStyledCheckbox, idShadowCheckbox:
		w.updatePresentationControls()
	case idColorPickerButton:
		w.chooseSecondaryColor()
	case idPrimaryColorPickerButton:
		w.choosePrimaryColor()
	case idShadowColorPickerButton:
		w.chooseShadowColor()
	}
}

func (w *nativeWindow) browse() {
	selected, ok := browseForFolder(w.hwnd)
	if !ok {
		return
	}
	normalized, err := w.service.ValidateGameRoot(selected)
	if err != nil {
		w.setStatus("Selected folder is not a supported KCD2 installation.")
		showMessage(w.hwnd, "Invalid game folder", err.Error(), mbOK|mbIconError)
		return
	}
	if err := w.refreshLanguageControls(normalized); err != nil {
		w.setStatus("Could not inspect installed localization languages.")
		showMessage(w.hwnd, "Language discovery", err.Error(), mbOK|mbIconError)
		return
	}
	w.setText(w.gameEdit, normalized)
	w.model.GameRoot = normalized
	w.model.AutoDetected = false
	if err := w.requireAtLeastTwoInstalledLanguages(); err != nil {
		w.setStatus(err.Error())
		return
	}
	w.setStatus(fmt.Sprintf("Game folder selected. Found %d supported subtitle languages; the generated pair will work with any of them.", len(w.languages)))
}

func (w *nativeWindow) generateAndInstall() {
	gameRoot := w.text(w.gameEdit)
	normalized, err := w.service.ValidateGameRoot(gameRoot)
	if err != nil {
		w.setStatus("Choose a valid KCD2 game folder first.")
		showMessage(w.hwnd, "Invalid game folder", err.Error(), mbOK|mbIconError)
		return
	}
	w.setText(w.gameEdit, normalized)
	if err := w.refreshLanguageControls(normalized); err != nil {
		w.setStatus("Could not inspect installed localization languages.")
		showMessage(w.hwnd, "Language discovery", err.Error(), mbOK|mbIconError)
		return
	}
	if err := w.requireAtLeastTwoInstalledLanguages(); err != nil {
		w.setStatus(err.Error())
		showMessage(w.hwnd, "Language selection", err.Error(), mbOK|mbIconError)
		return
	}

	main, ok := w.selectedLanguage(w.mainCombo)
	if !ok {
		showMessage(w.hwnd, "Language selection", "Select a main language.", mbOK|mbIconError)
		return
	}
	secondary, ok := w.selectedLanguage(w.secondaryCombo)
	if !ok {
		showMessage(w.hwnd, "Language selection", "Select a secondary language.", mbOK|mbIconError)
		return
	}
	if main == secondary {
		w.setStatus("Main and secondary languages must differ.")
		showMessage(w.hwnd, "Language selection", application.ErrSameLanguage.Error(), mbOK|mbIconError)
		return
	}

	input := w.currentPresentationInput()
	presentation, err := input.hudPresentation()
	if err != nil {
		w.setStatus("Fix the subtitle presentation settings before generating.")
		showMessage(w.hwnd, "Subtitle presentation", err.Error(), mbOK|mbIconError)
		return
	}

	w.presentation = input
	w.setBusy(true)
	w.setStatus("Generating and installing bilingual subtitle patch...")
	result, err := w.service.GenerateAndInstallWithPresentation(normalized, main, secondary, presentation)
	w.setBusy(false)
	if err != nil {
		w.setStatus("Generation failed. No successful replacement was published.")
		showMessage(w.hwnd, "Generation failed", err.Error(), mbOK|mbIconError)
		return
	}

	w.model.GameRoot = normalized
	w.model.Installed = true
	w.model.InstallationKnown = true
	w.model.InstallPath = result.InstallPath
	w.setText(w.generateButton, w.model.GenerateButtonLabel())
	w.enable(w.uninstallButton, true)
	w.setStatus(fmt.Sprintf("Installed successfully. Bilingual rows: %d; patch rows: %d; active language slots: %d.", result.Stats.Bilingual, result.PatchRows, result.LocalizationTargets))
}

func (w *nativeWindow) uninstall() {
	answer := showMessage(
		w.hwnd,
		"Uninstall KCD2 Dual Subtitles",
		"Remove the generated KCD2 Dual Subtitles mod and its mod_order.txt entry? Other mods will not be changed.",
		mbYesNo|mbIconQuestion,
	)
	if answer != idYes {
		return
	}

	w.setBusy(true)
	w.setStatus("Uninstalling...")
	result, err := w.service.Uninstall()
	w.setBusy(false)
	if err != nil {
		w.setStatus("Uninstall failed.")
		showMessage(w.hwnd, "Uninstall failed", err.Error(), mbOK|mbIconError)
		return
	}

	w.model.Installed = false
	w.model.InstallationKnown = true
	w.model.InstallPath = result.Path
	w.setText(w.generateButton, w.model.GenerateButtonLabel())
	w.enable(w.uninstallButton, false)
	if result.RemovedMod || result.UpdatedModOrder {
		w.setStatus("KCD2 Dual Subtitles was uninstalled.")
	} else {
		w.setStatus("KCD2 Dual Subtitles is already uninstalled.")
	}
}

func (w *nativeWindow) selectedLanguage(combo uintptr) (localization.Language, bool) {
	selected, _, _ := procSendMessageW.Call(combo, cbGetCurSel, 0, 0)
	index := int(selected)
	if index < 0 || index >= len(w.languages) {
		return "", false
	}
	return w.languages[index].Language, true
}

func (w *nativeWindow) currentPresentationInput() presentationInput {
	return presentationInput{
		Styled:           w.checked(w.styledCheckbox),
		ShowLanguageTags: w.checked(w.tagsCheckbox),
		PrimaryColor:     w.text(w.primaryColorEdit),
		PrimarySize:      w.text(w.primarySizeEdit),
		PrimaryItalic:    w.checked(w.primaryItalicCheckbox),
		SecondaryColor:   w.text(w.colorEdit),
		SecondarySize:    w.text(w.sizeEdit),
		SecondaryItalic:  w.checked(w.italicCheckbox),
		Outline:          w.checked(w.outlineCheckbox),
		Shadow:           w.checked(w.shadowCheckbox),
		ShadowColor:      w.text(w.shadowColorEdit),
	}
}

func (w *nativeWindow) setBusy(busy bool) {
	w.busy = busy
	enabled := !busy
	w.enable(w.gameEdit, enabled)
	w.enable(w.mainCombo, enabled && len(w.languages) > 0)
	w.enable(w.secondaryCombo, enabled && len(w.languages) > 1)
	w.enable(w.styledCheckbox, enabled)
	w.enable(w.generateButton, enabled && len(w.languages) >= 2)
	w.enable(w.uninstallButton, enabled && w.model.InstallationKnown && w.model.Installed)
	w.updatePresentationControls()
}

func (w *nativeWindow) updatePresentationControls() {
	enabled := !w.busy && w.checked(w.styledCheckbox)
	shadowColorEnabled := enabled && w.checked(w.shadowCheckbox)
	w.enable(w.tagsCheckbox, enabled)
	w.enable(w.outlineCheckbox, enabled)
	w.enable(w.shadowCheckbox, enabled)
	w.enable(w.shadowColorEdit, shadowColorEnabled)
	w.enable(w.shadowColorPickerButton, shadowColorEnabled)
	w.enable(w.primaryColorEdit, enabled)
	w.enable(w.primaryColorPickerButton, enabled)
	w.enable(w.primarySizeEdit, enabled)
	w.enable(w.primaryItalicCheckbox, enabled)
	w.enable(w.colorEdit, enabled)
	w.enable(w.colorPickerButton, enabled)
	w.enable(w.sizeEdit, enabled)
	w.enable(w.italicCheckbox, enabled)
}

func (w *nativeWindow) checked(hwnd uintptr) bool {
	if hwnd == 0 {
		return false
	}
	value, _, _ := procSendMessageW.Call(hwnd, bmGetCheck, 0, 0)
	return value == bstChecked
}

func (w *nativeWindow) setChecked(hwnd uintptr, checked bool) {
	value := uintptr(bstUnchecked)
	if checked {
		value = bstChecked
	}
	procSendMessageW.Call(hwnd, bmSetCheck, value, 0)
}

func (w *nativeWindow) setStatus(status string) {
	w.model.Status = status
	w.setText(w.statusLabel, status)
	if w.statusLabel != 0 {
		procUpdateWindowGUI.Call(w.statusLabel)
	}
}

func (w *nativeWindow) setText(hwnd uintptr, value string) {
	if hwnd == 0 {
		return
	}
	ptr := mustUTF16(value)
	procSetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(ptr)))
}

func (w *nativeWindow) text(hwnd uintptr) string {
	length, _, _ := procGetWindowTextLengthW.Call(hwnd)
	buffer := make([]uint16, int(length)+1)
	if len(buffer) == 0 {
		return ""
	}
	procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)))
	return syscall.UTF16ToString(buffer)
}

func (w *nativeWindow) enable(hwnd uintptr, enabled bool) {
	value := uintptr(0)
	if enabled {
		value = 1
	}
	procEnableWindow.Call(hwnd, value)
}

func browseForFolder(owner uintptr) (string, bool) {
	var displayName [260]uint16
	title := mustUTF16("Select the KCD2 Content folder or its immediate parent")
	info := browseInfo{
		Owner:       owner,
		DisplayName: &displayName[0],
		Title:       title,
		Flags:       bifReturnOnlyFSDirs | bifEditBox | bifNewDialogStyle,
	}
	pidl, _, _ := procSHBrowseForFolderW.Call(uintptr(unsafe.Pointer(&info)))
	if pidl == 0 {
		return "", false
	}
	defer procCoTaskMemFree.Call(pidl)

	var path [260]uint16
	ok, _, _ := procSHGetPathFromIDListW.Call(pidl, uintptr(unsafe.Pointer(&path[0])))
	if ok == 0 {
		return "", false
	}
	return syscall.UTF16ToString(path[:]), true
}

func showMessage(owner uintptr, title, text string, flags uintptr) int {
	titlePtr := mustUTF16(title)
	textPtr := mustUTF16(text)
	result, _, _ := procMessageBoxW.Call(
		owner,
		uintptr(unsafe.Pointer(textPtr)),
		uintptr(unsafe.Pointer(titlePtr)),
		flags,
	)
	return int(result)
}

func mustUTF16(value string) *uint16 {
	ptr, err := syscall.UTF16PtrFromString(value)
	if err != nil {
		panic(err)
	}
	return ptr
}
