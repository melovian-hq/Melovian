---
name: melovian-codebase
description: Melovian architecture map, module boundaries, and where to implement features. Use when navigating the repo or planning changes across Go and Svelte.
---

# Melovian codebase map

## Product shape

One app shell (Wails desktop or HTTP server) serves a Svelte 5 SPA. The Go backend proxies Subsonic APIs, indexes local folders, stores accounts and settings in SQLite by default (Postgres via `MELOVIAN_DATABASE_URL`), and streams audio. The frontend holds playback state in `frontend/src/lib/config/music.svelte.ts` (reactive facade) with logic in `frontend/src/lib/config/music/*-ops.ts` and context binders in `contexts.ts`.

The frontend uses its own router (`frontend/src/lib/router/`, routes in `frontend/src/routes.ts`), not SvelteKit. Pages are lazy-loaded with `load: () => import(...)`.

## Frontend map

```text
frontend/src/pages/*.svelte          UI routes (lazy imports in routes.ts)
frontend/src/routes.ts               Route table: path, load/component, title, description
frontend/src/lib/router/             Custom router: match.ts, RouteOutlet, Link, RouteBoundary
frontend/src/lib/components/         Reusable UI by area:
  ui/                                  primitives (dialogs, select, command palette, toggles)
  music/                               player chrome, EQ, playlist pickers, share dialogs
  settings/panels/                     one Settings*Panel.svelte per settings tab
  layout/ home/ instances/ desktop/    shell, home feed, server picker, window controls
  metadata/ notifications/ onboarding/ tag editor UI, notification panel, first-run flow
frontend/src/lib/config/
  music.svelte.ts                      Playback store (orchestrator facade)
  music/                               Music store modules (one *-ops.ts per concern)
    contexts.ts                          Lazy context binders for ops hosts
    queue-ops.ts                         Queue mutations
    playback-transport-ops.ts            Play/pause/seek/volume/next/prev
    playback-core-ops.ts                 Core playback wiring and helpers
    play-launch-ops.ts                   Start playback / launch paths
    player-chrome-ops.ts                 Player chrome visibility and layout
    favorites-ops.ts                     Starred tracks, albums, artists
    playlist-ops.ts                      Local and server playlists
    library-browse-ops.ts                Home data, search, genres, artists
    library-refresh-ops.ts               Library refresh and watch/poll
    settings-ops.ts                      Music settings persistence
    lyrics-ops.ts                        Lyrics fetch and cache
    eq-ops.ts                            Equalizer settings
    cache-ops.ts                         Download / offline cache
    mix-ops.ts radio-ops.ts              Personal mix and radio controls
    connect-ops.ts                       Server connect / reconnect
    engine-ops.ts                        Audio engine bind / teardown
    track-boundary-ops.ts                Track end, continuous refill, restore, scrobble
    helpers.ts types.ts                  Shared store helpers and types
  runtime.ts remote-server.ts          Runtime detection, remote server config
  demo-guards.ts                       Demo-mode write guards
frontend/src/lib/features/           Vertical feature modules (api + store.svelte.ts + types):
  auth/ filesystem/ instances/         login, folder pickers, saved Subsonic servers
  local-libraries/ metadata-editor/    local folder management, tag editor
  sources/                             source activation (which library is live)
frontend/src/lib/music/              Domain logic (~90 modules): audio-engine,
  native-audio-engine, playback-*, mixes, personal-radio, taste-score,
  mix-generator/, smart-playlist/, offline-*, lyrics*, metadata-enhancement,
  device-sync, listen history, search, related tracks, transcoding settings
frontend/src/lib/core/               Primitives: http/ (client, errors, api-paths),
  bounded-cache, collection, detail-cache, dom/, events/ws.svelte.ts, logger, sentry
frontend/src/lib/subsonic/           Subsonic HTTP client and types
frontend/src/lib/desktop/            window-chrome, window-close, graphics-settings,
  close-prompt, folder-dialog (desktop-only paths, lazy @bindings imports)
frontend/src/lib/extensions/         Extension registry, assets, styles, features
frontend/src/lib/ui/                 toast.svelte.ts, confirm, command-palette,
  keyboard-help, boot-error, focus-trap, client-error
frontend/src/lib/theme/              tokens.css (--jb-* custom properties), custom.css,
  theme.svelte.ts, contrast.ts
frontend/src/lib/media/              native.svelte.ts, native-media-sync.ts
frontend/src/lib/notifications/      notifications.svelte.ts + api.ts
frontend/src/lib/profile/            profile.svelte.ts, profile-settings.ts
frontend/src/lib/backup/             backup.ts (export/import user data)
frontend/src/lib/video/              video feature state + api
frontend/src/lib/settings/           settings tabs.ts, settings-page.css
frontend/src/lib/tasks/              tasks.svelte.ts (background task tracking)
frontend/src/lib/app/bootstrap.ts    App.svelte lifecycle effects
frontend/src/lib/demo/               static-api.ts (static demo mode)
frontend/src/lib/brand.ts            APP_NAME, APP_SLUG, storage keys
frontend/src/lib/compat/             client-side X-Melovian-* wire headers
frontend/bindings/                   Generated Wails bindings (do not hand-edit)
```

## Backend map

```text
main.go                Desktop entry: application.New, services, window, tray, landlock
main_server.go         Server-mode entry (-tags server)
main_mobile.go         Mobile entry (-tags android/ios)
services/              Wails services: MediaService (per-OS files), AudioService,
                       UpdateService, tray hooks, window chrome, graphics models
internal/api/          HTTP handlers, auth, middleware, assets, Last.fm /
                       ListenBrainz / Rocksky submit endpoints, api-routes
                       contract testdata
internal/subsonic/     Upstream Subsonic proxy + cache
internal/subsonicserver/ Melovian serving a Subsonic-compatible /rest API itself
                       (exposes the local library to other Subsonic clients)
internal/navidrome/    Navidrome-specific client (smart playlists, X-ND-Authorization)
internal/localmusic/   Folder scan, catalog, cover cache
internal/metaloader/   Tag reading across formats, hashing, scanner
internal/metadata/     Tag editing: artwork, filename rules, issue reporting
internal/smartplaylist/ Smart playlist evaluation
internal/store/        SQLite/Postgres persistence (modernc.org/sqlite, pgx)
internal/desktop/      Window state, close/quit handlers, tray manager,
                       graphics settings, WebKit env fixes, media keys
internal/libmpv/       libmpv audio engine
internal/libvlc/       libvlc audio engine (alternative native backend)
internal/pcmsink/      PCM routing: device/stdout/fifo/tcp/unix sinks, fanout, mixer
internal/jukebox/      Subsonic jukebox-mode playback controller
internal/dlna/         DLNA/UPnP MediaServer (SSDP advertise + content directory)
internal/transcode/    ffmpeg-based transcoding (optional, detected at runtime)
internal/lyrics/       LRCLIB + custom lyrics providers
internal/lastfm/       Last.fm Audioscrobbler 2.0 client
internal/rocksky/      Rocksky listen submission client
internal/extensions/   Bundled extension install. Bundles: lastfm,
                       listenbrainz, lyrics, lyrics-whisper, metadata, rocksky
internal/video/        YouTube Data API + Invidious search
internal/update/       Self-update pipeline: discovery, delta, apply
internal/sandbox/      Landlock filesystem sandbox (Linux), MELOVIAN_LANDLOCK toggle
internal/appconfig/    Env loading, dotenv, Sentry config
internal/brand/        brand.Name / brand.Slug (ldflags-overridable)
internal/compat/       X-Melovian-* wire protocol headers and version contracts
internal/democatalog/  Built-in fake Subsonic catalog for demo mode
internal/cache/        Generic cache helpers
internal/httputil/     JSON/error helpers, redaction
internal/melog/        App logging + crash handler
internal/observability/ Sentry init and flush
internal/seo/          Per-route HTML meta for the SPA shell (server mode)
internal/osutil/       Path helpers, reveal-in-file-manager
internal/termout/      Terminal output helpers
```

## Wiring between layers

- Frontend calls the Go backend two ways: REST under `/api/*` (see `frontend/src/lib/core/http/api-paths.ts`) and Wails service bindings.
- Wails bindings are imported lazily: `await import("@bindings/melovian/services/index.js")` gives `MediaService`, `AudioService`, `UpdateService`. Lazy import keeps server/web mode from loading desktop-only code. `@bindings` is a Vite alias for `frontend/bindings/`.
- `@wailsio/runtime` provides `Events.On/Emit`, `Dialogs`, `Window`. The `wails("./bindings")` Vite plugin injects typed event definitions.
- Three services are registered in `main.go`: MediaService, AudioService, UpdateService. Add new ones under `services/` and register in `application.Options.Services`, then run `task generate:bindings`.

## Multi-instance model

Users can save several Subsonic servers ("instances"). Connection state, credentials, and per-instance data flow through `internal/store` and `frontend/src/lib/features/instances/`. Local libraries can be per-user when auth is enabled.

## Audio paths

- Web playback: HTML audio via `frontend/src/lib/music/audio-engine.ts`
- Desktop native: libmpv (`internal/libmpv/`) or libvlc (`internal/libvlc/`) through the AudioService binding (`SetPreferredBackend` takes "auto", "mpv", "vlc"); decoded PCM routes through `internal/pcmsink/` (see `.agents/knowledgebase/audio.md`)
- Server mode: no window. API and static frontend only

## Generated code

- `frontend/bindings/` comes from `wails3 generate bindings -ts` (via `task generate:bindings`). `task test:bindings-drift` checks drift in CI. Do not hand-edit bindings.

## Tests

- Frontend unit tests: Vitest 4 under `frontend/src/**/*.test.ts`
- Extra Vitest configs: `vitest.contract.config.ts`, `vitest.acceptance.config.ts`, `vitest.exploratory.config.ts`, `vitest.oracle.config.ts` (`pnpm test:property` etc.)
- Playwright e2e: `frontend/e2e/`, `pnpm test:e2e`
- Go tests: colocated `*_test.go`. Use `go test ./internal/... ./services/...`; root `go test .` needs `frontend/dist` built.

When unsure which file owns a behavior, search for the user-visible string or API path in `internal/api/` and `frontend/src/lib/config/music.svelte.ts` first.
