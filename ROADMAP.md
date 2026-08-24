# KCD2 Dual Subtitles — development plan

## Goal

Build a small, dependency-light Windows tool that creates bilingual dialogue subtitles for Kingdom Come: Deliverance II from the localization files installed on the user's machine.

The first validated target is Russian + English on the Xbox / Microsoft Store PC build.

For the detailed current v0.3 state, retail evidence, failed experiments, AVM1/CryPak lessons, and resume instructions, read [`docs/v0.3-development-handoff.md`](docs/v0.3-development-handoff.md) before starting a new implementation slice.

## Development rules

1. Plan each implementation stage before changing production code.
2. Keep `main` releasable; implementation work happens in branches and pull requests.
3. All automated verification runs in GitHub Actions CI. Local test/build results are not acceptance evidence.
4. Every behavior change gets automated coverage where practical.
5. Prefer the Go standard library and keep dependencies minimal.
6. Keep one Windows executable; CLI remains available even when a GUI frontend exists.
7. Never modify the game's original localization files.
8. Never commit or redistribute proprietary game localization/GFX assets.
9. Manual in-game/UI checks are reserved for behavior CI cannot prove.

## v0.1 — complete

- [x] Stage 0 — repository and CI bootstrap
- [x] Stage 1 — localization model and synthetic fixtures
- [x] Stage 2 — PAK/ZIP reader
- [x] Stage 3 — robust dialogue XML parser
- [x] Stage 4 — deterministic bilingual merge
- [x] Stage 5 — mod archive builder
- [x] Stage 6 — CLI UX
- [x] Stage 7 — current KCD2 mod-format validation
- [x] Stage 8 — Xbox / Microsoft Store retail acceptance
- [x] Stage 9 — first stable release

Stable release: `v0.1.0`.

## Accepted v0.1 runtime contract

Stage 8 was completed with `v0.1.0-rc.4` on KCD2 1.5.6 Xbox / Microsoft Store.

Confirmed in the retail game:

- generated mod installed under the real Windows Documents Known Folder;
- existing `mod_order.txt` enabled the mod correctly;
- generated language PAK opened successfully;
- `text_ui__kcd_dual_subtitles.xml` loaded as a localization patch;
- ordinary NPC dialogue displayed both languages;
- story/cutscene dialogue displayed both languages;
- literal `\\n` rendered as a line break;
- no observed CryPak/XML/localization errors attributable to the generated mod.

## v0.2 — usability

Tracked by issue #36.

Goal: normal Windows use should require no console commands while keeping the proven v0.1 generation/mod format unchanged.

Implementation status:

- [x] Xbox / Microsoft Store KCD2 autodetection across fixed drives;
- [x] best-effort custom Xbox game-root discovery through `.GamingRoot`;
- [x] validated manual `Browse...` fallback;
- [x] application service above generator/installer internals;
- [x] safe generated-mod uninstall and `mod_order.txt` cleanup;
- [x] minimal native Win32 GUI with explicit main/secondary language selectors;
- [x] Generate and install / Regenerate state;
- [x] Uninstall state;
- [x] preserve all existing CLI commands and portable ZIP mode;
- [x] Windows no-argument launch opens GUI while explicit CLI commands keep normal console semantics;
- [x] full Linux/native-Windows CI plus CLI/GUI smoke tests on the release binary;
- [x] publish `v0.2.0-rc.1` through release-candidate CI;
- [ ] manually exercise autodetection, Browse, generation/regeneration and uninstall on the validated Xbox environment;
- [ ] publish stable `v0.2.0` only after that manual acceptance.

Explicitly out of scope for v0.2:

- automatic language inference/selection;
- Steam/GOG/Epic autodetection or compatibility claims;
- application self-update;
- persistent settings/history;
- custom Scaleform subtitle UI;
- Authenticode certificate acquisition.

## v0.3 — styled subtitles and runtime secondary language

Primary plan: issue #39. Detailed continuity notes: [`docs/v0.3-development-handoff.md`](docs/v0.3-development-handoff.md).

### Retail-proven foundation

- [x] deterministic derived HUD override from the user's installed `hud.gfx` without shipping proprietary GFX;
- [x] CryPak-compatible generated Data PAK;
- [x] standard bottom subtitle styling through post-vanilla `htmlText` restoration;
- [x] overhead NPC bubble styling through its separate retail render path;
- [x] secondary color, italic, and independent smaller size visibly render in KCD2 retail;
- [x] forced `<p align='center'>` removed after it was proven to disturb dialogue-choice layout;
- [x] `v0.3.0-rc.10` live check confirmed normal dialogue-choice alignment after removing forced centering.

### Current active work

- [ ] #55 Stage 2 — generator-owned presentation options: secondary color, size, italic, tags on/off;
- [ ] #55 Stage 3 — expose the same presentation configuration in the native Windows GUI, with no duplicate formatting implementation;
- [ ] #54 — support the separate narrative/cinematic subtitle path (`fc_setNarrativeSubtitles`), including the opening caption case such as `Несколько недель назад`;
- [ ] #39 — live-prove project-owned namespaced localization lookup through retail `TextExtension.translateString`;
- [ ] #39 — generate universal available-language data instead of one fixed secondary pair;
- [ ] #39 — runtime session state for secondary language/style, including Off and safe fallbacks;
- [ ] #39 — in-game Menu.gfx settings page after runtime lookup/state is proven;
- [ ] #39 — installer/updater integration for the universal runtime mod;
- [ ] full retail acceptance and stable v0.3 release.

Important architecture note: the current direct localization HTML is a successful retail proof/interim fixed-pair mode, but it is not the final runtime-language transport because the same localization rows are also consumed by dialogue-choice UI. Final runtime styling should be applied inside subtitle render paths.

## Later work

Potential later features include:

- additional bilingual UI categories such as quests, items, skills, tutorials, or encyclopedia;
- broader Steam/GOG/Epic live validation;
- Authenticode code signing;
- third-party translation patch inputs;
- retail-safe persistence if a dependency-free mechanism is demonstrated.

## Workflow for each stage

1. Write/update the stage plan and acceptance criteria first.
2. Create a dedicated branch and PR.
3. Implement the smallest coherent change.
4. Use GitHub Actions CI for all automated checks/builds.
5. Fix CI failures in the same PR.
6. Record manual acceptance separately where required.
7. Merge only when required CI is green.
