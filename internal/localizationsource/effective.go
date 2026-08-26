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

type genericDialogueValue struct {
	resource string
	text     string
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
		updated, used, err := overlayLocalizationPAK(result.Rows, candidate.pak, candidate.modID)
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
	activeOrderIDs := make(map[string]struct{}, len(order))
	if orderExists {
		for _, id := range order {
			activeOrderIDs[id] = struct{}{}
		}
	}

	candidates := make([]modCandidate, 0)
	for _, entry := range entries {
		if !entry.IsDir() || isProjectOwnedFolder(entry.Name()) {
			continue
		}
		dir := filepath.Join(modsRoot, entry.Name())
		modID, name, active, err := readManifestIdentity(dir, entry.Name(), orderExists, activeOrderIDs, gameVersion, gameVersionErr)
		if err != nil {
			return nil, fmt.Errorf("read localization mod identity from %q: %w", dir, err)
		}
		if !active || isProjectOwnedIdentity(entry.Name(), modID) {
			continue
		}

		pakPath := filepath.Join(dir, "Localization", pakFilename)
		info, err := os.Lstat(pakPath)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("inspect active localization PAK %q: %w", pakPath, err)
		}
		if orderExists && modID == "" {
			return nil, fmt.Errorf("localization mod %q has no explicit mod ID required by %s", entry.Name(), modinstall.ModOrderFilename)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return nil, fmt.Errorf("active localization PAK is not a regular file: %q", pakPath)
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

	byModID := make(map[string][]modCandidate, len(candidates))
	for _, candidate := range candidates {
		byModID[candidate.modID] = append(byModID[candidate.modID], candidate)
	}

	ordered := make([]modCandidate, 0, len(candidates))
	seen := make(map[string]struct{}, len(candidates))
	for _, id := range order {
		matches := byModID[id]
		if len(matches) == 0 {
			continue
		}
		if len(matches) > 1 {
			return nil, fmt.Errorf("ambiguous active localization mod ID %q between %q and %q", id, matches[0].path, matches[1].path)
		}
		candidate := matches[0]
		if _, duplicate := seen[candidate.path]; duplicate {
			return nil, fmt.Errorf("duplicate active localization mod ID %q in %s", id, modinstall.ModOrderFilename)
		}
		seen[candidate.path] = struct{}{}
		ordered = append(ordered, candidate)
	}
	return ordered, nil
}

func readManifestIdentity(modDir, folder string, requireModID bool, activeOrderIDs map[string]struct{}, gameVersion string, gameVersionErr error) (modID, name string, active bool, err error) {
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
	if modID == "" && name == "" {
		return "", "", false, nil
	}
	if modID != "" && !validModID(modID) {
		return "", "", false, nil
	}
	if requireModID {
		if modID == "" {
			// KCD2 can generate an ID from <name>, but its normalization is not
			// documented. Defer the fail-closed decision until the caller proves
			// this mod actually contains the selected language PAK.
			return "", name, true, nil
		}
		if _, listed := activeOrderIDs[modID]; !listed {
			return modID, name, false, nil
		}
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
		name = modID
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
	for index, line := range lines {
		line = strings.TrimSpace(line)
		if index == 0 {
			line = strings.TrimPrefix(line, "\uFEFF")
		}
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		order = append(order, line)
	}
	return order, true, nil
}

func overlayLocalizationPAK(base []localization.DialogueRow, pakPath, modID string) ([]localization.DialogueRow, bool, error) {
	reader, err := zip.OpenReader(pakPath)
	if err != nil {
		return nil, false, fmt.Errorf("open localization PAK: %w", err)
	}
	defer reader.Close()

	resources, err := supportedLocalizationResources(reader.File, modID)
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
	genericValues := make(map[string]genericDialogueValue)

	for _, resource := range resources {
		data, err := readZipEntry(resource.file)
		if err != nil {
			return nil, false, fmt.Errorf("read %q: %w", resource.file.Name, err)
		}
		patchRows, err := localization.ParseDialogueXML(data)
		if err != nil {
			return nil, false, fmt.Errorf("parse %q: %w", resource.file.Name, err)
		}

		if resource.dialogue {
			// An explicit dialogue table can introduce new dialogue IDs. Duplicate
			// IDs there are ambiguous and rejected.
			if err := requireUniqueIDs(patchRows); err != nil {
				return nil, false, fmt.Errorf("validate %q: %w", resource.file.Name, err)
			}
			for _, row := range patchRows {
				if i, ok := index[row.ID]; ok {
					if rows[i].Text != row.Text {
						rows[i] = row
					}
					continue
				}
				index[row.ID] = len(rows)
				rows = append(rows, row)
			}
			continue
		}

		// A generic localization resource may repeat a key internally; the last
		// occurrence in that resource is its effective value. Unknown IDs are not
		// admitted because the same file can contain UI/items/quest strings.
		finalRows := make(map[string]localization.DialogueRow)
		for _, row := range patchRows {
			if _, known := index[row.ID]; known {
				finalRows[row.ID] = row
			}
		}
		ids := make([]string, 0, len(finalRows))
		for id := range finalRows {
			ids = append(ids, id)
		}
		sort.Strings(ids)

		for _, id := range ids {
			row := finalRows[id]
			if previous, seen := genericValues[id]; seen && previous.text != row.Text {
				return nil, false, fmt.Errorf(
					"dialogue ID %q has conflicting values in generic localization resources %q and %q; KCD2 cross-resource override order is undocumented",
					id,
					previous.resource,
					resource.file.Name,
				)
			}
			if _, seen := genericValues[id]; !seen {
				genericValues[id] = genericDialogueValue{resource: resource.file.Name, text: row.Text}
			}

			i := index[id]
			// Warhorse documents the second cell as irrelevant to displayed text.
			// Preserve the inherited row when only that non-display field differs.
			if rows[i].Text != row.Text {
				rows[i] = row
			}
		}
	}
	return rows, !dialogueRowsEqual(base, rows), nil
}

func supportedLocalizationResources(files []*zip.File, modID string) ([]localizationResource, error) {
	resources := make([]localizationResource, 0)
	seen := make(map[string]string)
	for _, file := range files {
		name := normalizedArchiveName(file.Name)
		if name == "." || strings.Contains(name, "/") {
			continue
		}
		dialogue := strings.EqualFold(name, localization.DialogueXMLArchivePath)
		if !dialogue && !isGenericLocalizationPatch(name, modID) {
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
	// PAK. Generic resources are sorted only to make parsing/errors deterministic.
	// If two generic resources disagree on one dialogue ID, overlay fails closed
	// instead of pretending that this sort order is KCD2's undocumented winner.
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
		if left[i].ID != right[i].ID || left[i].Text != right[i].Text {
			return false
		}
	}
	return true
}

func normalizedArchiveName(name string) string {
	return path.Clean(strings.ReplaceAll(name, "\\", "/"))
}

func isGenericLocalizationPatch(name, modID string) bool {
	lower := strings.ToLower(path.Base(name))
	if !strings.HasSuffix(lower, ".xml") {
		return false
	}
	stem := strings.TrimSuffix(lower, ".xml")
	expectedModID := strings.ToLower(strings.TrimSpace(modID))
	if expectedModID != "" {
		return strings.HasSuffix(stem, "_"+expectedModID)
	}

	// Warhorse can auto-generate a missing manifest modid from the human name,
	// but its exact normalization is undocumented. Without an explicit ID we
	// therefore accept the documented syntactic anything_<modid>.xml shape and
	// still constrain its contents to already-known dialogue IDs.
	return strings.Contains(stem, "_")
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
