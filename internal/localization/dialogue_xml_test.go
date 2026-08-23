package localization

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseDialogueXMLSyntheticFixtures(t *testing.T) {
	tests := []struct {
		name    string
		fixture string
		want    []DialogueRow
	}{
		{
			name:    "English",
			fixture: "dialog_english.xml",
			want: []DialogueRow{
				{ID: "dialog_regular", Source: "synthetic-source", Text: "Hello, traveller."},
				{ID: "dialog_entities", Source: "synthetic-source", Text: "Fish & chips <sample> \"quoted\"."},
				{ID: "dialog_multiline", Source: "synthetic-source", Text: "First line.\nSecond line."},
				{ID: "dialog_identical", Source: "synthetic-source", Text: "[pause]"},
				{ID: "dialog_empty", Source: "synthetic-source", Text: ""},
			},
		},
		{
			name:    "Russian",
			fixture: "dialog_russian.xml",
			want: []DialogueRow{
				{ID: "dialog_regular", Source: "synthetic-source", Text: "Здравствуй, путник."},
				{ID: "dialog_entities", Source: "synthetic-source", Text: "Хлеб & сыр <пример> \"в кавычках\"."},
				{ID: "dialog_multiline", Source: "synthetic-source", Text: "Первая строка.\nВторая строка."},
				{ID: "dialog_identical", Source: "synthetic-source", Text: "[pause]"},
				{ID: "dialog_missing_secondary", Source: "synthetic-source", Text: "Эта строка есть только в основном языке."},
				{ID: "dialog_empty", Source: "synthetic-source", Text: ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := readDialogueFixture(t, tt.fixture)
			got, err := ParseDialogueXML(data)
			if err != nil {
				t.Fatalf("ParseDialogueXML() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ParseDialogueXML() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestParseDialogueXMLRejectsMalformedInput(t *testing.T) {
	tests := []struct {
		name string
		xml  string
		want error
	}{
		{name: "invalid XML", xml: "<Table><Row>", want: ErrInvalidDialogueXML},
		{name: "wrong root", xml: "<Root></Root>", want: ErrInvalidDialogueXML},
		{name: "two cells", xml: "<Table><Row><Cell>id</Cell><Cell>source</Cell></Row></Table>", want: ErrInvalidDialogueRow},
		{name: "four cells", xml: "<Table><Row><Cell>id</Cell><Cell>source</Cell><Cell>text</Cell><Cell>extra</Cell></Row></Table>", want: ErrInvalidDialogueRow},
		{name: "empty ID", xml: "<Table><Row><Cell></Cell><Cell>source</Cell><Cell>text</Cell></Row></Table>", want: ErrInvalidDialogueRow},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseDialogueXML([]byte(tt.xml))
			if !errors.Is(err, tt.want) {
				t.Fatalf("ParseDialogueXML() error = %v, want errors.Is(..., %v)", err, tt.want)
			}
		})
	}
}

func TestMarshalDialogueXMLRoundTripAndEscaping(t *testing.T) {
	want := []DialogueRow{
		{
			ID:     "dialog_special",
			Source: "A & B <source>",
			Text:   "Русский & English <sample>\nSecond line.",
		},
		{ID: "dialog_empty", Source: "synthetic-source", Text: ""},
	}

	first, err := MarshalDialogueXML(want)
	if err != nil {
		t.Fatalf("MarshalDialogueXML() error = %v", err)
	}
	second, err := MarshalDialogueXML(want)
	if err != nil {
		t.Fatalf("second MarshalDialogueXML() error = %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("MarshalDialogueXML() output is not deterministic")
	}

	encoded := string(first)
	if !strings.Contains(encoded, "A &amp; B &lt;source&gt;") {
		t.Fatalf("MarshalDialogueXML() did not XML-escape source: %s", encoded)
	}
	if !strings.Contains(encoded, "Русский &amp; English &lt;sample&gt;&#xA;Second line.") {
		t.Fatalf("MarshalDialogueXML() did not preserve/escape text: %s", encoded)
	}

	got, err := ParseDialogueXML(first)
	if err != nil {
		t.Fatalf("ParseDialogueXML(marshaled) error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round trip = %#v, want %#v", got, want)
	}
}

func TestSyntheticFixtureSemanticRoundTrip(t *testing.T) {
	data := readDialogueFixture(t, "dialog_english.xml")
	rows, err := ParseDialogueXML(data)
	if err != nil {
		t.Fatalf("ParseDialogueXML() error = %v", err)
	}

	marshaled, err := MarshalDialogueXML(rows)
	if err != nil {
		t.Fatalf("MarshalDialogueXML() error = %v", err)
	}
	roundTripped, err := ParseDialogueXML(marshaled)
	if err != nil {
		t.Fatalf("ParseDialogueXML(round trip) error = %v", err)
	}
	if !reflect.DeepEqual(roundTripped, rows) {
		t.Fatalf("semantic round trip = %#v, want %#v", roundTripped, rows)
	}
}

func TestMarshalDialogueXMLRejectsEmptyID(t *testing.T) {
	_, err := MarshalDialogueXML([]DialogueRow{{Source: "source", Text: "text"}})
	if !errors.Is(err, ErrInvalidDialogueRow) {
		t.Fatalf("MarshalDialogueXML() error = %v, want errors.Is(..., ErrInvalidDialogueRow)", err)
	}
}

func readDialogueFixture(t *testing.T, name string) []byte {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}
