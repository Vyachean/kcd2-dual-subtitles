package application

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/gamedetect"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/generator"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/modinstall"
)

var (
	ErrSameLanguage        = errors.New("main and secondary languages must differ")
	ErrGameRootNotSelected = errors.New("KCD2 game root is not selected")
)

type detectFunc func() (gamedetect.Result, error)
type inspectFunc func(string) (modinstall.Status, error)
type generateFunc func(generator.Request) (generator.Result, error)
type uninstallFunc func(string) (modinstall.UninstallResult, error)

type serviceState struct {
	mu               sync.RWMutex
	gameRoot         string
	modsRootOverride string
}

// Service exposes the small set of application operations needed by the CLI
// and native GUI without leaking archive or load-order implementation details.
// state is shared across value copies so detection/Browse/Generate/Uninstall
// stay bound to the same selected installation and Mods root.
type Service struct {
	version   string
	detect    detectFunc
	inspect   inspectFunc
	generate  generateFunc
	uninstall uninstallFunc
	state     *serviceState
}

// New creates the production application service for one executable version.
func New(version string) Service {
	return Service{
		version:   strings.TrimSpace(version),
		detect:    gamedetect.Detect,
		inspect:   modinstall.InspectInModsRoot,
		generate:  generator.Generate,
		uninstall: modinstall.UninstallFromModsRoot,
		state:     &serviceState{},
	}
}

// DetectGame returns all KCD2 candidates found by the current best-effort
// Windows discovery strategies. Store identity is not part of compatibility;
// callers may validate any compatible installation through ValidateGameRoot.
func (s Service) DetectGame() (gamedetect.Result, error) {
	result, err := s.detect()
	if err == nil {
		if root, ok := result.Unique(); ok {
			s.rememberGameRoot(root)
		}
	}
	return result, err
}

// ValidateGameRoot accepts either a compatible KCD2 root or its immediate
// parent/Content wrapper and remembers the normalized root for later state and
// uninstall operations. Selecting a different game installation clears any
// custom Mods-root override so it cannot leak across installations.
func (s Service) ValidateGameRoot(path string) (string, error) {
	normalized, err := gamedetect.NormalizeSelection(path)
	if err != nil {
		return "", err
	}
	s.rememberGameRoot(normalized)
	return normalized, nil
}

// SelectedModsLocation returns the one Mods root that every mod-facing
// operation must use for the selected game. A user override wins over automatic
// layout resolution until the game root changes or ResetModsRootOverride is
// called.
func (s Service) SelectedModsLocation() (modinstall.InstallLocation, error) {
	root, override := s.currentSelection()
	if root == "" {
		return modinstall.InstallLocation{}, ErrGameRootNotSelected
	}
	if override != "" {
		return modinstall.InstallLocation{ModsRoot: override, Layout: modinstall.InstallLayoutCustom}, nil
	}
	return modinstall.ResolveModSourceLocation(root)
}

// SetModsRootOverride validates and remembers an explicit Mods folder for the
// currently selected game installation.
func (s Service) SetModsRootOverride(path string) (modinstall.InstallLocation, error) {
	if s.currentGameRoot() == "" {
		return modinstall.InstallLocation{}, ErrGameRootNotSelected
	}
	location, err := modinstall.ValidateCustomModsRoot(path)
	if err != nil {
		return modinstall.InstallLocation{}, err
	}
	if s.state == nil {
		return modinstall.InstallLocation{}, errors.New("application service state is unavailable")
	}
	s.state.mu.Lock()
	s.state.modsRootOverride = location.ModsRoot
	s.state.mu.Unlock()
	return location, nil
}

// ResetModsRootOverride returns the selected game to its layout-aware automatic
// Mods root.
func (s Service) ResetModsRootOverride() (modinstall.InstallLocation, error) {
	if s.state == nil {
		return modinstall.InstallLocation{}, errors.New("application service state is unavailable")
	}
	s.state.mu.Lock()
	root := s.state.gameRoot
	s.state.modsRootOverride = ""
	s.state.mu.Unlock()
	if root == "" {
		return modinstall.InstallLocation{}, ErrGameRootNotSelected
	}
	return modinstall.ResolveModSourceLocation(root)
}

// InspectInstallation returns this tool's generated-mod state for the selected
// KCD2 installation and selected Mods root.
func (s Service) InspectInstallation() (modinstall.Status, error) {
	location, err := s.SelectedModsLocation()
	if err != nil {
		return modinstall.Status{}, err
	}
	return s.inspect(location.ModsRoot)
}

// InspectInstallationForGameRoot validates/selects gameRoot and inspects the
// exact target automatic generation would use. Changing gameRoot resets a prior
// custom Mods override before this inspection.
func (s Service) InspectInstallationForGameRoot(gameRoot string) (modinstall.Status, error) {
	if _, err := s.ValidateGameRoot(gameRoot); err != nil {
		return modinstall.Status{}, err
	}
	return s.InspectInstallation()
}

// GenerateAndInstall preserves the existing non-HUD tagged GUI/application
// behavior. Styled generation uses GenerateAndInstallWithPresentation.
func (s Service) GenerateAndInstall(gameRoot string, main, secondary localization.Language) (generator.Result, error) {
	return s.GenerateAndInstallWithPresentation(gameRoot, main, secondary, nil)
}

// GenerateAndInstallWithPresentation validates the selected game root,
// explicit language choices and optional HUD presentation, then performs the
// existing safe generator/install operation. A nil presentation preserves the
// legacy tagged path; a non-nil presentation explicitly selects HUD mode.
func (s Service) GenerateAndInstallWithPresentation(gameRoot string, main, secondary localization.Language, presentation *generator.HUDPresentationConfig) (generator.Result, error) {
	if main == secondary {
		return generator.Result{}, ErrSameLanguage
	}

	var normalizedPresentation *generator.HUDPresentationConfig
	if presentation != nil {
		normalized, err := generator.NormalizeHUDPresentationConfig(*presentation)
		if err != nil {
			return generator.Result{}, fmt.Errorf("validate subtitle presentation: %w", err)
		}
		normalizedPresentation = &normalized
	}

	normalized, err := s.ValidateGameRoot(gameRoot)
	if err != nil {
		return generator.Result{}, err
	}
	_, modsRootOverride := s.currentSelection()
	version := s.version
	if version == "" {
		version = "dev"
	}
	request := generator.Request{
		GameRoot:          normalized,
		ModsRoot:          modsRootOverride,
		MainLanguage:      main,
		SecondaryLanguage: secondary,
		Version:           version,
	}
	if normalizedPresentation != nil {
		request.SubtitleStyle = generator.SubtitleStyleHUD
		request.HUDPresentation = normalizedPresentation
	}
	result, err := s.generate(request)
	if err != nil {
		return generator.Result{}, fmt.Errorf("generate and install: %w", err)
	}
	return result, nil
}

// Uninstall removes only this tool's generated mod and load-order entries from
// the currently selected Mods root.
func (s Service) Uninstall() (modinstall.UninstallResult, error) {
	location, err := s.SelectedModsLocation()
	if err != nil {
		return modinstall.UninstallResult{}, err
	}
	return s.uninstall(location.ModsRoot)
}

func (s Service) rememberGameRoot(gameRoot string) {
	if s.state == nil {
		return
	}
	gameRoot = strings.TrimSpace(gameRoot)
	s.state.mu.Lock()
	if s.state.gameRoot != "" && !samePathSelection(s.state.gameRoot, gameRoot) {
		s.state.modsRootOverride = ""
	}
	s.state.gameRoot = gameRoot
	s.state.mu.Unlock()
}

func samePathSelection(left, right string) bool {
	return strings.EqualFold(strings.TrimSpace(left), strings.TrimSpace(right))
}

func (s Service) currentSelection() (gameRoot, modsRootOverride string) {
	if s.state == nil {
		return "", ""
	}
	s.state.mu.RLock()
	defer s.state.mu.RUnlock()
	return s.state.gameRoot, s.state.modsRootOverride
}

func (s Service) currentGameRoot() string {
	root, _ := s.currentSelection()
	return root
}
