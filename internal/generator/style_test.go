package generator

import (
	"errors"
	"testing"
)

func TestNormalizeHUDPresentationUsesLiveProvenDefaults(t *testing.T) {
	got, err := normalizeHUDPresentation(SubtitleStyleHUD, nil)
	if err != nil {
		t.Fatalf("normalizeHUDPresentation() error = %v", err)
	}
	want := DefaultHUDPresentationConfig()
	if got != want {
		t.Fatalf("presentation = %+v, want %+v", got, want)
	}
	if !got.SecondaryItalic || !got.ShowLanguageTags {
		t.Fatalf("default booleans = %+v, want italic and language tags enabled", got)
	}
}

func TestNormalizeHUDPresentationAcceptsExplicitPresentation(t *testing.T) {
	configured := HUDPresentationConfig{
		SecondaryColor:   "  #a1B2c3  ",
		SecondarySize:    MinHUDSecondarySize,
		SecondaryItalic:  false,
		ShowLanguageTags: false,
	}
	got, err := normalizeHUDPresentation(SubtitleStyleHUD, &configured)
	if err != nil {
		t.Fatalf("normalizeHUDPresentation() error = %v", err)
	}
	if got.SecondaryColor != "#a1B2c3" || got.SecondarySize != MinHUDSecondarySize || got.SecondaryItalic || got.ShowLanguageTags {
		t.Fatalf("presentation = %+v, want normalized explicit values", got)
	}
}

func TestNormalizeHUDPresentationRejectsInvalidOptions(t *testing.T) {
	valid := DefaultHUDPresentationConfig()
	tests := []struct {
		name  string
		style SubtitleStyle
		edit  func(*HUDPresentationConfig)
	}{
		{
			name:  "options outside HUD mode",
			style: SubtitleStyleTagged,
		},
		{
			name:  "short color",
			style: SubtitleStyleHUD,
			edit:  func(config *HUDPresentationConfig) { config.SecondaryColor = "#FFF" },
		},
		{
			name:  "missing color hash",
			style: SubtitleStyleHUD,
			edit:  func(config *HUDPresentationConfig) { config.SecondaryColor = "112233" },
		},
		{
			name:  "non-hex color",
			style: SubtitleStyleHUD,
			edit:  func(config *HUDPresentationConfig) { config.SecondaryColor = "#12GG33" },
		},
		{
			name:  "size below minimum",
			style: SubtitleStyleHUD,
			edit:  func(config *HUDPresentationConfig) { config.SecondarySize = MinHUDSecondarySize - 1 },
		},
		{
			name:  "size above maximum",
			style: SubtitleStyleHUD,
			edit:  func(config *HUDPresentationConfig) { config.SecondarySize = MaxHUDSecondarySize + 1 },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configured := valid
			if tt.edit != nil {
				tt.edit(&configured)
			}
			_, err := normalizeHUDPresentation(tt.style, &configured)
			if !errors.Is(err, ErrInvalidRequest) {
				t.Fatalf("error = %v, want ErrInvalidRequest", err)
			}
		})
	}
}

func TestNormalizeHUDPresentationAcceptsInclusiveSizeBounds(t *testing.T) {
	for _, size := range []int{MinHUDSecondarySize, MaxHUDSecondarySize} {
		configured := DefaultHUDPresentationConfig()
		configured.SecondarySize = size
		got, err := normalizeHUDPresentation(SubtitleStyleHUD, &configured)
		if err != nil {
			t.Fatalf("size %d: normalizeHUDPresentation() error = %v", size, err)
		}
		if got.SecondarySize != size {
			t.Fatalf("size = %d, want %d", got.SecondarySize, size)
		}
	}
}
