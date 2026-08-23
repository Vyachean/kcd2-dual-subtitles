# KCD2 Dual Subtitles — development plan

## Goal

Build a small, dependency-light Windows tool that creates bilingual dialogue subtitles for Kingdom Come: Deliverance II from the localization files installed on the user's machine.

The first validated target is Russian + English on the Xbox / Microsoft Store PC build.

## Development rules

1. Plan each implementation stage before changing production code.
2. Keep `main` releasable; implementation work happens in branches and pull requests.
3. All automated verification runs in GitHub Actions CI. Local test/build results are not acceptance evidence.
4. Every behavior change gets automated coverage where practical.
5. Prefer the Go standard library and keep dependencies minimal.
6. Keep one Windows executable; CLI remains available even when a GUI frontend exists.
7. Never modify the game's original localization files.
8. Manual in-game/UI checks are reserved for behavior CI cannot prove.

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

Implementation target:

- [ ] Xbox / Microsoft Store KCD2 autodetection across fixed drives;
- [ ] best-effort custom Xbox game-root discovery through `.GamingRoot`;
- [ ] validated manual `Browse...` fallback;
- [ ] application service above generator/installer internals;
- [ ] safe generated-mod uninstall and `mod_order.txt` cleanup;
- [ ] minimal native Win32 GUI with explicit main/secondary language selectors;
- [ ] Generate and install / Regenerate state;
- [ ] Uninstall state;
- [ ] preserve all existing CLI commands and portable ZIP mode;
- [ ] Windows no-argument launch opens GUI while explicit CLI commands keep normal console semantics;
- [ ] full Linux/native-Windows CI plus CLI smoke tests on the release binary;
- [ ] publish `v0.2.0-rc.1` through release-candidate CI;
- [ ] manually exercise autodetection, Browse, generation/regeneration and uninstall on the validated Xbox environment;
- [ ] publish stable `v0.2.0` only after that manual acceptance.

Explicitly out of scope for v0.2:

- automatic language inference/selection;
- Steam/GOG/Epic autodetection or compatibility claims;
- application self-update;
- persistent settings/history;
- custom Scaleform subtitle UI;
- Authenticode certificate acquisition.

## Later work

Potential later features include:

- custom Scaleform subtitle styling / separate visual treatment for the two languages;
- additional bilingual UI categories such as quests, items, skills, tutorials, or encyclopedia;
- additional languages;
- broader Steam/GOG/Epic live validation;
- Authenticode code signing;
- third-party translation patch inputs.

## Workflow for each stage

1. Write/update the stage plan and acceptance criteria first.
2. Create a dedicated branch and PR.
3. Implement the smallest coherent change.
4. Use GitHub Actions CI for all automated checks/builds.
5. Fix CI failures in the same PR.
6. Record manual acceptance separately where required.
7. Merge only when required CI is green.
