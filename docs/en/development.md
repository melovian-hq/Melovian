# Development

Everyday commands use the standard toolchain. The Taskfile wraps them (`task --list` for the full list, `task verify` for the composite gate).

| Command | What it does |
|------|----------------|
| `cp .env.example .env && cd frontend && pnpm install` | First-time bootstrap (`bash build/scripts/setup.sh` adds toolchain hints and lefthook hooks) |
| `wails3 dev -config ./build/config.yml -port 9245` | Desktop app with hot reload |
| `bash build/scripts/dev-server.sh` | HTTP API (`MELOVIAN_LISTEN`, default `127.0.0.1:17337`) plus Vite on port 9245 with `/api` proxy |
| `go test ./internal/... ./services/...` | Backend unit tests |
| `cd frontend && pnpm test` | Frontend unit tests |
| `cd frontend && pnpm check` | Frontend typecheck (`svelte-check`) |
| `cd frontend && pnpm lint` / `pnpm lint:fix` | ESLint, optionally autofix |
| `golangci-lint run ./...` | Go lint |
| `gofmt -w .` / `cd frontend && pnpm format` | Format Go / frontend |
| `wails3 generate bindings -clean=true -ts && bash build/scripts/bindings-postprocess.sh` | Regenerate `frontend/bindings` after Go service changes |
| `bash build/scripts/check-bindings-drift.sh` | Fail when committed bindings differ from generated output |
| `bash build/scripts/go-vendor.sh` | Refresh `./vendor` after Go dependency changes |
| `go test -bench=. -benchmem -count=1 ./internal/cache/ ./internal/localmusic/ ./internal/subsonic/ ./internal/httputil/ ./internal/video/` | Go allocation benchmarks |
| `cd frontend && pnpm test:coverage && bash build/scripts/coverage-go.sh` | Frontend + Go coverage floors |
| `cd frontend && pnpm lighthouse:ci` | Lighthouse CI (expects a built `frontend/dist`; `task lighthouse` builds the static demo first) |
| `cd scripts/showcase && pnpm install && pnpm exec playwright install chromium && node capture.mjs` | Capture demo screenshots into `showcase/` (needs `bin/melovian-server`) |

The pre-merge gate is `task verify`, or run its parts by hand: `bash build/scripts/check-bindings-drift.sh`, `bash build/scripts/check-gofmt.sh`, `golangci-lint run ./...`, `go test -race ./...`, `cd frontend && pnpm test:property && pnpm lint && pnpm check && pnpm test`. Full CI still runs e2e smoke, mutation, coverage floors, Lighthouse, and mobile packaging on top of that.

## Toolchain pins

| Tool | Pin |
|------|-----|
| Go | `go.mod` (1.26.x) |
| Node | `.nvmrc` / mise (`22`) |
| pnpm | `frontend/package.json` `packageManager` (`11.1.2`) |
| Task / golangci-lint / Wails CLI / lefthook | `mise.toml` (optional) |

Install [mise](https://mise.jdx.dev/) and run `mise install` to sync optional CLIs. Without mise, install Go, Node 22, and pnpm from their own docs. Task is optional. Wails CLI matches `go.mod`:

```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.16
```

Git hooks (format + lint on staged files):

```bash
go install github.com/evilmartians/lefthook@v1.11.13
lefthook install
```

Skip hooks for one commit with `LEFTHOOK=0 git commit ...`.

## Dev Container

[`.devcontainer/devcontainer.json`](../../.devcontainer/devcontainer.json) is a server-mode / web container (Go + Node + Task, no libmpv desktop stack). Open the folder in a Dev Container, then:

```bash
bash build/scripts/setup.sh
bash build/scripts/dev-server.sh
```

UI on port 9245, API on 17337.

## Editor

- [`.editorconfig`](../../.editorconfig)
- [`.vscode/extensions.json`](../../.vscode/extensions.json) and settings (format on save, ESLint in `frontend/`, golangci-lint)

AI agents and contributors: see [AGENTS.md](../../AGENTS.md) for repo layout, skills under `.agents/skills/`, and writing rules.

## Tech stack

- **Backend:** Go, SQLite, Subsonic API client, optional OIDC
- **Frontend:** Svelte 5, Vite, Tailwind CSS
- **Desktop:** Wails v3, libmpv on Linux
- **Deploy:** Docker Compose for headless / web use
