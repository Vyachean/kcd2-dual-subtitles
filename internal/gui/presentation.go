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
}

func defaultPresentationInput() presentationInput {
	return presentationInput{}
}

func (input presentationInput) presentationConfig() (*generator.HUDPresentationConfig, error) {
	// Language tags are a localization-only option and do not require the HUD
	// customization path.
	if !input.Styled {
		if !input.ShowLanguageTags {
			return nil, nil
		}
		return &generator.HUDPresentationConfig{ShowLanguageTags: true}, nil
	}

	secondarySize, err := parseOptionalSecondarySize(input.SecondarySize)
	if err != nil {
		return nil, err
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
	})
	if err != nil {
		return nil, err
	}
	return &presentation, nil
}

func parseOptionalPrimarySize(value string) (int, error) {
	return parseOptionalSize(value, "primary", generator.MinHUDPrimarySize, generator.MaxHUDPrimarySize)
}

func parseOptionalSecondarySize(value string) (int, error) {
	return parseOptionalSize(value, "secondary", generator.MinHUDSecondarySize, generator.MaxHUDSecondarySize)
}

func parseOptionalSize(value, line string, minSize, maxSize int) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	size, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s subtitle size must be empty for vanilla size or a whole number between %d and %d", line, minSize, maxSize)
	}
	return size, nil
}
