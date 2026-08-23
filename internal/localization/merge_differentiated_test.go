package localization

import (
	"strings"
	"testing"
)

func TestMergeDialogueRowsDifferentiatedPreservesLanguageOrder(t *testing.T) {
	tests := []struct {
		name          string
		mainText      string
		secondaryText string
		mainTag       string
		secondaryTag  string
		want          string
	}{
		{
			name:          "Russian main",
			mainText:      "Привет.",
			secondaryText: "Hello.",
			mainTag:       "RU",
			secondaryTag:  "EN",
			want:          `[RU] Привет.\n<font color='#A8A8A8'><i>[EN] Hello.</i></font>`,
		},
		{
			name:          "English main",
			mainText:      "Hello.",
			secondaryText: "Привет.",
			mainTag:       "EN",
			secondaryTag:  "RU",
			want:          `[EN] Hello.\n<font color='#A8A8A8'><i>[RU] Привет.</i></font>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows, stats, err := MergeDialogueRowsDifferentiated(
				[]DialogueRow{{ID: "id", Text: tt.mainText}},
				[]DialogueRow{{ID: "id", Text: tt.secondaryText}},
				tt.mainTag,
				tt.secondaryTag,
			)
			if err != nil {
				t.Fatalf("MergeDialogueRowsDifferentiated() error = %v", err)
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

func TestMergeDialogueRowsDifferentiatedLeavesFallbackRowsUnstyled(t *testing.T) {
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

	rows, stats, err := MergeDialogueRowsDifferentiated(main, secondary, "RU", "EN")
	if err != nil {
		t.Fatalf("MergeDialogueRowsDifferentiated() error = %v", err)
	}
	if stats.Identical != 1 || stats.MissingSecondary != 1 || stats.MainEmptyFallback != 1 || stats.SecondaryEmptyFallback != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}

	want := []string{"[pause]", "Только основной", "Secondary only", "Основной текст"}
	for i, wantText := range want {
		if rows[i].Text != wantText {
			t.Fatalf("row %d text = %q, want %q", i, rows[i].Text, wantText)
		}
		if strings.Contains(rows[i].Text, "<font") || strings.Contains(rows[i].Text, "<i>") {
			t.Fatalf("row %d unexpectedly styled: %q", i, rows[i].Text)
		}
	}
}

func TestMergeDialogueRowsDifferentiatedPreservesExistingInlineMarkup(t *testing.T) {
	rows, _, err := MergeDialogueRowsDifferentiated(
		[]DialogueRow{{ID: "id", Text: "First<br/>Second"}},
		[]DialogueRow{{ID: "id", Text: "Uno<br/>Dos"}},
		"EN",
		"ES",
	)
	if err != nil {
		t.Fatalf("MergeDialogueRowsDifferentiated() error = %v", err)
	}

	want := `[EN] First<br/>Second\n<font color='#A8A8A8'><i>[ES] Uno<br/>Dos</i></font>`
	if got := rows[0].Text; got != want {
		t.Fatalf("merged text = %q, want %q", got, want)
	}
}
