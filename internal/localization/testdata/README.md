# Synthetic localization fixtures

These files are intentionally small and entirely synthetic. They are not copied from Kingdom Come: Deliverance II localization data.

Both XML files use the three-`Cell` row shape expected by later parser stages:

1. dialogue row ID;
2. synthetic source/auxiliary value;
3. displayed localized text.

Covered cases:

- `dialog_regular` — ordinary English/Russian Unicode text;
- `dialog_entities` — XML entities that must decode and re-encode correctly;
- `dialog_multiline` — a real newline inside the third `<Cell>`;
- `dialog_identical` — the same displayed value in both languages;
- `dialog_missing_secondary` — present only in the Russian/main fixture;
- `dialog_empty` — an explicitly empty displayed value.

Later stages should use these fixtures as regression inputs rather than adding real game localization dumps to the repository.
