package localization

import "strings"

// Language identifies a supported KCD2 localization language.
type Language string

const (
	English Language = "English"
	Russian Language = "Russian"
)

// LanguageInfo contains the game-facing metadata for a supported language.
type LanguageInfo struct {
	Language    Language
	PakFilename string
	SubtitleTag string
}

var supportedLanguages = []LanguageInfo{
	{Language: English, PakFilename: "English_xml.pak", SubtitleTag: "EN"},
	{Language: Russian, PakFilename: "Russian_xml.pak", SubtitleTag: "RU"},
}

// SupportedLanguages returns the supported languages in stable display order.
func SupportedLanguages() []LanguageInfo {
	languages := make([]LanguageInfo, len(supportedLanguages))
	copy(languages, supportedLanguages)
	return languages
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

	return "", false
}
