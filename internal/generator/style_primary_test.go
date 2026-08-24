package generator

import (
	"errors"
	"testing"
)

func TestDefaultHUDPresentationLeavesPrimaryVanilla(t *testing.T) {
	presentation := DefaultHUDPresentationConfig()
	if presentation.PrimaryColor != "" || presentation.PrimarySize != 0 || presentation.PrimaryItalic {
		t.Fatalf("default primary presentation = %+v, want vanilla properties", presentation)
	}
}

func TestNormalizeHUDPresentationAcceptsPartialPrimaryOverrides(t *testing.T) {
	presentation := DefaultHUDPresentationConfig()
	presentation.PrimaryColor = "  #aBcDeF  "
	presentation.PrimarySize = MinHUDPrimarySize
	presentation.PrimaryItalic = true

	got, err := NormalizeHUDPresentationConfig(presentation)
	if err != nil {
		t.Fatalf("NormalizeHUDPresentationConfig() error = %v", err)
	}
	if got.PrimaryColor != "#aBcDeF" || got.PrimarySize != MinHUDPrimarySize || !got.PrimaryItalic {
		t.Fatalf("normalized primary presentation = %+v", got)
	}
}

func TestNormalizeHUDPresentationRejectsInvalidPrimaryOverrides(t *testing.T) {
	tests := []struct {
		name string
		edit func(*HUDPresentationConfig)
	}{
		{name: "invalid primary color", edit: func(config *HUDPresentationConfig) { config.PrimaryColor = "white" }},
		{name: "primary size below minimum", edit: func(config *HUDPresentationConfig) { config.PrimarySize = MinHUDPrimarySize - 1 }},
		{name: "primary size above maximum", edit: func(config *HUDPresentationConfig) { config.PrimarySize = MaxHUDPrimarySize + 1 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			presentation := DefaultHUDPresentationConfig()
			tt.edit(&presentation)
			_, err := NormalizeHUDPresentationConfig(presentation)
			if !errors.Is(err, ErrInvalidRequest) {
				t.Fatalf("error = %v, want ErrInvalidRequest", err)
			}
		})
	}
}
