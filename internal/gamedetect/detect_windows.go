//go:build windows

package gamedetect

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

const (
	driveFixed = 3

	regKeyRead      = 0x20019
	regSZ           = 1
	regExpandSZ     = 2
	regSuccess      = 0
	regMoreData     = 234
	regNoMoreItems  = 259
	hkeyCurrentUser = uintptr(0x80000001)
	hkeyLocalMachine = uintptr(0x80000002)
)

var (
	kernel32                   = syscall.NewLazyDLL("kernel32.dll")
	advapi32                   = syscall.NewLazyDLL("advapi32.dll")
	procGetLogicalDriveStrings = kernel32.NewProc("GetLogicalDriveStringsW")
	procGetDriveType           = kernel32.NewProc("GetDriveTypeW")
	procRegOpenKeyExW          = advapi32.NewProc("RegOpenKeyExW")
	procRegQueryValueExW       = advapi32.NewProc("RegQueryValueExW")
	procRegEnumKeyExW          = advapi32.NewProc("RegEnumKeyExW")
	procRegCloseKey            = advapi32.NewProc("RegCloseKey")
)

// Detect performs best-effort Windows discovery and returns every structurally
// compatible KCD2 root it finds. Launcher/store metadata is used only to find
// candidate paths; compatibility is always decided by NormalizeSelection /
// IsGameRoot and therefore remains store-neutral.
func Detect() (Result, error) {
	drives, err := logicalFixedDrives()
	if err != nil {
		return Result{}, err
	}

	results := []Result{
		detectInInstallRoots(xboxRootsForDrives(drives)),
		detectCandidatePaths(steamCommonCandidates(steamLibraryRoots(steamInstallRoots(drives)))),
		detectCandidatePaths(epicInstalledCandidates()),
		detectCandidatePaths(gogInstalledCandidates(drives)),
	}
	return mergeDetectionResults(results...), nil
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

func detectCandidatePaths(paths []string) Result {
	candidates := make([]string, 0, len(paths))
	for _, candidate := range uniquePaths(paths) {
		normalized, err := NormalizeSelection(candidate)
		if err != nil {
			continue
		}
		candidates = append(candidates, normalized)
	}
	return Result{Candidates: uniquePaths(candidates)}
}

func mergeDetectionResults(results ...Result) Result {
	all := make([]string, 0)
	for _, result := range results {
		all = append(all, result.Candidates...)
	}
	return Result{Candidates: uniquePaths(all)}
}

func uniquePaths(paths []string) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		path = filepath.Clean(path)
		key := strings.ToLower(path)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, path)
	}
	sort.Slice(result, func(i, j int) bool {
		return strings.ToLower(result[i]) < strings.ToLower(result[j])
	})
	return result
}

func steamInstallRoots(drives []string) []string {
	roots := make([]string, 0)
	if path, ok := readRegistryString(hkeyCurrentUser, `Software\Valve\Steam`, "SteamPath"); ok {
		roots = append(roots, path)
	}
	if path, ok := readRegistryString(hkeyLocalMachine, `SOFTWARE\WOW6432Node\Valve\Steam`, "InstallPath"); ok {
		roots = append(roots, path)
	}
	for _, variable := range []string{"ProgramFiles(x86)", "ProgramFiles"} {
		if base := strings.TrimSpace(os.Getenv(variable)); base != "" {
			roots = append(roots, filepath.Join(base, "Steam"))
		}
	}
	for _, drive := range drives {
		roots = append(roots,
			filepath.Join(drive, "Program Files (x86)", "Steam"),
			filepath.Join(drive, "Program Files", "Steam"),
		)
	}
	return uniquePaths(roots)
}

func steamLibraryRoots(steamRoots []string) []string {
	libraries := append([]string(nil), steamRoots...)
	for _, steamRoot := range steamRoots {
		for _, relative := range []string{
			filepath.Join("config", "libraryfolders.vdf"),
			filepath.Join("steamapps", "libraryfolders.vdf"),
		} {
			data, err := os.ReadFile(filepath.Join(steamRoot, relative))
			if err != nil {
				continue
			}
			libraries = append(libraries, parseSteamLibraryFolders(data)...)
		}
	}
	return uniquePaths(libraries)
}

func parseSteamLibraryFolders(data []byte) []string {
	tokens := parseVDFQuotedStrings(string(data))
	roots := make([]string, 0)
	for i := 0; i+1 < len(tokens); i++ {
		key := tokens[i]
		value := tokens[i+1]
		if strings.EqualFold(key, "path") {
			roots = append(roots, value)
			continue
		}
		if _, err := strconv.Atoi(key); err == nil && filepath.IsAbs(value) {
			roots = append(roots, value)
		}
	}
	return uniquePaths(roots)
}

func parseVDFQuotedStrings(data string) []string {
	result := make([]string, 0)
	for i := 0; i < len(data); {
		if data[i] != '"' {
			i++
			continue
		}
		i++
		var value strings.Builder
		for i < len(data) {
			if data[i] == '"' {
				i++
				break
			}
			if data[i] == '\\' && i+1 < len(data) && (data[i+1] == '\\' || data[i+1] == '"') {
				value.WriteByte(data[i+1])
				i += 2
				continue
			}
			value.WriteByte(data[i])
			i++
		}
		result = append(result, value.String())
	}
	return result
}

func steamCommonCandidates(libraries []string) []string {
	commonRoots := make([]string, 0, len(libraries))
	for _, library := range libraries {
		commonRoots = append(commonRoots, filepath.Join(library, "steamapps", "common"))
	}
	return childDirectories(commonRoots)
}

func epicInstalledCandidates() []string {
	programData := strings.TrimSpace(os.Getenv("ProgramData"))
	if programData == "" {
		return nil
	}
	return epicInstallLocationsFromManifestDir(filepath.Join(programData, "Epic", "EpicGamesLauncher", "Data", "Manifests"))
}

func epicInstallLocationsFromManifestDir(manifestDir string) []string {
	entries, err := os.ReadDir(manifestDir)
	if err != nil {
		return nil
	}
	locations := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".item") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(manifestDir, entry.Name()))
		if err != nil {
			continue
		}
		var manifest struct {
			InstallLocation string `json:"InstallLocation"`
		}
		if json.Unmarshal(data, &manifest) != nil || strings.TrimSpace(manifest.InstallLocation) == "" {
			continue
		}
		locations = append(locations, manifest.InstallLocation)
	}
	return uniquePaths(locations)
}

func gogInstalledCandidates(drives []string) []string {
	candidates := append([]string(nil), gogRegistryGamePaths()...)
	programData := strings.TrimSpace(os.Getenv("ProgramData"))
	if programData != "" {
		if root, ok := gogLibraryRootFromConfig(filepath.Join(programData, "GOG.com", "Galaxy", "config.json")); ok {
			candidates = append(candidates, childDirectories([]string{root})...)
		}
	}
	for _, drive := range drives {
		candidates = append(candidates, childDirectories([]string{filepath.Join(drive, "GOG Games")})...)
	}
	return uniquePaths(candidates)
}

func gogLibraryRootFromConfig(configPath string) (string, bool) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", false
	}
	var config struct {
		LibraryPath string `json:"libraryPath"`
	}
	if json.Unmarshal(data, &config) != nil || strings.TrimSpace(config.LibraryPath) == "" {
		return "", false
	}
	return filepath.Clean(config.LibraryPath), true
}

func gogRegistryGamePaths() []string {
	paths := make([]string, 0)
	for _, base := range []string{
		`SOFTWARE\WOW6432Node\GOG.com\Games`,
		`SOFTWARE\GOG.com\Games`,
	} {
		for _, subkey := range enumerateRegistrySubkeys(hkeyLocalMachine, base) {
			if path, ok := readRegistryString(hkeyLocalMachine, base+`\`+subkey, "path"); ok {
				paths = append(paths, path)
			}
		}
	return uniquePaths(paths)
}

func childDirectories(roots []string) []string {
	children := make([]string, 0)
	for _, root := range uniquePaths(roots) {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				children = append(children, filepath.Join(root, entry.Name()))
			}
		}
	}
	return uniquePaths(children)
}

func readRegistryString(root uintptr, subkey, valueName string) (string, bool) {
	subkeyPtr, err := syscall.UTF16PtrFromString(subkey)
	if err != nil {
		return "", false
	}
	var key uintptr
	status, _, _ := procRegOpenKeyExW.Call(
		root,
		uintptr(unsafe.Pointer(subkeyPtr)),
		0,
		regKeyRead,
		uintptr(unsafe.Pointer(&key)),
	)
	if status != regSuccess {
		return "", false
	}
	defer procRegCloseKey.Call(key)

	valuePtr, err := syscall.UTF16PtrFromString(valueName)
	if err != nil {
		return "", false
	}
	var valueType uint32
	var size uint32
	status, _, _ = procRegQueryValueExW.Call(
		key,
		uintptr(unsafe.Pointer(valuePtr)),
		0,
		uintptr(unsafe.Pointer(&valueType)),
		0,
		uintptr(unsafe.Pointer(&size)),
	)
	if status != regSuccess || size < 2 || (valueType != regSZ && valueType != regExpandSZ) {
		return "", false
	}

	buffer := make([]uint16, int(size/2)+1)
	status, _, _ = procRegQueryValueExW.Call(
		key,
		uintptr(unsafe.Pointer(valuePtr)),
		0,
		uintptr(unsafe.Pointer(&valueType)),
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(unsafe.Pointer(&size)),
	)
	if status != regSuccess {
		return "", false
	}
	value := strings.TrimSpace(syscall.UTF16ToString(buffer))
	if value == "" {
		return "", false
	}
	return filepath.Clean(value), true
}

func enumerateRegistrySubkeys(root uintptr, subkey string) []string {
	subkeyPtr, err := syscall.UTF16PtrFromString(subkey)
	if err != nil {
		return nil
	}
	var key uintptr
	status, _, _ := procRegOpenKeyExW.Call(
		root,
		uintptr(unsafe.Pointer(subkeyPtr)),
		0,
		regKeyRead,
		uintptr(unsafe.Pointer(&key)),
	)
	if status != regSuccess {
		return nil
	}
	defer procRegCloseKey.Call(key)

	names := make([]string, 0)
	for index := uintptr(0); ; index++ {
		buffer := make([]uint16, 512)
		nameLength := uint32(len(buffer))
		status, _, _ := procRegEnumKeyExW.Call(
			key,
			index,
			uintptr(unsafe.Pointer(&buffer[0])),
			uintptr(unsafe.Pointer(&nameLength)),
			0,
			0,
			0,
			0,
		)
		switch status {
		case regSuccess:
			names = append(names, syscall.UTF16ToString(buffer[:nameLength]))
		case regNoMoreItems:
			return names
		case regMoreData:
			continue
		default:
			return names
		}
	}
}
