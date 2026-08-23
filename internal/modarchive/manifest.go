package modarchive

import "strings"

const ModID = "kcd_dual_subtitles"

func manifestForVersion(version string) []byte {
	version = strings.TrimSpace(version)
	if version == "" {
		version = "dev"
	}
	version = escapeXMLText(version)

	return []byte(`<?xml version="1.0" encoding="utf-8"?>
<kcd_mod>
    <info>
        <name>KCD2 Dual Subtitles</name>
        <modid>` + ModID + `</modid>
        <description>Generated bilingual subtitle localization.</description>
        <author>Vyachean</author>
        <version>` + version + `</version>
        <created_on>2026-08-23</created_on>
        <modifies_level>false</modifies_level>
    </info>
</kcd_mod>
`)
}

func escapeXMLText(value string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&apos;",
	)
	return replacer.Replace(value)
}
