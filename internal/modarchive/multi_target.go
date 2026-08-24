package modarchive

import (
	"archive/zip"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
)

var ErrTargetLanguageRequired = errors.New("at least one target localization language is required")

// BuildVersionedForLanguages writes one generated localization payload under
// every supplied game-facing localization PAK name. This deliberately
// separates the languages used as text sources from the localization slot that
// the currently active game language loads.
func BuildVersionedForLanguages(outputPath string, targetLanguages []localization.Language, rows []localization.DialogueRow, version string) error {
	infos, err := targetLanguageInfos(targetLanguages)
	if err != nil {
		return err
	}
	localizationPAK, err := buildLocalizationPAK(rows)
	if err != nil {
		return fmt.Errorf("build localization PAK: %w", err)
	}

	entries := []archiveEntry{{name: modArchivePath(ManifestFilename), data: manifestForVersion(version)}}
	entries = append(entries, localizationTargetEntries(infos, localizationPAK)...)
	archiveData, err := buildZip(entries, zip.Deflate)
	if err != nil {
		return err
	}
	return publishArchive(outputPath, archiveData)
}

// BuildVersionedWithHUDForLanguages is the HUD equivalent of
// BuildVersionedForLanguages. The same generated dialogue patch is available
// regardless of which installed game localization is active.
func BuildVersionedWithHUDForLanguages(outputPath string, targetLanguages []localization.Language, rows []localization.DialogueRow, hud []byte, version string) error {
	if len(hud) == 0 {
		return ErrHUDRequired
	}
	infos, err := targetLanguageInfos(targetLanguages)
	if err != nil {
		return err
	}
	localizationPAK, err := buildLocalizationPAK(rows)
	if err != nil {
		return fmt.Errorf("build localization PAK: %w", err)
	}
	dataPAK, err := buildHUDPAK(hud)
	if err != nil {
		return fmt.Errorf("build HUD data PAK: %w", err)
	}

	entries := []archiveEntry{{name: modArchivePath(ManifestFilename), data: manifestForVersion(version)}}
	entries = append(entries, localizationTargetEntries(infos, localizationPAK)...)
	entries = append(entries, archiveEntry{name: modArchivePath(filepath.ToSlash(filepath.Join("Data", DataPAKFilename))), data: dataPAK})
	archiveData, err := buildZip(entries, zip.Deflate)
	if err != nil {
		return err
	}
	return publishArchive(outputPath, archiveData)
}

// WriteDirectoryVersionedForLanguages writes the localization-only variant
// into an installer-owned empty staging directory.
func WriteDirectoryVersionedForLanguages(directory string, targetLanguages []localization.Language, rows []localization.DialogueRow, version string) error {
	return writeDirectoryVersionedForLanguages(directory, targetLanguages, rows, nil, version)
}

// WriteDirectoryVersionedWithHUDForLanguages writes the HUD variant into an
// installer-owned empty staging directory.
func WriteDirectoryVersionedWithHUDForLanguages(directory string, targetLanguages []localization.Language, rows []localization.DialogueRow, hud []byte, version string) error {
	if len(hud) == 0 {
		return ErrHUDRequired
	}
	return writeDirectoryVersionedForLanguages(directory, targetLanguages, rows, hud, version)
}

func writeDirectoryVersionedForLanguages(directory string, targetLanguages []localization.Language, rows []localization.DialogueRow, hud []byte, version string) error {
	info, err := os.Stat(directory)
	if err != nil {
		return fmt.Errorf("inspect mod output directory %q: %w", directory, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("mod output path is not a directory: %q", directory)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return fmt.Errorf("read mod output directory %q: %w", directory, err)
	}
	if len(entries) != 0 {
		return fmt.Errorf("%w: %q", ErrDirectoryNotEmpty, directory)
	}

	infos, err := targetLanguageInfos(targetLanguages)
	if err != nil {
		return err
	}
	localizationPAK, err := buildLocalizationPAK(rows)
	if err != nil {
		return fmt.Errorf("build localization PAK: %w", err)
	}

	localizationDir := filepath.Join(directory, "Localization")
	if err := os.Mkdir(localizationDir, 0o755); err != nil {
		return fmt.Errorf("create localization directory %q: %w", localizationDir, err)
	}
	if err := os.WriteFile(filepath.Join(directory, ManifestFilename), manifestForVersion(version), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", ManifestFilename, err)
	}
	for _, languageInfo := range infos {
		pakPath := filepath.Join(localizationDir, languageInfo.PakFilename)
		if err := os.WriteFile(pakPath, localizationPAK, 0o644); err != nil {
			return fmt.Errorf("write localization PAK %q: %w", pakPath, err)
		}
	}

	if len(hud) == 0 {
		return nil
	}
	dataPAK, err := buildHUDPAK(hud)
	if err != nil {
		return fmt.Errorf("build HUD data PAK: %w", err)
	}
	dataDir := filepath.Join(directory, "Data")
	if err := os.Mkdir(dataDir, 0o755); err != nil {
		return fmt.Errorf("create data directory %q: %w", dataDir, err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, DataPAKFilename), dataPAK, 0o644); err != nil {
		return fmt.Errorf("write HUD data PAK: %w", err)
	}
	return nil
}

func targetLanguageInfos(targetLanguages []localization.Language) ([]localization.LanguageInfo, error) {
	if len(targetLanguages) == 0 {
		return nil, ErrTargetLanguageRequired
	}
	infos := make([]localization.LanguageInfo, 0, len(targetLanguages))
	seen := make(map[localization.Language]bool, len(targetLanguages))
	for _, language := range targetLanguages {
		if seen[language] {
			continue
		}
		info, ok := localization.LookupLanguage(language)
		if !ok {
			return nil, fmt.Errorf("%w: %q", ErrUnsupportedLanguage, language)
		}
		seen[language] = true
		infos = append(infos, info)
	}
	if len(infos) == 0 {
		return nil, ErrTargetLanguageRequired
	}
	return infos, nil
}

func localizationTargetEntries(infos []localization.LanguageInfo, localizationPAK []byte) []archiveEntry {
	entries := make([]archiveEntry, 0, len(infos))
	for _, info := range infos {
		entries = append(entries, archiveEntry{
			name: modArchivePath(filepath.ToSlash(filepath.Join("Localization", info.PakFilename))),
			data: localizationPAK,
		})
	}
	return entries
}
