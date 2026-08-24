package localization

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLookupLanguageCoversCurrentLocalizationRegistry(t *testing.T) {
	want := []LanguageInfo{
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

	got := SupportedLanguages()
	if len(got) != len(want) {
		t.Fatalf("len(SupportedLanguages()) = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("SupportedLanguages()[%d] = %+v, want %+v", i, got[i], want[i])
		}
		info, ok := LookupLanguage(want[i].Language)
		if !ok || info != want[i] {
			t.Fatalf("LookupLanguage(%q) = %+v, %v; want %+v, true", want[i].Language, info, ok, want[i])
		}
	}

	got[0] = LanguageInfo{Language: Language("Changed"), PakFilename: "Changed.pak", SubtitleTag: "XX"}
	fresh := SupportedLanguages()
	if fresh[0] != want[0] {
		t.Fatalf("caller mutation changed supported language registry: got %+v, want %+v", fresh[0], want[0])
	}
}

func TestLookupLanguageRejectsUnsupported(t *testing.T) {
	if info, ok := LookupLanguage(Language("Klingon")); ok {
		t.Fatalf("unsupported language unexpectedly resolved to %+v", info)
	}
}

func TestParseLanguageCanonicalAndStoreFacingNames(t *testing.T) {
	tests := []struct {
		input string
		want  Language
	}{
		{input: "English", want: English},
		{input: " english ", want: English},
		{input: "German", want: German},
		{input: "rUsSiAn", want: Russian},
		{input: "Portuguese (Brazil)", want: Portuguese},
		{input: "Portuguese - Brazil", want: Portuguese},
		{input: "Chinese (Simplified)", want: ChineseSimplified},
		{input: "Simplified Chinese", want: ChineseSimplified},
		{input: "Traditional Chinese", want: ChineseTraditional},
		{input: "Spanish - Spain", want: Spanish},
		{input: "Vietnamese", want: Vietnamese},
	}

	for _, tt := range tests {
		got, ok := ParseLanguage(tt.input)
		if !ok {
			t.Fatalf("ParseLanguage(%q) returned unsupported", tt.input)
		}
		if got != tt.want {
			t.Fatalf("ParseLanguage(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParseLanguageRejectsUnsupported(t *testing.T) {
	for _, input := range []string{"", "Rus", "English (US)", "Klingon"} {
		if got, ok := ParseLanguage(input); ok {
			t.Fatalf("ParseLanguage(%q) unexpectedly resolved to %q", input, got)
		}
	}
}

func TestInstalledLanguagesReturnsOnlyKnownPresentPAKsInRegistryOrder(t *testing.T) {
	root := t.TempDir()
	localizationDir := filepath.Join(root, "Localization")
	if err := os.MkdirAll(localizationDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, filename := range []string{
		"Russian_xml.pak",
		"German_xml.pak",
		"Portuguese_xml.pak",
		"Unknown_xml.pak",
	} {
		if err := os.WriteFile(filepath.Join(localizationDir, filename), []byte("fixture"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	got, err := InstalledLanguages(root)
	if err != nil {
		t.Fatalf("InstalledLanguages() error = %v", err)
	}
	want := []LanguageInfo{
		{Language: German, PakFilename: "German_xml.pak", SubtitleTag: "DE"},
		{Language: Portuguese, PakFilename: "Portuguese_xml.pak", SubtitleTag: "PT"},
		{Language: Russian, PakFilename: "Russian_xml.pak", SubtitleTag: "RU"},
	}
	if len(got) != len(want) {
		t.Fatalf("InstalledLanguages() = %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("InstalledLanguages()[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestInstalledLanguagesEmptyRootIsEmpty(t *testing.T) {
	got, err := InstalledLanguages("  ")
	if err != nil || len(got) != 0 {
		t.Fatalf("InstalledLanguages(empty) = %+v, %v; want empty, nil", got, err)
	}
}

func TestInstalledLanguagesRejectsDirectoryAtKnownPAKPath(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "Localization", "English_xml.pak")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := InstalledLanguages(root); err == nil {
		t.Fatal("InstalledLanguages() error = nil for directory at PAK path")
	}
}
