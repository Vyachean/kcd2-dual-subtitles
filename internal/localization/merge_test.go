package localization

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestMergeDialogueRows(t *testing.T) {
	main := []DialogueRow{
		{ID: "first", Source: "main-source-1", Text: "Основной"},
		{ID: "identical", Source: "main-source-2", Text: "[pause]"},
		{ID: "missing", Source: "main-source-3", Text: "Только основной"},
		{ID: "empty-main", Source: "main-source-4", Text: ""},
		{ID: "empty-secondary", Source: "main-source-5", Text: "Основной текст"},
		{ID: "both-empty", Source: "main-source-6", Text: ""},
	}
	secondary := []DialogueRow{
		{ID: "empty-secondary", Source: "secondary-source-5", Text: ""},
		{ID: "first", Source: "secondary-source-1", Text: "Secondary"},
		{ID: "identical", Source: "secondary-source-2", Text: "[pause]"},
		{ID: "empty-main", Source: "secondary-source-4", Text: "Secondary only text"},
		{ID: "both-empty", Source: "secondary-source-6", Text: ""},
		{ID: "secondary-only", Source: "secondary-source-extra", Text: "Unused"},
	}
	mainBefore := append([]DialogueRow(nil), main...)
	secondaryBefore := append([]DialogueRow(nil), secondary...)

	got, stats, err := MergeDialogueRows(main, secondary)
	if err != nil {
		t.Fatalf("MergeDialogueRows() error = %v", err)
	}

	want := []DialogueRow{
		{ID: "first", Source: "main-source-1", Text: "Основной\\nSecondary"},
		{ID: "identical", Source: "main-source-2", Text: "[pause]"},
		{ID: "missing", Source: "main-source-3", Text: "Только основной"},
		{ID: "empty-main", Source: "main-source-4", Text: "\\nSecondary only text"},
		{ID: "empty-secondary", Source: "main-source-5", Text: "Основной текст\\n"},
		{ID: "both-empty", Source: "main-source-6", Text: ""},
	}
	wantStats := MergeStats{
		Processed:        6,
		Bilingual:        3,
		Identical:        2,
		MissingSecondary: 1,
		SecondaryOnly:    1,
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("MergeDialogueRows() rows = %#v, want %#v", got, want)
	}
	if stats != wantStats {
		t.Fatalf("MergeDialogueRows() stats = %+v, want %+v", stats, wantStats)
	}
	if !reflect.DeepEqual(main, mainBefore) {
		t.Fatalf("MergeDialogueRows() mutated main input: got %#v, want %#v", main, mainBefore)
	}
	if !reflect.DeepEqual(secondary, secondaryBefore) {
		t.Fatalf("MergeDialogueRows() mutated secondary input: got %#v, want %#v", secondary, secondaryBefore)
	}

	secondRows, secondStats, err := MergeDialogueRows(main, secondary)
	if err != nil {
		t.Fatalf("second MergeDialogueRows() error = %v", err)
	}
	if !reflect.DeepEqual(secondRows, got) || secondStats != stats {
		t.Fatalf("MergeDialogueRows() is not deterministic: second rows=%#v stats=%+v", secondRows, secondStats)
	}
}

func TestMergeDialogueRowsUsesLiteralGameSeparator(t *testing.T) {
	rows, _, err := MergeDialogueRows(
		[]DialogueRow{{ID: "id", Text: "main"}},
		[]DialogueRow{{ID: "id", Text: "secondary"}},
	)
	if err != nil {
		t.Fatalf("MergeDialogueRows() error = %v", err)
	}

	if rows[0].Text != `main\nsecondary` {
		t.Fatalf("merged text = %q, want literal game separator", rows[0].Text)
	}
	if strings.ContainsRune(rows[0].Text, '\n') {
		t.Fatalf("merged text contains a real newline: %q", rows[0].Text)
	}
}

func TestMergeDialogueRowsRejectsDuplicateIDs(t *testing.T) {
	tests := []struct {
		name      string
		main      []DialogueRow
		secondary []DialogueRow
		wantSide  string
		wantID    string
	}{
		{
			name: "main",
			main: []DialogueRow{
				{ID: "duplicate", Text: "one"},
				{ID: "duplicate", Text: "two"},
			},
			wantSide: "main",
			wantID:   "duplicate",
		},
		{
			name: "secondary",
			main: []DialogueRow{{ID: "main", Text: "main"}},
			secondary: []DialogueRow{
				{ID: "duplicate", Text: "one"},
				{ID: "duplicate", Text: "two"},
			},
			wantSide: "secondary",
			wantID:   "duplicate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows, stats, err := MergeDialogueRows(tt.main, tt.secondary)
			if !errors.Is(err, ErrDuplicateDialogueID) {
				t.Fatalf("MergeDialogueRows() error = %v, want errors.Is(..., ErrDuplicateDialogueID)", err)
			}
			if rows != nil {
				t.Fatalf("MergeDialogueRows() rows = %#v, want nil on duplicate", rows)
			}
			if stats != (MergeStats{}) {
				t.Fatalf("MergeDialogueRows() stats = %+v, want zero stats on duplicate", stats)
			}
			if !strings.Contains(err.Error(), tt.wantSide) || !strings.Contains(err.Error(), tt.wantID) {
				t.Fatalf("duplicate error = %q, want side %q and ID %q", err, tt.wantSide, tt.wantID)
			}
		})
	}
}

func TestMergeDialogueRowsSyntheticFixtures(t *testing.T) {
	russian, err := ParseDialogueXML(readDialogueFixture(t, "dialog_russian.xml"))
	if err != nil {
		t.Fatalf("parse Russian fixture: %v", err)
	}
	english, err := ParseDialogueXML(readDialogueFixture(t, "dialog_english.xml"))
	if err != nil {
		t.Fatalf("parse English fixture: %v", err)
	}

	merged, stats, err := MergeDialogueRows(russian, english)
	if err != nil {
		t.Fatalf("MergeDialogueRows() error = %v", err)
	}

	wantStats := MergeStats{
		Processed:        6,
		Bilingual:        3,
		Identical:        2,
		MissingSecondary: 1,
		SecondaryOnly:    0,
	}
	if stats != wantStats {
		t.Fatalf("fixture stats = %+v, want %+v", stats, wantStats)
	}
	if len(merged) != len(russian) {
		t.Fatalf("merged fixture rows = %d, want %d", len(merged), len(russian))
	}
	if merged[0].ID != "dialog_regular" || merged[0].Text != "Здравствуй, путник.\\nHello, traveller." {
		t.Fatalf("regular merged row = %+v", merged[0])
	}
	if merged[2].Text != "Первая строка.\nВторая строка.\\nFirst line.\nSecond line." {
		t.Fatalf("multiline merged text = %q", merged[2].Text)
	}
	if merged[3].Text != "[pause]" {
		t.Fatalf("identical merged text = %q", merged[3].Text)
	}
	if merged[4].Text != "Эта строка есть только в основном языке." {
		t.Fatalf("missing-secondary merged text = %q", merged[4].Text)
	}
	if merged[5].Text != "" {
		t.Fatalf("both-empty merged text = %q", merged[5].Text)
	}
}
