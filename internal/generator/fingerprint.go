package generator

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/gameassets"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
)

// GenerationFingerprint identifies only source inputs that can affect one
// generated mod. It deliberately hashes extracted dialogue XML and hud.gfx
// bytes rather than timestamps or entire game PAKs.
type GenerationFingerprint struct {
	MainLanguage            localization.Language   `json:"mainLanguage"`
	SecondaryLanguage       localization.Language   `json:"secondaryLanguage"`
	MainDialogueSHA256      string                  `json:"mainDialogueSha256"`
	SecondaryDialogueSHA256 string                  `json:"secondaryDialogueSha256"`
	TargetLanguages         []localization.Language `json:"targetLanguages"`
	StyledHUD               bool                    `json:"styledHud"`
	RetailHUDSHA256         string                  `json:"retailHudSha256,omitempty"`
}

func fingerprintFromDialogueInputs(main, secondary localization.Language, mainXML, secondaryXML []byte, targets []localization.Language, styled bool) GenerationFingerprint {
	return GenerationFingerprint{
		MainLanguage:            main,
		SecondaryLanguage:       secondary,
		MainDialogueSHA256:      sha256Hex(mainXML),
		SecondaryDialogueSHA256: sha256Hex(secondaryXML),
		TargetLanguages:         append([]localization.Language(nil), targets...),
		StyledHUD:               styled,
	}
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// InspectGenerationFingerprint computes the current fingerprint without
// generating or installing a mod. It is used by the GUI to classify an
// existing generated installation as fresh or stale after KCD2 updates.
func InspectGenerationFingerprint(request Request) (GenerationFingerprint, error) {
	mainInfo, secondaryInfo, err := validateRequest(request)
	if err != nil {
		return GenerationFingerprint{}, err
	}
	style, err := normalizeSubtitleStyle(request.SubtitleStyle)
	if err != nil {
		return GenerationFingerprint{}, err
	}

	installedLanguages, err := localization.InstalledLanguages(request.GameRoot)
	if err != nil {
		return GenerationFingerprint{}, fmt.Errorf("discover installed localization languages: %w", err)
	}
	targetLanguages := make([]localization.Language, 0, len(installedLanguages))
	for _, info := range installedLanguages {
		targetLanguages = append(targetLanguages, info.Language)
	}

	localizationDir := filepath.Join(request.GameRoot, "Localization")
	mainXML, err := localization.ReadDialogueXML(filepath.Join(localizationDir, mainInfo.PakFilename))
	if err != nil {
		return GenerationFingerprint{}, fmt.Errorf("read main language %s: %w", request.MainLanguage, err)
	}
	secondaryXML, err := localization.ReadDialogueXML(filepath.Join(localizationDir, secondaryInfo.PakFilename))
	if err != nil {
		return GenerationFingerprint{}, fmt.Errorf("read secondary language %s: %w", request.SecondaryLanguage, err)
	}

	fingerprint := fingerprintFromDialogueInputs(request.MainLanguage, request.SecondaryLanguage, mainXML, secondaryXML, targetLanguages, style == SubtitleStyleHUD)
	if style == SubtitleStyleHUD {
		hud, err := gameassets.ReadHUD(request.GameRoot)
		if err != nil {
			return GenerationFingerprint{}, fmt.Errorf("read retail HUD from %s: %w", gameassets.GameDataPAKRelativePath, err)
		}
		fingerprint.RetailHUDSHA256 = sha256Hex(hud)
	}
	return fingerprint, nil
}
