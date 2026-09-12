---
name: melovian-debugging
description: Melovian debugging playbook: reproducing issues, log locations, dev modes, environment toggles, and tracing bugs across the Go/Svelte boundary. Use when investigating a bug or crash.
---

# Melovian debugging

## Method

1. Reproduce first. Pick the smallest environment that shows it: static demo (`VITE_STATIC_DEMO=true pnpm build`), `task dev:server`, or full `task dev` desktop.
2. Trace the path before editing. Find the owning layer with a search for the user-visible string or API path: `internal/api/` for the route, `frontend/src/lib/config/music.svelte.ts` + `*-ops.ts` for playback, `services/` for desktop-only behavior.
3. Add temporary logging at the boundary where behavior diverges from expectation, then remove it before finishing.
4. Fix the root cause, pin it with a `.regression.test.ts` or `*_test.go` case, and verify the test fails without the fix.

## Log locations

Desktop data dir defaults to `~/.local/share/melovian/` (XDG). Server mode uses `MELOVIAN_DATA`.

| File | Contents |
|---|---|
| `logs/melovian.log` | Go `slog` output |
| `logs/client.log` | Frontend errors and client log posts |
| `logs/crashes/` | Panic reports from `melog.DeferredPanicHandler` |

Tuning env vars:

- `MELOVIAN_LOG_LEVEL=debug|info|warn|error` (default info)
- `MELOVIAN_LOG_FILE` overrides the main log path
- Frontend errors post to `POST /api/client-log` (unauthenticated) and land in `client.log`. Boot failures render through `lib/ui/boot-error.ts` / `boot-error-screen.ts`.

## Dev modes

```bash
task dev          # wails3 dev: full desktop, Vite HMR on :9245, Go rebuild on change
task dev:server   # API on :17337 + Vite proxy, no webview. Best for API/frontend bugs
```

- `build/config.yml` `dev_mode` watches `*.go`, rebuilds with `wails3 build DEV=true`, runs `common:dev:frontend` in the background.
- Server-mode requests can go straight at `127.0.0.1:17337` with curl; `/health`, `/metrics` are open. Prometheus metrics come from `prometheus/client_golang`.
- Demo data without a real server: `MELOVIAN_DEMO_MODE=true` serves `internal/democatalog`, or `NAVIDROME_SERVER=fake://melovian-demo`.

## Desktop-specific issues

- Blank or white WebView on Linux: WebKitGTK GPU path. `desktop.ApplyWebKitStabilityEnv` already applies repo-known fixes; `WEBKIT_DISABLE_DMABUF_RENDERER=1` is the next lever (NVIDIA in particular).
- File access unexpectedly denied: the Landlock sandbox (`internal/sandbox/`) restricts filesystem reads. `MELOVIAN_LANDLOCK=false` disables it to confirm the cause. Startup logs the applied status via `sandbox.LogStatus`.
- Tray/window/close weirdness lives in `internal/desktop/` + `services/mediaservice_*.go`; frontend side is `lib/desktop/window-close.ts` listening for `WINDOW_CLOSE_REQUESTED_EVENT` / `APP_QUIT_REQUESTED_EVENT` via `Events.On`.
- Native audio problems: check which backend resolved (`AudioService` caps expose `mpvAvailable`/`vlcAvailable`), then look at `internal/libmpv/`, `internal/libvlc/`, and `internal/pcmsink/` for the PCM path.

## Backend issues

- `slog` fields are the first breadcrumb: `slog.Error("...", "err", err)` calls carry structured context.
- Subsonic proxy problems: `internal/subsonic/` caches upstream responses; check `internal/cache` behavior and whether the bug is upstream or in the proxy transform.
- DB problems: `internal/store/` abstracts SQLite (`modernc.org/sqlite`) and Postgres (`pgx`). `MELOVIAN_DATABASE_URL` switches backends; a bug that only shows under Postgres usually means a dialect assumption in a query.
- `task test:fuzz` covers parsers (metaloader signatures, query ints, auth injection, lyrics normalize, filename sanitize, cache keys). A crash report pointing at parsing is a fuzz-target candidate.

## Frontend issues

- `pnpm check` catches type drift before runtime. `pnpm test` on the owning suite reproduces logic bugs in jsdom without a browser.
- jsdom stubs live in `vitest.setup.ts`; a component test failing on `animate`/`matchMedia`/media methods means the stub is missing, not that the component is broken.
- Wails-dependent code in tests needs `vi.mock("@bindings/...")` / `vi.mock("@wailsio/runtime")`; a "cannot find module" or undefined-service error in a test usually means a missing mock.
- Reactive bugs in `.svelte.ts` stores: effects in `lib/app/bootstrap.ts` own app lifecycle; `contexts.ts` binds ops hosts lazily, so a nil-context error usually means the binder did not run.

## Observability

- Sentry: `internal/observability` (Go) and `lib/core/sentry.ts` (frontend). Configured via `internal/appconfig/sentry.go`; check whether error reporting is enabled before assuming silence means health.
- Crash reports: `melog` writes panics to `logs/crashes/`; the deferred handler must stay at the top of `main`.
