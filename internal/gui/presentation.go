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
	SecondaryColor   string
	SecondarySize    string
	SecondaryItalic  bool
}

func defaultPresentationInput() presentationInput {
	defaults := generator.DefaultHUDPresentationConfig()
	return presentationInput{
		Styled:           false,
		ShowLanguageTags: defaults.ShowLanguageTags,
		SecondaryColor:   defaults.SecondaryColor,
		SecondarySize:    strconv.Itoa(defaults.SecondarySize),
		SecondaryItalic:  defaults.SecondaryItalic,
	}
}

func (input presentationInput) hudPresentation() (*generator.HUDPresentationConfig, error) {
	if !input.Styled {
		return nil, nil
	}

	sizeText := strings.TrimSpace(input.SecondarySize)
	size, err := strconv.Atoi(sizeText)
	if err != nil {
		return nil, fmt.Errorf("secondary subtitle size must be a whole number between %d and %d", generator.MinHUDSecondarySize, generator.MaxHUDSecondarySize)
	}
	presentation, err := generator.NormalizeHUDPresentationConfig(generator.HUDPresentationConfig{
		SecondaryColor:   input.SecondaryColor,
		SecondarySize:    size,
		SecondaryItalic:  input.SecondaryItalic,
		ShowLanguageTags: input.ShowLanguageTags,
	})
	if err != nil {
		return nil, err
	}
	return &presentation, nil
}
