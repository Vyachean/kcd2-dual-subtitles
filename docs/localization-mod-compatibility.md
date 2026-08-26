# Localization mod compatibility

KCD2 Dual Subtitles builds Main and Secondary text from the effective installed localization, not only from the stock game PAK.

For each selected language the source stack is:

1. stock `<game-root>/Localization/<language>_xml.pak`;
2. active local mods from the same resolved Mods root used by installation;
3. later active overrides win for the same dialogue localization ID.

The stock table remains the fallback for rows that a localization mod does not override.

## Active mod order

For a standard PC installation, local mods are read from `<game-root>/Mods`. Microsoft GDK installations use the same resolved `Documents/kingdomcome_mods` root as automatic installation.

When `mod_order.txt` is absent, applicable local mod directories are applied in deterministic alphabetical folder order. When `mod_order.txt` exists, it is treated as the active whitelist and explicit order. Manifest `modid` values are used when available, so a mod folder does not need to have the same name as its ID.

KCD2 Dual Subtitles excludes its own canonical mod directory and known legacy staging names from source discovery. Regeneration therefore never consumes an older generated bilingual localization as an input.

## Supported localization resources

Inside an active mod's exact `Localization/<language>_xml.pak`, the source resolver recognizes:

- `text_ui_dialog.xml` as an explicit dialogue table; partial overrides are allowed and new dialogue IDs are retained after inherited stock rows;
- `text_ui__*.xml` localization patch resources. Only IDs already known to the effective dialogue table are consumed from these generic `text_ui` patches, preventing unrelated UI strings from being reclassified as dialogue.

Malformed supported dialogue resources and duplicate IDs inside one resource fail generation with the mod/PAK/resource context instead of silently producing a partial merge.

A mod that does not contain the selected language PAK, or whose PAK contains no supported dialogue resource, is irrelevant to that language and is skipped.

## Example: Chineses Fix

Nexus Mods #3108 (`Chineses Fix`) installs a Simplified Chinese localization override under a normal KCD2 mod directory, for example:

```text
Mods/chinesesfixptf/Localization/Chineses_xml.pak
```

When Simplified Chinese is selected as Main or Secondary and the mod is active in the resolved Mods root, compatible dialogue corrections from that PAK are composed over the stock `Localization/Chineses_xml.pak` before Dual Subtitles creates its bilingual patch.

The implementation is generic and contains no special case for this mod, Nexus ID, language, or release version.

## Scope

This feature composes local mods visible in the resolved KCD2 Mods root. It does not claim Steam Workshop localization discovery unless Workshop content is also present through that same resolved local-mod layout.

Original game and third-party mod files are never modified.
