//go:build !windows

package generator

import (
	"errors"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/modinstall"
)

func TestGenerateWithoutOutputRejectsAutomaticInstallOutsideWindows(t *testing.T) {
	gameRoot := createGameRoot(t, true, true)

	_, err := Generate(Request{
		GameRoot:          gameRoot,
		MainLanguage:      localization.Russian,
		SecondaryLanguage: localization.English,
	})
	if !errors.Is(err, modinstall.ErrAutomaticInstallUnsupported) {
		t.Fatalf("Generate() error = %v, want ErrAutomaticInstallUnsupported", err)
	}
}
