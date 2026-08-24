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
	mu       sync.RWMutex
	gameRoot string
}

// Service exposes the small set of application operations needed by the CLI
// and native GUI without leaking archive or load-order implementation details.
// state is shared across value copies so detection/Browse/Generate/Uninstall
// stay bound to the same selected installation.
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
		inspect:   modinstall.InspectForGameRoot,
		generate:  generator.Generate,
		uninstall: modinstall.UninstallForGameRoot,
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
// uninstall operations.
func (s Service) ValidateGameRoot(path string) (string, error) {
	normalized, err := gamedetect.NormalizeSelection(path)
	if err != nil {
		return "", err
	}
	s.rememberGameRoot(normalized)
	return normalized, nil
}

// InspectInstallation returns this tool's generated-mod state for the selected
// KCD2 installation, using the same target resolver as generation.
func (s Service) InspectInstallation() (modinstall.Status, error) {
	root := s.currentGameRoot()
	if root == "" {
		return modinstall.Status{}, ErrGameRootNotSelected
	}
	return s.inspect(root)
}

// InspectInstallationForGameRoot validates/selects gameRoot and inspects the
// exact target automatic generation would use.
func (s Service) InspectInstallationForGameRoot(gameRoot string) (modinstall.Status, error) {
	normalized, err := s.ValidateGameRoot(gameRoot)
	if err != nil {
		return modinstall.Status{}, err
	}
	return s.inspect(normalized)
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
	version := s.version
	if version == "" {
		version = "dev"
	}
	request := generator.Request{
		GameRoot:          normalized,
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
// the currently selected KCD2 installation target.
func (s Service) Uninstall() (modinstall.UninstallResult, error) {
	root := s.currentGameRoot()
	if root == "" {
		return modinstall.UninstallResult{}, ErrGameRootNotSelected
	}
	return s.uninstall(root)
}

func (s Service) rememberGameRoot(gameRoot string) {
	if s.state == nil {
		return
	}
	s.state.mu.Lock()
	s.state.gameRoot = strings.TrimSpace(gameRoot)
	s.state.mu.Unlock()
}

func (s Service) currentGameRoot() string {
	if s.state == nil {
		return ""
	}
	s.state.mu.RLock()
	defer s.state.mu.RUnlock()
	return s.state.gameRoot
}
