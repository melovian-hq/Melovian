---
name: melovian-wails
description: Wails v3 specifics for Melovian: service registration, bindings generation, @wailsio/runtime API, build/config.yml, and desktop gotchas. Use when editing Go services, bindings, or desktop window behavior.
---

# Melovian Wails v3

Wails v3 is in beta. `go.mod` pins `v3.0.0-beta.16`; `frontend/package.json` pins `@wailsio/runtime` `3.0.0-beta.8`. The `wails3` CLI must match `go.mod` (`mise.toml` pins it). Docs: `v3.wails.io` (the `v3alpha` site is stale).

## Service registration

Services are plain Go structs in `services/`. Register them in `main.go`:

```go
app := application.New(application.Options{
    Name: brand.Name,
    Services: []application.Service{
        application.NewService(mediaSvc),
        application.NewService(audioSvc),
        application.NewService(updateSvc),
    },
    Assets: application.AssetOptions{
        Handler: &api.CombinedHandler{
            API:    apiServer.Handler(),
            Assets: application.BundledAssetFileServer(assets),
        },
    },
})
```

- `application.NewService[T](*T)` wraps a pointer to a concrete named type.
- Services start in registration order, shut down in reverse order.
- Implement `ServiceStartup(ctx, application.ServiceOptions) error` for init; returning an error aborts startup.
- Only exported methods with serializable types become JS bindings.
- Melovian services: `MediaService` (window chrome, tray, graphics, media keys; per-OS files `mediaservice_{darwin,linux,windows,stub}.go`), `AudioService` (native audio engines), `UpdateService` (self-update hooks wired to `internal/update` via `api.DesktopUpdateHooks`).

## Bindings

```bash
task generate:bindings   # wails3 generate bindings -f '<flags>' -clean=true -ts
                         # then build/scripts/bindings-postprocess.sh
```

- Output: `frontend/bindings/`, mirroring Go import paths (`melovian/services/audioservice.ts`, `models.ts`, `index.ts`).
- Generated calls use `$Call.ByID(id, args...)` returning `$CancellablePromise<T>`.
- Do not hand-edit `frontend/bindings/**`. `task test:bindings-drift` fails CI on drift. Commit regenerated bindings with the Go change.
- The `wails("./bindings")` Vite plugin in `frontend/vite.config.ts` injects typed event definitions. A missing or stale `bindings/` dir breaks `vite build` and `vite dev`.

## Frontend usage

Lazy-import bindings so server/web mode never loads desktop code:

```ts
const { MediaService } = await import("@bindings/melovian/services/index.js");
await MediaService.SomeMethod(arg);

import type { GraphicsSettings } from "@bindings/melovian/services/models.js";
```

`@bindings` is a Vite alias for `frontend/bindings/`.

`@wailsio/runtime` provides:

- `Call.ByID(id, ...)` / `Call.ByName("pkg.Struct.Method", ...)`; `Call.RuntimeError` distinguishes a Go-returned error from transport failure.
- `Events.On(name, cb)`, `Events.Emit(name, data)`, `Events.Off`. Used for `WINDOW_CLOSE_REQUESTED_EVENT` and `APP_QUIT_REQUESTED_EVENT` in `frontend/src/lib/desktop/window-close.ts`.
- `Dialogs`, `Window`, `Clipboard`, `Browser`, `Screens` (e.g. `Dialogs` in `lib/desktop/folder-dialog.ts`).
- `Stream(name)` pairs with Go `app.HandleStream(name, handler)` (added beta.8) for byte streams; becomes a real WebSocket in `-tags server` builds.

Tests mock both modules: `vi.mock("@bindings/melovian/services/index.js")`, `vi.mock("@wailsio/runtime")`.

## Config and dev mode

- `build/config.yml` is the v3 project config: `info:` metadata, `ios:` overrides, `dev_mode:` watcher + command orchestration, `fileAssociations:`.
- Dev loop: `task dev` runs `wails3 dev -config ./build/config.yml -port 9245`. The port must match `WAILS_VITE_PORT` and `frontend/vite.config.ts` (`strictPort`).
- After changing `info` or `fileAssociations`, run `wails3 task common:update:build-assets` (overwrites generated assets).
- `wails3 generate icons` refreshes platform icons from `build/appicon.png`.
- Entry points by build tag: `main.go` (desktop, `!server && !android && !ios`), `main_server.go` (`server`), `main_mobile.go` + `app_options_*.go` (android/ios).

## Window and platform notes from this repo

- `application.WebviewWindowOptions` with `Frameless: runtime.GOOS != "darwin"`; macOS uses hidden-inset title bar. Custom window controls live in `frontend/src/lib/components/desktop/WindowControls.svelte`.
- `Linux: application.LinuxOptions{ ProgramName: brand.Slug, DisableQuitOnLastWindowClosed: true }` keeps the tray alive.
- `desktop.ApplyWebKitStabilityEnv(cfg.DataDir)` sets WebKit env fixes before `application.New`. On NVIDIA/WebKitGTK rendering failures the fallback is `WEBKIT_DISABLE_DMABUF_RENDERER=1`.
- Window state persists via `desktop.NewWindowStateStore` + `desktop.ApplyWindowState`.
- `//go:embed all:frontend/dist` in `main.go` means root-level `go build`/`go test .` needs the frontend built; package-scoped tests do not.
- GTK4 + WebKitGTK 6.0 is the default Linux stack. GTK3/WebKit2GTK 4.1 still works via `wails3 build -tags gtk3` but is slated for removal.
