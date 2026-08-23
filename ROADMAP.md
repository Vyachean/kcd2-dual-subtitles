# KCD2 Dual Subtitles — development plan

## Goal

Build a small, dependency-light command-line tool that creates a bilingual subtitle mod for Kingdom Come: Deliverance II from the localization files installed on the user's machine.

The first supported target is Russian + English on the current Xbox Store / Xbox app PC build, while keeping the implementation platform-independent enough for Steam, GOG, and Epic layouts.

## Development rules

1. Plan each implementation stage before changing production code.
2. Keep `main` releasable; implementation work happens in branches and pull requests.
3. All automated verification runs in GitHub Actions CI. Do not rely on local test/build results as acceptance evidence.
4. Every behavior change should have automated coverage where practical.
5. Prefer the Go standard library and keep dependencies minimal.
6. No GUI in v0.1. The deliverable is a single Windows CLI executable.
7. Do not modify the game's original localization files; generate a standalone mod archive.
8. Manual in-game checks are reserved for behavior CI cannot prove, especially Xbox Store loading and subtitle rendering.

## v0.1 acceptance target

Given a KCD2 game root and two supported languages, the tool must:

- locate the required `Localization/*_xml.pak` files without depending on a Steam-specific executable path;
- read the current `text_ui_dialog.xml` data from both language PAKs;
- match dialogue rows deterministically by ID;
- preserve Unicode, XML entities, and real multiline content;
- produce `main language + newline + secondary language` for differing translations;
- keep a single copy for identical text;
- keep the main-language text when the secondary row is missing;
- fail explicitly on malformed input instead of silently dropping data;
- create a valid standalone KCD2 mod archive without modifying original game files;
- print a useful generation summary;
- return meaningful exit codes and errors;
- build as a single Windows x64 executable in CI;
- pass all automated checks in CI;
- work in the current Xbox Store PC build for normal dialogue and at least one story cutscene.

## Stage 0 — repository and CI bootstrap

- [ ] Add a minimal Go module and CLI entry point.
- [ ] Add `.gitignore`, `LICENSE`, and a concise `README.md`.
- [ ] Add GitHub Actions CI for:
  - `gofmt` verification;
  - `go vet`;
  - `go test ./...`;
  - Windows amd64 build;
  - artifact upload for the Windows executable.
- [ ] Require future PR work to use the same CI checks as acceptance evidence.

## Stage 1 — localization model and fixtures

- [ ] Define supported language metadata and PAK filename mapping.
- [ ] Add representative synthetic fixtures for KCD2 dialogue XML.
- [ ] Cover Russian/English Unicode text.
- [ ] Cover XML entities and punctuation.
- [ ] Cover real newline characters inside `<Cell>` values.
- [ ] Cover missing/empty/identical translations.

Fixtures must be minimal and synthetic; do not commit copyrighted game localization dumps.

## Stage 2 — PAK/ZIP reader

- [ ] Open KCD2 language `.pak` files using the ZIP reader.
- [ ] Locate and read `text_ui_dialog.xml`.
- [ ] Produce explicit errors for missing files, malformed archives, and duplicate/ambiguous entries.
- [ ] Add focused automated tests.

## Stage 3 — robust dialogue XML parser

- [ ] Parse rows structurally rather than with line-based regular expressions.
- [ ] Preserve the three-cell row semantics used by KCD2 dialogue localization.
- [ ] Preserve multiline content.
- [ ] Preserve/produce valid XML escaping.
- [ ] Detect malformed rows explicitly instead of silently omitting them.
- [ ] Add round-trip and regression tests.

## Stage 4 — deterministic bilingual merge

- [ ] Match rows by dialogue ID.
- [ ] Define deterministic merge behavior:
  - identical text -> one copy;
  - different text -> main + `\\n` + secondary in the game-facing value;
  - missing secondary -> main only;
  - duplicate IDs -> explicit error unless real fixtures prove another rule is required.
- [ ] Preserve main-language ordering.
- [ ] Produce statistics: processed, bilingual, identical, missing-secondary, errors.
- [ ] Add table-driven tests.

## Stage 5 — mod archive builder

- [ ] Generate the nested localization PAK expected by KCD2.
- [ ] Generate `mod.manifest` with clear project identity and attribution where appropriate.
- [ ] Generate the final distributable ZIP.
- [ ] Write through a temporary file and publish atomically on success.
- [ ] Never leave a partial success artifact after an error.
- [ ] Add tests that inspect the complete generated archive structure and contents.

## Stage 6 — CLI UX

- [ ] Support explicit non-interactive arguments suitable for scripting and CI fixtures.
- [ ] Provide a simple interactive mode when launched without sufficient arguments.
- [ ] Validate game root and language PAK availability before generation.
- [ ] Print clear errors and generation statistics.
- [ ] Return stable non-zero exit codes on failure.
- [ ] Add `--help` and `--version`.

Proposed non-interactive shape:

```text
kcd2-dual-subtitles.exe generate \
  --game "C:\\XboxGames\\Kingdom Come- Deliverance II\\Content" \
  --main Russian \
  --secondary English \
  --output "."
```

## Stage 7 — current KCD2 mod-format validation

- [ ] Verify the generated nested archive structure against current KCD2 mod behavior.
- [ ] Confirm the expected generated XML filename and merge behavior.
- [ ] Confirm whether any additional metadata such as `mod_order.txt` is needed.
- [ ] Adjust implementation only from observed/current behavior, not assumptions copied from old tools.

## Stage 8 — Xbox Store acceptance

Manual acceptance that cannot be replaced by CI:

- [ ] Use the real Xbox app KCD2 `Content` directory as input.
- [ ] Generate Russian + English from the installed current localization PAKs.
- [ ] Install the generated mod without replacing the original `Russian_xml.pak`.
- [ ] Verify bilingual subtitles in normal dialogue.
- [ ] Verify bilingual subtitles in a story cutscene.
- [ ] Record the tested KCD2 version and installation path convention in the PR/release notes.

## Stage 9 — release pipeline and documentation

- [ ] Add tag-driven GitHub Release workflow.
- [ ] Build Windows amd64 release artifact in CI.
- [ ] Generate SHA-256 checksums in CI.
- [ ] Document Xbox Store, Steam, GOG, and Epic input-path guidance where verified.
- [ ] Document installation and regeneration after game updates.
- [ ] Publish known limitations.

## Post-v0.1

Do not block the first reliable release on these features:

- additional bilingual UI categories (quests, items, skills, tutorials, encyclopedia);
- per-category separators;
- automatic mod installation;
- GUI;
- third-party translation patch inputs;
- additional output formats.

## Workflow for each stage

For every implementation stage:

1. Write/update the stage plan and acceptance criteria first.
2. Create a dedicated branch and PR.
3. Implement the smallest coherent change.
4. Use GitHub Actions CI for all automated checks/builds.
5. Inspect CI failures and fix them in the same PR.
6. Record any required manual acceptance separately.
7. Merge only when required CI is green and the stage criteria are satisfied.
