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
	return resolveFromModsRoot(stockRows, location.ModsRoot, info.PakFilename)
}

func resolveFromModsRoot(stockRows []localization.DialogueRow, modsRoot, pakFilename string) (Result, error) {
	rows := append([]localization.DialogueRow(nil), stockRows...)
	candidates, err := activeLocalizationMods(modsRoot, pakFilename)
	if err != nil {
		return Result{}, err
	}

	result := Result{Rows: rows}
	for _, candidate := range candidates {
		updated, used, err := overlayLocalizationPAK(result.Rows, candidate.pak)
		if err != nil {
			return Result{}, fmt.Errorf("apply localization mod %q (%s): %w", candidate.modID, candidate.pak, err)
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

func activeLocalizationMods(modsRoot, pakFilename string) ([]modCandidate, error) {
	entries, err := os.ReadDir(modsRoot)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read mods root %q: %w", modsRoot, err)
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

		modID, name, err := readManifestIdentity(dir, entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read localization mod identity from %q: %w", dir, err)
		}
		if isProjectOwnedIdentity(entry.Name(), modID) {
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

	order, exists, err := readModOrder(modsRoot)
	if err != nil {
		return nil, err
	}
	if !exists {
		sort.Slice(candidates, func(i, j int) bool {
			return strings.ToLower(candidates[i].folder) < strings.ToLower(candidates[j].folder)
		})
		return candidates, nil
	}

	byIdentity := make(map[string]modCandidate, len(candidates)*2)
	for _, candidate := range candidates {
		for _, identity := range []string{candidate.modID, candidate.folder} {
			key := strings.ToLower(strings.TrimSpace(identity))
			if key == "" {
				continue
			}
			if previous, duplicate := byIdentity[key]; duplicate && previous.path != candidate.path {
				return nil, fmt.Errorf("ambiguous localization mod identity %q between %q and %q", identity, previous.path, candidate.path)
			}
			byIdentity[key] = candidate
		}
	}

	ordered := make([]modCandidate, 0, len(candidates))
	seen := make(map[string]struct{}, len(candidates))
	for _, id := range order {
		candidate, ok := byIdentity[strings.ToLower(id)]
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

func readManifestIdentity(modDir, fallback string) (modID, name string, err error) {
	manifestPath := filepath.Join(modDir, "mod.manifest")
	data, err := os.ReadFile(manifestPath)
	if errors.Is(err, os.ErrNotExist) {
		return fallback, fallback, nil
	}
	if err != nil {
		return "", "", err
	}
	var parsed manifest
	if err := xml.Unmarshal(data, &parsed); err != nil {
		return "", "", fmt.Errorf("parse %q: %w", manifestPath, err)
	}
	modID = strings.TrimSpace(parsed.Info.ModID)
	if modID == "" {
		modID = fallback
	}
	name = strings.TrimSpace(parsed.Info.Name)
	if name == "" {
		name = modID
	}
	return modID, name, nil
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

	files := make([]*zip.File, 0)
	for _, file := range reader.File {
		name := path.Clean(strings.ReplaceAll(file.Name, "\\", "/"))
		if strings.Contains(name, "/") {
			continue
		}
		if name == localization.DialogueXMLArchivePath || isTextUIPatch(name) {
			files = append(files, file)
		}
	}
	if len(files) == 0 {
		return base, false, nil
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })

	rows := append([]localization.DialogueRow(nil), base...)
	index := make(map[string]int, len(rows))
	for i, row := range rows {
		index[row.ID] = i
	}
	used := false
	for _, file := range files {
		data, err := readZipEntry(file)
		if err != nil {
			return nil, false, fmt.Errorf("read %q: %w", file.Name, err)
		}
		patchRows, err := localization.ParseDialogueXML(data)
		if err != nil {
			return nil, false, fmt.Errorf("parse %q: %w", file.Name, err)
		}
		if err := requireUniqueIDs(patchRows); err != nil {
			return nil, false, fmt.Errorf("validate %q: %w", file.Name, err)
		}

		allowNew := path.Base(file.Name) == localization.DialogueXMLArchivePath
		for _, row := range patchRows {
			if i, ok := index[row.ID]; ok {
				rows[i] = row
				used = true
				continue
			}
			if !allowNew {
				continue
			}
			index[row.ID] = len(rows)
			rows = append(rows, row)
			used = true
		}
	}
	return rows, used, nil
}

func isTextUIPatch(name string) bool {
	lower := strings.ToLower(path.Base(name))
	return strings.HasPrefix(lower, "text_ui__") && strings.HasSuffix(lower, ".xml")
}

func readZipEntry(file *zip.File) ([]byte, error) {
	entry, err := file.Open()
	if err != nil {
		return nil, err
	}
	data, readErr := io.ReadAll(entry)
	closeErr := entry.Close()
	if readErr != nil {
		return nil, readErr
	}
	if closeErr != nil {
		return nil, closeErr
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
