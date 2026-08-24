# KCD2 Dual Subtitles — generated mod format

This document records the game-facing layout and compatibility rules used by the stable v0.3 generation path.

## Principles

- Never modify the original KCD2 installation.
- Generate only project-owned patch resources from files already installed by the user.
- Never redistribute retail localization tables or a prebuilt proprietary `hud.gfx`.
- Keep game-facing PAK/ZIP metadata deterministic and compatible with KCD2 CryPak.
- Fail closed on unknown HUD structure or a foreign HUD override.

## Input data

For each selected source language, the generator reads:

```text
<Content>\Localization\<language>_xml.pak
└── text_ui_dialog.xml
```

The supported language filename/tag registry is explicit in `internal/localization/language.go`. Unknown future `*_xml.pak` files are not assigned guessed metadata.

The selected Main and Secondary languages are **text sources only**. They do not determine which single localization slot the generated mod targets.

## Generated installation layout

Automatic installation writes under the Windows Documents Known Folder:

```text
<Documents>\kingdomcome_mods\kcd_dual_subtitles\
├── mod.manifest
├── Localization\
│   ├── <installed-supported-language-1>_xml.pak
│   ├── <installed-supported-language-2>_xml.pak
│   └── ...
└── Data\
    └── kcd_dual_subtitles.pak      # styled HUD mode only
```

Each generated localization PAK contains exactly:

```text
text_ui__kcd_dual_subtitles.xml
```

Styled mode additionally creates:

```text
Data\kcd_dual_subtitles.pak
└── Libs/UI/hud.gfx
```

The HUD is derived from the user's current installed retail HUD. It is not stored as a source fixture or shipped as a static binary by the project.

## Why localization is emitted under every installed supported language slot

KCD2 loads localization patches through the game's currently active language slot. A patch written only as `Czech_xml.pak`, for example, is ignored when the game is currently using English text.

Stable v0.3 therefore emits the same generated bilingual patch under **every supported localization PAK actually present in the selected installation**.

This makes the chosen subtitle pair independent from the game's active UI/text language.

The payload is still generated from only the selected Main/Secondary source tables.

## Patch XML

`text_ui__kcd_dual_subtitles.xml` contains only dialogue rows whose generated value differs from the selected Main-language source value.

Rows that remain unchanged are omitted. The merge contract preserves:

- identical translations;
- missing secondary translations;
- empty main/secondary fallback cases;
- secondary-only cases according to generator rules.

Normal output contains no private/game-specific diagnostics.

## Tagged bilingual format

The non-styled path emits compact labels separated by the literal game-facing `\n` sequence, for example:

```text
[EN] Primary text\n[DE] Secondary text
```

The literal two-character `\n` separator is intentional. Retail KCD2 acceptance confirmed that it renders as a line break in dialogue subtitles.

Identical/single-language fallback rows remain untagged.

## Styled bilingual format

Styled mode generates controlled HTML in the localization row and restores that HTML after vanilla subtitle sizing in the derived HUD.

Conceptually, a default bilingual row is:

```html
[EN] Primary text<br/><font color='#7FDBFF' size='24'><i>[DE] Secondary text</i></font>
```

There is deliberately no forced `<p align='center'>` wrapper. A retail RC showed that forced centering leaked into dialogue-choice consumers and displaced their layout.

Presentation options can alter:

- primary color;
- primary size;
- primary italic;
- secondary color;
- secondary size;
- secondary italic;
- language tags.

Outline and shadow are not encoded as HTML. They are optional whole-TextField properties applied by the derived HUD transformation.

## Dialogue-row escaping

Game/user text must be escaped before being embedded into project-owned HTML so literal markup characters cannot become unintended formatting.

Existing line-break behavior is preserved according to the formatter's tested contract.

## Derived HUD contract

Styled mode reads the user's current:

```text
Data\IPL_GameData.pak
└── Libs/UI/hud.gfx
```

and applies narrow semantic AVM1 transformations.

The stable patch handles two retail render paths independently:

```text
fc_setSubtitles(text, speakerName, isPlayer)
fc_setBubbleText(bubbleId, bubbleText, speakerName, playerDistance)
```

The transforms:

- locate functions structurally;
- validate expected parameters/registers/member/string anchors;
- preserve the vanilla function body;
- save the original generated HTML before vanilla processing;
- restore `htmlText` at normal fallthrough after vanilla sizing;
- refresh only the required position/background geometry;
- include explicit project idempotence markers;
- reject missing/ambiguous/incompatible targets.

The separate narrative path (`fc_setNarrativeSubtitles`) is not part of the stable v0.3 HUD transformation.

## Readability properties

Optional readability uses direct Scaleform TextField extensions.

Current stable configuration:

### Outline

```text
outline = 1
```

### Shadow

```text
shadowColor    = 0x000000
shadowAlpha    = 1
shadowBlurX    = 2
shadowBlurY    = 2
shadowAngle    = 45
shadowDistance = 1
shadowQuality  = 1
shadowStrength = 1
```

The effects apply to the complete bilingual TextField. When both outline and shadow are disabled, the generator uses the existing direct-HTML patch path without readability-property writes.

## Localization PAK ZIP contract

Generated localization PAKs use a deliberately conservative representation:

- Store compression;
- no general-purpose bit 3 / data descriptor;
- CRC-32 and sizes precomputed in the local header;
- matching CRC/sizes in the central directory;
- zero local-header extra-field length;
- zero central-directory extra-field length;
- deterministic DOS timestamp;
- no ZIP64 for generated entries.

CI parses raw ZIP bytes independently of Go's ZIP reader and separately verifies ordinary ZIP readability.

## Data PAK CryPak contract

A ZIP can be valid to ordinary tools while still being rejected by KCD2 CryPak. The generated HUD Data PAK therefore uses the project's dedicated game-facing writer with tested ZIP version/header metadata.

Do not replace it with a generic ZIP writer without preserving the raw-header contract covered by tests.

## `mod.manifest`

`mod.manifest` is loose at the generated mod root and receives the executable/release build version.

Development builds use `dev`; release and release-candidate workflows inject their expected tag identity and test that the built executable reports the same version.

## `mod_order.txt`

The installer does not create `mod_order.txt` when absent.

If an existing file is present:

- an existing `kcd_dual_subtitles` entry is preserved;
- if missing, exactly one project entry is appended;
- unrelated entries are never removed or reordered;
- newline style is preserved where practical;
- publication/load-order changes participate in staged rollback behavior.

Uninstall removes only this project's matching entries.

## Foreign HUD conflicts

Because styled mode supplies `Libs/UI/hud.gfx`, the installer scans other installed mods for a foreign HUD supplied either as a loose file or inside a Data PAK.

If a foreign HUD is found, styled installation stops with an explicit conflict rather than silently changing load precedence or patching an unvalidated third-party binary.

## Installation publication and OneDrive fallback

Normal installation is built in a staging directory under `kingdomcome_mods` and published by same-volume rename.

Windows/OneDrive can transiently deny the final rename even when all writes succeeded. For retryable permission/sharing failures the installer:

1. performs bounded rename retries;
2. if they remain unsuccessful, uses a guarded recursive copy into an absent target;
3. rejects symlinks, non-regular staged files and unexpected target state;
4. removes partial output and restores the previous installation on failure.

The guarded copy path is a compatibility fallback, not the normal publication mechanism.

## Acceptance canary

For controlled diagnostics only, `--canary-id <localization-row-id>` prefixes that selected existing generated row with:

```text
[KCD2DS TEST] 
```

Unknown IDs are rejected. No retail row ID or proprietary dialogue content is committed as a permanent fixture.

## Historical evidence

The localization patch filename/layout was established by v0.1 retail acceptance on KCD2 1.5.6 Xbox / Microsoft Store. The v0.3 HUD/Data PAK path was then developed through narrow RC proofs separating CryPak loading, HUD resource precedence, AVM1 execution and final TextField styling.

For those experiments and their lessons, see [`v0.3-development-handoff.md`](v0.3-development-handoff.md).
