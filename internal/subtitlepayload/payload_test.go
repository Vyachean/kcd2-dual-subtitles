package subtitlepayload

import (
	"strings"
	"testing"
)

func TestEncodeSecondaryHTMLEscapesMarkupButPreservesDialogueBreaks(t *testing.T) {
	got := EncodeSecondaryHTML(`A < B & C <br/> <b>bold</b> -- tail`)
	want := `A &lt; B &amp; C <br/> &lt;b&gt;bold&lt;/b&gt; -&#45; tail`
	if got != want {
		t.Fatalf("EncodeSecondaryHTML() = %q, want %q", got, want)
	}
	if strings.Contains(got, "--") {
		t.Fatalf("encoded comment payload still contains --: %q", got)
	}
}

func TestWrapSecondaryUsesVersionedInvisibleMarker(t *testing.T) {
	got := WrapSecondary("[EN] Hello")
	want := Prefix + "[EN] Hello" + Suffix
	if got != want {
		t.Fatalf("WrapSecondary() = %q, want %q", got, want)
	}
}
