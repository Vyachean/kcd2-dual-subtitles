# KCD2 Dual Subtitles

<!-- nexus-description:start -->

KCD2 Dual Subtitles is a small Windows tool that generates and installs bilingual subtitles for **Kingdom Come: Deliverance II** using any two supported languages already present in the user's game installation.

It reads localization and HUD data from the installed game, does not redistribute proprietary KCD2 localization or UI assets, and does not overwrite the original game files.

## Features

- choose independent **Main** and **Secondary** subtitle languages from the supported languages installed with the game;
- ordinary bottom dialogue subtitles in both selected languages;
- story and cutscene dialogue that uses the normal subtitle path;
- overhead NPC subtitle bubbles;
- optional language tags;
- optional per-line color, size and italic styling;
- optional outline and shadow for readability;
- automatic Mods-folder resolution for the selected game installation, with the resolved path shown in the GUI and an explicit **Change... / Reset** override when needed;
- active local localization/correction mods from that selected Mods folder are incorporated into the Main and Secondary dialogue sources;
- **Regenerate** after KCD2 updates instead of waiting for a prebuilt localization patch;
- crash-safe replacement: interrupted generation is recovered before the next install/uninstall operation;
- safe **Uninstall** that removes only this project's files and load-order entry;
- store-neutral compatibility based on the KCD2 file layout rather than a specific launcher or storefront.

## Installation

1. Download the latest `kcd2-dual-subtitles-vX.Y.Z-windows-x64.zip` release and extract it anywhere.
2. Run `kcd2-dual-subtitles.exe`.
3. If KCD2 is not detected automatically, use **Browse...** and select the game installation.
4. Check the displayed **Mods folder**. If it is not the mod location used by your installation/setup, click **Change...** and select the active folder; **Reset** returns to automatic detection.
5. Select the **Main** and **Secondary** subtitle languages.
6. Configure subtitle appearance if desired.
7. Click **Generate**.
8. Start Kingdom Come: Deliverance II normally.

The tool is a standalone generator and installer. Do not copy the executable itself into the game's `Mods` directory, and do not install the executable through a mod manager.

## Regenerate after KCD2 updates

The generator reads the localization and HUD assets currently installed with the game each time it runs, including compatible active local localization/correction mods from the displayed Mods folder. After KCD2 or one of those localization mods is updated, fully close the game, run the application again and click **Regenerate**, then restart KCD2.

Generation and replacement are transactional. A failed or interrupted build is not reported as a successful new installation, and the previous installed version is preserved or recovered when possible.

## Supported languages

The tool currently recognizes the known KCD2 PC localization PAK set and shows only languages actually present in the selected installation:

- English
- Italian
- French
- German
- Spanish
- Czech
- Japanese
- Korean
- Polish
- Portuguese (Brazil)
- Chinese (Simplified)
- Chinese (Traditional)
- Turkish
- Russian
- Ukrainian
- Vietnamese

At least two supported installed languages are required. No particular language pair is mandatory, and the game's own interface language does not have to match either selected subtitle language.

## Compatibility

The application is **store-neutral**. Compatibility is determined by the KCD2 game-file layout, not by whether the game came from Steam, GOG, Epic Games Store, Xbox / Microsoft Store, or another Windows distribution.

Automatic discovery currently looks for Microsoft GDK/Xbox installations, Steam libraries, Epic Games Store installed-game manifests and GOG/Galaxy metadata/default library locations. Every detected candidate still passes the same structural KCD2 validation before it can be selected. **Browse...** remains available for compatible installations that automatic discovery misses.

For each selected subtitle language, the stock game localization remains the base/fallback and compatible active localization mods from the **displayed Mods folder** are applied on top. Without `mod_order.txt`, local mods follow KCD2's alphabetical folder order; when `mod_order.txt` exists, its whitelist and explicit order are respected. Installation keeps `kcd_dual_subtitles` as the final active entry of an existing order file so the generated bilingual patch loads after the localization sources it composed. The tool does not create `mod_order.txt` when it is absent; if alphabetical load order would let a contributing localization mod overwrite the generated patch, automatic installation fails with an explanation instead of creating a new whitelist. Steam Workshop content is not separately discovered by this feature unless it is also present through the selected local-mod layout.

The v0.3 retail test cycle was performed on **Kingdom Come: Deliverance II 1.5.6, Xbox / Microsoft Store PC**. Other compatible Windows storefront builds are supported by the same file-layout rules but have not all been separately retail-tested by this project.

## Subtitle styles and other HUD mods

With appearance customization disabled, the tool uses a simple tagged bilingual format such as:

```text
[EN] Primary text
[DE] Secondary text
```

With appearance customization enabled, the generator derives a bilingual HUD from the user's installed `hud.gfx`. Main and Secondary lines can be styled independently for color, size and italic treatment, with common outline and shadow options.

Styled mode therefore needs to supply `Libs/UI/hud.gfx`. If another installed mod also replaces that HUD file in the selected Mods folder, KCD2 Dual Subtitles fails closed instead of silently overwriting or combining an unknown foreign HUD. Remove the conflict or use the non-styled tagged mode.

## Known limitations

- the language pair is selected at generation time; there is no in-game secondary-language switch yet;
- standalone narrative/cinematic captions routed separately from the proven standard/bubble subtitle paths may remain single-language or unstyled;
- subtitle-source composition is limited to dialogue localization; items, quests, tutorials, codex and general UI text are outside the current scope;
- new dialogue introduced only by another mod's generic localization patch is not automatically classified as dialogue; safe support would require following that content mod's dialogue references rather than treating all new localization IDs as subtitles;
- automatic game and Mods-folder discovery is best-effort, but both can be reviewed in the GUI and the Mods folder can be overridden explicitly;
- not every Windows storefront build has been separately retail-tested even though compatibility and installation targeting are store-neutral;
- the executable is not Authenticode-signed;
- there is no application self-update or persisted presentation/Mods-folder profile.

## Windows SmartScreen

Release binaries are currently **not Authenticode-signed**, so Windows may show a Microsoft Defender SmartScreen reputation warning for a newly downloaded executable.

Release and release-candidate binaries are built by GitHub Actions and published with SHA-256 checksums. If Microsoft Defender Antivirus reports an actual named malware/threat detection rather than a SmartScreen reputation warning, report the exact detection name and release version. Do not disable protection as a workaround.

## Source code and support

- Source code and releases: <https://github.com/Vyachean/kcd2-dual-subtitles>
- Bug reports and feature requests: <https://github.com/Vyachean/kcd2-dual-subtitles/issues>

<!-- nexus-description:end -->

---

The section between `nexus-description:start` and `nexus-description:end` is the canonical public description source. Nexus Mods uses BBCode rather than GitHub-flavored Markdown, so [`NEXUS_DESCRIPTION.bbcode.txt`](NEXUS_DESCRIPTION.bbcode.txt) is generated from that block by [`scripts/render-nexus-description.ps1`](scripts/render-nexus-description.ps1) and is the copy/paste source for the Nexus description. Packaging CI fails if that generated file is stale.

Detailed implementation, diagnostics and maintainer notes follow below.

## Technical details

### Windows GUI

Run or double-click:

```text
kcd2-dual-subtitles.exe
```

The native Win32 application provides best-effort KCD2 autodetection, Game-folder `Browse...`, a visible **Mods folder** with **Change... / Reset**, Main and Secondary selectors, optional styled presentation, Generate / Regenerate, Uninstall, operation status and native error messages.

The Mods-folder field is read-only so displayed state cannot drift from backend state. **Change...** validates and selects an existing custom directory; **Reset** restores the layout-aware automatic location. Source discovery, installation, status, Regenerate, Uninstall, HUD-conflict scanning and `mod_order.txt` all use this one selected Mods folder. Selecting a different Game folder clears a previous custom Mods-folder override.

When no previous valid selection is available, the GUI prefers **English as Main** when installed and chooses the first other installed supported language as Secondary. It never silently selects the same language twice.

Generate/Regenerate runs outside the Win32 UI thread. The window remains responsive during generation, while controls that could change the captured game/Mods/language/presentation selection stay disabled until the operation finishes. Closing during an active generation asks for confirmation.

### Styled subtitle details

Leaving primary color or size blank preserves the game's default primary property.

The derived HUD path is generated from the user's installed `hud.gfx`; the project does not ship a prebuilt proprietary HUD file.

Unknown future localization PAK names are ignored rather than guessed.

### Game language is independent from the selected pair

Main and Secondary are **subtitle text sources**, not the language slot that KCD2 must currently be using.

The generated localization patch is written under every supported localization PAK present in the selected installation. For example, a Czech + German subtitle pair continues to work while KCD2 itself is configured to use English text.

### Localization correction mods

For each Main/Secondary language, generation starts with the stock `Localization/<language>_xml.pak` and then composes compatible dialogue overrides from active local mods in the selected KCD2 Mods folder. The same KCD2 order rules are used: alphabetical local-mod folder order by default, or the `mod_order.txt` whitelist/order when that file exists.

Warhorse documents text localization resources inside `<language>_xml.pak` as any number of XML tables named `<anything>_<modid>.xml`; `text__<modid>.xml` and `text_ui__<modid>.xml` are only examples, not exclusive prefixes. The first cell is `stringId`, the final cell is displayed text, and the middle source/authoring cell is optional and not loaded. Generic resources therefore accept both two-cell (`ID`, `text`) and three-cell (`ID`, `source`, `text`) rows. The resolver follows the general filename suffix contract case-insensitively when the manifest provides an explicit `modid`. The inspected Chineses Fix 20260727 archive uses three-cell rows in `text__chinesesfixptf.xml`. A root-level `text_ui_dialog.xml` remains supported as an explicit dialogue-table replacement/overlay and is intentionally parsed with the stricter three-cell dialogue schema.

Generic localization patches are restricted to IDs already known to the effective dialogue table so unrelated UI/items/quest localization is not accidentally treated as subtitle dialogue. Explicit `text_ui_dialog.xml` tables may introduce new dialogue IDs and must have unique IDs. Generic localization patches may repeat keys inside one resource; the last occurrence of an already-known dialogue ID in that resource wins. If two separate generic resources assign different displayed text to the same known dialogue ID, generation fails closed because Warhorse does not document a cross-resource winner. Supported resource filenames are case-folded for ambiguity detection, malformed XML fails with source context, and XML reads are size-limited before parsing.

When `mod_order.txt` exists, entries are matched to explicit manifest `modid` values exactly and unlisted explicit-ID mods are excluded before their localization PAK or `<supports>` declaration can affect generation. A repeated active localization-mod ID fails closed. A manifest without an explicit `modid` can still participate when `mod_order.txt` is absent: the game's exact auto-generated ID normalization is undocumented, so the resolver accepts the documented syntactic generic-localization shape while retaining the known-dialogue-only filter. With an explicit order file, a missing manifest ID fails closed only when that mod actually contains the selected-language localization PAK and exact identity is required.

A mod is reported as a contribution only when its complete supported-resource stack changes displayed effective dialogue; a difference only in the optional source/authoring cell is ignored. Source composition inspects the matching PAK for the selected installed source language. Warhorse mentions an English fallback when a user selects a language they do not have, but the available documentation does not define enough cross-language override precedence to safely apply an English correction PAK over another installed stock language, so this tool does not guess that behavior.

The project's own installed localization and known legacy staging residue are excluded from source discovery so Regenerate cannot consume previously generated bilingual text. See [`docs/localization-mod-compatibility.md`](docs/localization-mod-compatibility.md) for the detailed official filename contract, inspected Chineses Fix archive metadata and compatibility rationale.

### Game folder selection

Automatic Windows discovery is best-effort. Store/launcher detection coverage does not define which installations the application supports.

If the game is not found automatically, use **Browse...**. The selector accepts a compatible KCD2 game root directly and also accepts the immediate parent of a `Content` root used by packaged layouts.

Examples can therefore look like either:

```text
C:\Games\KingdomComeDeliverance2
```

or:

```text
C:\XboxGames\Kingdom Come- Deliverance II\Content
```

The normalized selected root is accepted solely by its KCD2 structure. There is no Steam/GOG/Epic/Xbox generation mode switch. Selecting a different normalized game root intentionally clears any custom Mods-folder override so a directory from one installation cannot silently carry into another.

### Mods folder and installation target

The selected game installation supplies the default Mods folder, but the GUI makes the resolved path visible and allows an explicit existing-directory override. Whichever path is displayed is the source of truth for **where local mods are read and where KCD2 Dual Subtitles is installed**.

For the normal PC KCD2 layout used by Steam/GOG/Epic and compatible distributions, the automatic path is:

```text
<game-root>\Mods\kcd_dual_subtitles\
```

For Microsoft GDK/Xbox packaged installations, the automatic path is:

```text
<Documents>\kingdomcome_mods\kcd_dual_subtitles\
```

GDK handling is selected from package artifacts in or next to the selected installation (`gamelaunchhelper.exe`, `MicrosoftGame.config`, or `appxmanifest.xml`), not from an `XboxGames` path name. A custom library location therefore does not change the rule, and the GUI override covers setups whose active Mods directory intentionally differs from the automatically resolved one.

The GDK Documents path is resolved through the real Windows **Documents Known Folder**, so redirected and OneDrive-backed Documents folders are supported. Publication uses a same-volume transaction workspace beside the selected mod root rather than a directory inside KCD2's scanned mod root; bounded rename retries and the guarded copy fallback remain available for cloud-backed filesystem failures.

A styled installation has the following shape beneath the selected `kcd_dual_subtitles` directory:

```text
kcd_dual_subtitles\
├── mod.manifest
├── Localization\
│   ├── English_xml.pak
│   ├── German_xml.pak
│   └── ... one generated patch PAK for each supported installed language slot
└── Data\
    └── kcd_dual_subtitles.pak
        └── Libs/UI/hud.gfx
```

The generated localization PAKs contain only the project's patch resource and changed dialogue rows. The generated Data PAK contains a deterministic transformation of the user's installed retail HUD when styled mode is enabled.

Before success is reported, the tool verifies generated portable ZIP contents, staged installation contents and the final published mod against the exact deterministic files requested for that generation. Styled mode requires the expected derived HUD Data PAK; localization-only mode rejects an unexpected HUD payload.

The original KCD2 files are never overwritten.

### `mod_order.txt`

If `mod_order.txt` already exists in the selected Mods folder, installation ensures exactly one `kcd_dual_subtitles` entry is the **final active entry** while preserving the relative order of unrelated entries. The order-file change is part of the same crash-safe install transaction as the generated mod directory.

This final position matters when localization/correction mods are used as sources: the generated bilingual patch must load after them or a later source mod could replace a bilingual row with its single-language value again.

For automatic layouts the file is therefore checked at either:

```text
<game-root>\Mods\mod_order.txt
```

or, for GDK:

```text
<Documents>\kingdomcome_mods\mod_order.txt
```

A custom GUI Mods folder uses its own `mod_order.txt` instead.

The tool does **not** create `mod_order.txt` when it is absent because KCD2 treats the file as a whitelist. In alphabetical mode, if a localization mod that actually contributes effective dialogue sorts after the `kcd_dual_subtitles` folder, automatic installation fails closed and explains the load-order conflict rather than creating a whitelist that could disable unrelated mods.

### Transaction and recovery model

Generation and replacement are transactional. Directory-shaped work is kept outside KCD2's scanned mod root, a previous installation is retained until the new one is committed, and an interrupted Generate/Regenerate is recovered on the next Generate/Regenerate or Uninstall.

Legacy `.kcd_dual_subtitles.staging-*` residue produced by v0.3.2 and earlier is recovered or cleaned before a new publication. After a successful Generate/Regenerate, fully restart KCD2 before testing the new language/style selection because a running game may still hold previously loaded localization/HUD resources in memory.

### Uninstall

The GUI **Uninstall** action removes only this project's directory from the **displayed Mods folder** and removes only this project's entries from that folder's existing `mod_order.txt`.

Other mods and their load-order entries are left alone.

## CLI

The same executable keeps a CLI for scripting, diagnostics and portable generated-mod ZIP creation.

Example:

```text
kcd2-dual-subtitles.exe generate --game "C:\path\to\KCD2-root" --main English --secondary German
```

Use the default tagged format:

```text
kcd2-dual-subtitles.exe generate --game "C:\path\to\KCD2-root" --main English --secondary German --subtitle-style tagged
```

Use the styled HUD defaults:

```text
kcd2-dual-subtitles.exe generate --game "C:\path\to\KCD2-root" --main English --secondary German --subtitle-style hud
```

Without `--output`, the Windows CLI resolves the automatic install target from `--game` using the same standard/GDK rules. The custom Mods-folder override introduced in this change is a native-GUI selection; CLI generation continues to use the automatically resolved mod environment.

Create a portable generated-mod ZIP instead of installing:

```text
kcd2-dual-subtitles.exe generate --game "C:\path\to\KCD2-root" --main English --secondary German --output "kcd2-dual-subtitles.zip"
```

Other entrypoints:

```text
kcd2-dual-subtitles.exe --help
kcd2-dual-subtitles.exe --version
```

`--canary-id` remains an acceptance/debugging option and should not be used for normal generation.

Automatic installation is Windows-only. Portable generated-mod ZIP creation remains available on other supported build environments.

## Troubleshooting

KCD2 writes `kcd.log` in the Windows Documents folder used by the game. A successfully loaded localization patch includes a line similar to:

```text
[Mod] Loading localization patch 'Localization\text_ui__kcd_dual_subtitles.xml'
```

If upgrading from v0.3.2 after an interrupted or apparently frozen Regenerate, fully close KCD2, run the current version and Regenerate once. The installer recognizes its legacy `.kcd_dual_subtitles.staging-*` transaction residue before publishing the replacement.

Useful failure indicators include CryPak errors such as `ReadFile returned 15`, XML parse failures, missing mod/localization load messages, duplicate `kcd_dual_subtitles`-style mod directories, an unsafe localization load-order warning, or an explicit foreign-HUD conflict reported by the installer.

Do not upload or commit full proprietary KCD2 localization or GFX files when reporting an issue.

## Release package

Stable releases and release candidates publish a versioned Windows x64 ZIP intended for GitHub and mod-hosting distribution. The archive contains only:

```text
kcd2-dual-subtitles.exe
README.md
LICENSE
SHA256SUMS.txt
```

The checksum file inside the archive verifies the executable, README and license included in that archive. GitHub releases also publish a top-level `SHA256SUMS.txt` that verifies the standalone executable and the versioned ZIP asset.

CI creates the same package shape with the `dev` version and validates both the exact archive file set and the internal checksums before uploading the artifact. The generated `NEXUS_DESCRIPTION.bbcode.txt` stays in the repository as maintainer-facing publication text and is intentionally not added to the user runtime package.

## Development

The project uses Go 1.27 and the standard library wherever practical. The Windows GUI uses Win32 directly; there is no Wails, React, Node or WebView runtime.

Automated acceptance, native Windows tests/builds, checksums and release publication run through GitHub Actions. Real KCD2 files are never committed; parser/patcher tests use synthetic fixtures.

See:

- [`ROADMAP.md`](ROADMAP.md) for current and future scope;
- [`docs/mod-format.md`](docs/mod-format.md) for the generated KCD2 mod/PAK contract;
- [`docs/localization-mod-compatibility.md`](docs/localization-mod-compatibility.md) for effective localization-source composition and third-party correction-mod support;
- [`docs/v0.3-development-handoff.md`](docs/v0.3-development-handoff.md) for v0.3 engineering continuity and retail evidence;
- [`docs/v0.3.3-maintenance-handoff.md`](docs/v0.3.3-maintenance-handoff.md) for the crash-safe publication/recovery architecture.

## License

MIT. See [`LICENSE`](LICENSE).
