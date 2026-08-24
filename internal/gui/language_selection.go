package gui

import "github.com/Vyachean/kcd2-dual-subtitles/internal/localization"

func languageIndex(languages []localization.LanguageInfo, language localization.Language) int {
	for i, info := range languages {
		if info.Language == language {
			return i
		}
	}
	return -1
}

func preferredLanguageIndexes(languages []localization.LanguageInfo, previousMain, previousSecondary localization.Language) (mainIndex, secondaryIndex int) {
	if len(languages) == 0 {
		return -1, -1
	}

	mainIndex = languageIndex(languages, previousMain)
	if mainIndex < 0 {
		mainIndex = languageIndex(languages, localization.Russian)
	}
	if mainIndex < 0 {
		mainIndex = 0
	}

	secondaryIndex = languageIndex(languages, previousSecondary)
	if secondaryIndex == mainIndex {
		secondaryIndex = -1
	}
	if secondaryIndex < 0 {
		english := languageIndex(languages, localization.English)
		if english >= 0 && english != mainIndex {
			secondaryIndex = english
		}
	}
	if secondaryIndex < 0 {
		for i := range languages {
			if i != mainIndex {
				secondaryIndex = i
				break
			}
		}
	}
	return mainIndex, secondaryIndex
}
