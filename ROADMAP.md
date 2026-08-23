# KCD2 Dual Subtitles — development plan

## Goal

Build a small, dependency-light command-line tool that creates bilingual dialogue subtitles for Kingdom Come: Deliverance II from the localization files installed on the user's machine.

The first validated target is Russian + English on the Xbox / Microsoft Store PC build.

## Development rules

1. Plan each implementation stage before changing production code.
2. Keep `main` releasable; implementation work happens in branches and pull requests.
3. All automated verification runs in GitHub Actions CI. Local test/build results are not acceptance evidence.
4. Every behavior change gets automated coverage where practical.
5. Prefer the Go standard library and keep dependencies minimal.
6. v0.1 is a single Windows CLI executable; no GUI.
7. Never modify the game's original localization files.
8. Manual in-game checks are reserved for behavior CI cannot prove.

## v0.1 stages

- [x] Stage 0 — repository and CI bootstrap
- [x] Stage 1 — localization model and synthetic fixtures
- [x] Stage 2 — PAK/ZIP reader
- [x] Stage 3 — robust dialogue XML parser
- [x] Stage 4 — deterministic bilingual merge
- [x] Stage 5 — mod archive builder
- [x] Stage 6 — CLI UX
- [x] Stage 7 — current KCD2 mod-format validation
- [x] Stage 8 — Xbox / Microsoft Store retail acceptance
- [ ] Stage 9 — first stable release

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

## Stage 9 — first stable release

Tracked by issue #32.

Implementation/release requirements:

- [x] add compact `[RU]` / `[EN]` labels to truly bilingual rows while preserving main-language-first order;
- [x] keep identical and single-language fallback rows untagged;
- [x] preserve patch minimization and the accepted literal `\\n` separator;
- [x] add stable GitHub Release workflow with Linux and native Windows gates;
- [x] embed and verify the stable version in the executable and generated manifest;
- [x] generate SHA-256 checksums in CI;
- [x] replace bootstrap README with validated install/use/update/uninstall/troubleshooting documentation;
- [x] document unsigned-executable / SmartScreen reputation behavior without bypass guidance;
- [ ] publish `v0.1.0` through the stable release workflow;

Stage 9 is complete only after the stable workflow successfully publishes `v0.1.0`.

## Post-v0.1

Do not block the first reliable release on:

- custom Scaleform subtitle styling / separate visual treatment for the two languages;
- additional bilingual UI categories such as quests, items, skills, tutorials, or encyclopedia;
- additional languages;
- GUI;
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
