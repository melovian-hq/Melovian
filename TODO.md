# TODO

Living list of unfinished work. Order is roughly priority for the alpha, not a promise. Check [CHANGELOG.md](CHANGELOG.md) for what already landed.

## Near term

- [ ] Finish and document the optional Postgres store path (`MELOVIAN_DATABASE_URL`). Cover migrations, backup notes, and CI coverage next to SQLite.
- [ ] Ship the extension installer end to end: Settings UI, zip validation, bundled reinstall, and asset serving for theme packs.
- [ ] Stabilize video watch mode in Now Playing (queue handoff, settings, and desktop vs web audio/video differences).
- [ ] Cut a tagged `v0.1.0` release once packaging and the docs below match the tree.

## Player and library

- [ ] Offline cache reliability on flaky networks (resume, partial file cleanup, clearer on-player errors).
- [ ] Equalizer and immersive audio settings parity across HTML audio and libmpv.
- [ ] Share inbox / password unlock edge cases (expiry, visit counts, revoked tokens).

## Server and Docker

- [ ] Hardening guide for reverse proxies (TLS, WebSocket upgrade, static asset caching).
- [ ] Metrics and health endpoints documented for operators (`/metrics`, readiness).
- [ ] Backup / restore of the data directory called out in operator docs, not only Settings UI.

## Desktop and mobile

- [ ] Linux WebKit graphics workarounds: keep defaults boring on NVIDIA + Wayland.
- [ ] Windows and macOS packaging smoke tests in CI where runners allow.
- [ ] Android and iOS: closer feature parity with desktop (media keys, background audio limits, install docs).

## Extensions and metadata

- [ ] Public extension manifest schema and example packages under `extensions/examples/`.
- [ ] Metadata editor: batch edits, conflict handling when the file and Subsonic tags disagree.
- [ ] Lyrics: provider failover UI and clearer empty-state copy when no lyrics exist.

## Tests and tooling

- [ ] Grow store driver tests so SQLite and Postgres share the same cases where SQL differs.
- [ ] Keep `task verify` green on race tests after store and auth changes.
- [ ] Bindings drift: regenerate Wails bindings whenever Go service signatures change.

## Docs

- [ ] Keep README, `docs/en/*`, CHANGELOG, and this file in sync when a feature leaves alpha-only status.
- [ ] Add a short operator FAQ for demo mode vs authenticated Docker deploys.

When you finish an item, delete the checkbox line or move a one-line note into CHANGELOG under Unreleased. Do not leave stale checked boxes forever.
