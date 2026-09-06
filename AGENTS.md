# AGENTS.md

Concise ground rules and a map for anyone editing Melovian with AI assistance.

## The shortest path in

1. Run `task --list` to see available commands. On a fresh clone, run `task setup`.
2. Before a broad change, load the relevant skill from `.agents/skills/`.
3. Run the smallest test that covers the change before finishing. For backend work that is `go test ./internal/... ./services/...`. For UI work that is `cd frontend && pnpm check`.
4. Do not commit secrets, `.env` values, or credentials.

## What Melovian is

A desktop and web music player for Subsonic-compatible servers (Navidrome and others) and local folders.

| Component | Stack |
|---|---|
| Backend | Go with Wails v3 bindings |
| Frontend | Svelte 5 SPA under Vite |
| Persistence | SQLite by default, Postgres via `MELOVIAN_DATABASE_URL` |
| Desktop audio | libmpv on Linux |
| Build | Task 3.x, pnpm 11.1.2, Go 1.26+ |

## Where code lives

| Area | Path | What is inside |
|---|---|---|
| HTTP API + business logic | `internal/` | `internal/api/`, `internal/subsonic/`, `internal/localmusic/`, `internal/store/`, `internal/libmpv/` |
| Wails service layer | `services/` | Desktop services exposed to the frontend |
| Svelte frontend | `frontend/src/` | Pages, components, music config, Subsonic client |
| Build + packaging | `build/`, `Taskfile.yml` | Wails config, per-OS task files, dev loop |
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
| `melovian-maintenance` | Updating deps, changing CI, or fixing build/test issues |

The `no-ai-slop` and `rossmann-voice` skills are vendored from [no_ai_slop_writing_rules](https://github.com/realrossmanngroup/no_ai_slop_writing_rules). Update them with:

```bash
curl -sL https://api.github.com/repos/realrossmanngroup/no_ai_slop_writing_rules/tarball/main | tar xz -C /tmp
cp -r /tmp/no_ai_slop_writing_rules-main/skills/no-ai-slop .agents/skills/
cp -r /tmp/no_ai_slop_writing_rules-main/skills/rossmann-voice .agents/skills/
```

## Where to put new work

| Feature type | Start here |
|---|---|
| New browse page | `frontend/src/pages/`, register in `frontend/src/routes.ts` |
| Player or queue behavior | `frontend/src/lib/config/music.svelte.ts`, `frontend/src/lib/config/music/*-ops.ts` |
| Settings tab UI | `frontend/src/lib/components/settings/panels/Settings*Panel.svelte` |
| Subsonic API call | `frontend/src/lib/subsonic/api.ts` or the library adapter |
| Persistent setting | Go handler in `internal/api/`, mirror in frontend prefs if needed |
| Desktop-only behavior | `internal/desktop/`, `services/`, graphics settings |
| Mix or radio algorithm | `frontend/src/lib/music/mix-generator/`, `personal-radio.ts`, `taste-score.ts` |

## Development loop

```bash
task setup              # first clone: .env, frontend deps, toolchain hints
task --list             # discover all tasks
task dev                # desktop app with Vite hot reload
task dev:server         # HTTP API + Vite (server mode)
task test               # Go tests + frontend Vitest
task verify             # format, lint, race tests, typecheck
task generate:bindings  # after Go Wails service signature changes
```

Package-scoped backend tests do not need the built frontend embed:

```bash
go test ./internal/... ./services/...
```

Frontend install and type check:

```bash
cd frontend && pnpm install  # pnpm 11.1.2
cd frontend && pnpm check
```

Root `go test .` fails unless `frontend/dist` is already built. Use package-scoped tests or `task test`.

## Writing rules

1. Read `.agents/skills/no-ai-slop/SKILL.md` before editing README, UI strings, toast text, or docs.
2. No emdashes in comments or docs. Use a period, comma, or parentheses.
3. No semicolons in code comments or doc comments. Use periods or separate sentences.
4. Do not add markdown files the user did not ask for.
5. Match existing code style. Small diffs beat drive-by refactors.

## Verification before finishing

- Run `task test` or the narrowest test subset that covers the change.
- For UI work, run `cd frontend && pnpm check` when types might break.
- Do not commit secrets, `.env` values, or user credentials.

## Common traps

| Symptom | Likely cause | Fix |
|---|---|---|
| `go test .` fails on embed | `frontend/dist` is missing | Build the frontend or use `go test ./internal/... ./services/...` |
| Wails dev will not start | Missing `build/` or CLI version mismatch | Run `task setup` and check the Wails CLI version against `go.mod` |
| Bindings drift in CI | Go service signature changed | Run `task generate:bindings` and commit `frontend/bindings/` |
| Desktop blank WebView on Linux | WebKit GPU issues | See the README logging section or try disabling GPU |
