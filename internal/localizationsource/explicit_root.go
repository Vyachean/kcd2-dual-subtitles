package localizationsource

import (
	"fmt"
	"path/filepath"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/modinstall"
)

// ResolveWithModsRoot builds one effective localization using an explicit Mods
// root selected by the application. Stock language data still comes from the
// selected game root; only the active mod environment is overridden.
func ResolveWithModsRoot(gameRoot, modsRoot string, language localization.Language) (Result, error) {
	info, ok := localization.LookupLanguage(language)
	if !ok {
		return Result{}, fmt.Errorf("unsupported localization language %q", language)
	}
	location, err := modinstall.ValidateCustomModsRoot(modsRoot)
	if err != nil {
		return Result{}, fmt.Errorf("validate localization mod root: %w", err)
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

	gameVersion, gameVersionErr := readGameVersion(gameRoot)
	return resolveFromModsRootWithVersion(stockRows, location.ModsRoot, info.PakFilename, gameVersion, gameVersionErr)
}
