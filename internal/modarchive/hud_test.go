package modarchive

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
)

func TestBuildVersionedWithHUDPackagesOneDerivedHUDDataPAK(t *testing.T) {
	output := filepath.Join(t.TempDir(), "mod.zip")
	hud := []byte("derived-hud")
	rows := []localization.DialogueRow{{ID: "id", Text: "text"}}
	if err := BuildVersionedWithHUD(output, localization.English, rows, hud, "v0.3.0-test"); err != nil {
		t.Fatalf("BuildVersionedWithHUD() error = %v", err)
	}
	outerFile, err := zip.OpenReader(output)
	if err != nil {
		t.Fatalf("open outer ZIP: %v", err)
	}
	defer outerFile.Close()

	wantDataPath := modArchivePath("Data/" + DataPAKFilename)
	wantLocalizationPath := modArchivePath("Localization/English_xml.pak")
	wantNames := []string{modArchivePath(ManifestFilename), wantLocalizationPath, wantDataPath}
	if got := zipNames(&outerFile.Reader); !reflect.DeepEqual(got, wantNames) {
		t.Fatalf("outer ZIP entries = %#v, want %#v", got, wantNames)
	}
	dataPAK := readZipEntry(t, &outerFile.Reader, wantDataPath)
	assertRetailDataPAKHeaders(t, dataPAK)

	nested := openZipBytes(t, dataPAK)
	if got := zipNames(nested); !reflect.DeepEqual(got, []string{HUDArchivePath}) {
		t.Fatalf("HUD PAK entries = %#v, want [%q]", got, HUDArchivePath)
	}
	if got := readZipEntry(t, nested, HUDArchivePath); !bytes.Equal(got, hud) {
		t.Fatalf("HUD bytes = %q, want %q", got, hud)
	}
	entry := nested.File[0]
	if entry.Method != zip.Store || entry.Flags&0x8 != 0 || len(entry.Extra) != 0 {
		t.Fatalf("HUD PAK does not use CryPak-safe raw Store contract: method=%d flags=%#x extra=%x", entry.Method, entry.Flags, entry.Extra)
	}
	if entry.CreatorVersion != kcd2WindowsCreatorVersion || entry.ReaderVersion != kcd2StoredZIPVersion {
		t.Fatalf("HUD PAK ZIP versions = creator=%d reader=%d, want creator=%d reader=%d", entry.CreatorVersion, entry.ReaderVersion, kcd2WindowsCreatorVersion, kcd2StoredZIPVersion)
	}
}

func TestDefaultBuildStillContainsNoDataPAK(t *testing.T) {
	archive, err := buildArchiveBytes(localization.Russian, []localization.DialogueRow{{ID: "id", Text: "text"}})
	if err != nil {
		t.Fatalf("buildArchiveBytes() error = %v", err)
	}
	outer := openZipBytes(t, archive)
	for _, name := range zipNames(outer) {
		if filepath.ToSlash(name) == modArchivePath("Data/"+DataPAKFilename) {
			t.Fatalf("default build unexpectedly contains HUD data PAK: %q", name)
		}
	}
}

func TestHUDPackagingRejectsEmptyHUDWithoutResidue(t *testing.T) {
	output := filepath.Join(t.TempDir(), "mod.zip")
	err := BuildVersionedWithHUD(output, localization.Russian, []localization.DialogueRow{{ID: "id", Text: "text"}}, nil, "dev")
	if !errors.Is(err, ErrHUDRequired) {
		t.Fatalf("BuildVersionedWithHUD() error = %v, want ErrHUDRequired", err)
	}
	if _, statErr := os.Stat(output); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("output exists after empty-HUD failure: %v", statErr)
	}
}

func assertRetailDataPAKHeaders(t *testing.T, data []byte) {
	t.Helper()
	if len(data) < 30 || !bytes.Equal(data[:4], []byte{'P', 'K', 0x03, 0x04}) {
		t.Fatalf("Data PAK has no ZIP local header")
	}
	if got := binary.LittleEndian.Uint16(data[4:6]); got != kcd2StoredZIPVersion {
		t.Fatalf("local version-needed = %d, want %d", got, kcd2StoredZIPVersion)
	}
	if got := binary.LittleEndian.Uint16(data[6:8]); got != 0 {
		t.Fatalf("local flags = %#x, want 0", got)
	}
	if got := binary.LittleEndian.Uint16(data[8:10]); got != zip.Store {
		t.Fatalf("local compression method = %d, want Store", got)
	}
	if extraLength := binary.LittleEndian.Uint16(data[28:30]); extraLength != 0 {
		t.Fatalf("local extra length = %d, want 0", extraLength)
	}

	centralOffset := bytes.Index(data, []byte{'P', 'K', 0x01, 0x02})
	if centralOffset < 0 || len(data)-centralOffset < 46 {
		t.Fatalf("Data PAK has no complete central-directory header")
	}
	central := data[centralOffset:]
	if got := binary.LittleEndian.Uint16(central[4:6]); got != kcd2WindowsCreatorVersion {
		t.Fatalf("central creator version = %d, want %d", got, kcd2WindowsCreatorVersion)
	}
	if got := binary.LittleEndian.Uint16(central[6:8]); got != kcd2StoredZIPVersion {
		t.Fatalf("central version-needed = %d, want %d", got, kcd2StoredZIPVersion)
	}
	if got := binary.LittleEndian.Uint16(central[8:10]); got != 0 {
		t.Fatalf("central flags = %#x, want 0", got)
	}
	if got := binary.LittleEndian.Uint16(central[10:12]); got != zip.Store {
		t.Fatalf("central compression method = %d, want Store", got)
	}
	if extraLength := binary.LittleEndian.Uint16(central[30:32]); extraLength != 0 {
		t.Fatalf("central extra length = %d, want 0", extraLength)
	}
}
