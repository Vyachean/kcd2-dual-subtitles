//go:build windows

package modinstall

import (
	"fmt"
	"syscall"
	"unsafe"
)

type guid struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

var folderIDDocuments = guid{
	Data1: 0xFDD39AD0,
	Data2: 0x238F,
	Data3: 0x46AF,
	Data4: [8]byte{0xAD, 0xB4, 0x6C, 0x85, 0x48, 0x03, 0x69, 0xC7},
}

var (
	shell32                  = syscall.NewLazyDLL("shell32.dll")
	ole32                    = syscall.NewLazyDLL("ole32.dll")
	procSHGetKnownFolderPath = shell32.NewProc("SHGetKnownFolderPath")
	procCoTaskMemFree        = ole32.NewProc("CoTaskMemFree")
	procCoInitializeEx       = ole32.NewProc("CoInitializeEx")
	procCoUninitialize       = ole32.NewProc("CoUninitialize")
)

const (
	coinitApartmentThreaded = 0x2
	sOK                     = 0x00000000
	sFalse                  = 0x00000001
	rpcEChangedMode         = 0x80010106
)

// documentsPath resolves FOLDERID_Documents for the current user through
// SHGetKnownFolderPath. This follows redirected and OneDrive-backed Documents
// folders instead of assuming %USERPROFILE%\Documents.
func documentsPath() (string, error) {
	hr, _, _ := procCoInitializeEx.Call(0, coinitApartmentThreaded)
	hresult := uint32(hr)
	initializedHere := hresult == sOK || hresult == sFalse
	if initializedHere {
		defer procCoUninitialize.Call()
	} else if hresult != rpcEChangedMode {
		return "", fmt.Errorf("initialize COM for Known Folders: HRESULT 0x%08X", hresult)
	}

	var path *uint16
	hr, _, _ = procSHGetKnownFolderPath.Call(
		uintptr(unsafe.Pointer(&folderIDDocuments)),
		0,
		0,
		uintptr(unsafe.Pointer(&path)),
	)
	if path != nil {
		defer procCoTaskMemFree.Call(uintptr(unsafe.Pointer(path)))
	}
	if failedHRESULT(hr) {
		return "", fmt.Errorf("resolve FOLDERID_Documents: HRESULT 0x%08X", uint32(hr))
	}
	if path == nil {
		return "", errorsNewKnownFolderReturnedNil()
	}

	resolved := utf16PtrToString(path)
	if resolved == "" {
		return "", fmt.Errorf("resolve FOLDERID_Documents: returned empty path")
	}
	return resolved, nil
}

func failedHRESULT(hr uintptr) bool {
	return int32(uint32(hr)) < 0
}

func utf16PtrToString(ptr *uint16) string {
	length := 0
	for {
		value := *(*uint16)(unsafe.Add(unsafe.Pointer(ptr), uintptr(length)*unsafe.Sizeof(*ptr)))
		if value == 0 {
			break
		}
		length++
	}
	return syscall.UTF16ToString(unsafe.Slice(ptr, length))
}

func errorsNewKnownFolderReturnedNil() error {
	return fmt.Errorf("resolve FOLDERID_Documents: returned nil path")
}
