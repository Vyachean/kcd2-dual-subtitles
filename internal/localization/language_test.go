package localization

import "testing"

func TestLookupLanguage(t *testing.T) {
	tests := []struct {
		name        string
		language    Language
		pakFilename string
	}{
		{name: "English", language: English, pakFilename: "English_xml.pak"},
		{name: "Russian", language: Russian, pakFilename: "Russian_xml.pak"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, ok := LookupLanguage(tt.language)
			if !ok {
				t.Fatalf("LookupLanguage(%q) returned unsupported", tt.language)
			}
			if info.Language != tt.language {
				t.Fatalf("Language = %q, want %q", info.Language, tt.language)
			}
			if info.PakFilename != tt.pakFilename {
				t.Fatalf("PakFilename = %q, want %q", info.PakFilename, tt.pakFilename)
			}
		})
	}
}

func TestLookupLanguageRejectsUnsupported(t *testing.T) {
	if info, ok := LookupLanguage(Language("German")); ok {
		t.Fatalf("unsupported language unexpectedly resolved to %+v", info)
	}
}

func TestParseLanguage(t *testing.T) {
	tests := []struct {
		input string
		want  Language
	}{
		{input: "English", want: English},
		{input: "english", want: English},
		{input: " ENGLISH ", want: English},
		{input: "Russian", want: Russian},
		{input: "rUsSiAn", want: Russian},
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
	for _, input := range []string{"", "German", "Rus", "English (US)"} {
		if got, ok := ParseLanguage(input); ok {
			t.Fatalf("ParseLanguage(%q) unexpectedly resolved to %q", input, got)
		}
	}
}

func TestSupportedLanguagesStableOrderAndCopy(t *testing.T) {
	got := SupportedLanguages()
	want := []LanguageInfo{
		{Language: English, PakFilename: "English_xml.pak"},
		{Language: Russian, PakFilename: "Russian_xml.pak"},
	}

	if len(got) != len(want) {
		t.Fatalf("len(SupportedLanguages()) = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("SupportedLanguages()[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}

	got[0] = LanguageInfo{Language: Language("Changed"), PakFilename: "Changed.pak"}
	fresh := SupportedLanguages()
	if fresh[0] != want[0] {
		t.Fatalf("caller mutation changed supported language registry: got %+v, want %+v", fresh[0], want[0])
	}
}
