package localization

import (
	"strings"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/subtitlepayload"
)

func TestMergeDialogueRowsHUDPrototypeEmitsCompleteScaleformHTML(t *testing.T) {
	rows, stats, err := MergeDialogueRowsHUDPrototype(
		[]DialogueRow{{ID: "id", Text: "Основной <br/> line"}},
		[]DialogueRow{{ID: "id", Text: "Secondary <b>unsafe</b> <br/> line"}},
		"RU",
		"EN",
	)
	if err != nil {
		t.Fatalf("MergeDialogueRowsHUDPrototype() error = %v", err)
	}
	if stats.Bilingual != 1 {
		t.Fatalf("Bilingual = %d, want 1", stats.Bilingual)
	}

	got := rows[0].Text
	for _, want := range []string{
		"[RU] Основной <br/> line",
		"<br/><font color='" + subtitlepayload.SecondaryColor + "' size='24'><i>",
		"[EN] Secondary &lt;b&gt;unsafe&lt;/b&gt; <br/> line",
		"</i></font>",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("merged text = %q, missing %q", got, want)
		}
	}
	if strings.Contains(got, "<p") || strings.Contains(got, "align=") {
		t.Fatalf("direct HTML payload unexpectedly forces paragraph alignment: %q", got)
	}
	if strings.Contains(got, subtitlepayload.Prefix) || strings.Contains(got, subtitlepayload.Suffix) {
		t.Fatalf("direct HTML payload unexpectedly contains carrier sentinel: %q", got)
	}
	if strings.Contains(got, BilingualSeparator) {
		t.Fatalf("direct HTML payload unexpectedly contains legacy separator: %q", got)
	}
}

func TestMergeDialogueRowsHUDPrototypeLeavesFallbacksPlain(t *testing.T) {
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
		if strings.Contains(row.Text, "<font") || strings.Contains(row.Text, "<p") {
			t.Fatalf("fallback row %d unexpectedly contains HUD HTML: %q", i, row.Text)
		}
	}
}
