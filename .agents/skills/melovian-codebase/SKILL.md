---
name: melovian-codebase
description: Melovian architecture map, module boundaries, and where to implement features. Use when navigating the repo or planning changes across Go and Svelte.
---

# Melovian codebase map

## Product shape

One app shell (Wails desktop or HTTP server) serves a Svelte SPA. The Go backend proxies Subsonic APIs, indexes local folders, stores accounts and settings in SQLite by default (Postgres via `MELOVIAN_DATABASE_URL`), and streams audio. The frontend holds playback state in `music.svelte.ts` (reactive facade) with logic in `frontend/src/lib/config/music/*-ops.ts` and context binders in `contexts.ts`.

## Layer diagram

```text
frontend/src/pages/*.svelte          UI routes
frontend/src/lib/components/         Reusable UI
frontend/src/lib/components/settings/panels/   Settings tab panels (one per tab)
frontend/src/lib/config/music.svelte.ts   Playback store (orchestrator)
frontend/src/lib/config/music/         Music store modules
  contexts.ts                          Lazy context binders for ops hosts
  queue-ops.ts                         Queue mutations
  playback-transport-ops.ts            Play/pause/seek/volume/next/prev
  playback-core-ops.ts                 Core playback wiring and helpers
  play-launch-ops.ts                   Start playback / launch paths
  player-chrome-ops.ts                 Player chrome visibility and layout
  favorites-ops.ts                     Starred tracks, albums, artists
  playlist-ops.ts                      Local and server playlists
  library-browse-ops.ts                Home data, search, genres, artists
  settings-ops.ts                      Music settings persistence
  lyrics-ops.ts                        Lyrics fetch and cache
  eq-ops.ts                            Equalizer settings
  cache-ops.ts                         Download / offline cache
  mix-ops.ts                           Personal mix controls
  radio-ops.ts                         Personal radio controls
  connect-ops.ts                       Server connect / reconnect
  engine-ops.ts                        Audio engine bind / teardown
  library-refresh-ops.ts               Library refresh and watch/poll
  track-boundary-ops.ts                Track end, continuous refill, restore
  helpers.ts, types.ts                 Shared store helpers and types
frontend/src/lib/core/http/            Shared fetch client, errors, API paths
frontend/src/lib/app/bootstrap.ts      App.svelte lifecycle effects
frontend/src/lib/music/mix-generator/  Personal mix generation (split module)
frontend/src/lib/subsonic/           Subsonic HTTP client and types
frontend/src/lib/music/              Domain logic (mixes, radio, cache, prefs)
        |
        |  REST /api/*  +  Wails bindings
        v
internal/api/                        HTTP handlers, auth, middleware
internal/subsonic/                   Upstream Subsonic proxy + cache
internal/localmusic/                 Folder scan and catalog
internal/store/                      SQLite/Postgres persistence
services/                            Wails service layer (audio, desktop)
```

## Where to put new work

| Feature type | Start here |
|--------------|------------|
| New browse page | `frontend/src/pages/`, register in `frontend/src/routes.ts` |
| Player or queue behavior | `frontend/src/lib/config/music.svelte.ts`, `frontend/src/lib/config/music/` |
| Settings tab UI | `frontend/src/lib/components/settings/panels/Settings*Panel.svelte` |
| Subsonic API call | `frontend/src/lib/subsonic/api.ts` or library adapter |
| Persistent setting | Go handler in `internal/api/`, mirror in frontend prefs if needed |
| Desktop-only behavior | `internal/desktop/`, `services/`, graphics settings |
| Mix or radio algorithm | `frontend/src/lib/music/mix-generator/`, `personal-radio.ts`, `taste-score.ts` |

## Multi-instance model

Users can save several Subsonic servers ("instances"). Connection state, credentials, and per-instance data flow through `internal/store/instances` and the frontend instances feature module. Local libraries can be per-user when auth is enabled.

## Audio paths

- Web playback: HTML audio via `frontend/src/lib/music/audio-engine.ts`
- Linux desktop native: libmpv through Wails (`internal/libmpv/`, native audio engine on frontend)
- Server mode: no window. API and static frontend only

## Generated code

- `frontend/bindings/` comes from Wails. Regenerate with the Wails CLI after Go binding changes. `task test:bindings-drift` checks drift.
- Do not hand-edit bindings unless you are fixing a one-off and will regenerate soon.

## Tests

- Frontend unit tests: Vitest under `frontend/src/**/*.test.ts`
- Go tests: colocated `*_test.go` and `internal/**/*_test.go`
- Property / oracle tests: separate vitest configs (`vitest.contract.config.ts`, etc.)

When unsure which file owns a behavior, search for the user-visible string or API path in `internal/api/` and `frontend/src/lib/config/music.svelte.ts` first.
