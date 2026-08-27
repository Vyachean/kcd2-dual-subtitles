# Localization mod compatibility

KCD2 Dual Subtitles builds Main and Secondary text from the effective installed localization, not only from the stock game PAK.

For each selected language the source stack is:

1. stock `<game-root>/Localization/<language>_xml.pak`;
2. active local mods from the currently selected KCD2 Mods folder;
3. later active mods win for the same dialogue localization ID.

The stock table remains the fallback for rows that an applicable localization mod does not override.

## Selected Mods folder

The mod root is not assumed to be a universal `<game-root>/Mods` path. By default the application uses the same `modinstall` layout resolver for source discovery, installation, status, Regenerate, Uninstall, HUD-conflict detection and `mod_order.txt`.

For standard PC layouts (Steam, GOG, Epic and compatible installations), the automatically resolved mod root is:

```text
<game-root>/Mods
```

For Microsoft GDK / Xbox PC layouts, it is:

```text
<Documents>/kingdomcome_mods
```

GDK is selected from package markers in or next to the selected game root, and the Documents path is resolved through the Windows Known Folder API. No localization-source code independently guesses or hardcodes a second mod location.

The Windows GUI displays this resolved path as **Mods folder**. If the user's active mod environment is elsewhere, **Change...** selects an existing custom Mods folder. **Reset** returns to the layout-aware automatic path. A custom selection becomes the single source of truth for source discovery, installation, status, Regenerate, Uninstall and HUD-conflict detection. Selecting a different Game folder clears the custom override so it cannot accidentally carry over to another KCD2 installation.

Custom roots are revalidated when used, not only when selected. A deleted, replaced or otherwise invalid custom path therefore fails instead of silently becoming a different target. Install transactions also record the normalized Mods-root owner. Recovery ignores transactions owned by a sibling Mods environment, and custom-root operations ignore old unowned transaction workspaces from pre-custom-root releases.

## Active mod order

When `mod_order.txt` is absent, applicable local mod directories are applied in deterministic alphabetical folder order. When `mod_order.txt` exists, it is treated as the active whitelist and explicit order. Entries are matched to the manifest `modid` exactly; folder names are not aliases for mod IDs and case is not normalized into another ID.

Whitelist activation is resolved before localization-PAK validation and before `<supports>` evaluation for an explicit-ID mod. An unlisted mod therefore cannot make generation fail merely because its unused selected-language PAK is malformed/non-regular or because its `<supports>` declaration cannot be evaluated. For a listed/otherwise active mod, the selected-language PAK must first exist before its `<supports>` declaration becomes relevant; unrelated language-only mods therefore do not make generation depend on `system.cfg`. Once a mod is relevant to the selected language, its PAK and activation metadata are validated strictly. A repeated active localization-mod ID in `mod_order.txt` fails closed rather than being silently collapsed. A UTF-8 BOM on the first order entry is tolerated and preserved by install/uninstall normalization.

KCD2 manifest activation is also respected. If an active relevant manifest contains `<supports>`, its version patterns are checked against `wh_sys_version` from the selected game's `system.cfg`; a mod that does not support the current game version is not used as a source. If an active relevant localization mod has `<supports>` but the current game version cannot be determined, generation fails closed rather than guessing whether that mod is active.

Warhorse documents that KCD2 can auto-generate a missing `modid` from the human-readable mod name. When `mod_order.txt` is absent, the exact generated ID is not needed for mod load order. For localization resource discovery, an explicit manifest ID is used to validate the documented filename suffix when available. If the ID is auto-generated, its normalization is not documented, so the resolver accepts the documented syntactic generic-localization shape and still constrains its contents to already-known dialogue IDs. When `mod_order.txt` exists, the exact generated ID cannot be reproduced safely; a manifest without explicit `modid` therefore fails closed if it actually contains the selected-language localization PAK. A manifest containing neither `name` nor `modid` is not treated as an active mod identity.

### Generated-mod precedence

Composing a localization mod is useful only if the generated bilingual patch wins the same localization IDs when KCD2 starts.

If an existing `mod_order.txt` is present, installation transactionally normalizes the project entry so exactly one `kcd_dual_subtitles` entry is the final active entry while preserving the relative order of unrelated entries. This makes the generated patch load after the localization sources it composed.

If `mod_order.txt` is absent, the tool does **not** create it: KCD2 treats an order file as a whitelist, so creating one could disable unrelated local or Workshop mods. In this mode, every active selected-language localization mod that can write a relevant dialogue ID participates in precedence safety, even if its displayed value happens to equal the already-effective text. If such a mod folder alphabetically loads after `kcd_dual_subtitles`, automatic installation fails closed with a load-order explanation instead of publishing a patch that KCD2 could later overwrite. Chineses Fix's documented `chinesesfixptf` folder sorts before `kcd_dual_subtitles`, so this specific representative layout does not require an order file for that reason.

KCD2 Dual Subtitles excludes its own canonical mod directory and known legacy staging names from source discovery. Regeneration therefore never consumes an older generated bilingual localization as an input.

## Official text-localization format

Warhorse documents mod text localization as `Localization/<language>_xml.pak`. Inside that text PAK, a mod may contain any number of localization XML files named:

```text
<anything>_<modid>.xml
```

The first cell is the localization `stringId`; the final cell is the text displayed by the game. Warhorse's Skald documentation describes the middle source-language/authoring cell as optional and not loaded, so generic mod-localization rows are accepted in either form:

```xml
<Table>
  <Row>
    <Cell>stringId</Cell>
    <Cell>actual localized text</Cell>
  </Row>
  <Row>
    <Cell>stringId_with_source</Cell>
    <Cell>optional source/authoring text</Cell>
    <Cell>actual localized text</Cell>
  </Row>
</Table>
```

`text__<modid>.xml` and `text_ui__<modid>.xml` are therefore only examples of the general `anything_<modid>.xml` contract, not the complete set of valid prefixes. Warhorse notes that before game patch 1.3 only the `text__<modid>.xml` form loaded reliably; current post-1.3 localization supports the general filename form. Files that do not follow the documented suffix convention may still be seen by the game at the wrong time and produce hash-clash log errors; they are not treated as a separate supported localization format here.

`Localization/<language>.pak` is the voiceover container and is outside this text/subtitle-source feature.

## Supported localization resources

Inside an active mod's selected-language `Localization/<language>_xml.pak`, the source resolver recognizes root-level resources case-insensitively:

- `text_ui_dialog.xml` as an explicit dialogue table/replacement compatibility form; it deliberately remains strict three-cell dialogue data, partial overrides are allowed and new dialogue IDs are retained after inherited stock rows;
- any documented generic `<anything>_<modid>.xml` localization patch. With an explicit manifest `modid`, the filename suffix must correspond to that mod ID; `text__<modid>.xml`, `text_ui__<modid>.xml`, and Chineses Fix's `text__chinesesfixptf.xml` are all instances of this rule. Generic rows accept the documented optional middle cell.

Generic localization patches can contain dialogue alongside items, quests, menus and other UI strings. They therefore may override only IDs already known to the accumulated dialogue table; unknown generic IDs are not reclassified as dialogue. This is what prevents a broad translation mod from turning ordinary interface labels into bilingual text. If an explicit `text_ui_dialog.xml` is present, it is applied first, followed by the mod's generic localization resources.

An explicit dialogue table must contain unique dialogue IDs because it is authoritative enough to introduce new rows. Generic localization patches are different: real KCD2 patch files can repeat localization keys. Within one generic XML, relevant known dialogue rows are processed in file order and the last occurrence of the same dialogue ID is its effective value. Duplicate unknown generic UI rows remain irrelevant because they are never admitted into the dialogue table.

Warhorse permits multiple generic localization XML files in one language PAK but does not document a cross-resource winner when two such files assign different text to the same `stringId`. The resolver therefore does not invent an alphabetical precedence rule: different generic resources may affect different dialogue IDs, and identical displayed text for the same ID is harmless, but conflicting displayed text for one dialogue ID across two generic resources fails closed with both resource names.

Malformed supported XML and case-insensitive duplicate supported resource filenames fail generation with mod/PAK/resource context. Individual supported XML resources are size-limited before parsing so a malformed third-party PAK cannot make generation allocate unbounded memory.

The optional middle XML cell is non-display metadata. A mod is reported in the user-facing applied-localization list only when it changes displayed dialogue text (or explicitly introduces a new dialogue row), not merely when that source/authoring field differs. For runtime precedence, however, any active supported resource that writes a relevant dialogue ID is tracked separately: writing the same single-language text can still overwrite the generated bilingual row if that mod loads later. A mod that does not contain the selected language PAK or whose PAK contains no supported relevant dialogue resource is irrelevant to that language.

### Semantic scope versus file-format support

Supporting every documented text-localization filename does **not** mean every localized string becomes bilingual. The project intentionally limits generic patches to dialogue IDs already known from the stock/effective dialogue table.

Therefore:

- corrections/retranslations of existing KCD2 dialogue and subtitle IDs are composed regardless of the valid `anything_<modid>.xml` prefix used by the translation mod;
- menu, inventory, quest-log, codex, tutorial and other UI strings remain single-language;
- brand-new dialogue introduced by another content/quest mod through only a generic localization patch is not automatically admitted, because the generic file itself does not identify which new IDs are dialogue rather than UI. Supporting that safely would require discovering dialogue references from the content mod, not merely accepting more localization filenames.

This boundary is deliberate: it preserves the product requirement that bilingual output appears only in subtitles/dialogue, not across the whole interface.

### Language fallback boundary

Source composition currently inspects the mod PAK for the **selected installed source language** only. Warhorse's localization documentation states that mods default to English when the user selects a language they do not have, but it does not specify enough runtime precedence semantics to safely infer that an `English_xml.pak` correction should be overlaid onto a different installed stock-language dialogue table. The generator does not invent that cross-language behavior.

This is intentionally conservative: the GUI selects only stock languages actually present in the installation, and a translation/correction mod for that selected language is composed from its matching `<language>_xml.pak`. Cross-language fallback can be added later if retail/runtime evidence establishes the exact KCD2 rule without risking English corrections being applied to the wrong source language.

## Example: Chineses Fix

Nexus Mods #3108 (`Chineses Fix`) installs a Simplified Chinese localization override beneath the active KCD2 mod root. In resolver-neutral form the relevant path is:

```text
<selected-mod-root>/chinesesfixptf/Localization/Chineses_xml.pak
```

That can mean, for example:

```text
<game-root>/Mods/chinesesfixptf/Localization/Chineses_xml.pak
```

on a standard installation, or:

```text
<Documents>/kingdomcome_mods/chinesesfixptf/Localization/Chineses_xml.pak
```

on a Microsoft GDK / Xbox PC installation, or an explicitly selected custom Mods folder in the GUI.

When Simplified Chinese is selected as Main or Secondary and the mod is active in the selected Mods folder, compatible dialogue corrections from that PAK are composed over the stock `Localization/Chineses_xml.pak` before Dual Subtitles creates its bilingual patch.

The implementation is generic and contains no special case for this mod, Nexus ID, language, release version, or storefront layout.

### Inspected 20260727 archive

A user-supplied original Nexus archive for Chineses Fix version `20260727` was inspected during PR #88 without committing or redistributing its proprietary localization text.

The inspected `Chineses_xml.pak` has SHA-256:

```text
d97f73111c834fc380ad28b89c9214e212c2ed77aa76e634e5098e8e5a7cac77
```

Its relevant structure is:

```text
Chineses_xml.pak
└── text__chinesesfixptf.xml
```

That XML is approximately 5.25 MiB uncompressed and contains 22,368 three-cell localization rows. It mixes broad UI/quest localization with spoken-dialogue rows, so treating the entire resource as dialogue would be incorrect. It also contains six repeated localization IDs, including repeats with differing values, which is why generic patches use sequential last-occurrence-wins handling within one resource instead of the strict uniqueness rule used for an explicit `text_ui_dialog.xml`.

Direct comparison with the retail Simplified-Chinese stock dialogue found 18,593 intersecting IDs, of which 17,974 change displayed dialogue text; 3,769 mod IDs are outside stock dialogue and are excluded. The final selected-pair build was then smoke-tested in KCD2 1.5.6 Xbox/GDK: the runtime log showed stock `Chineses_xml.pak`, Chineses Fix, and the generated `kcd_dual_subtitles` localization loading in the required order, and bilingual subtitles worked in game.

## Scope

This feature composes local mods visible in the selected KCD2 Mods folder. It does not claim Steam Workshop localization discovery unless Workshop content is also present through that same selected local-mod layout.

Original game and third-party mod files are never modified.
