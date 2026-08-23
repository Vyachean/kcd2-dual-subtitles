# KCD2 Dual Subtitles

A lightweight command-line tool for generating bilingual subtitle mods for Kingdom Come: Deliverance II from the localization files installed with the game.

## Status

Early development. The current repository only contains the project bootstrap; KCD2 localization parsing and mod generation are not implemented yet.

The v0.1 target is Russian + English generation from the current Xbox Store / Xbox app PC build, with platform-independent input handling suitable for Steam, GOG, and Epic where their localization layout matches the supported format.

See [`ROADMAP.md`](ROADMAP.md) and the roadmap issue for the development plan.

## Design goals

- single Windows CLI executable;
- minimal dependencies, preferring the Go standard library;
- no modification of original game localization files;
- deterministic output and explicit errors;
- automated verification and Windows builds in GitHub Actions CI;
- no GUI in v0.1.

## Development

Go 1.27 or newer is used for the project toolchain.

All automated acceptance checks run in GitHub Actions. Local test/build results are not used as merge acceptance evidence.

## License

MIT. See [`LICENSE`](LICENSE).
