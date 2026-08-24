package generator

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/gameassets"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/gfxpatch"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/modarchive"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/modinstall"
)

const CanaryPrefix = "[KCD2DS TEST] "

var (
	ErrInvalidRequest         = errors.New("invalid generation request")
	readRetailHUD             = gameassets.ReadHUD
	patchRetailHUD            = gfxpatch.PatchHUDDirectHTMLAll
	patchRetailHUDReadability = gfxpatch.PatchHUDDirectHTMLAllWithReadability
)

// Request describes one end-to-end mod generation operation. An empty
// OutputPath selects automatic installation; a non-empty OutputPath writes a
// portable mod ZIP instead. CanaryID enables an explicit acceptance-only
// marker on one existing localization row. An empty SubtitleStyle keeps the
// accepted tagged format for backward compatibility. HUDPresentation is used
// only with SubtitleStyleHUD; nil preserves the live-proven rc.10 defaults.
type Request struct {
	GameRoot          string
	MainLanguage      localization.Language
	SecondaryLanguage localization.Language
	SubtitleStyle     SubtitleStyle
	HUDPresentation   *HUDPresentationConfig
	OutputPath        string
	Version           string
	CanaryID          string
}

// Result describes a successfully generated mod destination.
type Result struct {
	OutputPath          string
	InstallPath         string
	Stats               localization.MergeStats
	PatchRows           int
	CanaryID            string
	SubtitleStyle       SubtitleStyle
	HUDOverride         bool
	LocalizationTargets int
}

// Generate reads installed localization PAKs, merges their dialogue rows and
// writes only changed rows as a KCD2 localization patch. Selected languages are
// text sources only; the resulting patch is published under every supported
// localization PAK present in the selected installation so it remains active
// regardless of the game's current language. The explicit HUD prototype also
// derives a HUD override from the user's installed IPL_GameData.pak. Base-game
// files are never modified.
func Generate(request Request) (Result, error) {
	mainInfo, secondaryInfo, err := validateRequest(request)
	if err != nil {
		return Result{}, err
	}
	style, err := normalizeSubtitleStyle(request.SubtitleStyle)
	if err != nil {
		return Result{}, err
	}
	presentation, err := normalizeHUDPresentation(style, request.HUDPresentation)
	if err != nil {
		return Result{}, err
	}

	localizationDir := filepath.Join(request.GameRoot, "Localization")
	if err := requireDirectory(localizationDir, "Localization directory"); err != nil {
		return Result{}, err
	}

	installedLanguages, err := localization.InstalledLanguages(request.GameRoot)
	if err != nil {
		return Result{}, fmt.Errorf("discover installed localization languages: %w", err)
	}
	targetLanguages := make([]localization.Language, 0, len(installedLanguages))
	for _, info := range installedLanguages {
		targetLanguages = append(targetLanguages, info.Language)
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

	mergedRows, stats, err := mergeRowsForStyle(style, presentation, mainRows, secondaryRows, mainInfo.SubtitleTag, secondaryInfo.SubtitleTag)
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
		Stats:               stats,
		PatchRows:           len(patchRows),
		CanaryID:            strings.TrimSpace(request.CanaryID),
		SubtitleStyle:       style,
		LocalizationTargets: len(targetLanguages),
	}

	var derivedHUD []byte
	if style == SubtitleStyleHUD {
		retailHUD, err := readRetailHUD(request.GameRoot)
		if err != nil {
			return Result{}, fmt.Errorf("read retail HUD from %s: %w", gameassets.GameDataPAKRelativePath, err)
		}
		if presentation.Outline || presentation.Shadow {
			shadowColor, colorErr := hudColorValue(presentation.ShadowColor)
			if colorErr != nil {
				return Result{}, fmt.Errorf("resolve shadow color: %w", colorErr)
			}
			derivedHUD, err = patchRetailHUDReadability(retailHUD, readabilityConfig(presentation, shadowColor))
		} else {
			// Preserve the already retail-proven patch path byte-for-byte when
			// neither experimental readability effect is enabled.
			derivedHUD, err = patchRetailHUD(retailHUD)
		}
		if err != nil {
			return Result{}, fmt.Errorf("derive experimental HUD override: %w", err)
		}
		if len(derivedHUD) == 0 {
			return Result{}, errors.New("derive experimental HUD override: patcher returned an empty HUD")
		}
		result.HUDOverride = true
	}

	if request.OutputPath != "" {
		if result.HUDOverride {
			if err := modarchive.BuildVersionedWithHUDForLanguages(request.OutputPath, targetLanguages, patchRows, derivedHUD, version); err != nil {
				return Result{}, fmt.Errorf("build HUD prototype mod archive: %w", err)
			}
		} else if err := modarchive.BuildVersionedForLanguages(request.OutputPath, targetLanguages, patchRows, version); err != nil {
			return Result{}, fmt.Errorf("build mod archive: %w", err)
		}
		result.OutputPath = request.OutputPath
		return result, nil
	}

	var installPath string
	if result.HUDOverride {
		installPath, err = modinstall.InstallVersionedWithHUDForLanguages(request.GameRoot, targetLanguages, patchRows, derivedHUD, version)
	} else {
		installPath, err = modinstall.InstallVersionedForLanguages(request.GameRoot, targetLanguages, patchRows, version)
	}
	if err != nil {
		return Result{}, fmt.Errorf("install generated mod: %w", err)
	}
	result.InstallPath = installPath
	return result, nil
}

func mergeRowsForStyle(style SubtitleStyle, presentation HUDPresentationConfig, mainRows, secondaryRows []localization.DialogueRow, mainTag, secondaryTag string) ([]localization.DialogueRow, localization.MergeStats, error) {
	switch style {
	case SubtitleStyleTagged:
		return localization.MergeDialogueRowsTagged(mainRows, secondaryRows, mainTag, secondaryTag)
	case SubtitleStyleDifferentiated:
		return localization.MergeDialogueRowsDifferentiated(mainRows, secondaryRows, mainTag, secondaryTag)
	case SubtitleStyleHUD:
		return localization.MergeDialogueRowsHUD(mainRows, secondaryRows, mainTag, secondaryTag, localization.HUDPresentationOptions{
			PrimaryColor:     presentation.PrimaryColor,
			PrimarySize:      presentation.PrimarySize,
			PrimaryItalic:    presentation.PrimaryItalic,
			SecondaryColor:   presentation.SecondaryColor,
			SecondarySize:    presentation.SecondarySize,
			SecondaryItalic:  presentation.SecondaryItalic,
			ShowLanguageTags: presentation.ShowLanguageTags,
		})
	default:
		return nil, localization.MergeStats{}, fmt.Errorf("%w: unsupported subtitle style %q", ErrInvalidRequest, style)
	}
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
