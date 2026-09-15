# AGENTS.md

Concise ground rules and a map for anyone editing Melovian with AI assistance.

## The shortest path in

1. On a fresh clone, copy `.env.example` to `.env` and run `cd frontend && pnpm install`. `task setup` does both plus toolchain hints, and `task --list` shows every wrapped command.
2. Before a broad change, load the relevant skill from `.agents/skills/`.
3. Run the smallest test that covers the change before finishing. For backend work that is `go test ./internal/... ./services/...`. For UI work that is `cd frontend && pnpm check`.
4. Do not commit secrets, `.env` values, or credentials.

## What Melovian is

A desktop and web music player for Subsonic-compatible servers (Navidrome and others) and local folders.

| Component | Stack |
|---|---|
| Backend | Go 1.26 with Wails v3 (beta.16) bindings |
| Frontend | Svelte 5 (runes) SPA under Vite 8, Tailwind v4, bits-ui, runed |
| Persistence | SQLite (modernc.org/sqlite) by default, Postgres via `MELOVIAN_DATABASE_URL` |
| Desktop audio | libmpv or libvlc, selectable backend via AudioService |
| Build | Task 3.x, pnpm 11.1.2, Node 22 |

## Where code lives

| Area | Path | What is inside |
|---|---|---|
| HTTP API + business logic | `internal/` | 30+ packages: `api/`, `subsonic/`, `subsonicserver/`, `localmusic/`, `store/`, `libmpv/`, `libvlc/`, `pcmsink/`, `dlna/`, `jukebox/`, `lastfm/`, `rocksky/`, `transcode/`, `update/`, `sandbox/`, `metadata/`, `metaloader/`, `smartplaylist/`, `extensions/`, `navidrome/`, `video/`, `democatalog/` and helpers. See `melovian-codebase` skill for the full map |
| Wails service layer | `services/` | `MediaService`, `AudioService`, `UpdateService` plus tray and window chrome |
| Svelte frontend | `frontend/src/` | `pages/` + custom router (`lib/router/`, `routes.ts`), `lib/components/`, `lib/config/music*` store, `lib/features/` vertical modules, `lib/music/` domain logic, `lib/subsonic/` client |
| Build + packaging | `build/`, `Taskfile.yml` | `build/config.yml` (Wails v3 config), per-OS task files, dev loop |
| Docker server mode | `docker/` | Headless API + static frontend |

## Branding and constants

Product identity is centralized for rebranding:

- `frontend/src/lib/brand.ts`: `APP_NAME`, `APP_SLUG`, storage keys, file names.
  Override at build time with `VITE_APP_NAME` and `VITE_APP_SLUG`.
- `internal/brand/brand.go`: `brand.Name` and `brand.Slug`. Override at link time
  with `-X melovian/internal/brand.Name=... -X melovian/internal/brand.Slug=...`.
- `index.html` uses `%VITE_APP_NAME%` / `%VITE_APP_SLUG%` tokens resolved by Vite.
- Changing the slug orphans localStorage keys, temp dirs, log files, and the
  extension manifest name. Keep it stable unless the rebrand intends a clean break.
- `X-Melovian-*` HTTP headers in `internal/compat` and `lib/compat` are wire
  protocol and intentionally stay fixed.

## .agents directory

Agent context lives in `.agents/`.

- `.agents/skills/`: task-specific skills. Load one before a broad change.
- `.agents/conventions/`: code, git, and security conventions.
- `.agents/rules/`: prose and process rules.
- `.agents/knowledgebase/`: deep notes on audio, scrobbling, and other domains.

## Skill trigger reference

Skills live in `.agents/skills/`. Load the relevant one before starting a broad change.

| Skill | Use when |
|---|---|
| `no-ai-slop` | Writing prose, UI strings, comments, docs, or commit messages humans will read |
| `rossmann-voice` | Writing long-form content that should sound like Louis Rossmann |
| `melovian-codebase` | Navigating the repo or planning a feature that crosses Go and Svelte |
| `melovian-frontend` | Writing frontend code: Svelte 5 runes, Tailwind v4, bits-ui v2, runed, Vitest 4, TS 6 syntax rules |
| `melovian-wails` | Editing Go services, Wails bindings, `build/config.yml`, or desktop window behavior |
| `melovian-testing` | Writing or running tests: file suffix taxonomy, mocking, coverage gates, e2e |
| `melovian-quality` | Applying the code quality bar or deciding if a change is done |
| `melovian-debugging` | Investigating a bug or crash: logs, dev modes, env toggles, tracing across layers |
| `melovian-maintenance` | Updating deps, changing CI, or fixing build/test issues |

The `no-ai-slop` and `rossmann-voice` skills are vendored from [no_ai_slop_writing_rules](https://github.com/realrossmanngroup/no_ai_slop_writing_rules). Update them with:

```bash
curl -sL https://api.github.com/repos/realrossmanngroup/no_ai_slop_writing_rules/tarball/main | tar xz -C /tmp
cp -r /tmp/no_ai_slop_writing_rules-main/skills/no-ai-slop .agents/skills/
cp -r /tmp/no_ai_slop_writing_rules-main/skills/rossmann-voice .agents/skills/
```

The local `no-ai-slop` files carry additions beyond upstream: the `.agents/rules/prose.md` pointer in SKILL.md, the "Newer patterns (2025-2026 model output)" section, and the extended sections in `references/ai-writing-detection.md` (significance inflation, -ing tails, copula avoidance, vague attribution, chatbot residue, marketing register, formatting bleed, expanded markup artifacts, vocabulary-by-era). Re-vendoring overwrites them; re-apply the local sections afterward or diff before copying.

## Where to put new work

| Feature type | Start here |
|---|---|
| New browse page | `frontend/src/pages/`, register in `frontend/src/routes.ts` |
| Player or queue behavior | `frontend/src/lib/config/music.svelte.ts`, `frontend/src/lib/config/music/*-ops.ts` |
| Settings tab UI | `frontend/src/lib/components/settings/panels/Settings*Panel.svelte`, register in `frontend/src/lib/settings/tabs.ts` |
| Subsonic API call | `frontend/src/lib/subsonic/api.ts` or the library adapter |
| Persistent setting | Go handler in `internal/api/`, mirror in frontend prefs if needed |
| config.toml key | `internal/appconfig/configfile.go` key map, see `docs/en/configuration.md` |
| Desktop-only behavior | `internal/desktop/`, `services/`, `frontend/src/lib/desktop/` |
| Wails service method | `services/`, register in `main.go`, then regenerate bindings (see below) |
| Mix or radio algorithm | `frontend/src/lib/music/mix-generator/`, `personal-radio.ts`, `taste-score.ts` |
| Scrobbling (Last.fm, ListenBrainz, Rocksky) | `internal/lastfm/`, `internal/api/{lastfm,listenbrainz,rocksky,scrobblers}.go`, see `.agents/knowledgebase/scrobbling.md` |
| Native audio output | `internal/libmpv/`, `internal/libvlc/`, `internal/pcmsink/`, see `.agents/knowledgebase/audio.md` |
| Serving library to other clients | `internal/subsonicserver/` (`/rest`), `internal/dlna/` |
| Extension feature | `internal/extensions/`, `frontend/src/lib/extensions/` |

## Development loop

```bash
cp .env.example .env                        # first clone
cd frontend && pnpm install                 # frontend deps (pnpm 11.1.2)
wails3 dev -config ./build/config.yml -port 9245   # desktop app with Vite hot reload
bash build/scripts/dev-server.sh            # HTTP API + Vite (server mode)
go test ./internal/... ./services/...       # backend tests, no frontend embed needed
cd frontend && pnpm test                    # frontend Vitest
cd frontend && pnpm check                   # svelte-check typecheck
wails3 generate bindings -clean=true -ts && bash build/scripts/bindings-postprocess.sh
```

Each command has a `task` wrapper (`task dev`, `task test`, `task verify`, `task generate:bindings`). `task verify` is the composite pre-merge gate: bindings drift, format check, golangci-lint, `go test -race ./...`, property tests, ESLint, svelte-check, Vitest.

Package-scoped backend tests do not need the built frontend embed:

```bash
go test ./internal/... ./services/...
```

Frontend install and type check:

```bash
cd frontend && pnpm install  # pnpm 11.1.2
cd frontend && pnpm check
```

Root `go test .` fails unless `frontend/dist` is already built. Use package-scoped tests or `cd frontend && pnpm test` for the frontend side.

## Writing rules

1. Read `.agents/skills/no-ai-slop/SKILL.md` before editing README, UI strings, toast text, or docs.
2. No emdashes in comments or docs. Use a period, comma, or parentheses.
3. No semicolons in code comments or doc comments. Use periods or separate sentences.
4. Do not add markdown files the user did not ask for.
5. Match existing code style. Small diffs beat drive-by refactors.

## Verification before finishing

- Run `go test ./internal/... ./services/...` or the narrowest test subset that covers the change. Frontend: `cd frontend && pnpm test`.
- For UI work, run `cd frontend && pnpm check` when types might break.
- Do not commit secrets, `.env` values, or user credentials.

## Common traps

| Symptom | Likely cause | Fix |
|---|---|---|
| `go test .` fails on embed | `frontend/dist` is missing | Build the frontend (`cd frontend && pnpm build`) or use `go test ./internal/... ./services/...` |
| Wails dev will not start | Missing `build/` or CLI version mismatch | Run `bash build/scripts/setup.sh` and check the Wails CLI version against `go.mod` |
| Bindings drift in CI | Go service signature changed | Run `wails3 generate bindings -clean=true -ts && bash build/scripts/bindings-postprocess.sh` and commit `frontend/bindings/` |
| Vite fails on `wails("./bindings")` | `frontend/bindings/` missing or stale | Regenerate bindings as above |
| Desktop blank WebView on Linux | WebKit GPU issues | `WEBKIT_DISABLE_DMABUF_RENDERER=1` or the graphics stability settings; see the README logging section |
