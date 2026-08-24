# KCD2 Dual Subtitles

KCD2 Dual Subtitles is a small Windows tool that generates and installs bilingual dialogue subtitles for **Kingdom Come: Deliverance II** from localization files already present in the user's game installation.

It does not redistribute KCD2 localization or UI assets and does not modify the original game files.

## Compatibility

The application is **store-neutral**. Compatibility is determined by the KCD2 game-file layout, not by whether the game came from Steam, GOG, Epic Games Store, Xbox / Microsoft Store, or another Windows distribution.

A compatible installation must provide the normal KCD2 `Content` layout, the required shared Data PAKs, and at least two localization PAKs supported by the tool. The application does not require English, Russian, or any other specific language pair to be installed.

Automatic discovery is only a convenience and is not the compatibility boundary. The current Windows autodetection strategy knows Microsoft GDK/Xbox roots; installations from other stores can be selected with `Browse...` and are validated by the same store-neutral structural rules.

The v0.3 retail test cycle was performed on **Kingdom Come: Deliverance II 1.5.6, Xbox / Microsoft Store PC**. That is test evidence, not a store restriction. Other Windows storefront builds are supported when they expose the same compatible KCD2 file structure; they have not all been separately retail-tested by this project.

Retail-tested behavior includes:

- ordinary bottom dialogue subtitles with two selected languages;
- story/cutscene dialogue using the normal subtitle path;
- overhead NPC subtitle bubbles;
- selectable installed-language pairs, including Czech + German while the game's own text language remained English;
- configurable primary and secondary subtitle presentation;
- optional language tags, italic, outline and shadow;
- native color selection;
- safe regeneration and installation through the Windows Documents folder.

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

If the game is not found automatically, use `Browse...`. It accepts either the KCD2 `Content` directory or its immediate parent, for example:

```text
...\KingdomComeDeliverance2\Content
```

The selected directory is accepted solely by its compatible KCD2 structure. There is no Steam/GOG/Epic/Xbox mode switch and no store-specific generation path.

## Generated mod

Automatic installation uses the real Windows **Documents Known Folder**:

```text
<Documents>\kingdomcome_mods\kcd_dual_subtitles\
```

Redirected and OneDrive-backed Documents folders are supported. Normal publication uses same-volume rename semantics; a guarded copy fallback is available for cloud-backed filesystem cases where Windows repeatedly refuses the final rename.

A styled installation has the following shape:

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

The original files under the KCD2 installation are never overwritten.

## Other HUD mods

Styled mode needs to supply `Libs/UI/hud.gfx`. The installer therefore checks installed mods for another HUD override and fails closed instead of silently overwriting or composing an unknown foreign HUD.

If another mod replaces the KCD2 HUD, remove the conflict or use the non-styled tagged mode.

## `mod_order.txt`

If `<Documents>\kingdomcome_mods\mod_order.txt` already exists, installation ensures `kcd_dual_subtitles` is present while preserving unrelated entries and their order.

The tool does not create `mod_order.txt` when it is absent.

## Regenerate after KCD2 updates

The generator reads the currently installed localization and HUD assets each time it runs. After a KCD2 update, run the application again and use **Regenerate** so the mod is rebuilt from the current game files.

Generation and replacement are staged. A failed build is not published as a successful new installation.

## Uninstall

The GUI `Uninstall` action removes only:

```text
<Documents>\kingdomcome_mods\kcd_dual_subtitles
```

and removes only this project's entries from an existing `mod_order.txt`. Other mods and their load-order entries are left alone.

## CLI

The same executable keeps a CLI for scripting, diagnostics and portable ZIP generation.

Example:

```text
kcd2-dual-subtitles.exe generate --game "C:\path\to\Content" --main English --secondary German
```

Use the default tagged format:

```text
kcd2-dual-subtitles.exe generate --game "C:\path\to\Content" --main English --secondary German --subtitle-style tagged
```

Use the styled HUD defaults:

```text
kcd2-dual-subtitles.exe generate --game "C:\path\to\Content" --main English --secondary German --subtitle-style hud
```

Create a portable ZIP instead of installing:

```text
kcd2-dual-subtitles.exe generate --game "C:\path\to\Content" --main English --secondary German --output "kcd2-dual-subtitles.zip"
```

Other entrypoints:

```text
kcd2-dual-subtitles.exe --help
kcd2-dual-subtitles.exe --version
```

`--canary-id` remains an acceptance/debugging option and should not be used for normal generation.

## Known limitations

- v0.3 uses a **generation-time fixed language pair**; there is no in-game secondary-language switch yet;
- standalone narrative/cinematic captions routed through `fc_setNarrativeSubtitles` are not yet handled by the proven standard/bubble HUD transformations and may remain single-language or unstyled;
- dialogue localization comes from `text_ui_dialog.xml`; items, quests, tutorials, codex and general UI text are outside the current scope;
- automatic discovery does not yet enumerate every possible launcher/library location, so `Browse...` may be required;
- not every Windows storefront build has been separately retail-tested even though compatibility is store-neutral;
- the executable is not Authenticode-signed;
- there is no application self-update or persisted presentation profile.

Future runtime-language/in-game-settings work is tracked separately from the stable v0.3 fixed-pair contract.

## Troubleshooting

KCD2 writes `kcd.log` in the Windows Documents folder used by the game. A successfully loaded localization patch includes a line similar to:

```text
[Mod] Loading localization patch 'Localization\text_ui__kcd_dual_subtitles.xml'
```

Useful failure indicators include CryPak errors such as `ReadFile returned 15`, XML parse failures, missing mod/localization load messages, or an explicit foreign-HUD conflict reported by the installer.

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
