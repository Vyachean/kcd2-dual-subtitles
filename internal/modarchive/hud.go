package modarchive

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
)

const (
	DataPAKFilename = ModID + ".pak"
	HUDArchivePath  = "Libs/UI/hud.gfx"
)

var ErrHUDRequired = errors.New("derived HUD is required")

// BuildVersionedWithHUD writes the explicit experimental HUD variant. Existing
// BuildVersioned callers remain localization-only and therefore cannot
// accidentally start overriding Libs/UI/hud.gfx.
func BuildVersionedWithHUD(outputPath string, mainLanguage localization.Language, rows []localization.DialogueRow, hud []byte, version string) error {
	if len(hud) == 0 {
		return ErrHUDRequired
	}
	languageInfo, ok := localization.LookupLanguage(mainLanguage)
	if !ok {
		return fmt.Errorf("%w: %q", ErrUnsupportedLanguage, mainLanguage)
	}
	localizationPAK, err := buildLocalizationPAK(rows)
	if err != nil {
		return fmt.Errorf("build localization PAK: %w", err)
	}
	dataPAK, err := buildHUDPAK(hud)
	if err != nil {
		return fmt.Errorf("build HUD data PAK: %w", err)
	}
	archiveData, err := buildZip([]archiveEntry{
		{name: modArchivePath(ManifestFilename), data: manifestForVersion(version)},
		{name: modArchivePath(filepath.ToSlash(filepath.Join("Localization", languageInfo.PakFilename))), data: localizationPAK},
		{name: modArchivePath(filepath.ToSlash(filepath.Join("Data", DataPAKFilename))), data: dataPAK},
	}, 8) // archive/zip.Deflate; numeric form avoids widening builder.go's API.
	if err != nil {
		return err
	}
	return publishArchive(outputPath, archiveData)
}

// WriteDirectoryVersionedWithHUD writes the experimental HUD variant into an
// installer-owned empty staging directory.
func WriteDirectoryVersionedWithHUD(directory string, mainLanguage localization.Language, rows []localization.DialogueRow, hud []byte, version string) error {
	if len(hud) == 0 {
		return ErrHUDRequired
	}
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

	languageInfo, ok := localization.LookupLanguage(mainLanguage)
	if !ok {
		return fmt.Errorf("%w: %q", ErrUnsupportedLanguage, mainLanguage)
	}
	localizationPAK, err := buildLocalizationPAK(rows)
	if err != nil {
		return fmt.Errorf("build localization PAK: %w", err)
	}
	dataPAK, err := buildHUDPAK(hud)
	if err != nil {
		return fmt.Errorf("build HUD data PAK: %w", err)
	}

	localizationDir := filepath.Join(directory, "Localization")
	dataDir := filepath.Join(directory, "Data")
	if err := os.Mkdir(localizationDir, 0o755); err != nil {
		return fmt.Errorf("create localization directory %q: %w", localizationDir, err)
	}
	if err := os.Mkdir(dataDir, 0o755); err != nil {
		return fmt.Errorf("create data directory %q: %w", dataDir, err)
	}
	if err := os.WriteFile(filepath.Join(directory, ManifestFilename), manifestForVersion(version), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", ManifestFilename, err)
	}
	if err := os.WriteFile(filepath.Join(localizationDir, languageInfo.PakFilename), localizationPAK, 0o644); err != nil {
		return fmt.Errorf("write localization PAK: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, DataPAKFilename), dataPAK, 0o644); err != nil {
		return fmt.Errorf("write HUD data PAK: %w", err)
	}
	return nil
}

func buildHUDPAK(hud []byte) ([]byte, error) {
	if len(hud) == 0 {
		return nil, ErrHUDRequired
	}
	return buildCryPak([]archiveEntry{{name: HUDArchivePath, data: hud}})
}
