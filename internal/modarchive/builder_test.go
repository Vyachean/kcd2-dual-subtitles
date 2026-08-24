package modarchive

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
)

func TestBuildCreatesExpectedArchive(t *testing.T) {
	output := filepath.Join(t.TempDir(), "mod.zip")
	rows := []localization.DialogueRow{
		{ID: "a", Text: "Привет"},
		{ID: "b", Text: "Hello"},
	}

	if err := Build(output, localization.Russian, rows); err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	reader, err := zip.OpenReader(output)
	if err != nil {
		t.Fatalf("zip.OpenReader() error = %v", err)
	}
	defer reader.Close()

	files := make(map[string]*zip.File)
	for _, file := range reader.File {
		files[file.Name] = file
	}

	manifestPath := "kcd_dual_subtitles/mod.manifest"
	pakPath := "kcd_dual_subtitles/Localization/Russian_xml.pak"
	if files[manifestPath] == nil {
		t.Fatalf("missing %s", manifestPath)
	}
	if files[pakPath] == nil {
		t.Fatalf("missing %s", pakPath)
	}

	manifest := readZipFile(t, files[manifestPath])
	if !strings.Contains(string(manifest), "<Name>KCD2 Dual Subtitles</Name>") {
		t.Fatalf("manifest = %q", manifest)
	}

	pakBytes := readZipFile(t, files[pakPath])
	pakReader, err := zip.NewReader(bytes.NewReader(pakBytes), int64(len(pakBytes)))
	if err != nil {
		t.Fatalf("zip.NewReader(PAK) error = %v", err)
	}
	if len(pakReader.File) != 1 || pakReader.File[0].Name != LocalizationPatchArchivePath {
		t.Fatalf("PAK entries = %+v", pakReader.File)
	}
	xmlBytes := readZipFile(t, pakReader.File[0])
	parsed, err := localization.ParseDialogueXML(xmlBytes)
	if err != nil {
		t.Fatalf("ParseDialogueXML() error = %v", err)
	}
	if len(parsed) != len(rows) {
		t.Fatalf("parsed rows = %d, want %d", len(parsed), len(rows))
	}
}

func TestBuildRejectsExistingOutputWithoutChangingIt(t *testing.T) {
	output := filepath.Join(t.TempDir(), "mod.zip")
	original := []byte("keep me")
	if err := os.WriteFile(output, original, 0o644); err != nil {
		t.Fatal(err)
	}

	err := Build(output, localization.English, []localization.DialogueRow{{ID: "id", Text: "text"}})
	if !errors.Is(err, ErrOutputExists) {
		t.Fatalf("Build() error = %v, want errors.Is(..., ErrOutputExists)", err)
	}
	current, readErr := os.ReadFile(output)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(current, original) {
		t.Fatalf("output changed to %q", current)
	}
}

func TestBuildArchiveBytesVersionedUsesProvidedVersion(t *testing.T) {
	archiveBytes, err := buildArchiveBytesVersioned(localization.English, []localization.DialogueRow{{ID: "id", Text: "text"}}, "v0.1.0-rc.1")
	if err != nil {
		t.Fatalf("buildArchiveBytesVersioned() error = %v", err)
	}

	reader, err := zip.NewReader(bytes.NewReader(archiveBytes), int64(len(archiveBytes)))
	if err != nil {
		t.Fatal(err)
	}
	var manifest []byte
	for _, file := range reader.File {
		if file.Name == "kcd_dual_subtitles/mod.manifest" {
			manifest = readZipFile(t, file)
			break
		}
	}
	if !strings.Contains(string(manifest), "<Version>v0.1.0-rc.1</Version>") {
		t.Fatalf("manifest = %q", manifest)
	}
}

func TestBuildArchiveBytesIsDeterministic(t *testing.T) {
	rows := []localization.DialogueRow{{ID: "id", Text: "text"}}
	first, err := buildArchiveBytes(localization.Russian, rows)
	if err != nil {
		t.Fatalf("first buildArchiveBytes() error = %v", err)
	}
	second, err := buildArchiveBytes(localization.Russian, rows)
	if err != nil {
		t.Fatalf("second buildArchiveBytes() error = %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("buildArchiveBytes() is not byte-deterministic for identical input")
	}
}

func TestBuildRejectsUnsupportedLanguage(t *testing.T) {
	output := filepath.Join(t.TempDir(), "mod.zip")

	err := Build(output, localization.Language("Klingon"), []localization.DialogueRow{{ID: "id"}})
	if !errors.Is(err, ErrUnsupportedLanguage) {
		t.Fatalf("Build() error = %v, want errors.Is(..., ErrUnsupportedLanguage)", err)
	}
	if _, statErr := os.Stat(output); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("output exists after unsupported-language failure: %v", statErr)
	}
}

func TestBuildRejectsInvalidDialogueRowWithoutResidue(t *testing.T) {
	directory := t.TempDir()
	output := filepath.Join(directory, "mod.zip")

	err := Build(output, localization.English, []localization.DialogueRow{{ID: "", Text: "text"}})
	if err == nil {
		t.Fatal("Build() error = nil")
	}
	if _, statErr := os.Stat(output); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("output exists after invalid-row failure: %v", statErr)
	}
	entries, readErr := os.ReadDir(directory)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("temporary residue after failed build: %+v", entries)
	}
}

func TestBuildLocalizationPAKUsesStoredEntryWithoutDescriptorOrExtraFields(t *testing.T) {
	pakBytes, err := buildLocalizationPAK([]localization.DialogueRow{{ID: "id", Text: "text"}})
	if err != nil {
		t.Fatalf("buildLocalizationPAK() error = %v", err)
	}
	assertRawZipContract(t, pakBytes, 0, 0)

	reader, err := zip.NewReader(bytes.NewReader(pakBytes), int64(len(pakBytes)))
	if err != nil {
		t.Fatal(err)
	}
	if len(reader.File) != 1 {
		t.Fatalf("entries = %d", len(reader.File))
	}
	file := reader.File[0]
	if file.Name != LocalizationPatchArchivePath {
		t.Fatalf("entry name = %q", file.Name)
	}
	if file.Method != zip.Store {
		t.Fatalf("method = %d, want Store", file.Method)
	}
	if file.Flags != 0 {
		t.Fatalf("flags = %#x, want 0", file.Flags)
	}
	if len(file.Extra) != 0 {
		t.Fatalf("extra field length = %d, want 0", len(file.Extra))
	}
}

func TestBuildDataCryPakUsesCryPakCompatibleVersionFields(t *testing.T) {
	pakBytes, err := buildDataCryPak([]archiveEntry{{name: "Libs/UI/hud.gfx", data: []byte("hud")}})
	if err != nil {
		t.Fatalf("buildDataCryPak() error = %v", err)
	}
	assertRawZipContract(t, pakBytes, kcd2WindowsCreatorVersion, kcd2StoredZIPVersion)
}

func assertRawZipContract(t *testing.T, data []byte, wantCreatorVersion, wantReaderVersion uint16) {
	t.Helper()
	const (
		localHeaderSignature   = 0x04034b50
		centralHeaderSignature = 0x02014b50
	)
	if len(data) < 30 {
		t.Fatalf("ZIP too short: %d", len(data))
	}
	if got := binary.LittleEndian.Uint32(data[:4]); got != localHeaderSignature {
		t.Fatalf("local header signature = %#x", got)
	}
	if got := binary.LittleEndian.Uint16(data[4:6]); got != wantReaderVersion {
		t.Fatalf("local reader version = %d, want %d", got, wantReaderVersion)
	}
	flags := binary.LittleEndian.Uint16(data[6:8])
	if flags != 0 {
		t.Fatalf("local flags = %#x, want 0", flags)
	}
	if method := binary.LittleEndian.Uint16(data[8:10]); method != zip.Store {
		t.Fatalf("local method = %d, want Store", method)
	}
	nameLen := int(binary.LittleEndian.Uint16(data[26:28]))
	extraLen := int(binary.LittleEndian.Uint16(data[28:30]))
	if extraLen != 0 {
		t.Fatalf("local extra length = %d, want 0", extraLen)
	}
	compressedSize := int(binary.LittleEndian.Uint32(data[18:22]))
	centralOffset := 30 + nameLen + compressedSize
	if centralOffset+46 > len(data) {
		t.Fatalf("central header offset %d outside %d-byte ZIP", centralOffset, len(data))
	}
	if got := binary.LittleEndian.Uint32(data[centralOffset : centralOffset+4]); got != centralHeaderSignature {
		t.Fatalf("central header signature = %#x", got)
	}
	if got := binary.LittleEndian.Uint16(data[centralOffset+4 : centralOffset+6]); got != wantCreatorVersion {
		t.Fatalf("central creator version = %d, want %d", got, wantCreatorVersion)
	}
	if got := binary.LittleEndian.Uint16(data[centralOffset+6 : centralOffset+8]); got != wantReaderVersion {
		t.Fatalf("central reader version = %d, want %d", got, wantReaderVersion)
	}
	if flags := binary.LittleEndian.Uint16(data[centralOffset+8 : centralOffset+10]); flags != 0 {
		t.Fatalf("central flags = %#x, want 0", flags)
	}
	if extraLen := binary.LittleEndian.Uint16(data[centralOffset+30 : centralOffset+32]); extraLen != 0 {
		t.Fatalf("central extra length = %d, want 0", extraLen)
	}
}

func readZipFile(t *testing.T, file *zip.File) []byte {
	t.Helper()
	reader, err := file.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func zipFileNames(files []*zip.File) []string {
	names := make([]string, 0, len(files))
	for _, file := range files {
		names = append(names, file.Name)
	}
	sort.Strings(names)
	return names
}
