package subtitlepayload

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
