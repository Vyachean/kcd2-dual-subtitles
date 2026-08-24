//go:build windows && amd64

package gui

import (
	"testing"
	"unsafe"
)

func TestWin32StructSizesAMD64(t *testing.T) {
	if got := unsafe.Sizeof(wndClass{}); got != 72 {
		t.Fatalf("WNDCLASSW-compatible size = %d, want 72", got)
	}
	if got := unsafe.Sizeof(message{}); got != 48 {
		t.Fatalf("MSG-compatible size = %d, want 48", got)
	}
	if got := unsafe.Sizeof(browseInfo{}); got != 64 {
		t.Fatalf("BROWSEINFOW-compatible size = %d, want 64", got)
	}
	if got := unsafe.Sizeof(chooseColor{}); got != 72 {
		t.Fatalf("CHOOSECOLORW-compatible size = %d, want 72", got)
	}
}

func TestUTF16Helper(t *testing.T) {
	ptr := mustUTF16("Русский / English")
	if ptr == nil {
		t.Fatal("mustUTF16 returned nil")
	}
}
