//go:build windows

package launcher

import (
	"os"
	"syscall"
	"unsafe"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/cli"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/gui"
)

const swHide = 0

var (
	launcherKernel32              = syscall.NewLazyDLL("kernel32.dll")
	launcherUser32                = syscall.NewLazyDLL("user32.dll")
	procGetConsoleWindow          = launcherKernel32.NewProc("GetConsoleWindow")
	procGetConsoleProcessList     = launcherKernel32.NewProc("GetConsoleProcessList")
	procShowWindow                = launcherUser32.NewProc("ShowWindow")
)

// Run selects the native GUI for an ordinary no-argument Windows launch and
// preserves the existing console CLI for every explicit command/flag.
func Run(args []string, version string) int {
	if len(args) == 0 {
		hideDedicatedConsoleWindow()
		return gui.Run(version)
	}
	return cli.Run(args, os.Stdin, os.Stdout, os.Stderr, version)
}

func hideDedicatedConsoleWindow() {
	hwnd, _, _ := procGetConsoleWindow.Call()
	if hwnd == 0 {
		return
	}
	processes := [2]uint32{}
	count, _, _ := procGetConsoleProcessList.Call(
		uintptr(unsafe.Pointer(&processes[0])),
		uintptr(len(processes)),
	)
	if count == 1 {
		procShowWindow.Call(hwnd, swHide)
	}
}
