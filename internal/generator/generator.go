package generator

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/modarchive"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/modinstall"
)

const CanaryPrefix = "[KCD2DS TEST] "

var ErrInvalidRequest = errors.New("invalid generation request")

// Request describes one end-to-end mod generation operation. An empty
// OutputPath selects automatic installation; a non-empty OutputPath writes a
// portable mod ZIP instead. CanaryID enables an explicit acceptance-only
// marker on one existing localization row.
type Request struct {
	GameRoot          string
	MainLanguage      localization.Language
	SecondaryLanguage localization.Language
	OutputPath        string
	Version           string
	CanaryID          string
}

// Result describes a successfully generated mod destination.
type Result struct {
	OutputPath  string
	InstallPath string
	Stats       localization.MergeStats
	PatchRows   int
	CanaryID    string
}

// Generate reads two installed localization PAKs, merges their dialogue rows,
// writes only changed rows as a KCD2 localization patch, then either installs
// the mod for the current Windows user or writes a standalone archive. It never
// modifies base-game files.
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

	patchRows, err := changedRows(mainRows, mergedRows, strings.TrimSpace(request.CanaryID))
	if err != nil {
		return Result{}, err
	}

	version := strings.TrimSpace(request.Version)
	if version == "" {
		version = "dev"
	}

	result := Result{
		Stats:     stats,
		PatchRows: len(patchRows),
		CanaryID:  strings.TrimSpace(request.CanaryID),
	}
	if request.OutputPath != "" {
		if err := modarchive.BuildVersioned(request.OutputPath, request.MainLanguage, patchRows, version); err != nil {
			return Result{}, fmt.Errorf("build mod archive: %w", err)
		}
		result.OutputPath = request.OutputPath
		return result, nil
	}

	installPath, err := modinstall.InstallVersioned(request.MainLanguage, patchRows, version)
	if err != nil {
		return Result{}, fmt.Errorf("install generated mod: %w", err)
	}
	result.InstallPath = installPath
	return result, nil
}

func changedRows(baseRows, mergedRows []localization.DialogueRow, canaryID string) ([]localization.DialogueRow, error) {
	if len(baseRows) != len(mergedRows) {
		return nil, fmt.Errorf("merge invariant violated: base rows=%d merged rows=%d", len(baseRows), len(mergedRows))
	}

	patchRows := make([]localization.DialogueRow, 0, len(mergedRows))
	canaryFound := canaryID == ""
	for i := range mergedRows {
		base := baseRows[i]
		merged := mergedRows[i]
		if base.ID != merged.ID {
			return nil, fmt.Errorf("merge invariant violated at row %d: base ID %q != merged ID %q", i, base.ID, merged.ID)
		}

		if canaryID != "" && merged.ID == canaryID {
			merged.Text = CanaryPrefix + merged.Text
			canaryFound = true
		}
		if merged.Text != base.Text {
			patchRows = append(patchRows, merged)
		}
	}

	if !canaryFound {
		return nil, fmt.Errorf("%w: canary localization ID %q was not found in the main language table", ErrInvalidRequest, canaryID)
	}
	return patchRows, nil
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
