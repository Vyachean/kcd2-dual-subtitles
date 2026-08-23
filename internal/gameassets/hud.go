package gameassets

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

const (
	GameDataPAKRelativePath = "Data/IPL_GameData.pak"
	HUDArchivePath          = "Libs/UI/hud.gfx"
)

var (
	ErrHUDNotFound  = errors.New("hud.gfx not found")
	ErrHUDDuplicate = errors.New("duplicate hud.gfx entry")
)

// ReadHUD reads the retail HUD from the user's installed IPL_GameData.pak.
// The caller owns any derived output; this function never modifies game files.
func ReadHUD(gameRoot string) ([]byte, error) {
	pakPath := filepath.Join(gameRoot, filepath.FromSlash(GameDataPAKRelativePath))
	reader, err := zip.OpenReader(pakPath)
	if err != nil {
		return nil, fmt.Errorf("open game data PAK %q: %w", pakPath, err)
	}

	data, readErr := readHUDFromArchive(reader.File, pakPath)
	closeErr := reader.Close()
	if readErr != nil {
		if closeErr != nil {
			return nil, errors.Join(readErr, fmt.Errorf("close game data PAK %q: %w", pakPath, closeErr))
		}
		return nil, readErr
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close game data PAK %q: %w", pakPath, closeErr)
	}
	return data, nil
}

func readHUDFromArchive(files []*zip.File, pakPath string) ([]byte, error) {
	var target *zip.File
	for _, file := range files {
		name := strings.ReplaceAll(file.Name, "\\", "/")
		if !strings.EqualFold(name, HUDArchivePath) {
			continue
		}
		if target != nil {
			return nil, fmt.Errorf("%w in game data PAK %q", ErrHUDDuplicate, pakPath)
		}
		target = file
	}
	if target == nil {
		return nil, fmt.Errorf("%w in game data PAK %q", ErrHUDNotFound, pakPath)
	}

	entry, err := target.Open()
	if err != nil {
		return nil, fmt.Errorf("open %s in game data PAK %q: %w", HUDArchivePath, pakPath, err)
	}
	data, readErr := io.ReadAll(entry)
	closeErr := entry.Close()
	if readErr != nil {
		if closeErr != nil {
			return nil, errors.Join(
				fmt.Errorf("read %s from game data PAK %q: %w", HUDArchivePath, pakPath, readErr),
				fmt.Errorf("close %s in game data PAK %q: %w", HUDArchivePath, pakPath, closeErr),
			)
		}
		return nil, fmt.Errorf("read %s from game data PAK %q: %w", HUDArchivePath, pakPath, readErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close %s in game data PAK %q: %w", HUDArchivePath, pakPath, closeErr)
	}
	return data, nil
}
