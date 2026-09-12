---
name: melovian-maintenance
description: Melovian maintenance workflow: dependencies, Task commands, tests, CI, and safe upgrade paths. Use when updating deps, fixing CI, or onboarding to the dev loop.
---

# Melovian maintenance

## Toolchain pins

Versions are pinned in `mise.toml`, `frontend/package.json` (`packageManager`), `.nvmrc`, and `go.mod`.

| Tool | Pinned | Source |
|------|--------|--------|
| Go | 1.26 (`go 1.26.6` in go.mod) | `mise.toml`, `go.mod` |
| Node | 22 | `.nvmrc`, `mise.toml` |
| pnpm | 11.1.2 | `packageManager` field |
| Task | 3.46.3 | `mise.toml` |
| Wails | v3.0.0-beta.16 | `go.mod`, `mise.toml` wails3 |
| golangci-lint | 2.13.2 | `mise.toml` |
| lefthook | 1.11.13 | `mise.toml` |
| goreleaser | 2.18.1 | `mise.toml` |

The `wails3` CLI must match the `go.mod` Wails version. Install with `go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.16`.

## Daily commands

```bash
task setup                # first clone: .env + pnpm install
task --list               # discover tasks
task dev                  # wails3 dev -config ./build/config.yml -port 9245
task dev:server           # HTTP API + vite (no Wails GUI)
task test                 # go test + pnpm test
task verify               # local pre-merge gate (format, lint, race, typecheck)
task generate:bindings    # after Go service signature changes
cd frontend && pnpm check # svelte-check / TS
```

Git hooks: `lefthook install` (see `lefthook.yml`). Dev Container: `.devcontainer/` (server-mode stack).

Linux desktop native builds need libmpv, libvlc, GTK 4, and WebKitGTK dev packages. See README.

## Frontend dependency status (as of 2026-09)

| Package | Pinned range | Latest | Note |
|---------|--------------|--------|------|
| svelte | ^5.56.8 | 5.57.x | Current |
| @sveltejs/vite-plugin-svelte | ^7.2.0 | 7.3.x | Requires Vite 8 + Svelte >=5.46.4 |
| vite | ^8.2.0 | 8.2.x | Rolldown + Oxc based, ESM-only |
| vitest | ^4.1.11 | 5.0.0 | Major behind. Vitest 5 changes defaults (clearMocks, extends). Stay on 4 unless deliberately upgrading |
| typescript | ~6.0.3 | 7.0.x | Major behind. TS 7 is the native Go port. Stay on 6 |
| eslint | ^10.8.0 | 10.9.x | Flat config only |
| eslint-plugin-svelte | ^3.22.0 | 3.23.x | Flat config only, ESM-only |
| tailwindcss / @tailwindcss/vite | ^4.3.3 | 4.3.x | Current |
| bits-ui | ^2.19.2 | 2.19.x | Current |
| runed | 0.37.1 | 0.37.x | Current |
| @wailsio/runtime | 3.0.0-beta.8 | 3.0.0-beta.9 | Keep pinned to a resolved version, never `latest` |
| @playwright/test | ^1.55.0 | 1.63.x | Range already resolves to latest 1.x |

## Go dependency status (as of 2026-09)

`modernc.org/sqlite` has v1.58.0 available (project has v1.57.0). `pgx/v5`, `coder/websocket`, `go-oidc/v3`, `fsnotify`, `prometheus/client_golang`, `sentry-go` are current. `oto/v3` v3.5.0 is ahead of the latest stable line; treat it as intentional.

## Dependency updates

**Frontend**

```bash
cd frontend && pnpm update
pnpm test && pnpm check
```

Pin `@wailsio/runtime` to a resolved version after major updates. Do not leave `latest` in `package.json`.

**Go**

```bash
go get -u ./...
go mod tidy
task go:vendor          # repo vendors modules into ./vendor
go test ./internal/... ./services/...
```

Wails bumps may require regenerating `frontend/bindings/` and running `task test:bindings-drift`. Bump the wails3 CLI in `mise.toml` to match `go.mod`.

## CI expectations

`.github/workflows/` run `task verify` or subsets. A change that passes `task test` locally but skips `verify` may still fail on race tests or svelte-check.

## Build folder

`build/` holds per-OS Taskfiles, `config.yml` (Wails v3 project config), packaging assets, and `build/scripts/bindings-postprocess.sh` which runs after bindings generation. `task dev` depends on `build/config.yml`. After changing `info` or `fileAssociations` in `config.yml`, run `wails3 task common:update:build-assets`.

## Docker server mode

```bash
cp .env.example .env
task build:docker
cd docker && docker compose up -d
```

Set `MELOVIAN_AUTH_SECRET` for multi-user web deploys. Demo mode uses `MELOVIAN_DEMO_MODE` plus `NAVIDROME_*` env vars, or the built-in `democatalog` when unset. Other env toggles in `.env.example`: `MELOVIAN_SUBSONIC_SERVER` (serve local library over `/rest`), `MELOVIAN_DLNA_SERVER`, `MELOVIAN_OIDC_*`, `MELOVIAN_LANDLOCK`.

## Data locations

- Desktop: XDG data dir, default `~/.local/share/melovian/`
- Logs: `{data}/logs/melovian.log`, `client.log`, `crashes/`
- SQLite DB and offline cache under the same data root

## Agent skills in this repo

- `.agents/skills/no-ai-slop` and `rossmann-voice` from [no_ai_slop_writing_rules](https://github.com/realrossmanngroup/no_ai_slop_writing_rules)
- `melovian-codebase`, `melovian-frontend`, `melovian-wails`, `melovian-testing`, `melovian-quality`, `melovian-debugging`, and this skill for repo-specific knowledge

When editing skills, keep YAML frontmatter `name` matching the directory name.

## Common failure modes

| Symptom | Likely cause |
|---------|----------------|
| `go test .` fails on embed | Build frontend first or test subpackages only |
| Wails dev won't start | Missing `build/`, or wails3 CLI version mismatch with `go.mod` |
| Bindings drift CI | Run `task generate:bindings` and commit `frontend/bindings/` |
| Desktop blank WebView on Linux | WebKit GPU issues; try `WEBKIT_DISABLE_DMABUF_RENDERER=1` or graphics stability settings (see README logging section) |
| Vite fails on `wails("./bindings")` plugin | `frontend/bindings/` missing; run `task generate:bindings` |
