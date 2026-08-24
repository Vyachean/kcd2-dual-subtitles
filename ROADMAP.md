# KCD2 Dual Subtitles — roadmap

## Product direction

KCD2 Dual Subtitles is a dependency-light Windows tool that builds bilingual dialogue subtitles from localization files already installed with Kingdom Come: Deliverance II.

The currently validated platform is KCD2 1.5.6 from Xbox / Microsoft Store on PC.

## Development rules

1. Keep `main` releasable; production changes go through branches and pull requests.
2. GitHub Actions is the automated acceptance source for formatting, tests, native Windows builds and release artifacts.
3. Use synthetic fixtures in the repository; never commit or redistribute proprietary KCD2 localization or GFX assets.
4. Never modify original game files.
5. Fail closed when game UI/PAK structure is incompatible or when another mod supplies an unknown HUD override.
6. Prefer the Go standard library and one Windows executable containing both GUI and CLI entrypoints.
7. Treat live in-game behavior as a separate acceptance gate when CI cannot prove it.

## v0.1 — localization baseline

Status: **complete**. Stable release: `v0.1.0`.

Delivered and retail-validated:

- localization PAK reading;
- dialogue XML parsing;
- deterministic bilingual merge;
- patch-only localization PAK generation;
- portable ZIP output;
- Windows Documents installation and safe `mod_order.txt` integration;
- ordinary and story/cutscene dialogue in the Xbox / Microsoft Store retail build.

## v0.2 — native Windows usability

The v0.2 implementation was completed and published as an RC, then absorbed into the larger v0.3 release line instead of receiving a separate stable tag.

Delivered:

- Xbox / Microsoft Store autodetection;
- `.GamingRoot` custom-root discovery;
- manual Browse fallback;
- native Win32 GUI;
- explicit language selectors;
- Generate / Regenerate / Uninstall;
- safe own-mod removal and load-order cleanup;
- preserved CLI and portable ZIP mode;
- native Windows CI/build/smoke coverage.

Issue #36 is historical/superseded by the v0.3 stable release.

## v0.3 — stable fixed-pair styled subtitles

Status: **release-ready** after the v0.3 RC retail acceptance cycle.

The stable v0.3 contract is intentionally a **generation-time fixed pair**, not the larger runtime-language architecture originally explored in #39.

### Languages and generation

- [x] explicit metadata for the current known KCD2 PC localization PAK set;
- [x] show only supported languages actually installed with the selected game root;
- [x] English-first neutral GUI default instead of a Russian-specific default;
- [x] selected Main/Secondary languages are text sources only;
- [x] generated localization patches are emitted for every supported installed game-language slot, so the selected pair works independently of KCD2's current text language;
- [x] preserve identical/missing/empty translation fallbacks;
- [x] safe portable ZIP and automatic installation paths.

### Styled HUD presentation

- [x] derive `hud.gfx` from the user's installed game instead of shipping proprietary UI assets;
- [x] deterministic semantic AVM1 patching with fail-closed anchors and idempotence;
- [x] standard bottom subtitle path;
- [x] overhead NPC bubble path;
- [x] primary color, optional size and italic;
- [x] secondary color, size and italic;
- [x] language tags on/off;
- [x] native Windows color picker;
- [x] common outline on/off;
- [x] common shadow on/off;
- [x] preserve the proven legacy tagged path when appearance customization is disabled;
- [x] foreign HUD conflict detection.

### Windows installation robustness

- [x] real Windows Documents Known Folder;
- [x] redirected/OneDrive Documents support;
- [x] bounded retry around Windows rename/sharing failures;
- [x] guarded copy fallback for cloud-backed Documents when rename remains unavailable;
- [x] staged replacement and rollback;
- [x] safe existing `mod_order.txt` preservation;
- [x] safe uninstall.

### Retail acceptance established during v0.3 RCs

- [x] ordinary styled bottom subtitles;
- [x] overhead styled bubbles;
- [x] secondary color/size/italic visibly render;
- [x] forced centering removed after it was shown to disturb dialogue-choice layout;
- [x] presentation GUI works in the retail environment;
- [x] language pair is independent of the game's active text language (for example Czech + German while KCD2 remains English);
- [x] outline/shadow path accepted in the retail test cycle.

## Known v0.3 limitations / future work

### Narrative/cinematic captions — #54

Some standalone narrative captions use the separate:

```text
fc_setNarrativeSubtitles(text, layout, fadeIn)
```

path rather than the proven `fc_setSubtitles` / `fc_setBubbleText` paths. This remains a future, research-first change. Do not guess its TextField/geometry/fade contract from the existing patchers.

### Runtime secondary-language switching and in-game settings — #39

The original v0.3 research issue described a larger architecture where one installed universal mod would switch secondary language/style inside KCD2.

That work is **not part of stable v0.3.0**. Future work should still begin with the narrow retail proof that a project-owned generated localization key can be resolved safely through `TextExtension.translateString`, then proceed to runtime session state and Menu.gfx integration only if that proof succeeds.

No KCSE, MCM, LuaDB, ASI loader or `dinput8.dll` dependency is planned for the Xbox-first baseline.

### Presentation preview

A live/approximate preview was deliberately deferred from v0.3.0. If implemented later, it should remain an approximation and must not become a second source of formatting rules.

### Other possible future work

- Steam/GOG/Epic autodetection and explicit retail validation;
- additional bilingual categories such as quests, items, tutorials or codex;
- persisted presentation profiles;
- application self-update;
- Authenticode signing;
- third-party translation inputs.

## Engineering continuity

For the detailed HUD/AVM1 evidence, failed RC experiments and exact future constraints, see [`docs/v0.3-development-handoff.md`](docs/v0.3-development-handoff.md).
