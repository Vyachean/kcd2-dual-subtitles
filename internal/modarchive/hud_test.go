package modarchive

import (
	"archive/zip"
	"bytes"
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
	nested := openZipBytes(t, dataPAK)
	if got := zipNames(nested); !reflect.DeepEqual(got, []string{HUDArchivePath}) {
		t.Fatalf("HUD PAK entries = %#v, want [%q]", got, HUDArchivePath)
	}
	if got := readZipEntry(t, nested, HUDArchivePath); !bytes.Equal(got, hud) {
		t.Fatalf("HUD bytes = %q, want %q", got, hud)
	}
	if nested.File[0].Method != zip.Store || nested.File[0].Flags&0x8 != 0 || len(nested.File[0].Extra) != 0 {
		t.Fatalf("HUD PAK does not use CryPak-safe raw Store contract: method=%d flags=%#x extra=%x", nested.File[0].Method, nested.File[0].Flags, nested.File[0].Extra)
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
