# Changelog

All notable stable-release changes are recorded here. Release candidates are development history and are not listed individually.

## [v0.3.2] - 2026-08-24

Automatic-install target correction completing the store-neutral v0.3 compatibility contract.

### Fixed

- Automatic installation now resolves the mod root from the selected KCD2 installation instead of always writing to the Microsoft GDK Documents path.
- Standard compatible PC layouts use `<game-root>\Mods\kcd_dual_subtitles`.
- Microsoft GDK packaged layouts continue to use the real Windows Documents Known Folder at `<Documents>\kingdomcome_mods\kcd_dual_subtitles`.
- GDK layout detection uses package artifacts in or next to the selected installation (`gamelaunchhelper.exe`, `MicrosoftGame.config`, or `appxmanifest.xml`) rather than an `XboxGames` path-name heuristic.
- Install, HUD-conflict detection, `mod_order.txt`, Regenerate status and Uninstall now share one resolved mod root.
- Switching the GUI to another installation with `Browse...` refreshes install state for that selected copy of KCD2 instead of reusing a different installation's status.
- Uninstall removes this project's mod only from the currently selected installation target.

### Notes

- Automatic installation remains Windows-only; portable ZIP generation is unchanged.
- The Microsoft GDK/Documents path remains the live retail-accepted environment from the v0.3 test cycle.
- Standard `<game-root>\Mods` targeting follows the KCD2 PC mod layout and is covered by automated resolver/install tests; not every storefront build has been separately live-tested.

## [v0.3.1] - 2026-08-24

Platform-neutral compatibility correction for the v0.3 release line.

### Fixed

- Game-root validation no longer requires `English_xml.pak` and `Russian_xml.pak` specifically.
- A compatible KCD2 installation is now identified by shared game Data PAKs plus at least two localization PAKs from the explicit supported-language registry.
- Manual `Browse...` selection is explicitly store-neutral: Steam, GOG, Epic Games Store, Xbox / Microsoft Store and other Windows distributions use the same validation and generation path when their KCD2 file structure is compatible.
- User-facing documentation no longer treats the Microsoft Store/Xbox build used for retail testing as the application's compatibility boundary.

### Notes

- Automatic game discovery remains best-effort and currently knows Microsoft GDK/Xbox install roots. Failure to auto-detect another launcher is not a compatibility failure; use `Browse...`.
- The existing v0.3 retail acceptance evidence still comes from KCD2 1.5.6 on the Xbox / Microsoft Store PC build. Other storefront builds are supported structurally but have not all been separately retail-tested by this project.

## [v0.3.0] - 2026-08-24

First stable release of the native Windows GUI and styled subtitle path.

### Added

- Native Win32 application with Xbox / Microsoft Store KCD2 autodetection, manual Browse fallback, Generate / Regenerate and Uninstall.
- Explicit support metadata for the current known KCD2 PC localization set: English, Italian, French, German, Spanish, Czech, Japanese, Korean, Polish, Portuguese (Brazil), Chinese Simplified/Traditional, Turkish, Russian, Ukrainian and Vietnamese.
- Installed-language discovery: the GUI shows only supported localization PAKs actually present in the selected KCD2 installation.
- Neutral language defaults: English is preferred as Main when installed; the first other installed supported language is used as Secondary.
- Styled subtitle mode derived from the user's installed `hud.gfx` without redistributing proprietary KCD2 UI assets.
- Independent primary and secondary color/size/italic presentation controls.
- Native Windows color pickers.
- Language tags on/off.
- Optional common outline and shadow readability effects.
- Styled support for standard bottom subtitles and overhead NPC subtitle bubbles.
- Foreign HUD conflict detection for styled mode.
- Portable ZIP generation alongside automatic installation.
- SHA-256 checksums for release binaries.

### Changed

- Main/Secondary selections are now text sources only. The generated localization patch is emitted under every supported installed localization slot, so the chosen pair works independently of KCD2's currently active text language.
- Removed forced subtitle centering after retail testing showed it displaced dialogue-choice layout.
- The v0.2 GUI/autodetection work is included directly in v0.3.0; no separate stable v0.2.0 is published.

### Fixed

- Czech + German and other non-RU/EN pairs no longer disappear when KCD2 itself is configured to another text language.
- OneDrive/redirected Documents publication now retries transient Windows rename/sharing failures and uses a guarded copy fallback when needed.
- Staged installation/rollback remains safe when publication or load-order updates fail.

### Known limitations

- Secondary language/style is selected at generation time; there is no in-game runtime language switch yet.
- Standalone narrative/cinematic captions using `fc_setNarrativeSubtitles` are outside the proven standard/bubble transformation and may remain unstyled or single-language.
- Dialogue localization only; quests, items, tutorials, codex and general UI text are not bilingualized.
- The executable is not Authenticode-signed.
- Presentation preview was deliberately deferred.

## [v0.1.0] - 2026-08-23

Initial stable localization-only release.

### Added

- Deterministic English/Russian dialogue merge from installed KCD2 localization PAKs.
- Patch-only localization PAK generation.
- Portable ZIP and automatic Windows Documents installation.
- Safe existing `mod_order.txt` integration.
- CLI generation and acceptance canary support.
- Retail validation on KCD2 1.5.6 Xbox / Microsoft Store PC.

[v0.3.2]: https://github.com/Vyachean/kcd2-dual-subtitles/releases/tag/v0.3.2
[v0.3.1]: https://github.com/Vyachean/kcd2-dual-subtitles/releases/tag/v0.3.1
[v0.3.0]: https://github.com/Vyachean/kcd2-dual-subtitles/releases/tag/v0.3.0
[v0.1.0]: https://github.com/Vyachean/kcd2-dual-subtitles/releases/tag/v0.1.0
