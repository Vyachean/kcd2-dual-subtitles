# Changelog

All notable stable-release changes are recorded here. Release candidates are development history and are not listed individually.

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
- Xbox / Microsoft Store PC is the live-validated target; Steam/GOG/Epic autodetection and explicit validation are not included.
- Release binaries are not Authenticode-signed.
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

[v0.3.0]: https://github.com/Vyachean/kcd2-dual-subtitles/releases/tag/v0.3.0
[v0.1.0]: https://github.com/Vyachean/kcd2-dual-subtitles/releases/tag/v0.1.0
