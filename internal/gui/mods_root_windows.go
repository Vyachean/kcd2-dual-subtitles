//go:build windows

package gui

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/modinstall"
)

const (
	idModsBrowseButton = 1007
	idModsResetButton  = 1008
)

func (w *nativeWindow) refreshModsRootControls() error {
	if w.modsEdit == 0 {
		return nil
	}
	if strings.TrimSpace(w.model.GameRoot) == "" {
		w.setText(w.modsEdit, "")
		w.setText(w.modsModeLabel, "Select a game folder to resolve its Mods folder.")
		w.enable(w.modsBrowseButton, false)
		w.enable(w.modsResetButton, false)
		return nil
	}
	location, err := w.service.SelectedModsLocation()
	if err != nil {
		return err
	}
	w.applyModsLocation(location)
	return nil
}

func (w *nativeWindow) applyModsLocation(location modinstall.InstallLocation) {
	w.setText(w.modsEdit, location.ModsRoot)
	custom := location.Layout == modinstall.InstallLayoutCustom
	if custom {
		w.setText(w.modsModeLabel, "Custom location. Source discovery, install, status and uninstall all use this folder.")
	} else {
		w.setText(w.modsModeLabel, "Automatically detected from the selected game. Use Change... only if this is not your active Mods folder.")
	}
	w.enable(w.modsBrowseButton, !w.busy)
	w.enable(w.modsResetButton, !w.busy && custom)
}

func (w *nativeWindow) browseModsRoot() {
	if strings.TrimSpace(w.model.GameRoot) == "" {
		w.setStatus("Choose a valid KCD2 game folder before changing the Mods folder.")
		showMessage(w.hwnd, "Mods folder", "Choose a valid KCD2 game folder first.", mbOK|mbIconError)
		return
	}
	selected, ok := browseForFolderWithTitle(w.hwnd, "Select the active KCD2 Mods folder")
	if !ok {
		return
	}
	location, err := w.service.SetModsRootOverride(selected)
	if err != nil {
		w.setStatus("Selected Mods folder is not usable.")
		showMessage(w.hwnd, "Invalid Mods folder", err.Error(), mbOK|mbIconError)
		return
	}
	w.applyModsLocation(location)
	if err := w.refreshLanguageControls(w.model.GameRoot); err != nil {
		w.setStatus("Mods folder changed, but installation status could not be refreshed.")
		showMessage(w.hwnd, "Mods folder", err.Error(), mbOK|mbIconError)
		return
	}
	w.setStatus(fmt.Sprintf("Using custom Mods folder: %s", location.ModsRoot))
}

func (w *nativeWindow) resetModsRoot() {
	location, err := w.service.ResetModsRootOverride()
	if err != nil {
		w.setStatus("Could not restore the automatically detected Mods folder.")
		showMessage(w.hwnd, "Mods folder", err.Error(), mbOK|mbIconError)
		return
	}
	w.applyModsLocation(location)
	if err := w.refreshLanguageControls(w.model.GameRoot); err != nil {
		w.setStatus("Automatic Mods folder restored, but installation status could not be refreshed.")
		showMessage(w.hwnd, "Mods folder", err.Error(), mbOK|mbIconError)
		return
	}
	w.setStatus(fmt.Sprintf("Using automatically detected Mods folder: %s", location.ModsRoot))
}

func browseForFolderWithTitle(owner uintptr, titleText string) (string, bool) {
	var displayName [260]uint16
	title := mustUTF16(titleText)
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

	var selectedPath [260]uint16
	ok, _, _ := procSHGetPathFromIDListW.Call(pidl, uintptr(unsafe.Pointer(&selectedPath[0])))
	if ok == 0 {
		return "", false
	}
	return syscall.UTF16ToString(selectedPath[:]), true
}
