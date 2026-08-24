package localization

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Language identifies a supported KCD2 localization language.
type Language string

const (
	English            Language = "English"
	Italian            Language = "Italian"
	French             Language = "French"
	German             Language = "German"
	Spanish            Language = "Spanish"
	Czech              Language = "Czech"
	Japanese           Language = "Japanese"
	Korean             Language = "Korean"
	Polish             Language = "Polish"
	Portuguese         Language = "Portuguese (Brazil)"
	ChineseSimplified  Language = "Chinese (Simplified)"
	ChineseTraditional Language = "Chinese (Traditional)"
	Turkish            Language = "Turkish"
	Russian            Language = "Russian"
	Ukrainian          Language = "Ukrainian"
	Vietnamese         Language = "Vietnamese"
)

// LanguageInfo contains the game-facing metadata for a supported language.
type LanguageInfo struct {
	Language    Language
	PakFilename string
	SubtitleTag string
}

// Keep this registry explicit: game-facing localization filenames and compact
// subtitle tags are a compatibility contract. Unknown future *_xml.pak files
// must not silently receive invented metadata.
var supportedLanguages = []LanguageInfo{
	{Language: English, PakFilename: "English_xml.pak", SubtitleTag: "EN"},
	{Language: Italian, PakFilename: "Italian_xml.pak", SubtitleTag: "IT"},
	{Language: French, PakFilename: "French_xml.pak", SubtitleTag: "FR"},
	{Language: German, PakFilename: "German_xml.pak", SubtitleTag: "DE"},
	{Language: Spanish, PakFilename: "Spanish_xml.pak", SubtitleTag: "ES"},
	{Language: Czech, PakFilename: "Czech_xml.pak", SubtitleTag: "CS"},
	{Language: Japanese, PakFilename: "Japanese_xml.pak", SubtitleTag: "JA"},
	{Language: Korean, PakFilename: "Korean_xml.pak", SubtitleTag: "KO"},
	{Language: Polish, PakFilename: "Polish_xml.pak", SubtitleTag: "PL"},
	{Language: Portuguese, PakFilename: "Portuguese_xml.pak", SubtitleTag: "PT"},
	{Language: ChineseSimplified, PakFilename: "Chineses_xml.pak", SubtitleTag: "ZH-S"},
	{Language: ChineseTraditional, PakFilename: "Chineset_xml.pak", SubtitleTag: "ZH-T"},
	{Language: Turkish, PakFilename: "Turkish_xml.pak", SubtitleTag: "TR"},
	{Language: Russian, PakFilename: "Russian_xml.pak", SubtitleTag: "RU"},
	{Language: Ukrainian, PakFilename: "Ukrainian_xml.pak", SubtitleTag: "UK"},
	{Language: Vietnamese, PakFilename: "Vietnamese_xml.pak", SubtitleTag: "VI"},
}

// SupportedLanguages returns the supported languages in stable display order.
func SupportedLanguages() []LanguageInfo {
	languages := make([]LanguageInfo, len(supportedLanguages))
	copy(languages, supportedLanguages)
	return languages
}

// InstalledLanguages returns only registry languages whose localization PAK is
// present in the selected KCD2 Content root. Unknown PAKs are ignored so
// generation never invents game-facing tags or filenames.
func InstalledLanguages(gameRoot string) ([]LanguageInfo, error) {
	root := strings.TrimSpace(gameRoot)
	if root == "" {
		return nil, nil
	}

	localizationDir := filepath.Join(root, "Localization")
	languages := make([]LanguageInfo, 0, len(supportedLanguages))
	for _, info := range supportedLanguages {
		path := filepath.Join(localizationDir, info.PakFilename)
		stat, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("inspect localization PAK %q: %w", path, err)
		}
		if stat.IsDir() {
			return nil, fmt.Errorf("localization PAK path is a directory: %q", path)
		}
		languages = append(languages, info)
	}
	return languages, nil
}

// LookupLanguage returns metadata for a supported language.
func LookupLanguage(language Language) (LanguageInfo, bool) {
	for _, info := range supportedLanguages {
		if info.Language == language {
			return info, true
		}
	}

	return LanguageInfo{}, false
}

// ParseLanguage resolves a user-facing language name case-insensitively.
func ParseLanguage(value string) (Language, bool) {
	value = strings.TrimSpace(value)
	for _, info := range supportedLanguages {
		if strings.EqualFold(value, string(info.Language)) {
			return info.Language, true
		}
	}

	// Accept the store-facing names for regional labels while keeping one
	// canonical Language value throughout the generator and GUI.
	switch {
	case strings.EqualFold(value, "Portuguese - Brazil"):
		return Portuguese, true
	case strings.EqualFold(value, "Simplified Chinese"):
		return ChineseSimplified, true
	case strings.EqualFold(value, "Traditional Chinese"):
		return ChineseTraditional, true
	case strings.EqualFold(value, "Spanish - Spain"):
		return Spanish, true
	}

	return "", false
}
