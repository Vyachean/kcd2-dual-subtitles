# Localization mod compatibility

KCD2 Dual Subtitles builds Main and Secondary text from the effective installed localization, not only from the stock game PAK.

For each selected language the source stack is:

1. stock `<game-root>/Localization/<language>_xml.pak`;
2. active local mods from the same resolved mod root used by installation/status/uninstall;
3. later active overrides win for the same dialogue localization ID.

The stock table remains the fallback for rows that a localization mod does not override.

## Resolved mod root

The mod root is not assumed to be a universal `<game-root>/Mods` path. Source discovery uses the same `modinstall` layout resolver as the rest of the application.

For standard PC layouts (Steam, GOG, Epic and compatible installations), the resolved mod root is:

```text
<game-root>/Mods
```

For Microsoft GDK / Xbox PC layouts, it is:

```text
<Documents>/kingdomcome_mods
```

GDK is selected from package markers in or next to the selected game root, and the Documents path is resolved through the Windows Known Folder API. No localization-source code independently guesses or hardcodes a second mod location.

## Active mod order

When `mod_order.txt` is absent, applicable local mod directories are applied in deterministic alphabetical folder order. When `mod_order.txt` exists, it is treated as the active whitelist and explicit order. Explicit manifest `modid` values are used, so a mod folder does not need to have the same name as its ID.

KCD2 manifest activation is also respected. If a manifest contains `<supports>`, its version patterns are checked against `wh_sys_version` from the selected game's `system.cfg`; a mod that does not support the current game version is not used as a source. If a relevant localization mod has `<supports>` but the current game version cannot be determined, generation fails closed rather than guessing whether that mod is active.

Warhorse documents that KCD2 can auto-generate a missing `modid` from the human-readable mod name, but the exact normalization contract is not documented. A relevant localization mod without an explicit `modid` therefore fails generation with a clear error instead of being silently ignored or assigned a guessed identity. A non-empty invalid `modid` is treated as not loadable by the normal KCD2 mod rules.

KCD2 Dual Subtitles excludes its own canonical mod directory and known legacy staging names from source discovery. Regeneration therefore never consumes an older generated bilingual localization as an input.

## Supported localization resources

Inside an active mod's exact `Localization/<language>_xml.pak`, the source resolver recognizes:

- `text_ui_dialog.xml` as an explicit dialogue table; partial overrides are allowed and new dialogue IDs are retained after inherited stock rows;
- `text_ui__*.xml` localization patch resources. Only IDs already known to the effective dialogue table are consumed from these generic `text_ui` patches, preventing unrelated UI strings from being reclassified as dialogue.

If one PAK contains both forms, `text_ui_dialog.xml` is applied first as the dialogue table and `text_ui__*.xml` resources are applied afterwards as patch layers. Multiple patch resources are ordered deterministically by archive path.

Malformed supported dialogue resources and duplicate IDs inside one resource fail generation with the mod/PAK/resource context instead of silently producing a partial merge.

A mod that does not contain the selected language PAK, or whose PAK contains no supported dialogue resource, is irrelevant to that language and is skipped.

## Example: Chineses Fix

Nexus Mods #3108 (`Chineses Fix`) installs a Simplified Chinese localization override beneath the active KCD2 mod root. In resolver-neutral form the relevant path is:

```text
<resolved-mod-root>/chinesesfixptf/Localization/Chineses_xml.pak
```

That means, for example:

```text
<game-root>/Mods/chinesesfixptf/Localization/Chineses_xml.pak
```

on a standard installation, or:

```text
<Documents>/kingdomcome_mods/chinesesfixptf/Localization/Chineses_xml.pak
```

on a Microsoft GDK / Xbox PC installation.

When Simplified Chinese is selected as Main or Secondary and the mod is active in the resolved mod root, compatible dialogue corrections from that PAK are composed over the stock `Localization/Chineses_xml.pak` before Dual Subtitles creates its bilingual patch.

The implementation is generic and contains no special case for this mod, Nexus ID, language, release version, or storefront layout.

## Scope

This feature composes local mods visible in the resolved KCD2 mod root. It does not claim Steam Workshop localization discovery unless Workshop content is also present through that same resolved local-mod layout.

Original game and third-party mod files are never modified.
