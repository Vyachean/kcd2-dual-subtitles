package modarchive

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
)

var ErrDirectoryNotEmpty = errors.New("mod output directory is not empty")

// WriteDirectory writes one complete mod tree into an existing empty directory.
// It is intended for an isolated staging directory managed by the installer.
func WriteDirectory(directory string, mainLanguage localization.Language, rows []localization.DialogueRow) error {
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

	localizationDir := filepath.Join(directory, "Localization")
	if err := os.Mkdir(localizationDir, 0o755); err != nil {
		return fmt.Errorf("create localization directory %q: %w", localizationDir, err)
	}
	if err := os.WriteFile(filepath.Join(directory, ManifestFilename), []byte(manifest), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", ManifestFilename, err)
	}
	pakPath := filepath.Join(localizationDir, languageInfo.PakFilename)
	if err := os.WriteFile(pakPath, localizationPAK, 0o644); err != nil {
		return fmt.Errorf("write localization PAK %q: %w", pakPath, err)
	}
	return nil
}
