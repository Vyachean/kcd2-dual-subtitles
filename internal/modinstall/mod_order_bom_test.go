package modinstall

import (
	"bytes"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/modarchive"
)

func TestModOrderWithEntryPreservesUTF8BOMAndMovesProjectLast(t *testing.T) {
	original := append(append([]byte(nil), utf8BOM...), []byte(modarchive.ModID+"\r\nother_mod\r\n")...)
	got := modOrderWithEntry(original, modarchive.ModID)
	want := append(append([]byte(nil), utf8BOM...), []byte("other_mod\r\n"+modarchive.ModID+"\r\n")...)
	if !bytes.Equal(got, want) {
		t.Fatalf("modOrderWithEntry() = %q, want %q", got, want)
	}
	if !modOrderContains(got, modarchive.ModID) {
		t.Fatalf("modOrderContains() = false for BOM-prefixed normalized order %q", got)
	}
}

func TestRemoveModOrderEntriesPreservesUTF8BOM(t *testing.T) {
	original := append(append([]byte(nil), utf8BOM...), []byte(modarchive.ModID+"\nother_mod\n")...)
	got, changed := removeModOrderEntries(original, modarchive.ModID)
	if !changed {
		t.Fatal("removeModOrderEntries() changed = false, want true")
	}
	want := append(append([]byte(nil), utf8BOM...), []byte("other_mod\n")...)
	if !bytes.Equal(got, want) {
		t.Fatalf("removeModOrderEntries() = %q, want %q", got, want)
	}
}
