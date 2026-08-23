//go:build windows

package gamedetect

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"unsafe"
)

const driveFixed = 3

var (
	kernel32                   = syscall.NewLazyDLL("kernel32.dll")
	procGetLogicalDriveStrings = kernel32.NewProc("GetLogicalDriveStringsW")
	procGetDriveType           = kernel32.NewProc("GetDriveTypeW")
)

// Detect searches Xbox / Microsoft Store GDK flat-file install roots on fixed
// drives and returns every structurally valid KCD2 Content root it finds.
func Detect() (Result, error) {
	drives, err := logicalFixedDrives()
	if err != nil {
		return Result{}, err
	}
	return detectInXboxRoots(xboxRootsForDrives(drives)), nil
}

func logicalFixedDrives() ([]string, error) {
	required, _, callErr := procGetLogicalDriveStrings.Call(0, 0)
	if required == 0 {
		return nil, fmt.Errorf("GetLogicalDriveStringsW: %w", callErr)
	}

	buffer := make([]uint16, required+1)
	written, _, callErr := procGetLogicalDriveStrings.Call(
		uintptr(len(buffer)),
		uintptr(unsafe.Pointer(&buffer[0])),
	)
	if written == 0 {
		return nil, fmt.Errorf("GetLogicalDriveStringsW: %w", callErr)
	}

	drives := make([]string, 0)
	for start := 0; start < int(written); {
		end := start
		for end < len(buffer) && buffer[end] != 0 {
			end++
		}
		if end == start {
			break
		}
		drive := syscall.UTF16ToString(buffer[start:end])
		ptr, err := syscall.UTF16PtrFromString(drive)
		if err == nil {
			driveType, _, _ := procGetDriveType.Call(uintptr(unsafe.Pointer(ptr)))
			if driveType == driveFixed {
				drives = append(drives, filepath.Clean(drive))
			}
		}
		start = end + 1
	}
	sort.Slice(drives, func(i, j int) bool { return strings.ToLower(drives[i]) < strings.ToLower(drives[j]) })
	return drives, nil
}

func xboxRootsForDrives(drives []string) []string {
	seen := make(map[string]struct{})
	roots := make([]string, 0, len(drives)*2)
	add := func(path string) {
		path = filepath.Clean(path)
		key := strings.ToLower(path)
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		roots = append(roots, path)
	}

	for _, drive := range drives {
		add(filepath.Join(drive, "XboxGames"))

		data, err := os.ReadFile(filepath.Join(drive, ".GamingRoot"))
		if err != nil {
			continue
		}
		customRoots, err := parseGamingRoot(data, drive)
		if err != nil {
			continue
		}
		for _, root := range customRoots {
			add(root)
		}
	}

	sort.Slice(roots, func(i, j int) bool { return strings.ToLower(roots[i]) < strings.ToLower(roots[j]) })
	return roots
}
