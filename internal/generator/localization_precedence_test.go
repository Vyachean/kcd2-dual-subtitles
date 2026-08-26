package generator

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localizationsource"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/modarchive"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/modinstall"
)

func TestValidateAutomaticLocalizationPrecedenceAllowsEarlierAlphabeticalMod(t *testing.T) {
	modsRoot := t.TempDir()
	contribution := localizationsource.Contribution{
		ModID: "chinesesfixptf",
		Name:  "Chineses Fix",
		Path:  filepath.Join(modsRoot, "chinesesfixptf"),
	}
	if err := validateAutomaticLocalizationPrecedence(Request{ModsRoot: modsRoot}, []localizationsource.Contribution{contribution}); err != nil {
		t.Fatalf("validateAutomaticLocalizationPrecedence() error = %v", err)
	}
}

func TestValidateAutomaticLocalizationPrecedenceRejectsLaterAlphabeticalModWithoutOrderFile(t *testing.T) {
	modsRoot := t.TempDir()
	contribution := localizationsource.Contribution{
		ModID: "russian_fix",
		Name:  "Russian Fix",
		Path:  filepath.Join(modsRoot, "russian_fix"),
	}
	err := validateAutomaticLocalizationPrecedence(Request{ModsRoot: modsRoot}, []localizationsource.Contribution{contribution})
	if !errors.Is(err, ErrUnsafeLocalizationLoadOrder) {
		t.Fatalf("error = %v, want ErrUnsafeLocalizationLoadOrder", err)
	}
}

func TestValidateAutomaticLocalizationPrecedenceAllowsExplicitOrderBecauseInstallerMakesProjectLast(t *testing.T) {
	modsRoot := t.TempDir()
	order := []byte("russian_fix\n" + modarchive.ModID + "\nother_mod\n")
	if err := os.WriteFile(filepath.Join(modsRoot, modinstall.ModOrderFilename), order, 0o600); err != nil {
		t.Fatal(err)
	}
	contribution := localizationsource.Contribution{
		ModID: "russian_fix",
		Name:  "Russian Fix",
		Path:  filepath.Join(modsRoot, "russian_fix"),
	}
	if err := validateAutomaticLocalizationPrecedence(Request{ModsRoot: modsRoot}, []localizationsource.Contribution{contribution}); err != nil {
		t.Fatalf("validateAutomaticLocalizationPrecedence() error = %v", err)
	}
}
