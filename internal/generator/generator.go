package generator

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/modarchive"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/modinstall"
)

var ErrInvalidRequest = errors.New("invalid generation request")

// Request describes one end-to-end mod generation operation. An empty
// OutputPath selects automatic installation; a non-empty OutputPath writes a
// portable mod ZIP instead.
type Request struct {
	GameRoot          string
	MainLanguage      localization.Language
	SecondaryLanguage localization.Language
	OutputPath        string
}

// Result describes a successfully generated mod destination.
type Result struct {
	OutputPath  string
	InstallPath string
	Stats       localization.MergeStats
}

// Generate reads two installed localization PAKs, merges their dialogue rows,
// then either installs the mod for the current Windows user or writes a
// standalone archive. It never modifies base-game files.
func Generate(request Request) (Result, error) {
	mainInfo, secondaryInfo, err := validateRequest(request)
	if err != nil {
		return Result{}, err
	}

	localizationDir := filepath.Join(request.GameRoot, "Localization")
	if err := requireDirectory(localizationDir, "Localization directory"); err != nil {
		return Result{}, err
	}

	mainPAK := filepath.Join(localizationDir, mainInfo.PakFilename)
	secondaryPAK := filepath.Join(localizationDir, secondaryInfo.PakFilename)
	if err := requireFile(mainPAK, fmt.Sprintf("main language PAK (%s)", request.MainLanguage)); err != nil {
		return Result{}, err
	}
	if err := requireFile(secondaryPAK, fmt.Sprintf("secondary language PAK (%s)", request.SecondaryLanguage)); err != nil {
		return Result{}, err
	}

	mainXML, err := localization.ReadDialogueXML(mainPAK)
	if err != nil {
		return Result{}, fmt.Errorf("read main language %s: %w", request.MainLanguage, err)
	}
	secondaryXML, err := localization.ReadDialogueXML(secondaryPAK)
	if err != nil {
		return Result{}, fmt.Errorf("read secondary language %s: %w", request.SecondaryLanguage, err)
	}

	mainRows, err := localization.ParseDialogueXML(mainXML)
	if err != nil {
		return Result{}, fmt.Errorf("parse main language %s: %w", request.MainLanguage, err)
	}
	secondaryRows, err := localization.ParseDialogueXML(secondaryXML)
	if err != nil {
		return Result{}, fmt.Errorf("parse secondary language %s: %w", request.SecondaryLanguage, err)
	}

	mergedRows, stats, err := localization.MergeDialogueRows(mainRows, secondaryRows)
	if err != nil {
		return Result{}, fmt.Errorf("merge dialogue rows: %w", err)
	}

	if request.OutputPath != "" {
		if err := modarchive.Build(request.OutputPath, request.MainLanguage, mergedRows); err != nil {
			return Result{}, fmt.Errorf("build mod archive: %w", err)
		}
		return Result{OutputPath: request.OutputPath, Stats: stats}, nil
	}

	installPath, err := modinstall.Install(request.MainLanguage, mergedRows)
	if err != nil {
		return Result{}, fmt.Errorf("install generated mod: %w", err)
	}
	return Result{InstallPath: installPath, Stats: stats}, nil
}

func validateRequest(request Request) (localization.LanguageInfo, localization.LanguageInfo, error) {
	if request.GameRoot == "" {
		return localization.LanguageInfo{}, localization.LanguageInfo{}, fmt.Errorf("%w: game root is required", ErrInvalidRequest)
	}
	if request.MainLanguage == request.SecondaryLanguage {
		return localization.LanguageInfo{}, localization.LanguageInfo{}, fmt.Errorf("%w: main and secondary languages must differ", ErrInvalidRequest)
	}

	mainInfo, ok := localization.LookupLanguage(request.MainLanguage)
	if !ok {
		return localization.LanguageInfo{}, localization.LanguageInfo{}, fmt.Errorf("%w: unsupported main language %q", ErrInvalidRequest, request.MainLanguage)
	}
	secondaryInfo, ok := localization.LookupLanguage(request.SecondaryLanguage)
	if !ok {
		return localization.LanguageInfo{}, localization.LanguageInfo{}, fmt.Errorf("%w: unsupported secondary language %q", ErrInvalidRequest, request.SecondaryLanguage)
	}

	return mainInfo, secondaryInfo, nil
}

func requireDirectory(path, label string) error {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%w: %s not found at %q", ErrInvalidRequest, label, path)
		}
		return fmt.Errorf("inspect %s %q: %w", label, path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%w: %s is not a directory at %q", ErrInvalidRequest, label, path)
	}
	return nil
}

func requireFile(path, label string) error {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%w: %s not found at %q", ErrInvalidRequest, label, path)
		}
		return fmt.Errorf("inspect %s %q: %w", label, path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("%w: %s is not a file at %q", ErrInvalidRequest, label, path)
	}
	return nil
}
