package localization

import (
	"strings"
	"testing"
)

func TestMergeDialogueRowsTaggedPreservesLanguageOrder(t *testing.T) {
	tests := []struct {
		name         string
		mainText     string
		secondaryText string
		mainTag      string
		secondaryTag string
		want         string
	}{
		{
			name:          "Russian main",
			mainText:      "Привет.",
			secondaryText: "Hello.",
			mainTag:       "RU",
			secondaryTag:  "EN",
			want:          `[RU] Привет.\n[EN] Hello.`,
		},
		{
			name:          "English main",
			mainText:      "Hello.",
			secondaryText: "Привет.",
			mainTag:       "EN",
			secondaryTag:  "RU",
			want:          `[EN] Hello.\n[RU] Привет.`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows, stats, err := MergeDialogueRowsTagged(
				[]DialogueRow{{ID: "id", Text: tt.mainText}},
				[]DialogueRow{{ID: "id", Text: tt.secondaryText}},
				tt.mainTag,
				tt.secondaryTag,
			)
			if err != nil {
				t.Fatalf("MergeDialogueRowsTagged() error = %v", err)
			}
			if stats.Bilingual != 1 {
				t.Fatalf("Bilingual = %d, want 1", stats.Bilingual)
			}
			if got := rows[0].Text; got != tt.want {
				t.Fatalf("merged text = %q, want %q", got, tt.want)
			}
			if strings.ContainsRune(rows[0].Text, '\n') {
				t.Fatalf("merged text contains a real newline: %q", rows[0].Text)
			}
		})
	}
}

func TestMergeDialogueRowsTaggedLeavesSingleLanguageRowsUntagged(t *testing.T) {
	main := []DialogueRow{
		{ID: "identical", Text: "[pause]"},
		{ID: "missing", Text: "Только основной"},
		{ID: "empty-main", Text: ""},
		{ID: "empty-secondary", Text: "Основной текст"},
	}
	secondary := []DialogueRow{
		{ID: "identical", Text: "[pause]"},
		{ID: "empty-main", Text: "Secondary only"},
		{ID: "empty-secondary", Text: ""},
	}

	rows, _, err := MergeDialogueRowsTagged(main, secondary, "RU", "EN")
	if err != nil {
		t.Fatalf("MergeDialogueRowsTagged() error = %v", err)
	}

	want := []string{"[pause]", "Только основной", "Secondary only", "Основной текст"}
	for i, wantText := range want {
		if rows[i].Text != wantText {
			t.Fatalf("row %d text = %q, want %q", i, rows[i].Text, wantText)
		}
		if strings.Contains(rows[i].Text, "[RU]") || strings.Contains(rows[i].Text, "[EN]") {
			t.Fatalf("row %d unexpectedly tagged: %q", i, rows[i].Text)
		}
	}
}

func TestSupportedLanguagesProvideStableSubtitleTags(t *testing.T) {
	english, ok := LookupLanguage(English)
	if !ok || english.SubtitleTag != "EN" {
		t.Fatalf("English metadata = %+v, ok=%v; want SubtitleTag EN", english, ok)
	}
	russian, ok := LookupLanguage(Russian)
	if !ok || russian.SubtitleTag != "RU" {
		t.Fatalf("Russian metadata = %+v, ok=%v; want SubtitleTag RU", russian, ok)
	}
}
