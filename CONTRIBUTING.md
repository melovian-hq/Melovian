# Contributing

Thanks for helping with Melovian. This file is for humans. AI agents should also read [AGENTS.md](AGENTS.md).

## Before you start

1. Read the alpha warning in [README.md](README.md). Behavior and APIs still move.
2. Install the toolchain listed under **What you need** in the README (Go 1.26+, Node 22+, pnpm 11, Wails v3 CLI, Task optional).
3. Copy `.env.example` to `.env` and run through [docs/en/getting-started.md](docs/en/getting-started.md).

Platform package lists: [docs/en/requirements.md](docs/en/requirements.md).

## Development loop

| Goal | Command |
|------|---------|
| First-time bootstrap | `cp .env.example .env && cd frontend && pnpm install` |
| Desktop app with hot reload | `wails3 dev -config ./build/config.yml -port 9245` |
| Server API + Vite (no GUI) | `bash build/scripts/dev-server.sh` |
| Backend unit tests | `go test ./internal/... ./services/...` |
| Frontend unit tests | `cd frontend && pnpm test` |
| Frontend typecheck | `cd frontend && pnpm check` |
| Frontend lint | `cd frontend && pnpm lint` |
| Regenerate Wails bindings | `wails3 generate bindings -clean=true -ts && bash build/scripts/bindings-postprocess.sh` |

A Taskfile wraps these and more (`task dev`, `task test`, `task --list` for the full list). `task verify` is the local pre-merge gate: format check, golangci-lint, race-enabled Go tests, property tests, ESLint, svelte-check, and Vitest. Run it before you open a pull request. Full CI also runs e2e smoke, mutation, coverage, Lighthouse, and mobile packaging.

Root `go test .` fails until the frontend embed exists. Use the package paths, or build the frontend with `cd frontend && pnpm build` first.

Go modules are vendored in `./vendor`. After changing `go.mod` / `go.sum`, run:

```bash
bash build/scripts/go-vendor.sh
```

That script patches Wails WebView2 embed files (see `third_party/wails-webview2loader/`) then runs `go mod vendor`. Builds and Docker images use `-mod=vendor`.

Do not load Google Fonts or other CSS/JS CDNs. Fonts stay in-repo (`frontend/public/Inter-Medium.ttf`, extension assets under `frontend/public/extensions/`). The UI font stack is system fonts in `frontend/src/lib/theme/tokens.css`.

More detail: [docs/en/development.md](docs/en/development.md).

## Where code lives

| Area | Path |
|------|------|
| HTTP API | `internal/api/` |
| SQLite / Postgres store | `internal/store/` |
| Subsonic client / proxy | `internal/subsonic/` |
| Local library scan | `internal/localmusic/` |
| Wails desktop services | `services/` |
| Svelte UI | `frontend/src/` |
| Playback store | `frontend/src/lib/config/music.svelte.ts` and `frontend/src/lib/config/music/` |

Do not hand-edit `frontend/bindings/` except as a short-lived fix. Regenerate with `wails3 generate bindings -clean=true -ts && bash build/scripts/bindings-postprocess.sh` and check drift with `bash build/scripts/check-bindings-drift.sh`.

## Pull requests

1. Fork (or branch) from the default branch.
2. Keep the change focused. One problem per PR when you can.
3. Match the style of the files you touch. Skip drive-by renames and unrelated refactors.
4. Add or update tests next to the code (`*_test.go`, `*.test.ts`).
5. Run `go test ./internal/... ./services/...` and `cd frontend && pnpm test` at minimum. Run `task verify` for anything that touches shared packages or CI-sensitive paths.
6. Write a short PR description that states what broke or what is missing, and how you checked the fix.

Commit messages in this repo are usually imperative and short (`Add video settings panel`, `Fix share token expiry`). Prefer that over changelog essays in the subject line.

## Docs and UI copy

User-facing strings, README edits, and docs follow the writing rules in [AGENTS.md](AGENTS.md):

- No emdashes
- No semicolons in comments or doc comments
- Concrete wording over marketing filler

Read `.agents/skills/no-ai-slop/SKILL.md` before large prose edits.

## Issues

Bug reports work best with:

- Melovian version or git commit
- OS and how you run it (desktop binary, `wails3 dev`, Docker, server binary)
- Steps to reproduce
- Relevant lines from `{data}/logs/melovian.log` (desktop default: `~/.local/share/melovian/logs/`)

Security problems: [SECURITY.md](SECURITY.md). Do not file those in public issues.

## License

By contributing you agree that your changes are licensed under the Apache License, Version 2.0, the same as the rest of the project. See [LICENSE](LICENSE) and [NOTICE](NOTICE).
