package gamedetect

import (
	"encoding/binary"
	"errors"
	"path/filepath"
	"strings"
	"unicode/utf16"
)

var errInvalidGamingRoot = errors.New("invalid .GamingRoot data")

const maxGamingRootBytes = 64 * 1024

// parseGamingRoot decodes the current Xbox .GamingRoot marker format: RGBX,
// a little-endian uint32 location count, then that many NUL-terminated UTF-16LE
// paths. It is deliberately strict; callers treat malformed markers as
// best-effort discovery misses and retain the default XboxGames root.
func parseGamingRoot(data []byte, driveRoot string) ([]string, error) {
	if len(data) < 10 || len(data) > maxGamingRootBytes || string(data[:4]) != "RGBX" {
		return nil, errInvalidGamingRoot
	}
	count := binary.LittleEndian.Uint32(data[4:8])
	if count == 0 || count > 64 {
		return nil, errInvalidGamingRoot
	}

	driveRoot = filepath.Clean(driveRoot)
	driveVolume := strings.ToLower(filepath.VolumeName(driveRoot))
	offset := 8
	roots := make([]string, 0, count)

	for i := uint32(0); i < count; i++ {
		if offset >= len(data) {
			return nil, errInvalidGamingRoot
		}

		units := make([]uint16, 0, 64)
		terminated := false
		for offset+1 < len(data) {
			unit := binary.LittleEndian.Uint16(data[offset : offset+2])
			offset += 2
			if unit == 0 {
				terminated = true
				break
			}
			units = append(units, unit)
		}
		if !terminated || len(units) == 0 {
			return nil, errInvalidGamingRoot
		}

		location := strings.TrimSpace(string(utf16.Decode(units)))
		if location == "" {
			continue
		}

		var resolved string
		switch {
		case filepath.IsAbs(location):
			resolved = filepath.Clean(location)
			if volume := strings.ToLower(filepath.VolumeName(resolved)); volume != "" && driveVolume != "" && volume != driveVolume {
				continue
			}
		case strings.HasPrefix(location, `\`):
			resolved = filepath.Clean(filepath.VolumeName(driveRoot) + location)
		default:
			resolved = filepath.Clean(filepath.Join(driveRoot, location))
		}
		roots = append(roots, resolved)
	}

	return roots, nil
}
