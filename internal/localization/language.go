package localization

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
}

var supportedLanguages = []LanguageInfo{
	{Language: English, PakFilename: "English_xml.pak"},
	{Language: Russian, PakFilename: "Russian_xml.pak"},
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
