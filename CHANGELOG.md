# Changelog

What changed in Melovian. Plain language. Newest first.

The project is still alpha. Things can change before a stable release.

## 0.1.0 (not released yet)

**Play music**

- Desktop app on Linux, Windows, and macOS
- Website mode with Docker (accounts for each person)
- Connect one or more Navidrome / Subsonic servers
- Play music from folders on your computer
- Local folders stay in sync while the app runs: new, changed, moved, or deleted files update the library without a manual rescan
- More audio formats for local folders: m4b, mka, mp2, WavPack, DSD (dsf/dsd), aifc, tta, multichannel ogg
- Genre browsing, starred list, similar-track suggestions, and lyrics lookup work for local files too
- Browse, search, playlists, favorites, queue
- Offline downloads so tracks work without the network
- Lyrics, mixes, personal radio, keyboard shortcuts, command palette
- Share a playlist or album with a link (password optional)
- Video watch panel next to the music player

**Server and access**

- Sign in with a password, or with OIDC
- Demo mode for a public read-only site
- Optional Postgres database (SQLite is still the default)
- IP allowlist and reverse-proxy settings
- Metrics at `/metrics`
- Optional error reporting (Sentry / GlitchTip)

**Extras**

- Extensions (install from zip, bundled feature toggles and themes)
- Metadata lookup from iTunes, MusicBrainz, Deezer, and TheAudioDB
- Optional Lyrics Whisper extension: generate synced lyrics by transcribing track audio with a whisper.cpp-compatible server or a client-side WASM engine
- Linux media keys (MPRIS), system tray, AppImage with bundled libmpv
- Android and iOS builds (HTML audio in a WebView)

**Developer tooling**

- `task setup`, `task dev:server`, `task generate:bindings`, `task format` / `task fmt` / `task format:check`
- golangci-lint v2 (`task lint:go`), lefthook pre-commit, EditorConfig, VS Code recommendations
- GoReleaser builds the cross-platform server binaries and writes grouped release notes (`task release:snapshot` to try it locally)
- mise / `.nvmrc` pins, Dev Container for server-mode work
- `task --list` shows root tasks with descriptions
