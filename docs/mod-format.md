# KCD2 mod format decisions

This document records the format assumptions used by `kcd2-dual-subtitles` and the evidence reviewed through Stage 8. Retail-game rendering remains the final acceptance authority.

## Evidence reviewed

### Official KCD2 modding wiki snapshot

Source mirror: <https://github.com/muyuanjin/kcd2-mod-docs>

Relevant pages:

- Structure of a Mod: <https://github.com/muyuanjin/kcd2-mod-docs/tree/main/official-wiki/KM-A-1%20Modding%20Kingdom%20Come%20Deliverance%202/KM-A-36%20Technical%20Overview/KM-A-3%20Structure%20of%20a%20Mod>
- Publishing a mod: <https://github.com/muyuanjin/kcd2-mod-docs/tree/main/official-wiki/KM-A-1%20Modding%20Kingdom%20Come%20Deliverance%202/KM-A-36%20Technical%20Overview/KM-A-58%20Publishing%20a%20mod>
- Installing mods: <https://github.com/muyuanjin/kcd2-mod-docs/tree/main/official-wiki/KM-A-56%20Installing%20mods>
- Skald localization workflow: <https://github.com/muyuanjin/kcd2-mod-docs/tree/main/official-wiki/KM-A-1%20Modding%20Kingdom%20Come%20Deliverance%202/KM-A-83%20Walkthroughs/KM-A-18%20Skald>

Validated facts:

- `mod.manifest` is loose at the mod root.
- Localization PAKs live at `Localization/<language>_xml.pak` inside the mod folder.
- Published-game localization content must be packed into PAK files.
- A mod id must contain only lowercase letters and underscores.
- Localization rows use three cells: string ID, optional source-language text, target-language text.
- If `mod_order.txt` exists, only mod IDs listed in it are loaded.
- KCD2 provides patch-style integration for data that would otherwise conflict between mods; surgical patches are preferred where supported.

### Real Xbox Store localization files

The Stage 8 user supplied the current Xbox Store `Russian_xml.pak` and `English_xml.pak` privately for structural analysis. Their contents were not committed or uploaded to the repository.

Observed:

- both are valid ZIP/PAK files;
- both contain `text_ui_dialog.xml` at the archive root;
- both contain 177,930 dialogue rows;
- every dialogue row has three cells;
- EN/RU ID sets and ordering match exactly;
- no duplicate dialogue IDs were found;
- the base-game PAK uses Deflate and a CryEngine-specific ZIP extra field `0x4450`.

The base `text_ui_dialog.xml` remains the input source for this tool.

### Current working localization-patch evidence

A current KCD2 mod, Better Arm Of Beowulf (uploaded 2026-08-16 and documented as tested on retail KCD2 1.5.6), modifies existing localization IDs through language PAKs containing a resource named:

```text
text_ui__better_arm_of_beowulf.xml
```

Its documented `kcd.log` success marker is:

```text
[Mod] Loading localization patch 'Localization\text_ui__better_arm_of_beowulf.xml'
```

This is stronger current evidence for existing-ID localization changes than the old Dual Dialog layout or a full replacement of `text_ui_dialog.xml`.

The same mod documents a CryPak failure mode where ordinary ZIP readers accept an archive but KCD2 logs `ReadFile returned 15` when ZIP local-header and central-directory metadata disagree. In particular, mismatched extra-field lengths can shift where CryPak expects file data to begin.

### Stage 8 history

`v0.1.0-rc.1` used a custom localization filename and had no visible effect in the retail Xbox Store game.

`rc.2`/`rc.3` changed to a full `text_ui_dialog.xml` override. That correction has not yet been live-tested, and subsequent compatibility research found a better current mechanism: `text_ui__<modid>.xml` localization patches.

The next release candidate therefore uses the patch mechanism instead of treating the untested full override as final.

### Historical Dual Dialog evidence

The original Dual Dialog generator combined main/secondary subtitle values using the literal two-character `\\n` separator:

<https://github.com/SDxBacon/kcd2-mod-dualdialog-tool/blob/master/export.go>

This remains historical evidence for the subtitle separator only. The project no longer treats the old tool as a current packaging oracle.

## Generated layout

Automatic Xbox Store installation creates:

```text
<Documents>\kingdomcome_mods\kcd_dual_subtitles\
├── mod.manifest
└── Localization\
    └── Russian_xml.pak
```

For an English-main generation, the PAK is `English_xml.pak` instead.

The localization PAK contains exactly:

```text
text_ui__kcd_dual_subtitles.xml
```

The patch XML contains only rows whose final generated text differs from the installed main-language row. Identical translations, missing-secondary fallbacks, and other unchanged rows are not emitted. This reduces conflicts with unrelated localization mods.

## CryPak ZIP contract

The game-facing localization PAK intentionally uses a conservative ZIP representation:

- Store compression;
- no general-purpose bit 3 / data descriptor;
- CRC-32 and compressed/uncompressed sizes precomputed in the local header;
- matching CRC/sizes in the central directory;
- zero local-header extra-field length;
- zero central-directory extra-field length;
- deterministic DOS timestamp;
- no ZIP64 for generated entries.

CI verifies these properties by parsing the raw ZIP bytes independently of Go's `archive/zip` reader, and separately verifies that the standard ZIP reader can still read the archive.

## `mod_order.txt`

The automatic installer does not create `mod_order.txt`.

If `Documents\kingdomcome_mods\mod_order.txt` already exists:

- an existing `kcd_dual_subtitles` entry is left byte-for-byte unchanged;
- if the entry is absent, the installer appends exactly one `kcd_dual_subtitles` line while preserving existing order and newline style where practical;
- unrelated entries are never removed or reordered;
- the update is staged and rollback-safe together with replacement of the tool's own mod directory.

This is required because KCD2 ignores mods not listed in an existing `mod_order.txt`.

## Release identity

`mod.manifest` receives the executable build version. Release-candidate CI verifies the RC version used by the Windows executable is also accepted by the generated manifest contract. Development builds use `dev`.

## Acceptance canary

Normal generation contains no diagnostic marker.

For a controlled live test, `--canary-id <localization-row-id>` prefixes that selected row with:

```text
[KCD2DS TEST] 
```

The ID must already exist in the selected main-language table. Unknown IDs are rejected. No private/game text or game-specific canary ID is committed to the repository.

The canary is intended only to distinguish "the localization patch did not load" from "the patch loaded but bilingual subtitle rendering still has a problem".

## Remaining Stage 8 assumptions

CI can validate archive bytes, source localization structure, merge output, installation safety, and load-order integration. It still cannot prove retail-game behavior. Manual acceptance remains required for:

- the Xbox Store build discovers the installed mod;
- `kcd.log` reports loading `text_ui__kcd_dual_subtitles.xml`;
- an explicit canary row changes when canary mode is enabled;
- literal `\\n` renders as a subtitle line break;
- Russian + English appears in ordinary dialogue;
- Russian + English appears in a story cutscene;
- the game remains stable with the mod enabled.
