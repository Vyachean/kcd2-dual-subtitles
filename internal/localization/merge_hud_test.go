package localization

import (
	"strings"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/subtitlepayload"
)

func TestMergeDialogueRowsHUDPrototypeCarriesSecondaryBeforeVisiblePrimary(t *testing.T) {
	rows, stats, err := MergeDialogueRowsHUDPrototype(
		[]DialogueRow{{ID: "id", Text: "Основной"}},
		[]DialogueRow{{ID: "id", Text: "Secondary <br/> line"}},
		"RU",
		"EN",
	)
	if err != nil {
		t.Fatalf("MergeDialogueRowsHUDPrototype() error = %v", err)
	}
	if stats.Bilingual != 1 {
		t.Fatalf("Bilingual = %d, want 1", stats.Bilingual)
	}
	want := subtitlepayload.Prefix + "[EN] Secondary <br/> line" + subtitlepayload.Suffix + "[RU] Основной"
	if rows[0].Text != want {
		t.Fatalf("merged text = %q, want %q", rows[0].Text, want)
	}
	if strings.Contains(rows[0].Text, BilingualSeparator) {
		t.Fatalf("HUD payload unexpectedly contains legacy separator: %q", rows[0].Text)
	}
}

func TestMergeDialogueRowsHUDPrototypeLeavesFallbacksUnmarked(t *testing.T) {
	main := []DialogueRow{
		{ID: "identical", Text: "same"},
		{ID: "missing", Text: "main only"},
		{ID: "empty-main", Text: ""},
		{ID: "empty-secondary", Text: "main"},
	}
	secondary := []DialogueRow{
		{ID: "identical", Text: "same"},
		{ID: "empty-main", Text: "secondary only"},
		{ID: "empty-secondary", Text: ""},
	}
	rows, _, err := MergeDialogueRowsHUDPrototype(main, secondary, "RU", "EN")
	if err != nil {
		t.Fatalf("MergeDialogueRowsHUDPrototype() error = %v", err)
	}
	for i, row := range rows {
		if strings.Contains(row.Text, subtitlepayload.Prefix) {
			t.Fatalf("fallback row %d unexpectedly contains HUD marker: %q", i, row.Text)
		}
	}
}
