package localization

import (
	"archive/zip"
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type testPakEntry struct {
	name string
	data []byte
}

func TestReadDialogueXMLPreservesRawBytes(t *testing.T) {
	want := []byte("<?xml version=\"1.0\"?><Table>\n<Row><Cell>id</Cell><Cell>source</Cell><Cell>Привет &amp; hello.\nSecond line.</Cell></Row>\n</Table>\n")
	pakPath := writeTestPak(t,
		testPakEntry{name: "before.txt", data: []byte("before")},
		testPakEntry{name: DialogueXMLArchivePath, data: want},
		testPakEntry{name: "after.txt", data: []byte("after")},
	)

	got, err := ReadDialogueXML(pakPath)
	if err != nil {
		t.Fatalf("ReadDialogueXML() error = %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("ReadDialogueXML() changed stored bytes\ngot:  %q\nwant: %q", got, want)
	}
}

func TestReadDialogueXMLMissingEntry(t *testing.T) {
	pakPath := writeTestPak(t, testPakEntry{name: "other.xml", data: []byte("<Table />")})

	_, err := ReadDialogueXML(pakPath)
	if !errors.Is(err, ErrDialogueXMLNotFound) {
		t.Fatalf("ReadDialogueXML() error = %v, want errors.Is(..., ErrDialogueXMLNotFound)", err)
	}
}

func TestReadDialogueXMLDuplicateEntry(t *testing.T) {
	pakPath := writeTestPak(t,
		testPakEntry{name: DialogueXMLArchivePath, data: []byte("first")},
		testPakEntry{name: DialogueXMLArchivePath, data: []byte("second")},
	)

	_, err := ReadDialogueXML(pakPath)
	if !errors.Is(err, ErrDialogueXMLDuplicate) {
		t.Fatalf("ReadDialogueXML() error = %v, want errors.Is(..., ErrDialogueXMLDuplicate)", err)
	}
}

func TestReadDialogueXMLNonexistentPak(t *testing.T) {
	pakPath := filepath.Join(t.TempDir(), "missing.pak")

	_, err := ReadDialogueXML(pakPath)
	if err == nil {
		t.Fatal("ReadDialogueXML() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "open localization PAK") {
		t.Fatalf("ReadDialogueXML() error = %q, want contextual open error", err)
	}
}

func TestReadDialogueXMLMalformedPak(t *testing.T) {
	pakPath := filepath.Join(t.TempDir(), "malformed.pak")
	if err := os.WriteFile(pakPath, []byte("this is not a zip archive"), 0o600); err != nil {
		t.Fatalf("write malformed PAK: %v", err)
	}

	_, err := ReadDialogueXML(pakPath)
	if err == nil {
		t.Fatal("ReadDialogueXML() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "open localization PAK") {
		t.Fatalf("ReadDialogueXML() error = %q, want contextual open error", err)
	}
}

func TestReadDialogueXMLRequiresExactRootEntry(t *testing.T) {
	pakPath := writeTestPak(t, testPakEntry{
		name: "nested/" + DialogueXMLArchivePath,
		data: []byte("nested"),
	})

	_, err := ReadDialogueXML(pakPath)
	if !errors.Is(err, ErrDialogueXMLNotFound) {
		t.Fatalf("ReadDialogueXML() error = %v, want exact root entry requirement", err)
	}
}

func writeTestPak(t *testing.T, entries ...testPakEntry) string {
	t.Helper()

	pakPath := filepath.Join(t.TempDir(), "fixture.pak")
	file, err := os.Create(pakPath)
	if err != nil {
		t.Fatalf("create test PAK: %v", err)
	}

	writer := zip.NewWriter(file)
	for _, entry := range entries {
		entryWriter, err := writer.Create(entry.name)
		if err != nil {
			_ = writer.Close()
			_ = file.Close()
			t.Fatalf("create ZIP entry %q: %v", entry.name, err)
		}
		if _, err := entryWriter.Write(entry.data); err != nil {
			_ = writer.Close()
			_ = file.Close()
			t.Fatalf("write ZIP entry %q: %v", entry.name, err)
		}
	}

	if err := writer.Close(); err != nil {
		_ = file.Close()
		t.Fatalf("close ZIP writer: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close test PAK: %v", err)
	}

	return pakPath
}
