package generator

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
)

func TestChangedRowsEmitsOnlyModifiedTexts(t *testing.T) {
	base := []localization.DialogueRow{
		{ID: "different", Source: "source", Text: "Основной"},
		{ID: "identical", Source: "source", Text: "[pause]"},
		{ID: "missing", Source: "source", Text: "Только основной"},
		{ID: "main_empty", Source: "source", Text: ""},
	}
	merged := []localization.DialogueRow{
		{ID: "different", Source: "source", Text: "Основной\\nSecondary"},
		{ID: "identical", Source: "source", Text: "[pause]"},
		{ID: "missing", Source: "source", Text: "Только основной"},
		{ID: "main_empty", Source: "source", Text: "Secondary only"},
	}

	got, err := changedRows(base, merged, "")
	if err != nil {
		t.Fatalf("changedRows() error = %v", err)
	}
	want := []localization.DialogueRow{merged[0], merged[3]}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("changedRows() = %#v, want %#v", got, want)
	}
	if got[0].Text != "Основной\\nSecondary" {
		t.Fatalf("bilingual separator changed: %q", got[0].Text)
	}
}

func TestChangedRowsCanaryForcesSelectedExistingRowIntoPatch(t *testing.T) {
	base := []localization.DialogueRow{
		{ID: "unchanged", Source: "source", Text: "Visible text"},
		{ID: "other", Source: "source", Text: "Other"},
	}
	merged := append([]localization.DialogueRow(nil), base...)

	got, err := changedRows(base, merged, "unchanged")
	if err != nil {
		t.Fatalf("changedRows() error = %v", err)
	}
	want := []localization.DialogueRow{{ID: "unchanged", Source: "source", Text: CanaryPrefix + "Visible text"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("canary patch rows = %#v, want %#v", got, want)
	}
}

func TestChangedRowsRejectsUnknownCanaryID(t *testing.T) {
	base := []localization.DialogueRow{{ID: "known", Text: "text"}}
	merged := append([]localization.DialogueRow(nil), base...)

	_, err := changedRows(base, merged, "not_present")
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("changedRows() error = %v, want errors.Is(..., ErrInvalidRequest)", err)
	}
}
