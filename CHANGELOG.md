# Changelog

All notable stable-release changes are recorded here. Release candidates are development history and are not listed individually.

## [v0.3.5] - 2026-08-27

Localization-mod compatibility and generation-transparency release.

### Added

- Main and Secondary dialogue sources now compose compatible active installed localization/correction mods over the stock game localization instead of reading only stock PAKs.
- Generic Warhorse `<anything>_<modid>.xml` localization resources are supported for already-known dialogue IDs, including two-cell and three-cell rows.
- The Windows GUI now displays the active **Mods folder** and allows an explicit custom root with **Change...** / **Reset**.
- The native GUI includes a selectable **Generation activity** log showing selected folders/languages, effective localization sources, applied localization mods, merge/patch counts, HUD mode, load-order validation, install destination and failure context.

### Changed

- Generated localization is emitted only for the selected Main and Secondary language slots instead of duplicating the same bilingual payload under every installed KCD2 language. On the 16-language retail test installation this reduces localization payload duplication from 16 PAK copies to 2 (about 8x less generated localization data).
- KCD2's active text/interface language must therefore be one of the selected Main/Secondary languages. After switching the game to a third language, Regenerate with that language selected.
- Localization source discovery, generation/install status, Regenerate, Uninstall, HUD-conflict detection and `mod_order.txt` now consistently use the same selected Mods root.
- Existing `mod_order.txt` files keep exactly one final `kcd_dual_subtitles` entry while preserving unrelated relative order. UTF-8 BOM is preserved and ignored for project-entry matching/removal.

### Fixed

- Load-order safety now considers every active localization mod capable of writing a relevant dialogue ID, including same-text writes that could still overwrite a bilingual row at runtime.
- `<supports>` is evaluated only after a selected-language PAK proves that a localization mod is relevant to that source.
- Generic localization patches no longer leak unrelated UI/items/quest strings into bilingual subtitles; unknown generic IDs remain excluded.
- Conflicting values for one dialogue ID across separate generic localization resources fail closed instead of inventing an undocumented winner.
- Custom Mods roots are revalidated before use.
- Install transactions now record their Mods-root owner so sibling custom Mods environments cannot recover each other's interrupted transaction.
- Generation failure text no longer overstates publication state when crash-recovery work may still be pending.

### Notes

- Real KCD2 1.5.6 Xbox/GDK acceptance was completed with Chineses Fix 20260727. KCD2 loaded the stock Simplified Chinese localization, then Chineses Fix, then the generated Dual Subtitles localization in the expected order.
- The observed missing CJK glyphs when KCD2 uses an English UI/font configuration are a separate game font limitation and are not changed in this release.
- No proprietary retail or third-party localization content is distributed with the application.

## [v0.3.4] - 2026-08-25

Distribution/package maintenance release. Runtime subtitle generation, HUD transformation and crash-safe install behavior are unchanged from v0.3.3.

### Added

- Versioned Windows x64 ZIP packages for stable releases, release candidates and CI artifacts.
- The distribution ZIP contains only `kcd2-dual-subtitles.exe`, `README.md`, `LICENSE` and an internal `SHA256SUMS.txt`.
- CI validates the exact archive file set and recomputes the packaged-file checksums after extracting the ZIP.
- Stable/RC GitHub releases publish a top-level `SHA256SUMS.txt` covering both the standalone executable and the versioned ZIP.
- README now contains a self-contained `nexus-description` section intended to be reused on Nexus Mods without maintaining a separate copy of the product documentation.

### Changed

- The versioned Windows x64 ZIP is the recommended normal-user download; the standalone executable remains available for advanced use and verification.
- Release-candidate notes are store-neutral and no longer contain obsolete v0.2 acceptance wording.

### Notes

- No KCD2 runtime behavior changes are included in this release.
- The package is suitable for GitHub Releases and Nexus Mods; users extract it anywhere and run the executable rather than installing the executable through a mod manager.

## [v0.3.3] - 2026-08-25

Crash-safe generation/install maintenance release, including broader Windows launcher autodetection already merged after v0.3.2.

### Added

- Best-effort automatic discovery now includes Steam libraries, Epic Games Store installed-game manifests and GOG/Galaxy metadata/default library locations in addition to the existing Microsoft GDK/Xbox discovery.
- Every launcher-derived candidate still passes the same store-neutral structural KCD2 validation before it can be selected.
- Generate/Regenerate/Uninstall mutations are serialized across application instances with a Windows file lock.
- Generated portable ZIPs, staged installations and final published installations are verified against the exact deterministic files requested by the generator before success is reported.

### Fixed

- Generate/Regenerate no longer creates a directory-shaped staging copy inside KCD2's scanned mod root. Transaction workspaces now live beside the resolved mod root on the same volume.
- Interrupted replacement is recoverable: the previous installation and `mod_order.txt` state are preserved until publication is committed, interrupted fresh publication removes its partial target, and committed publication keeps the verified new target.
- Legacy `.kcd_dual_subtitles.staging-*` / `.previous` and interrupted legacy `mod_order.txt` residue from v0.3.2 and earlier are recovered or cleaned before a new publication.
- The generated styled installation is byte-verified to contain the exact localization PAKs and exact derived HUD Data PAK; localization-only mode rejects an accidental HUD payload.
- A corrupt guarded-copy publication now rolls back to the previous installation rather than being reported as successful.
- The native GUI performs Generate/Regenerate off the Win32 UI thread, stays responsive during long generation, and confirms before closing an active operation.
- Game-root/language/appearance controls, including `Browse...`, remain stable while background generation uses the captured selection.
- Successful GUI status now identifies the installed mode/path and explicitly tells the user to restart KCD2 before testing the regenerated mod.
- Uninstall recovers any interrupted generation transaction first and keeps its temporary directory backup outside the scanned mod root.

### Notes

- Retail `kcd.log` evidence identified the v0.3.2 failure source: a leaked `.kcd_dual_subtitles.staging-*` directory could be discovered as a duplicate mod and win localization/Data PAK loading over the canonical `kcd_dual_subtitles` directory.
- The retail-proven direct-HTML Scaleform subtitle transformation is unchanged in this release; the correction is in publication, recovery and activation hygiene.
- When upgrading from v0.3.2 after a failed/interrupted Regenerate, fully close KCD2, run v0.3.3 and Regenerate once before starting the game again.
- Automatic installation remains Windows-only; portable ZIP generation is unchanged.

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

[v0.3.5]: https://github.com/Vyachean/kcd2-dual-subtitles/releases/tag/v0.3.5
[v0.3.4]: https://github.com/Vyachean/kcd2-dual-subtitles/releases/tag/v0.3.4
[v0.3.3]: https://github.com/Vyachean/kcd2-dual-subtitles/releases/tag/v0.3.3
[v0.3.2]: https://github.com/Vyachean/kcd2-dual-subtitles/releases/tag/v0.3.2
[v0.3.1]: https://github.com/Vyachean/kcd2-dual-subtitles/releases/tag/v0.3.1
[v0.3.0]: https://github.com/Vyachean/kcd2-dual-subtitles/releases/tag/v0.3.0
[v0.1.0]: https://github.com/Vyachean/kcd2-dual-subtitles/releases/tag/v0.1.0
