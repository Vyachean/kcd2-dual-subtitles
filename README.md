# KCD2 Dual Subtitles

<!-- nexus-description:start -->

KCD2 Dual Subtitles is a small Windows tool that generates and installs bilingual subtitles for **Kingdom Come: Deliverance II** using any two supported languages already present in the user's game installation.

It reads the required localization and HUD data from the installed game, does not redistribute proprietary KCD2 localization or UI assets, and does not overwrite the original game files.

## Features

- choose independent **Main** and **Secondary** subtitle languages from the supported languages installed with the game;
- ordinary bottom dialogue subtitles in both selected languages;
- story and cutscene dialogue that uses the normal subtitle path;
- overhead NPC subtitle bubbles;
- optional language tags;
- optional per-line color, size and italic styling;
- optional outline and shadow for readability;
- automatic installation to the correct mod location for the selected game installation;
- **Regenerate** after KCD2 updates instead of waiting for a prebuilt localization patch;
- safe **Uninstall** that removes only this project's files and load-order entry;
- store-neutral compatibility based on the KCD2 file layout rather than a specific launcher or storefront.

## Installation

1. Download the latest Windows ZIP release and extract it anywhere.
2. Run `kcd2-dual-subtitles.exe`.
3. If KCD2 is not detected automatically, use **Browse...** and select the game installation.
4. Select the **Main** and **Secondary** subtitle languages.
5. Configure subtitle appearance if desired.
6. Click **Generate**.
7. Start Kingdom Come: Deliverance II normally.

The tool is a standalone generator and installer. Do not copy the executable itself into the game's `Mods` directory, and do not install the executable through a mod manager.

## Regenerate after KCD2 updates

The generator reads the localization and HUD assets currently installed with the game each time it runs. After KCD2 is updated, run the application again and click **Regenerate** so the mod is rebuilt from the current game files.

Generation and replacement are staged. A failed build is not published as a successful new installation.

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

A compatible selected game root contains the shared KCD2 Data PAKs and at least two localization PAKs supported by the tool.

Automatic discovery is only a convenience and is not the compatibility boundary. The current Windows autodetection strategy knows Microsoft GDK/Xbox roots; installations from other stores can be selected with **Browse...** and are validated by the same store-neutral structural rules.

The v0.3 retail test cycle was performed on **Kingdom Come: Deliverance II 1.5.6, Xbox / Microsoft Store PC**. Other compatible Windows storefront builds are supported by the same file-layout rules but have not all been separately retail-tested by this project.

## Subtitle styles and other HUD mods

With appearance customization disabled, the tool uses a simple tagged bilingual format such as:

```text
[EN] Primary text
[DE] Secondary text
```

With appearance customization enabled, the generator derives a bilingual HUD from the user's installed `hud.gfx`. Main and Secondary lines can be styled independently for color, size and italic treatment, with common outline and shadow options.

Styled mode therefore needs to supply `Libs/UI/hud.gfx`. If another installed mod also replaces that HUD file, KCD2 Dual Subtitles fails closed instead of silently overwriting or combining an unknown foreign HUD. Remove the conflict or use the non-styled tagged mode.

## Known limitations

- the language pair is selected at generation time; there is no in-game secondary-language switch yet;
- standalone narrative/cinematic captions routed separately from the proven standard/bubble subtitle paths may remain single-language or unstyled;
- dialogue localization comes from `text_ui_dialog.xml`; items, quests, tutorials, codex and general UI text are outside the current scope;
- automatic discovery does not yet enumerate every possible launcher/library location, so **Browse...** may be required;
- not every Windows storefront build has been separately retail-tested even though compatibility and installation targeting are store-neutral;
- the executable is not Authenticode-signed;
- there is no application self-update or persisted presentation profile.

## Windows SmartScreen

Release binaries are currently **not Authenticode-signed**, so Windows may show a Microsoft Defender SmartScreen reputation warning for a newly downloaded executable.

Release and release-candidate binaries are built by GitHub Actions and published with SHA-256 checksums. If Microsoft Defender Antivirus reports an actual named malware/threat detection rather than a SmartScreen reputation warning, report the exact detection name and release version. Do not disable protection as a workaround.

## Source code and support

- Source code and releases: <https://github.com/Vyachean/kcd2-dual-subtitles>
- Bug reports and feature requests: <https://github.com/Vyachean/kcd2-dual-subtitles/issues>

<!-- nexus-description:end -->

---

The section between `nexus-description:start` and `nexus-description:end` is intentionally written as a self-contained public description that can also be copied to mod-hosting pages. Detailed implementation and maintenance notes follow below.

## Technical details

### Game language is independent from the selected pair

Main and Secondary are **subtitle text sources**, not the language slot that KCD2 must currently be using.

The generated localization patch is written under every supported localization PAK present in the selected installation. For example, a Czech + German subtitle pair continues to work while KCD2 itself is configured to use English text.

Unknown future localization PAK names are ignored rather than guessed.

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

The normalized selected root is accepted solely by its KCD2 structure. There is no Steam/GOG/Epic/Xbox generation mode switch.

### Automatic installation target

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

The GDK Documents path is resolved through the real Windows **Documents Known Folder**, so redirected and OneDrive-backed Documents folders are supported. Publication remains staged and guarded; the bounded retry/copy fallback for cloud-backed filesystem failures remains available there.

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

The original KCD2 files are never overwritten.

### `mod_order.txt`

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

### Uninstall

The GUI **Uninstall** action removes only this project's directory from the currently selected installation's resolved mod root and removes only this project's entries from that root's existing `mod_order.txt`.

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

Useful failure indicators include CryPak errors such as `ReadFile returned 15`, XML parse failures, missing mod/localization load messages, or an explicit foreign-HUD conflict reported by the installer.

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

## Development

The project uses Go 1.27 and the standard library wherever practical. The Windows GUI uses Win32 directly; there is no Wails, React, Node or WebView runtime.

Automated acceptance, native Windows tests/builds, checksums and release publication run through GitHub Actions. Real KCD2 files are never committed; parser/patcher tests use synthetic fixtures.

See:

- [`ROADMAP.md`](ROADMAP.md) for current and future scope;
- [`docs/mod-format.md`](docs/mod-format.md) for the generated KCD2 mod/PAK contract;
- [`docs/v0.3-development-handoff.md`](docs/v0.3-development-handoff.md) for engineering continuity and retail evidence.

## License

MIT. See [`LICENSE`](LICENSE).
