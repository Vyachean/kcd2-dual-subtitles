# KCD2 Dual Subtitles

A lightweight Windows CLI that generates and installs bilingual dialogue subtitles for Kingdom Come: Deliverance II from the localization files already installed with the game.

## v0.1.0 status

Validated in the retail **Kingdom Come: Deliverance II 1.5.6 Xbox / Microsoft Store PC build**.

Confirmed in live testing:

- ordinary NPC dialogue shows both languages;
- story/cutscene dialogue shows both languages;
- the two languages render on separate lines;
- the generated `text_ui__kcd_dual_subtitles.xml` localization patch is loaded by the game;
- the generated CryPak produces no observed CryPak/XML/localization errors;
- automatic installation works with a redirected OneDrive Documents folder and an existing `mod_order.txt`.

Steam, GOG, and Epic have not yet been live-validated. The input layout may work when their `Localization/*_xml.pak` files match the supported format, but v0.1.0 only claims the Xbox / Microsoft Store PC build above.

## Subtitle format

Bilingual rows are labeled so the languages are easier to distinguish:

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

v0.1.0 supports **Russian** and **English**.

## Download

Use the Windows executable from the latest stable GitHub Release:

- `kcd2-dual-subtitles.exe`
- `SHA256SUMS.txt`

The executable is built and tested by GitHub Actions from this public source repository.

## Generate and install

The default languages are Russian main + English secondary.

```text
kcd2-dual-subtitles.exe generate --game "C:\path\to\Kingdom Come- Deliverance II\Content"
```

Without `--output`, the Windows build automatically installs the generated mod into the real Windows **Documents Known Folder**:

```text
<Documents>\kingdomcome_mods\kcd_dual_subtitles\
```

This intentionally does not assume `%USERPROFILE%\Documents`; redirected and OneDrive Documents folders are supported through the Windows Known Folders API.

The generated mod contains:

```text
kcd_dual_subtitles\
├── mod.manifest
└── Localization\
    └── <main-language>_xml.pak
        └── text_ui__kcd_dual_subtitles.xml
```

If `Documents\kingdomcome_mods\mod_order.txt` already exists, the installer ensures `kcd_dual_subtitles` is listed exactly once without reordering or deleting unrelated mods. If `mod_order.txt` does not exist, the tool does not create it.

The installer replaces only its own `kcd_dual_subtitles` directory and does not modify the game's original localization PAKs.

## Language order

Russian first, English second:

```text
kcd2-dual-subtitles.exe generate --game "C:\path\to\Content" --main Russian --secondary English
```

English first, Russian second:

```text
kcd2-dual-subtitles.exe generate --game "C:\path\to\Content" --main English --secondary Russian
```

The game's currently selected text language should match the generated **main** language so KCD2 opens the corresponding mod localization PAK.

## Portable ZIP mode

To create a ZIP instead of installing automatically:

```text
kcd2-dual-subtitles.exe generate --game "C:\path\to\Content" --output "kcd2-dual-subtitles.zip"
```

The ZIP contains the complete `kcd_dual_subtitles` mod folder.

## Regenerate after game updates

The tool reads the localization files installed with the game each time it runs. After a KCD2 update changes localization data, run the generator again so the patch is rebuilt from the current language PAKs.

Automatic install mode safely replaces the previous generated `kcd_dual_subtitles` folder.

## Uninstall

Remove only:

```text
<Documents>\kingdomcome_mods\kcd_dual_subtitles
```

If your existing `mod_order.txt` contains `kcd_dual_subtitles`, remove that one line when permanently uninstalling the mod. Do not modify the original game localization PAKs.

## Troubleshooting

KCD2 writes `kcd.log` in the Windows Documents folder used by the game. A successful localization-patch load contains:

```text
[Mod] Loading localization patch 'Localization\text_ui__kcd_dual_subtitles.xml'
```

Useful failure indicators include CryPak errors such as `ReadFile returned 15`, XML parse failures, or the absence of the mod/localization patch from the log.

Do not upload or commit full proprietary game localization files when reporting an issue.

## Microsoft Defender SmartScreen

The v0.1.0 executable is **not Authenticode-signed**. Windows may therefore show a Microsoft Defender SmartScreen reputation warning such as "Windows protected your PC" / "unrecognized app" for a newly downloaded release executable.

This project does not attempt to bypass or suppress Windows security checks. To verify the downloaded file, compare its SHA-256 hash with `SHA256SUMS.txt` from the same GitHub Release and inspect the public source/CI history if desired.

A future trusted Authenticode code-signing certificate is the appropriate way to improve executable identity/reputation. A self-signed certificate is not presented as a substitute.

If Microsoft Defender Antivirus reports an actual named malware/threat detection instead of the SmartScreen reputation warning above, report the exact detection name and release version. That should be investigated as a separate false-positive/security case rather than solved by disabling protection.

## Current limitations

- Russian + English only;
- dialogue localization from `text_ui_dialog.xml` only;
- no bilingual item names, quest text, tutorials, codex, or general UI text;
- language separation is text-based (`[RU]` / `[EN]`), not a custom Scaleform subtitle UI;
- Steam/GOG/Epic are not yet live-validated;
- executable is currently unsigned.

## Development

The project uses Go 1.27 and the standard library wherever practical.

All automated acceptance checks, Windows tests, builds, checksums, and release publication run through GitHub Actions. Local test/build results are not used as merge or release acceptance evidence.

## License

MIT. See [`LICENSE`](LICENSE).
