# KCD2 mod format decisions

This document records the format assumptions used by `kcd2-dual-subtitles` and the evidence reviewed for Stage 7. In-game Xbox Store verification remains Stage 8.

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

- A manually installed mod is a subdirectory below the game's `Mods/` directory.
- `mod.manifest` is loose at the mod root.
- Localization PAKs live at `Localization/<language>_xml.pak` inside that mod folder.
- Published-game localization content must be packed into PAK files.
- A mod id must contain only lowercase letters and underscores.
- `mod_order.txt` is optional. If it exists, only listed mod ids load; without it, ordinary `Mods/` folders are loaded alphabetically after Steam Workshop mods.
- The documented localization row is three cells: string ID, optional source-language text, target-language text.

The official publishing page describes PAKs as ZIP archives without compression. The maintained tooling below reports that both Store and Deflate are accepted by current CryPak; this project uses Store for localization PAKs because it is the conservative intersection of both sources and the file is small enough that compression is not important.

### Current KCD2 PAK tooling

`muyuanjin/kcd2-mod-docs` includes a current `kcd2_pak.py` validator/packer, updated 2026-06-28:

<https://github.com/muyuanjin/kcd2-mod-docs/blob/main/.agents/skills/kcd2-mod-workflow/scripts/kcd2_pak.py>

It enforces `^[a-z_]+$` for mod ids and accepts Store or Deflate compression.

`tkhquang/kcd-pak-action`, reviewed in August 2026, documents a more specific CryPak compatibility issue: ZIP extended timestamp metadata written by some archivers can make a PAK unloadable, while Store and Deflate both work. It produces a deploy-ready `<modid>/` folder containing `mod.manifest` and localization PAKs:

<https://github.com/tkhquang/kcd-pak-action>

Go's `archive/zip.FileHeader.Modified` documentation states that an extended timestamp is always emitted when writing a non-zero `Modified` value. Therefore game-facing PAK entries in this project intentionally leave `Modified` zero and populate only the legacy MS-DOS date/time fields. CI tests inspect the resulting PAK extra fields so this does not regress.

### Current localization mod evidence

Loot Info 1.4.1, updated 2025-11-18 and requiring KCD2 1.5.x, ships independent per-mod localization XML files such as `Localization/Russian_xml/text_ui_LootInfo.xml`. This demonstrates that a mod localization PAK can contribute an additional XML table rather than replacing the base `text_ui_dialog.xml` file.

Nexus page/documentation:
<https://www.nexusmods.com/kingdomcomedeliverance2/mods/1124>

The generated dialogue table is therefore kept as its own file. Its project-owned name is:

`text_kcd_dual_subtitles.xml`

This also aligns the generated filename with the project mod id instead of retaining the historical `text_dualdialog.xml` name from another mod.

### Historical Dual Dialog evidence

The original Dual Dialog generator used a separate `text_dualdialog.xml` entry and combined main/secondary subtitle values using the literal two-character `\\n` separator:

<https://github.com/SDxBacon/kcd2-mod-dualdialog-tool/blob/master/export.go>

Its published mod reports support for ordinary dialogue, overhead dialogue, and dialogue sequences. This is useful evidence for the separator and separate localization-table approach, but it is not treated as authoritative for current KCD2 behavior.

The literal `\\n` separator remains in v0.1. Actual rendered line separation in the current Xbox Store build is an explicit Stage 8 acceptance item.

## Validated generated layout

The distributable ZIP is directly extractable into the game's `Mods/` directory:

```text
kcd_dual_subtitles/
├── mod.manifest
└── Localization/
    └── Russian_xml.pak
```

For an English-main generation, the PAK is `English_xml.pak` instead.

The localization PAK contains:

```text
text_kcd_dual_subtitles.xml
```

The PAK entry uses ZIP Store compression and contains no NTFS (`0x000a`) or extended timestamp (`0x5455`) extra fields.

## `mod_order.txt`

The generator does not create or modify `mod_order.txt`.

This is intentional because the official installation documentation makes it optional. If a user's installation already has `mod_order.txt`, they must add `kcd_dual_subtitles` to it or use their mod manager to enable the mod. Installation guidance is finalized in Stages 8-9.

## Remaining Stage 8 assumptions

CI can validate archive bytes and structure, but it cannot prove retail-game behavior. The following remain manual acceptance items on the current Xbox Store PC build:

- the mod folder is discovered and loaded at the documented Xbox-compatible installation location;
- `text_kcd_dual_subtitles.xml` is merged by the retail build;
- literal `\\n` renders as a subtitle line break;
- Russian + English appears in ordinary dialogue and a story cutscene.
