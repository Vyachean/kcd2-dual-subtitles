package modstate

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/generator"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
)

const (
	SettingsFilename        = "settings.json"
	GenerationStateFilename = "generation-state.json"
	settingsSchemaVersion   = 1
	generationSchemaVersion = 1
)

// Settings is user-editable GUI state scoped to one installed KCD2 copy.
type Settings struct {
	MainLanguage      localization.Language
	SecondaryLanguage localization.Language
	Styled            bool
	Presentation      generator.HUDPresentationConfig
}

// GenerationState records the exact source fingerprint used by the last
// successful generation published into the same mod directory.
type GenerationState struct {
	Schema      int                             `json:"schema"`
	ToolVersion string                          `json:"toolVersion"`
	Fingerprint generator.GenerationFingerprint `json:"fingerprint"`
}

// Freshness is intentionally tri-state: partial/malformed legacy evidence can
// never be reported as fresh.
type Freshness string

const (
	FreshnessUnknown Freshness = "unknown"
	FreshnessFresh   Freshness = "fresh"
	FreshnessStale   Freshness = "stale"
)

type settingsFile struct {
	Schema             int                             `json:"schema"`
	MainLanguage       localization.Language           `json:"mainLanguage"`
	SecondaryLanguage  localization.Language           `json:"secondaryLanguage"`
	Styled             bool                            `json:"styled"`
	PrimaryColor       string                          `json:"primaryColor"`
	PrimarySize        int                             `json:"primarySize"`
	PrimaryItalic      bool                            `json:"primaryItalic"`
	SecondaryColor     string                          `json:"secondaryColor"`
	SecondarySize      int                             `json:"secondarySize"`
	SecondaryItalic    bool                            `json:"secondaryItalic"`
	ShowLanguageTags   bool                            `json:"showLanguageTags"`
	Outline            bool                            `json:"outline"`
	Shadow             bool                            `json:"shadow"`
	ShadowColor        string                          `json:"shadowColor"`
	ShadowIntensity    generator.HUDShadowIntensity    `json:"shadowIntensity"`
}

// DefaultSettings returns generator-owned presentation defaults and leaves the
// language pair empty so installed-language selection can choose a valid pair.
func DefaultSettings() Settings {
	return Settings{Presentation: generator.DefaultHUDPresentationConfig()}
}

func SaveSettings(modPath string, settings Settings) error {
	presentation, err := generator.NormalizeHUDPresentationConfig(settings.Presentation)
	if err != nil {
		return fmt.Errorf("validate persisted presentation: %w", err)
	}
	file := settingsFile{
		Schema:            settingsSchemaVersion,
		MainLanguage:      settings.MainLanguage,
		SecondaryLanguage: settings.SecondaryLanguage,
		Styled:            settings.Styled,
		PrimaryColor:      presentation.PrimaryColor,
		PrimarySize:       presentation.PrimarySize,
		PrimaryItalic:     presentation.PrimaryItalic,
		SecondaryColor:    presentation.SecondaryColor,
		SecondarySize:     presentation.SecondarySize,
		SecondaryItalic:   presentation.SecondaryItalic,
		ShowLanguageTags:  presentation.ShowLanguageTags,
		Outline:           presentation.Outline,
		Shadow:            presentation.Shadow,
		ShadowColor:       presentation.ShadowColor,
		ShadowIntensity:   presentation.ShadowIntensity,
	}
	return writeJSON(filepath.Join(modPath, SettingsFilename), file)
}

// LoadSettings overlays independently valid fields onto current defaults.
// Missing/wrong-typed/invalid individual fields fall back without discarding
// the rest of the file. Malformed JSON or an unknown schema returns defaults
// plus a warning error; callers should keep the GUI usable.
func LoadSettings(modPath string) (Settings, error) {
	defaults := DefaultSettings()
	data, err := os.ReadFile(filepath.Join(modPath, SettingsFilename))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return defaults, nil
		}
		return defaults, fmt.Errorf("read settings: %w", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return defaults, fmt.Errorf("parse settings: %w", err)
	}
	var schema int
	if !decodeField(fields, "schema", &schema) || schema != settingsSchemaVersion {
		return defaults, fmt.Errorf("unsupported settings schema")
	}

	settings := defaults
	var language localization.Language
	if decodeField(fields, "mainLanguage", &language) {
		if _, ok := localization.LookupLanguage(language); ok {
			settings.MainLanguage = language
		}
	}
	language = ""
	if decodeField(fields, "secondaryLanguage", &language) {
		if _, ok := localization.LookupLanguage(language); ok {
			settings.SecondaryLanguage = language
		}
	}
	_ = decodeField(fields, "styled", &settings.Styled)

	applyPresentationField(fields, "primaryColor", &settings.Presentation, func(p *generator.HUDPresentationConfig, raw json.RawMessage) bool {
		return json.Unmarshal(raw, &p.PrimaryColor) == nil
	})
	applyPresentationField(fields, "primarySize", &settings.Presentation, func(p *generator.HUDPresentationConfig, raw json.RawMessage) bool {
		return json.Unmarshal(raw, &p.PrimarySize) == nil
	})
	_ = decodeField(fields, "primaryItalic", &settings.Presentation.PrimaryItalic)
	applyPresentationField(fields, "secondaryColor", &settings.Presentation, func(p *generator.HUDPresentationConfig, raw json.RawMessage) bool {
		return json.Unmarshal(raw, &p.SecondaryColor) == nil
	})
	applyPresentationField(fields, "secondarySize", &settings.Presentation, func(p *generator.HUDPresentationConfig, raw json.RawMessage) bool {
		return json.Unmarshal(raw, &p.SecondarySize) == nil
	})
	_ = decodeField(fields, "secondaryItalic", &settings.Presentation.SecondaryItalic)
	_ = decodeField(fields, "showLanguageTags", &settings.Presentation.ShowLanguageTags)
	_ = decodeField(fields, "outline", &settings.Presentation.Outline)
	_ = decodeField(fields, "shadow", &settings.Presentation.Shadow)
	applyPresentationField(fields, "shadowColor", &settings.Presentation, func(p *generator.HUDPresentationConfig, raw json.RawMessage) bool {
		return json.Unmarshal(raw, &p.ShadowColor) == nil
	})
	applyPresentationField(fields, "shadowIntensity", &settings.Presentation, func(p *generator.HUDPresentationConfig, raw json.RawMessage) bool {
		return json.Unmarshal(raw, &p.ShadowIntensity) == nil
	})

	return settings, nil
}

func applyPresentationField(fields map[string]json.RawMessage, name string, presentation *generator.HUDPresentationConfig, apply func(*generator.HUDPresentationConfig, json.RawMessage) bool) {
	raw, ok := fields[name]
	if !ok {
		return
	}
	candidate := *presentation
	if !apply(&candidate, raw) {
		return
	}
	normalized, err := generator.NormalizeHUDPresentationConfig(candidate)
	if err == nil {
		*presentation = normalized
	}
}

func SaveGenerationState(modPath, toolVersion string, fingerprint generator.GenerationFingerprint) error {
	state := GenerationState{
		Schema:      generationSchemaVersion,
		ToolVersion: strings.TrimSpace(toolVersion),
		Fingerprint: fingerprint,
	}
	return writeJSON(filepath.Join(modPath, GenerationStateFilename), state)
}

func LoadGenerationState(modPath string) (GenerationState, error) {
	data, err := os.ReadFile(filepath.Join(modPath, GenerationStateFilename))
	if err != nil {
		return GenerationState{}, err
	}
	var state GenerationState
	if err := json.Unmarshal(data, &state); err != nil {
		return GenerationState{}, fmt.Errorf("parse generation state: %w", err)
	}
	if state.Schema != generationSchemaVersion {
		return GenerationState{}, fmt.Errorf("unsupported generation-state schema %d", state.Schema)
	}
	if strings.TrimSpace(state.ToolVersion) == "" || !validFingerprint(state.Fingerprint) {
		return GenerationState{}, fmt.Errorf("incomplete generation state")
	}
	return state, nil
}

func ClassifyFreshness(saved GenerationState, current generator.GenerationFingerprint, currentToolVersion string) Freshness {
	if saved.Schema != generationSchemaVersion || strings.TrimSpace(saved.ToolVersion) == "" || !validFingerprint(saved.Fingerprint) || !validFingerprint(current) {
		return FreshnessUnknown
	}
	if saved.ToolVersion != strings.TrimSpace(currentToolVersion) {
		return FreshnessStale
	}
	if !reflect.DeepEqual(saved.Fingerprint, current) {
		return FreshnessStale
	}
	return FreshnessFresh
}

func validFingerprint(fingerprint generator.GenerationFingerprint) bool {
	if fingerprint.MainLanguage == "" || fingerprint.SecondaryLanguage == "" || fingerprint.MainDialogueSHA256 == "" || fingerprint.SecondaryDialogueSHA256 == "" || len(fingerprint.TargetLanguages) == 0 {
		return false
	}
	if fingerprint.StyledHUD && fingerprint.RetailHUDSHA256 == "" {
		return false
	}
	if !fingerprint.StyledHUD && fingerprint.RetailHUDSHA256 != "" {
		return false
	}
	return true
}

func decodeField(fields map[string]json.RawMessage, name string, target any) bool {
	raw, ok := fields[name]
	if !ok {
		return false
	}
	return json.Unmarshal(raw, target) == nil
}

func writeJSON(path string, value any) error {
	modPath := filepath.Dir(path)
	info, err := os.Lstat(modPath)
	if err != nil {
		return fmt.Errorf("inspect mod state directory %q: %w", modPath, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("unsafe mod state directory %q", modPath)
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", filepath.Base(path), err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", filepath.Base(path), err)
	}
	return nil
}
