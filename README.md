# KCD2 Dual Subtitles

A lightweight Windows tool that generates and installs bilingual dialogue subtitles for Kingdom Come: Deliverance II from localization files already installed with the game.

## Status

Stable `v0.1.0` was validated in the retail **Kingdom Come: Deliverance II 1.5.6 Xbox / Microsoft Store PC build**.

Confirmed in live testing:

- ordinary NPC dialogue shows both languages;
- story/cutscene dialogue shows both languages;
- the two languages render on separate lines;
- the generated `text_ui__kcd_dual_subtitles.xml` localization patch is loaded by the game;
- the generated CryPak produces no observed CryPak/XML/localization errors;
- automatic installation works with a redirected OneDrive Documents folder and an existing `mod_order.txt`.

`v0.2` adds a native Windows GUI, Xbox / Microsoft Store game autodetection, regeneration and safe uninstall. The new GUI/autodetection flow remains release-candidate functionality until it is manually exercised on the validated Xbox environment.

Steam, GOG, and Epic have not yet been live-validated and are not automatically detected.

## Subtitle format

Bilingual rows are labeled so the languages are easy to distinguish:

```text
[RU] Русский текст
[EN] English text
```

If English is selected as the main language, the order is reversed:

```text
[EN] English text
[RU] Русский текст
```

Identical translations and single-language fallback rows remain untagged.

Russian and English are currently supported.

## Normal Windows use

Run `kcd2-dual-subtitles.exe` normally or double-click it.

On Windows with no command-line arguments, a small native Win32 window opens. It provides:

- automatically detected KCD2 game folder when exactly one Xbox / Microsoft Store installation is found;
- `Browse...` fallback for manual selection;
- Main language selector;
- Secondary language selector;
- `Generate and install` / `Regenerate`;
- `Uninstall`;
- operation status and errors.

The default selections are Russian main + English secondary. The application does not infer or automatically swap languages; the two selections must differ.

### Xbox / Microsoft Store autodetection

Current Microsoft GDK flat-file games are searched in Xbox game roots on fixed Windows drives. The detector checks the default `<drive>:\XboxGames` root and best-effort custom roots recorded by `.GamingRoot`.

Only immediate game directories are inspected; the application does not recursively search whole disks. A candidate is accepted only when the expected KCD2 structure is present, including the supported localization PAKs and core Data PAKs.

If no unique installation can be determined, choose the game folder manually. `Browse...` accepts either:

```text
...\Kingdom Come- Deliverance II\Content
```

or its immediate parent directory.

## Generated mod installation

The Windows build installs into the real Windows **Documents Known Folder**:

```text
<Documents>\kingdomcome_mods\kcd_dual_subtitles\
```

Redirected and OneDrive Documents folders are supported through the Windows Known Folders API; the tool does not assume `%USERPROFILE%\Documents`.

The generated mod contains:

```text
kcd_dual_subtitles\
├── mod.manifest
└── Localization\
    └── <main-language>_xml.pak
        └── text_ui__kcd_dual_subtitles.xml
```

If `Documents\kingdomcome_mods\mod_order.txt` already exists, installation ensures `kcd_dual_subtitles` is listed without reordering or deleting unrelated mods. If the file does not exist, the tool does not create it.

The original game localization PAKs are never modified.

## Regenerate after game updates

The generator reads the currently installed localization files every time it runs. After KCD2 updates localization data, open the application and use `Regenerate` to rebuild the patch from the current PAKs.

Replacement is staged so a failed generation does not publish a partial new mod.

## Uninstall

Use the GUI `Uninstall` button.

It removes only:

```text
<Documents>\kingdomcome_mods\kcd_dual_subtitles
```

and removes only `kcd_dual_subtitles` entries from an existing `mod_order.txt`. Other mod entries and their order are preserved. The tool does not create `mod_order.txt` during uninstall.

## CLI

The existing CLI remains available for scripting, portable ZIP generation and diagnostics.

Russian first, English second:

```text
kcd2-dual-subtitles.exe generate --game "C:\path\to\Content" --main Russian --secondary English
```

English first, Russian second:

```text
kcd2-dual-subtitles.exe generate --game "C:\path\to\Content" --main English --secondary Russian
```

Portable ZIP instead of automatic installation:

```text
kcd2-dual-subtitles.exe generate --game "C:\path\to\Content" --output "kcd2-dual-subtitles.zip"
```

Other preserved CLI entrypoints:

```text
kcd2-dual-subtitles.exe --help
kcd2-dual-subtitles.exe --version
```

The acceptance-only `--canary-id` option also remains available for controlled localization diagnostics.

## Main language and KCD2 language

The game's currently selected text language should match the generated **main** language so KCD2 opens the corresponding mod localization PAK.

## Troubleshooting

KCD2 writes `kcd.log` in the Windows Documents folder used by the game. A successful localization-patch load contains:

```text
[Mod] Loading localization patch 'Localization\text_ui__kcd_dual_subtitles.xml'
```

Useful failure indicators include CryPak errors such as `ReadFile returned 15`, XML parse failures, or the absence of the mod/localization patch from the log.

Do not upload or commit full proprietary game localization files when reporting an issue.

## Microsoft Defender SmartScreen

The executable is currently **not Authenticode-signed**. Windows may therefore show Microsoft Defender SmartScreen reputation warnings for a newly downloaded build.

The project does not attempt to bypass Windows security checks. Stable/release-candidate binaries are built by GitHub Actions and published with `SHA256SUMS.txt` so the downloaded file can be verified against the corresponding release.

If Microsoft Defender Antivirus reports an actual named malware/threat detection rather than a SmartScreen reputation warning, report the exact detection name and release version for separate false-positive/security investigation instead of disabling protection.

## Current limitations

- Russian + English only;
- dialogue localization from `text_ui_dialog.xml` only;
- no bilingual item names, quest text, tutorials, codex, or general UI text;
- language separation is text-based (`[RU]` / `[EN]`), not a custom Scaleform subtitle UI;
- Xbox / Microsoft Store is the only live-validated store target;
- Steam/GOG/Epic autodetection is not implemented;
- executable is currently unsigned;
- no application self-update or persistent settings.

## Development

The project uses Go 1.27 and the standard library wherever practical. The Windows GUI uses Win32 APIs directly and does not require Wails, React, Node or WebView.

All automated acceptance checks, Windows tests, builds, checksums, and release publication run through GitHub Actions. Local test/build results are not used as merge or release acceptance evidence.

## License

MIT. See [`LICENSE`](LICENSE).
