package subtitlepayload

import (
	"html"
	"strings"
)

const (
	// Prefix/Suffix deliberately use plain text for the next retail diagnostic.
	// If the derived HUD is not active, the marker remains visibly present in the
	// subtitle and proves that localization payload transport is working. If the
	// wrapper is active, it can detect the same marker and append the styled
	// secondary line. This avoids the ambiguity of the rc.2 HTML-comment carrier,
	// which vanilla rendering could hide even when the HUD override was absent.
	Prefix = "[KCD2DS1|"
	Suffix = "|KCD2DS1]"

	HUDWrapperMarker = "KCD2DS_HUD_WRAPPER_V1"

	// Deliberately obvious acceptance styling. The production palette can be
	// toned down after the direct-HUD path is proven in retail.
	SecondaryColor = "#7FDBFF"
	SecondarySize  = 24
)

// EncodeSecondaryHTML turns localization text into safe htmlText content for
// the HUD wrapper. The dialogue table already uses <br/> intentionally, so the
// known line-break forms are preserved while arbitrary markup is escaped.
func EncodeSecondaryHTML(text string) string {
	encoded := html.EscapeString(text)
	encoded = strings.ReplaceAll(encoded, "&lt;br/&gt;", "<br/>")
	encoded = strings.ReplaceAll(encoded, "&lt;br /&gt;", "<br />")
	encoded = strings.ReplaceAll(encoded, "&lt;br&gt;", "<br>")
	return encoded
}

func WrapSecondary(secondary string) string {
	return Prefix + EncodeSecondaryHTML(secondary) + Suffix
}
