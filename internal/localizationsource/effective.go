package localizationsource

import (
	"archive/zip"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/modarchive"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/modinstall"
)

const maxLocalizationResourceBytes int64 = 128 << 20

// Contribution identifies one active installed mod that changed the effective
// dialogue source for a selected language.
type Contribution struct {
	ModID string
	Name  string
	Path  string
}

// Result is the stock dialogue table after active localization overrides have
// been applied in KCD2 load order.
type Result struct {
	Rows          []localization.DialogueRow
	Contributions []Contribution
}

type modCandidate struct {
	folder string
	path   string
	modID  string
	name   string
	pak    string
}

type manifest struct {
	Info struct {
		Name  string `xml:"name"`
		ModID string `xml:"modid"`
	} `xml:"info"`
	Supports struct {
		Versions []string `xml:"version"`
	} `xml:"supports"`
}

type localizationResource struct {
	file     *zip.File
	name     string
	dialogue bool
}

// Resolve builds the effective dialogue table for language from the stock game
// localization plus active local-mod overrides. Original files are read-only.
func Resolve(gameRoot string, language localization.Language) (Result, error) {
	info, ok := localization.LookupLanguage(language)
	if !ok {
		return Result{}, fmt.Errorf("unsupported localization language %q", language)
	}

	stockPAK := filepath.Join(gameRoot, "Localization", info.PakFilename)
	stockXML, err := localization.ReadDialogueXML(stockPAK)
	if err != nil {
		return Result{}, fmt.Errorf("read stock localization %q: %w", stockPAK, err)
	}
	stockRows, err := localization.ParseDialogueXML(stockXML)
	if err != nil {
		return Result{}, fmt.Errorf("parse stock localization %q: %w", stockPAK, err)
	}
	if err := requireUniqueIDs(stockRows); err != nil {
		return Result{}, fmt.Errorf("validate stock localization %q: %w", stockPAK, err)
	}

	location, err := modinstall.ResolveModSourceLocation(gameRoot)
	if err != nil {
		return Result{}, fmt.Errorf("resolve localization mod root: %w", err)
	}
	gameVersion, gameVersionErr := readGameVersion(gameRoot)
	return resolveFromModsRootWithVersion(stockRows, location.ModsRoot, info.PakFilename, gameVersion, gameVersionErr)
}

func resolveFromModsRoot(stockRows []localization.DialogueRow, modsRoot, pakFilename string) (Result, error) {
	return resolveFromModsRootWithVersion(stockRows, modsRoot, pakFilename, "", nil)
}

func resolveFromModsRootWithVersion(stockRows []localization.DialogueRow, modsRoot, pakFilename, gameVersion string, gameVersionErr error) (Result, error) {
	rows := append([]localization.DialogueRow(nil), stockRows...)
	candidates, err := activeLocalizationMods(modsRoot, pakFilename, gameVersion, gameVersionErr)
	if err != nil {
		return Result{}, err
	}

	result := Result{Rows: rows}
	for _, candidate := range candidates {
		updated, used, err := overlayLocalizationPAK(result.Rows, candidate.pak)
		if err != nil {
			return Result{}, fmt.Errorf("apply localization mod %q (%s): %w", candidateLabel(candidate), candidate.pak, err)
		}
		if !used {
			continue
		}
		result.Rows = updated
		result.Contributions = append(result.Contributions, Contribution{
			ModID: candidate.modID,
			Name:  candidate.name,
			Path:  candidate.path,
		})
	}
	return result, nil
}

func candidateLabel(candidate modCandidate) string {
	if strings.TrimSpace(candidate.modID) != "" {
		return candidate.modID
	}
	if strings.TrimSpace(candidate.name) != "" {
		return candidate.name
	}
	return candidate.folder
}

func activeLocalizationMods(modsRoot, pakFilename, gameVersion string, gameVersionErr error) ([]modCandidate, error) {
	entries, err := os.ReadDir(modsRoot)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read mods root %q: %w", modsRoot, err)
	}

	order, orderExists, err := readModOrder(modsRoot)
	if err != nil {
		return nil, err
	}

	candidates := make([]modCandidate, 0)
	for _, entry := range entries {
		if !entry.IsDir() || isProjectOwnedFolder(entry.Name()) {
			continue
		}
		dir := filepath.Join(modsRoot, entry.Name())
		pakPath := filepath.Join(dir, "Localization", pakFilename)
		info, err := os.Lstat(pakPath)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("inspect localization PAK %q: %w", pakPath, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return nil, fmt.Errorf("localization PAK is not a regular file: %q", pakPath)
		}

		modID, name, active, err := readManifestIdentity(dir, entry.Name(), orderExists, gameVersion, gameVersionErr)
		if err != nil {
			return nil, fmt.Errorf("read localization mod identity from %q: %w", dir, err)
		}
		if !active || isProjectOwnedIdentity(entry.Name(), modID) {
			continue
		}
		candidates = append(candidates, modCandidate{
			folder: entry.Name(),
			path:   dir,
			modID:  modID,
			name:   name,
			pak:    pakPath,
		})
	}

	if !orderExists {
		sort.Slice(candidates, func(i, j int) bool {
			left := strings.ToLower(candidates[i].folder)
			right := strings.ToLower(candidates[j].folder)
			if left == right {
				return candidates[i].folder < candidates[j].folder
			}
			return left < right
		})
		return candidates, nil
	}

	byModID := make(map[string]modCandidate, len(candidates))
	for _, candidate := range candidates {
		key := strings.ToLower(strings.TrimSpace(candidate.modID))
		if key == "" {
			return nil, fmt.Errorf("localization mod %q has no explicit mod ID required by %s", candidate.folder, modinstall.ModOrderFilename)
		}
		if previous, duplicate := byModID[key]; duplicate && previous.path != candidate.path {
			return nil, fmt.Errorf("ambiguous localization mod ID %q between %q and %q", candidate.modID, previous.path, candidate.path)
		}
		byModID[key] = candidate
	}

	ordered := make([]modCandidate, 0, len(candidates))
	seen := make(map[string]struct{}, len(candidates))
	for _, id := range order {
		candidate, ok := byModID[strings.ToLower(id)]
		if !ok {
			continue
		}
		if _, duplicate := seen[candidate.path]; duplicate {
			continue
		}
		seen[candidate.path] = struct{}{}
		ordered = append(ordered, candidate)
	}
	return ordered, nil
}

func readManifestIdentity(modDir, folder string, requireModID bool, gameVersion string, gameVersionErr error) (modID, name string, active bool, err error) {
	manifestPath := filepath.Join(modDir, "mod.manifest")
	data, err := os.ReadFile(manifestPath)
	if errors.Is(err, os.ErrNotExist) {
		return "", "", false, nil
	}
	if err != nil {
		return "", "", false, err
	}
	var parsed manifest
	if err := xml.Unmarshal(data, &parsed); err != nil {
		// KCD2 requires a valid mod.manifest for a local folder to function
		// properly. Invalid XML cannot be reproduced as an active source.
		return "", "", false, nil
	}
	modID = strings.TrimSpace(parsed.Info.ModID)
	name = strings.TrimSpace(parsed.Info.Name)
	if modID == "" && requireModID {
		return "", name, false, errors.New("manifest omits <modid>; an explicit ID is required to reproduce the active mod_order.txt whitelist safely")
	}
	if modID != "" && !validModID(modID) {
		return "", "", false, nil
	}
	if len(parsed.Supports.Versions) != 0 {
		if gameVersionErr != nil {
			return "", "", false, fmt.Errorf("determine current game version for manifest <supports>: %w", gameVersionErr)
		}
		if strings.TrimSpace(gameVersion) == "" {
			return "", "", false, errors.New("determine current game version for manifest <supports>: wh_sys_version is empty")
		}
		if !supportsGameVersion(parsed.Supports.Versions, gameVersion) {
			return "", "", false, nil
		}
	}
	if name == "" {
		if modID != "" {
			name = modID
		} else {
			name = folder
		}
	}
	return modID, name, true, nil
}

func validModID(modID string) bool {
	if modID == "" {
		return false
	}
	for _, r := range modID {
		if (r < 'a' || r > 'z') && r != '_' {
			return false
		}
	}
	return true
}

func supportsGameVersion(patterns []string, gameVersion string) bool {
	gameVersion = strings.TrimSpace(gameVersion)
	for _, pattern := range patterns {
		if wildcardVersionMatch(strings.TrimSpace(pattern), gameVersion) {
			return true
		}
	}
	return false
}

func wildcardVersionMatch(pattern, value string) bool {
	if pattern == "" {
		return false
	}
	if !strings.Contains(pattern, "*") {
		return pattern == value
	}

	parts := strings.Split(pattern, "*")
	position := 0
	if parts[0] != "" {
		if !strings.HasPrefix(value, parts[0]) {
			return false
		}
		position = len(parts[0])
	}
	for i := 1; i < len(parts)-1; i++ {
		if parts[i] == "" {
			continue
		}
		relative := strings.Index(value[position:], parts[i])
		if relative < 0 {
			return false
		}
		position += relative + len(parts[i])
	}
	last := parts[len(parts)-1]
	if last == "" {
		return true
	}
	return strings.HasSuffix(value[position:], last)
}

func readGameVersion(gameRoot string) (string, error) {
	configPath := filepath.Join(gameRoot, "system.cfg")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("read %q: %w", configPath, err)
	}

	var version string
	for _, rawLine := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "--") || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(key) != "wh_sys_version" {
			continue
		}
		value = strings.TrimSpace(value)
		if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
			value = value[1 : len(value)-1]
		}
		value = strings.TrimSpace(value)
		if value == "" {
			return "", fmt.Errorf("parse %q: wh_sys_version is empty", configPath)
		}
		if version != "" && version != value {
			return "", fmt.Errorf("parse %q: conflicting wh_sys_version values %q and %q", configPath, version, value)
		}
		version = value
	}
	if version == "" {
		return "", fmt.Errorf("parse %q: wh_sys_version not found", configPath)
	}
	return version, nil
}

func readModOrder(modsRoot string) ([]string, bool, error) {
	orderPath := filepath.Join(modsRoot, modinstall.ModOrderFilename)
	data, err := os.ReadFile(orderPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("read mod load order %q: %w", orderPath, err)
	}

	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	order := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		order = append(order, line)
	}
	return order, true, nil
}

func overlayLocalizationPAK(base []localization.DialogueRow, pakPath string) ([]localization.DialogueRow, bool, error) {
	reader, err := zip.OpenReader(pakPath)
	if err != nil {
		return nil, false, fmt.Errorf("open localization PAK: %w", err)
	}
	defer reader.Close()

	resources, err := supportedLocalizationResources(reader.File)
	if err != nil {
		return nil, false, err
	}
	if len(resources) == 0 {
		return base, false, nil
	}

	rows := append([]localization.DialogueRow(nil), base...)
	index := make(map[string]int, len(rows))
	for i, row := range rows {
		index[row.ID] = i
	}
	for _, resource := range resources {
		data, err := readZipEntry(resource.file)
		if err != nil {
			return nil, false, fmt.Errorf("read %q: %w", resource.file.Name, err)
		}
		patchRows, err := localization.ParseDialogueXML(data)
		if err != nil {
			return nil, false, fmt.Errorf("parse %q: %w", resource.file.Name, err)
		}
		if err := requireUniqueIDs(patchRows); err != nil {
			return nil, false, fmt.Errorf("validate %q: %w", resource.file.Name, err)
		}

		for _, row := range patchRows {
			if i, ok := index[row.ID]; ok {
				rows[i] = row
				continue
			}
			// Generic text_ui__*.xml resources can also contain items/menu/UI
			// localization. New IDs are therefore accepted only from an explicit
			// dialogue table; patch resources may modify IDs already proven to be
			// dialogue by stock or an earlier explicit dialogue table.
			if !resource.dialogue {
				continue
			}
			index[row.ID] = len(rows)
			rows = append(rows, row)
		}
	}
	return rows, !dialogueRowsEqual(base, rows), nil
}

func supportedLocalizationResources(files []*zip.File) ([]localizationResource, error) {
	resources := make([]localizationResource, 0)
	seen := make(map[string]string)
	for _, file := range files {
		name := normalizedArchiveName(file.Name)
		if name == "." || strings.Contains(name, "/") {
			continue
		}
		dialogue := strings.EqualFold(name, localization.DialogueXMLArchivePath)
		if !dialogue && !isTextUIPatch(name) {
			continue
		}
		canonical := strings.ToLower(name)
		if previous, duplicate := seen[canonical]; duplicate {
			return nil, fmt.Errorf("duplicate localization resource %q conflicts with %q", file.Name, previous)
		}
		seen[canonical] = file.Name
		resources = append(resources, localizationResource{file: file, name: name, dialogue: dialogue})
	}

	// An explicit text_ui_dialog.xml is the base layer inside one localization
	// PAK. Generic patch resources follow in deterministic case-insensitive name
	// order. Case-fold duplicates were rejected above, so the ordering is total.
	sort.Slice(resources, func(i, j int) bool {
		if resources[i].dialogue != resources[j].dialogue {
			return resources[i].dialogue
		}
		left := strings.ToLower(resources[i].name)
		right := strings.ToLower(resources[j].name)
		if left == right {
			return resources[i].name < resources[j].name
		}
		return left < right
	})
	return resources, nil
}

func dialogueRowsEqual(left, right []localization.DialogueRow) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func normalizedArchiveName(name string) string {
	return path.Clean(strings.ReplaceAll(name, "\\", "/"))
}

func isTextUIPatch(name string) bool {
	lower := strings.ToLower(path.Base(name))
	return strings.HasPrefix(lower, "text_ui__") && strings.HasSuffix(lower, ".xml")
}

func readZipEntry(file *zip.File) ([]byte, error) {
	return readZipEntryLimited(file, maxLocalizationResourceBytes)
}

func readZipEntryLimited(file *zip.File, limit int64) ([]byte, error) {
	if limit <= 0 {
		return nil, errors.New("localization resource size limit must be positive")
	}
	if file.UncompressedSize64 > uint64(limit) {
		return nil, fmt.Errorf("localization resource exceeds %d-byte size limit", limit)
	}
	entry, err := file.Open()
	if err != nil {
		return nil, err
	}
	data, readErr := io.ReadAll(io.LimitReader(entry, limit+1))
	closeErr := entry.Close()
	if readErr != nil {
		return nil, readErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("localization resource exceeds %d-byte size limit", limit)
	}
	return data, nil
}

func requireUniqueIDs(rows []localization.DialogueRow) error {
	seen := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		if _, duplicate := seen[row.ID]; duplicate {
			return fmt.Errorf("duplicate dialogue ID %q", row.ID)
		}
		seen[row.ID] = struct{}{}
	}
	return nil
}

func isProjectOwnedFolder(folder string) bool {
	lower := strings.ToLower(strings.TrimSpace(folder))
	return lower == modarchive.ModID || strings.HasPrefix(lower, "."+modarchive.ModID+".") || strings.HasPrefix(lower, "."+modarchive.ModID+"-")
}

func isProjectOwnedIdentity(folder, modID string) bool {
	return isProjectOwnedFolder(folder) || strings.EqualFold(strings.TrimSpace(modID), modarchive.ModID)
}
