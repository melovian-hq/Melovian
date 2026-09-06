---
name: melovian-maintenance
description: Melovian maintenance workflow: dependencies, Task commands, tests, CI, and safe upgrade paths. Use when updating deps, fixing CI, or onboarding to the dev loop.
---

# Melovian maintenance

## Toolchain pins

| Tool | Version / source |
|------|------------------|
| Go | 1.26+ (`go.mod`) |
| Node | 22 in CI |
| pnpm | 11.1.2 (`frontend/package.json` `packageManager`) |
| Wails | v3 alpha/beta (`go.mod`, Wails CLI for dev) |
| Task | 3.x |

## Daily commands

```bash
task setup                # first clone: .env + pnpm install
task --list               # discover tasks
task dev                  # desktop + vite
task dev:server           # HTTP API + vite (no Wails GUI)
task test                 # go test + pnpm test
task verify               # local pre-merge gate (format, lint, race, typecheck)
task generate:bindings    # after Go service signature changes
cd frontend && pnpm check # svelte-check / TS
```

Optional pins: `.nvmrc`, `mise.toml`. Git hooks: `lefthook install` (see `lefthook.yml`). Dev Container: `.devcontainer/` (server-mode stack).

Linux desktop native builds need libmpv, GTK 4, and WebKitGTK dev packages. See README.

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
go test ./internal/... ./services/...
```

Wails bumps may require regenerating `frontend/bindings/` and running `task test:bindings-drift`.

## CI expectations

`.github/workflows/` run `task verify` or subsets. A change that passes `task test` locally but skips `verify` may still fail on race tests or svelte-check.

## Build folder

`build/` holds per-OS Taskfiles, Wails config, and packaging assets. If `build/` is missing, restore from git. `task dev` depends on `build/config.yml`.

## Docker server mode

```bash
cp .env.example .env
task build:docker
cd docker && docker compose up -d
```

Set `MELOVIAN_AUTH_SECRET` for multi-user web deploys. Demo mode uses `MELOVIAN_DEMO_MODE` plus Navidrome env vars.

## Data locations

- Desktop: XDG data dir, default `~/.local/share/melovian/`
- Logs: `{data}/logs/melovian.log`, `client.log`, `crashes/`
- SQLite DB and offline cache under the same data root

## Agent skills in this repo

- `.agents/skills/no-ai-slop` and `rossmann-voice` from [no_ai_slop_writing_rules](https://github.com/realrossmanngroup/no_ai_slop_writing_rules)
- `melovian-codebase` and this skill for repo-specific knowledge

When editing skills, keep YAML frontmatter `name` matching the directory name.

## Common failure modes

| Symptom | Likely cause |
|---------|----------------|
| `go test .` fails on embed | Build frontend first or test subpackages only |
| Wails dev won't start | Missing `build/`, wrong Wails CLI version |
| Bindings drift CI | Run Wails generate and commit `frontend/bindings/` |
| Desktop blank WebView on Linux | Graphics stability settings, WebKit GPU issues (see README logging section) |
