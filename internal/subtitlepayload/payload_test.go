package subtitlepayload

import "testing"

func TestEncodeSecondaryHTMLEscapesMarkupButPreservesDialogueBreaks(t *testing.T) {
	got := EncodeSecondaryHTML(`A < B & C <br/> <b>bold</b> -- tail`)
	want := `A &lt; B &amp; C <br/> &lt;b&gt;bold&lt;/b&gt; -- tail`
	if got != want {
		t.Fatalf("EncodeSecondaryHTML() = %q, want %q", got, want)
	}
}

func TestWrapSecondaryUsesVersionedVisibleDiagnosticMarker(t *testing.T) {
	got := WrapSecondary("[EN] Hello")
	want := Prefix + "[EN] Hello" + Suffix
	if got != want {
		t.Fatalf("WrapSecondary() = %q, want %q", got, want)
	}
	if got[0] != '[' {
		t.Fatalf("diagnostic marker unexpectedly looks like hidden markup: %q", got)
	}
}
