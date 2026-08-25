# KCD2 Dual Subtitles

KCD2 Dual Subtitles is a small Windows tool that generates and installs bilingual dialogue subtitles for **Kingdom Come: Deliverance II** from localization files already present in the user's game installation.

It does not redistribute KCD2 localization or UI assets and does not modify the original game files.

## Compatibility

The application is **store-neutral**. Compatibility is determined by the KCD2 game-file layout, not by whether the game came from Steam, GOG, Epic Games Store, Xbox / Microsoft Store, or another Windows distribution.

A compatible selected game root contains the shared KCD2 Data PAKs and at least two localization PAKs supported by the tool. The application does not require English, Russian, or any other specific language pair to be installed.

Automatic discovery is only a convenience and is not the compatibility boundary. The current Windows autodetection strategy looks for Microsoft GDK/Xbox installations, Steam libraries, Epic Games Store installed-game manifests and GOG/Galaxy metadata/default library locations. Every candidate is still accepted only after the same store-neutral structural KCD2 validation. `Browse...` remains the authoritative fallback for compatible installations that launcher discovery does not find.

The v0.3 retail test cycle was performed on **Kingdom Come: Deliverance II 1.5.6, Xbox / Microsoft Store PC**. That is test evidence, not a store restriction. Other Windows storefront builds are supported when they expose the compatible KCD2 file structure; they have not all been separately retail-tested by this project.

Retail-tested behavior includes:

- ordinary bottom dialogue subtitles with two selected languages;
- story/cutscene dialogue using the normal subtitle path;
- overhead NPC subtitle bubbles;
- selectable installed-language pairs, including Czech + German while the game's own text language remained English;
- configurable primary and secondary subtitle presentation;
- optional language tags, italic, outline and shadow;
- native color selection;
- safe regeneration and installation on the Microsoft GDK/Documents layout.

## Windows GUI

Run or double-click:

```text
kcd2-dual-subtitles.exe
```

The native Win32 application provides:

- best-effort KCD2 autodetection;
- `Browse...` selection for any compatible Windows installation;
- Main and Secondary subtitle language selectors;
- optional styled subtitle presentation;
- primary color, size and italic controls;
- secondary color, size and italic controls;
- native Windows color pickers;
- language tags on/off;
- outline on/off;
- shadow on/off;
- Generate / Regenerate;
- Uninstall;
- operation status and native error messages.

When no previous valid selection is available, the GUI prefers **English as Main** when installed and chooses the first other installed supported language as Secondary. It never silently selects the same language twice.

Generate/Regenerate runs outside the Win32 UI thread. The window remains responsive during generation, while controls that could change the captured game/language/presentation selection stay disabled until the operation finishes. Closing during an active generation asks for confirmation.

### Styled subtitles

When appearance customization is disabled, the tool keeps the simple tagged bilingual format:

```text
[EN] Primary text
[DE] Secondary text
```

When appearance customization is enabled, the generator creates the bilingual HTML consumed by KCD2's Scaleform HUD. Primary and secondary lines can be styled independently for color, size and italic treatment. Outline and shadow are common readability effects applied to the complete subtitle field.

Leaving primary color or size blank preserves the game's default primary property.

The derived HUD path is generated from the user's installed `hud.gfx`; the project does not ship a prebuilt proprietary HUD file.

## Supported languages

The tool has explicit metadata for the current known KCD2 PC localization PAK set and shows only languages actually present in the selected installation:

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

Unknown future localization PAK names are ignored rather than guessed.

At least two supported installed languages are required because the generated result is bilingual. No particular two languages are mandatory.

## Game language is independent from the selected pair

Main and Secondary are **subtitle text sources**, not the language slot that KCD2 must currently be using.

The generated localization patch is written under every supported localization PAK present in the selected installation. For example, a Czech + German subtitle pair continues to work while KCD2 itself is configured to use English text.

## Game folder selection

Automatic Windows discovery is best-effort. Store/launcher detection coverage does not define which installations the application supports.

If the game is not found automatically, use `Browse...`. The selector accepts a compatible KCD2 game root directly and also accepts the immediate parent of a `Content` root used by packaged layouts.

Examples can therefore look like either:

```text
C:\Games\KingdomComeDeliverance2
```

or:

```text
C:\XboxGames\Kingdom Come- Deliverance II\Content
```

The normalized selected root is accepted solely by its KCD2 structure. There is no Steam/GOG/Epic/Xbox generation mode switch.

## Automatic installation target

The selected game installation is also the source of truth for **where the mod is installed**. The application resolves one mod root and uses that same root for installation, HUD-conflict detection, `mod_order.txt`, Regenerate status and Uninstall.

For the normal PC KCD2 layout used by Steam/GOG/Epic and compatible distributions:

```text
<game-root>\Mods\kcd_dual_subtitles\
```

For Microsoft GDK/Xbox packaged installations:

```text
<Documents>\kingdomcome_mods\kcd_dual_subtitles\
```

GDK handling is selected from package artifacts in or next to the selected installation (`gamelaunchhelper.exe`, `MicrosoftGame.config`, or `appxmanifest.xml`), not from an `XboxGames` path name. A custom library location therefore does not change the rule.

The GDK Documents path is resolved through the real Windows **Documents Known Folder**, so redirected and OneDrive-backed Documents folders are supported. Publication uses a same-volume transaction workspace beside the resolved mod root rather than a directory inside KCD2's scanned mod root; bounded rename retries and the guarded copy fallback remain available for cloud-backed filesystem failures.

A styled installation has the following shape beneath the resolved `kcd_dual_subtitles` directory:

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

## Other HUD mods

Styled mode needs to supply `Libs/UI/hud.gfx`. The installer checks the **resolved mod root for the selected installation** for another HUD override and fails closed instead of silently overwriting or composing an unknown foreign HUD.

If another mod replaces the KCD2 HUD, remove the conflict or use the non-styled tagged mode.

## `mod_order.txt`

If `mod_order.txt` already exists in the resolved mod root, installation ensures `kcd_dual_subtitles` is present while preserving unrelated entries and their order.

That means the file is checked at either:

```text
<game-root>\Mods\mod_order.txt
```

or, for GDK:

```text
<Documents>\kingdomcome_mods\mod_order.txt
```

The tool does not create `mod_order.txt` when it is absent.

## Regenerate after KCD2 updates

The generator reads the currently installed localization and HUD assets each time it runs. After a KCD2 update, run the application again and use **Regenerate** so the mod is rebuilt from the current game files.

Regenerate status is checked against the currently selected installation's resolved mod root, so switching `Browse...` to another KCD2 installation does not reuse the status of a different copy of the game.

Generation and replacement are transactional. Directory-shaped work is kept outside KCD2's scanned mod root, a previous installation is retained until the new one is committed, and an interrupted Generate/Regenerate is recovered on the next Generate/Regenerate or Uninstall. Legacy `.kcd_dual_subtitles.staging-*` residue produced by v0.3.2 and earlier is also recovered or cleaned before a new publication.

After a successful Generate/Regenerate, **fully restart KCD2 before testing the new language/style selection**. A running game may still hold the previously loaded localization/HUD resources in memory.

## Uninstall

The GUI `Uninstall` action removes only this project's directory from the **currently selected installation's resolved mod root** and removes only this project's entries from that root's existing `mod_order.txt`.

Other mods and their load-order entries are left alone.

## CLI

The same executable keeps a CLI for scripting, diagnostics and portable ZIP generation.

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

Without `--output`, the Windows build resolves the automatic install target from `--game` using the same rules as the GUI.

Create a portable ZIP instead of installing:

```text
kcd2-dual-subtitles.exe generate --game "C:\path\to\KCD2-root" --main English --secondary German --output "kcd2-dual-subtitles.zip"
```

Other entrypoints:

```text
kcd2-dual-subtitles.exe --help
kcd2-dual-subtitles.exe --version
```

`--canary-id` remains an acceptance/debugging option and should not be used for normal generation.

Automatic installation is Windows-only. Portable ZIP generation remains available on other supported build environments.

## Known limitations

- v0.3 uses a **generation-time fixed language pair**; there is no in-game secondary-language switch yet;
- standalone narrative/cinematic captions routed through `fc_setNarrativeSubtitles` are not yet handled by the proven standard/bubble HUD transformations and may remain single-language or unstyled;
- dialogue localization comes from `text_ui_dialog.xml`; items, quests, tutorials, codex and general UI text are outside the current scope;
- automatic discovery is best-effort and can still miss unusual/custom launcher metadata or library layouts, so `Browse...` remains available;
- not every Windows storefront build has been separately retail-tested even though compatibility and installation targeting are store-neutral;
- the executable is not Authenticode-signed;
- there is no application self-update or persisted presentation profile.

Future runtime-language/in-game-settings work is tracked separately from the stable v0.3 fixed-pair contract.

## Troubleshooting

KCD2 writes `kcd.log` in the Windows Documents folder used by the game. A successfully loaded localization patch includes a line similar to:

```text
[Mod] Loading localization patch 'Localization\text_ui__kcd_dual_subtitles.xml'
```

If upgrading from v0.3.2 after an interrupted or apparently frozen Regenerate, fully close KCD2, run the current version and Regenerate once. The installer recognizes its legacy `.kcd_dual_subtitles.staging-*` transaction residue before publishing the replacement.

Useful failure indicators include CryPak errors such as `ReadFile returned 15`, XML parse failures, missing mod/localization load messages, duplicate `kcd_dual_subtitles`-style mod directories, or an explicit foreign-HUD conflict reported by the installer.

Do not upload or commit full proprietary KCD2 localization or GFX files when reporting an issue.

## Microsoft Defender SmartScreen

Release binaries are currently **not Authenticode-signed**, so Windows may show a Microsoft Defender SmartScreen reputation warning for a newly downloaded executable.

Release and release-candidate binaries are built by GitHub Actions and published with `SHA256SUMS.txt`.

If Microsoft Defender Antivirus reports an actual named malware/threat detection rather than a SmartScreen reputation warning, report the exact detection name and release version. Do not disable protection as a workaround.

## Development

The project uses Go 1.27 and the standard library wherever practical. The Windows GUI uses Win32 directly; there is no Wails, React, Node or WebView runtime.

Automated acceptance, native Windows tests/builds, checksums and release publication run through GitHub Actions. Real KCD2 files are never committed; parser/patcher tests use synthetic fixtures.

See:

- [`ROADMAP.md`](ROADMAP.md) for current and future scope;
- [`docs/mod-format.md`](docs/mod-format.md) for the generated KCD2 mod/PAK contract;
- [`docs/v0.3-development-handoff.md`](docs/v0.3-development-handoff.md) for engineering continuity and retail evidence.

## License

MIT. See [`LICENSE`](LICENSE).
