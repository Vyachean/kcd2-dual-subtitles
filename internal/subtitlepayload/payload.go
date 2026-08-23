package subtitlepayload

import (
	"html"
	"strings"
)

const (
	// Prefix/Suffix wrap the secondary subtitle in an HTML comment. The retail
	// HUD prototype keeps this metadata invisible to the vanilla htmlText path,
	// then extracts it from the original fc_setSubtitles text argument.
	Prefix = "<!--KCD2DS1:"
	Suffix = "-->"

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
	// HTML comments cannot contain --. Encode the second dash so the payload
	// cannot terminate its own invisible marker; htmlText decodes it back.
	encoded = strings.ReplaceAll(encoded, "--", "-&#45;")
	return encoded
}

func WrapSecondary(secondary string) string {
	return Prefix + EncodeSecondaryHTML(secondary) + Suffix
}
