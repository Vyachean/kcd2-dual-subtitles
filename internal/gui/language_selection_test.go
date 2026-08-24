package gui

import (
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
)

func TestPreferredLanguageIndexesUsesEnglishNeutralDefault(t *testing.T) {
	languages := []localization.LanguageInfo{
		{Language: localization.English},
		{Language: localization.German},
		{Language: localization.Russian},
	}
	main, secondary := preferredLanguageIndexes(languages, "", "")
	if main != 0 || secondary != 1 {
		t.Fatalf("indexes = %d, %d; want EN=0 and first distinct language=1", main, secondary)
	}
}

func TestPreferredLanguageIndexesDoesNotPreferRussian(t *testing.T) {
	languages := []localization.LanguageInfo{
		{Language: localization.German},
		{Language: localization.Russian},
		{Language: localization.French},
	}
	main, secondary := preferredLanguageIndexes(languages, "", "")
	if main != 0 || secondary != 1 {
		t.Fatalf("indexes = %d, %d; want first two installed languages 0,1", main, secondary)
	}
}

func TestPreferredLanguageIndexesPreservesAvailablePreviousChoices(t *testing.T) {
	languages := []localization.LanguageInfo{
		{Language: localization.English},
		{Language: localization.German},
		{Language: localization.Russian},
	}
	main, secondary := preferredLanguageIndexes(languages, localization.German, localization.Russian)
	if main != 1 || secondary != 2 {
		t.Fatalf("indexes = %d, %d; want previous choices 1,2", main, secondary)
	}
}

func TestPreferredLanguageIndexesFallsBackToDistinctLanguages(t *testing.T) {
	languages := []localization.LanguageInfo{
		{Language: localization.German},
		{Language: localization.French},
	}
	main, secondary := preferredLanguageIndexes(languages, "", "")
	if main != 0 || secondary != 1 {
		t.Fatalf("indexes = %d, %d; want first two distinct languages", main, secondary)
	}
}

func TestPreferredLanguageIndexesHandlesInsufficientLanguages(t *testing.T) {
	main, secondary := preferredLanguageIndexes(nil, "", "")
	if main != -1 || secondary != -1 {
		t.Fatalf("empty indexes = %d, %d", main, secondary)
	}
	main, secondary = preferredLanguageIndexes([]localization.LanguageInfo{{Language: localization.English}}, "", "")
	if main != 0 || secondary != -1 {
		t.Fatalf("single-language indexes = %d, %d", main, secondary)
	}
}
