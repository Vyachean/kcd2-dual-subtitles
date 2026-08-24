package localization

import "testing"

func TestMergeDialogueRowsHUDStylesPrimaryPropertiesIndependently(t *testing.T) {
	rows, _, err := MergeDialogueRowsHUD(
		[]DialogueRow{{ID: "id", Text: "Primary <b>unsafe</b>"}},
		[]DialogueRow{{ID: "id", Text: "Secondary"}},
		"RU",
		"EN",
		HUDPresentationOptions{
			PrimaryColor:      "#FFEEDD",
			PrimarySize:       30,
			PrimaryItalic:     true,
			SecondaryColor:    "#123ABC",
			SecondarySize:     18,
			SecondaryItalic:   false,
			ShowLanguageTags:  false,
		},
	)
	if err != nil {
		t.Fatalf("MergeDialogueRowsHUD() error = %v", err)
	}
	want := "<font color='#FFEEDD' size='30'><i>Primary &lt;b&gt;unsafe&lt;/b&gt;</i></font><br/><font color='#123ABC' size='18'>Secondary</font>"
	if rows[0].Text != want {
		t.Fatalf("merged text = %q, want %q", rows[0].Text, want)
	}
}

func TestMergeDialogueRowsHUDAllowsPartialPrimaryOverrides(t *testing.T) {
	tests := []struct {
		name         string
		presentation HUDPresentationOptions
		wantPrimary  string
	}{
		{
			name: "color only",
			presentation: HUDPresentationOptions{
				PrimaryColor:   "#ABCDEF",
				SecondaryColor: "#123ABC",
				SecondarySize:  18,
			},
			wantPrimary: "<font color='#ABCDEF'>Primary</font>",
		},
		{
			name: "size only",
			presentation: HUDPresentationOptions{
				PrimarySize:     28,
				SecondaryColor: "#123ABC",
				SecondarySize:  18,
			},
			wantPrimary: "<font size='28'>Primary</font>",
		},
		{
			name: "italic only",
			presentation: HUDPresentationOptions{
				PrimaryItalic:  true,
				SecondaryColor: "#123ABC",
				SecondarySize:  18,
			},
			wantPrimary: "<i>Primary</i>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows, _, err := MergeDialogueRowsHUD(
				[]DialogueRow{{ID: "id", Text: "Primary"}},
				[]DialogueRow{{ID: "id", Text: "Secondary"}},
				"RU",
				"EN",
				t.presentation,
			)
			if err != nil {
				t.Fatalf("MergeDialogueRowsHUD() error = %v", err)
			}
			want := tt.wantPrimary + "<br/><font color='#123ABC' size='18'>Secondary</font>"
			if rows[0].Text != want {
				t.Fatalf("merged text = %q, want %q", rows[0].Text, want)
			}
		})
	}
}
