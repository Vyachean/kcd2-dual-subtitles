package application

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/gamedetect"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/generator"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/modinstall"
)

var ErrSameLanguage = errors.New("main and secondary languages must differ")

type detectFunc func() (gamedetect.Result, error)
type inspectFunc func() (modinstall.Status, error)
type generateFunc func(generator.Request) (generator.Result, error)
type uninstallFunc func() (modinstall.UninstallResult, error)

// Service exposes the small set of application operations needed by the CLI
// and native GUI without leaking archive or load-order implementation details.
type Service struct {
	version   string
	detect    detectFunc
	inspect   inspectFunc
	generate  generateFunc
	uninstall uninstallFunc
}

// New creates the production application service for one executable version.
func New(version string) Service {
	return Service{
		version:   strings.TrimSpace(version),
		detect:    gamedetect.Detect,
		inspect:   modinstall.Inspect,
		generate:  generator.Generate,
		uninstall: modinstall.Uninstall,
	}
}

// DetectGame returns all structurally valid Xbox / Microsoft Store KCD2
// candidates. Callers may auto-select only when Result.Unique succeeds.
func (s Service) DetectGame() (gamedetect.Result, error) {
	return s.detect()
}

// ValidateGameRoot accepts either Content or its immediate parent.
func (s Service) ValidateGameRoot(path string) (string, error) {
	return gamedetect.NormalizeSelection(path)
}

// InspectInstallation returns this tool's current generated-mod state.
func (s Service) InspectInstallation() (modinstall.Status, error) {
	return s.inspect()
}

// GenerateAndInstall validates the selected game root and explicit language
// choices, then performs the existing safe generator/install operation.
func (s Service) GenerateAndInstall(gameRoot string, main, secondary localization.Language) (generator.Result, error) {
	if main == secondary {
		return generator.Result{}, ErrSameLanguage
	}
	normalized, err := s.ValidateGameRoot(gameRoot)
	if err != nil {
		return generator.Result{}, err
	}
	version := s.version
	if version == "" {
		version = "dev"
	}
	result, err := s.generate(generator.Request{
		GameRoot:          normalized,
		MainLanguage:      main,
		SecondaryLanguage: secondary,
		Version:           version,
	})
	if err != nil {
		return generator.Result{}, fmt.Errorf("generate and install: %w", err)
	}
	return result, nil
}

// Uninstall removes only this tool's generated mod and load-order entries.
func (s Service) Uninstall() (modinstall.UninstallResult, error) {
	return s.uninstall()
}
