# KCD2 mod format decisions

This document records the format assumptions used by `kcd2-dual-subtitles` and the evidence reviewed through Stage 8. Retail-game rendering remains the final acceptance authority.

## Evidence reviewed

### Official KCD2 modding wiki snapshot

Source mirror: <https://github.com/muyuanjin/kcd2-mod-docs>

The repository mirrors the Warhorse Studios YouTrack modding wiki and records its offline snapshot as 2026-06-23.

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
- The documented localization row is three cells: string ID, optional source-language text, target-language text.
- A base-game resource is overridden by placing a file with the identical internal path/name in the mod PAK.

The last point is critical for this project. We modify existing dialogue string IDs from the base `text_ui_dialog.xml`, so the generated localization PAK must itself contain `text_ui_dialog.xml` at the archive root.

The official publishing page describes PAKs as ZIP archives without compression. This project therefore uses ZIP Store for the game-facing localization PAK.

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

The project continues to use Store without timestamp extra fields because that is the conservative format documented by the official publishing guide. The real files prove that Deflate is also used by the game itself, but compression is not required for this generated PAK.

### Stage 8 retail-game failure and correction

`v0.1.0-rc.1` generated the merged existing dialogue IDs under a new file name:

`text_kcd_dual_subtitles.xml`

The mod was installed in the correct Xbox Store location, `Documents\kingdomcome_mods`, but had no visible effect anywhere in the retail game.

That Stage 7 choice conflated two localization use cases:

1. a mod that introduces its own localization keys can ship its own localization XML table;
2. this tool changes existing base-game dialogue IDs and therefore needs to override the original dialogue resource.

The official identical-path override rule, the real Xbox PAK layout, and current bilingual-localization tooling all support using:

`text_ui_dialog.xml`

The generator therefore writes the merged table under the original resource name starting with the Stage 8 correction after `rc.1`.

### Historical Dual Dialog evidence

The original Dual Dialog generator used a separate `text_dualdialog.xml` entry and combined main/secondary subtitle values using the literal two-character `\\n` separator:

<https://github.com/SDxBacon/kcd2-mod-dualdialog-tool/blob/master/export.go>

That behavior remains historical supporting evidence only. The current Xbox Store retail build is the acceptance authority for whether literal `\\n` still renders as a subtitle line break.

## Generated layout

The distributable ZIP contains:

```text
kcd_dual_subtitles/
├── mod.manifest
└── Localization/
    └── Russian_xml.pak
```

For an English-main generation, the PAK is `English_xml.pak` instead.

The localization PAK contains exactly:

```text
text_ui_dialog.xml
```

That matches the internal resource name present in the base localization PAK and is intended to override the existing dialogue table when this mod is loaded after the base game.

The PAK entry uses ZIP Store compression and contains no NTFS (`0x000a`) or extended timestamp (`0x5455`) extra fields.

## Installation location

Installation is platform-specific and is separate from archive generation.

For the Xbox app / Microsoft Store PC build, Stage 8 testing uses:

```text
Documents\kingdomcome_mods\kcd_dual_subtitles\
```

Automatic installation is tracked separately and must not block proving the generated localization override works.

## `mod_order.txt`

The generator does not currently create or modify `mod_order.txt`.

If a user's mod setup already uses an explicit load-order file, `kcd_dual_subtitles` must be enabled through that existing flow. Automatic installation/load-order handling is tracked separately.

## Remaining Stage 8 assumptions

CI can validate archive bytes and structure, but it cannot prove retail-game rendering. After the `text_ui_dialog.xml` correction the remaining manual acceptance items are:

- the current Xbox Store build loads the generated localization override;
- literal `\\n` renders as a subtitle line break;
- Russian + English appears in ordinary dialogue;
- Russian + English appears in a story cutscene;
- the game remains stable with the mod enabled.
