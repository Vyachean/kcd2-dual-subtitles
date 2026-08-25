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
)

var (
	procPostMessageW  = guiUser32.NewProc("PostMessageW")
	generationResults = make(chan generationOutcome, 1)
)

type generationOutcome struct {
	normalized string
	result     generator.Result
	err        error
}

func (w *nativeWindow) startGeneration(normalized string, main, secondary localization.Language, presentation *generator.HUDPresentationConfig) {
	if w.busy {
		return
	}
	w.setBusy(true)
	w.setStatus("Generating and installing bilingual subtitle patch... The window will remain responsive.")

	service := w.service
	hwnd := w.hwnd
	go func() {
		result, err := service.GenerateAndInstallWithPresentation(normalized, main, secondary, presentation)
		generationResults <- generationOutcome{normalized: normalized, result: result, err: err}
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

	w.setBusy(false)
	if outcome.err != nil {
		w.setStatus("Generation failed. No successful replacement was published.")
		showMessage(w.hwnd, "Generation failed", outcome.err.Error(), mbOK|mbIconError)
		return
	}

	w.model.GameRoot = outcome.normalized
	w.model.Installed = true
	w.model.InstallationKnown = true
	w.model.InstallPath = outcome.result.InstallPath
	w.setText(w.generateButton, w.model.GenerateButtonLabel())
	w.enable(w.uninstallButton, true)
	w.setStatus(fmt.Sprintf("Installed successfully. Bilingual rows: %d; patch rows: %d; active language slots: %d.", outcome.result.Stats.Bilingual, outcome.result.PatchRows, outcome.result.LocalizationTargets))
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
