//go:build windows

package gui

import (
	"fmt"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/generator"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
)

const (
	wmClose              = 0x0010
	wmApp                = 0x8000
	wmGenerationComplete = wmApp + 1

	generationWindowWidth  = 780
	generationWindowHeight = 890
	generationEditMultiline = 0x0004
	generationEditAutoVScroll = 0x0040
	generationEditReadOnly = 0x0800
	generationSWPNoMove   = 0x0002
	generationSWPNoZOrder = 0x0004
)

var (
	procPostMessageW   = guiUser32.NewProc("PostMessageW")
	procGetDlgItem     = guiUser32.NewProc("GetDlgItem")
	procSetWindowPos   = guiUser32.NewProc("SetWindowPos")
	generationResults  = make(chan generationOutcome, 1)
	generationLogGroup uintptr
	generationLogEdit  uintptr
)

type generationOutcome struct {
	normalized string
	context    generationLogContext
	result     generator.Result
	err        error
}

func (w *nativeWindow) startGeneration(normalized string, main, secondary localization.Language, presentation *generator.HUDPresentationConfig) {
	if w.busy {
		return
	}

	context := generationLogContext{
		GameRoot:  normalized,
		ModsRoot:  w.text(w.modsEdit),
		Main:      main,
		Secondary: secondary,
		Styled:    presentation != nil,
	}
	if err := w.ensureGenerationLogControls(); err != nil {
		w.setStatus("Generation log could not be shown: " + err.Error())
	} else {
		w.setGenerationLog(formatGenerationStarted(context))
		w.setStatus("Generating and installing... See Generation activity below.")
	}
	w.setGenerationBusy(true)

	service := w.service
	hwnd := w.hwnd
	go func() {
		result, err := service.GenerateAndInstallWithPresentation(normalized, main, secondary, presentation)
		generationResults <- generationOutcome{normalized: normalized, context: context, result: result, err: err}
		_, _, _ = procPostMessageW.Call(hwnd, wmGenerationComplete, 0, 0)
	}()
}

func (w *nativeWindow) finishGeneration() {
	var outcome generationOutcome
	select {
	case outcome = <-generationResults:
	default:
		return
	}

	w.setGenerationBusy(false)
	if outcome.err != nil {
		w.setGenerationLog(formatGenerationFailed(outcome.context, outcome.err))
		w.setStatus("Generation failed. See Generation activity below for details.")
		showMessage(w.hwnd, "Generation failed", outcome.err.Error(), mbOK|mbIconError)
		return
	}

	w.model.GameRoot = outcome.normalized
	w.model.Installed = true
	w.model.InstallationKnown = true
	w.model.InstallPath = outcome.result.InstallPath
	w.setText(w.generateButton, w.model.GenerateButtonLabel())
	w.enable(w.uninstallButton, true)
	w.setGenerationLog(formatGenerationSucceeded(outcome.context, outcome.result))
	w.setStatus("Generation completed. Restart KCD2 before testing.")
}

// ensureGenerationLogControls expands the fixed native window once and creates
// a selectable read-only activity log below the existing status line. The UI is
// intentionally singleton-based, matching activeWindow and the rest of this
// Win32 presentation layer.
func (w *nativeWindow) ensureGenerationLogControls() error {
	if generationLogEdit != 0 {
		return nil
	}
	resized, _, resizeErr := procSetWindowPos.Call(
		w.hwnd,
		0,
		0,
		0,
		generationWindowWidth,
		generationWindowHeight,
		generationSWPNoMove|generationSWPNoZOrder,
	)
	if resized == 0 {
		return fmt.Errorf("resize window: %v", resizeErr)
	}

	group, err := w.createControl("BUTTON", "Generation activity", wsChild|wsVisible|bsGroupBox, 16, 664, 728, 174, 0)
	if err != nil {
		return err
	}
	edit, err := w.createControl(
		"EDIT",
		"",
		wsChild|wsVisible|wsTabStop|wsBorder|wsVScroll|generationEditMultiline|generationEditAutoVScroll|generationEditReadOnly,
		30,
		690,
		700,
		132,
		0,
	)
	if err != nil {
		return err
	}
	generationLogGroup = group
	generationLogEdit = edit
	return nil
}

func (w *nativeWindow) setGenerationLog(value string) {
	if generationLogEdit == 0 {
		w.setStatus(value)
		return
	}
	w.setText(generationLogEdit, value)
	procUpdateWindowGUI.Call(generationLogEdit)
}

// setGenerationBusy keeps every game-selection control stable while the
// background operation uses the captured game root. Browse is a sibling button
// rather than one of nativeWindow's stored controls, so toggle it explicitly in
// addition to the common busy-state controls.
func (w *nativeWindow) setGenerationBusy(busy bool) {
	w.setBusy(busy)
	browseButton, _, _ := procGetDlgItem.Call(w.hwnd, idBrowseButton)
	w.enable(browseButton, !busy)
}

func (w *nativeWindow) confirmCloseWhileBusy() bool {
	answer := showMessage(
		w.hwnd,
		"Generation in progress",
		"Generation or installation is still running. Closing now will stop the application. Any interrupted install transaction will be recovered automatically on the next generation. Close anyway?",
		mbYesNo|mbIconQuestion,
	)
	return answer == idYes
}
