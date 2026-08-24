package gui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/generator"
)

type presentationInput struct {
	Styled           bool
	ShowLanguageTags bool
	PrimaryColor     string
	PrimarySize      string
	PrimaryItalic    bool
	SecondaryColor   string
	SecondarySize    string
	SecondaryItalic  bool
	Outline          bool
	Shadow           bool
	ShadowColor      string
}

func defaultPresentationInput() presentationInput {
	defaults := generator.DefaultHUDPresentationConfig()
	return presentationInput{
		Styled:           false,
		ShowLanguageTags: defaults.ShowLanguageTags,
		PrimaryColor:     defaults.PrimaryColor,
		PrimarySize:      "",
		PrimaryItalic:    defaults.PrimaryItalic,
		SecondaryColor:   defaults.SecondaryColor,
		SecondarySize:    strconv.Itoa(defaults.SecondarySize),
		SecondaryItalic:  defaults.SecondaryItalic,
		Outline:          defaults.Outline,
		Shadow:           defaults.Shadow,
		ShadowColor:      defaults.ShadowColor,
	}
}

func (input presentationInput) hudPresentation() (*generator.HUDPresentationConfig, error) {
	if !input.Styled {
		return nil, nil
	}

	secondarySizeText := strings.TrimSpace(input.SecondarySize)
	secondarySize, err := strconv.Atoi(secondarySizeText)
	if err != nil {
		return nil, fmt.Errorf("secondary subtitle size must be a whole number between %d and %d", generator.MinHUDSecondarySize, generator.MaxHUDSecondarySize)
	}

	primarySize, err := parseOptionalPrimarySize(input.PrimarySize)
	if err != nil {
		return nil, err
	}

	presentation, err := generator.NormalizeHUDPresentationConfig(generator.HUDPresentationConfig{
		PrimaryColor:     input.PrimaryColor,
		PrimarySize:      primarySize,
		PrimaryItalic:    input.PrimaryItalic,
		SecondaryColor:   input.SecondaryColor,
		SecondarySize:    secondarySize,
		SecondaryItalic:  input.SecondaryItalic,
		ShowLanguageTags: input.ShowLanguageTags,
		Outline:          input.Outline,
		Shadow:           input.Shadow,
		ShadowColor:      input.ShadowColor,
	})
	if err != nil {
		return nil, err
	}
	return &presentation, nil
}

func parseOptionalPrimarySize(value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	size, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("primary subtitle size must be empty for vanilla size or a whole number between %d and %d", generator.MinHUDPrimarySize, generator.MaxHUDPrimarySize)
	}
	return size, nil
}
