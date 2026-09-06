# Development

| Task | What it does |
|------|----------------|
| `task setup` | Copy `.env` if missing, `pnpm install`, print toolchain hints, install lefthook hooks when present |
| `task --list` | List root tasks (each has a one-line `desc`) |
| `task dev` | Run the desktop app with hot reload |
| `task dev:server` | HTTP API (`MELOVIAN_LISTEN`, default `127.0.0.1:17337`) plus Vite on port 9245 with `/api` proxy |
| `task test` | Backend and frontend unit tests |
| `task verify` | Local gate: bindings drift, format check, golangci-lint, Go race tests, property tests, ESLint, svelte-check, Vitest |
| `task go:vendor` | Refresh `./vendor` after Go dependency changes |
| `task generate:bindings` | Regenerate `frontend/bindings` (fix drift with this after Go service changes) |
| `task lint` | Frontend ESLint + golangci-lint |
| `task lint:frontend` / `task lint:go` / `task lint:fix` | Split or autofix lint |
| `task check` | Frontend typecheck (`svelte-check`) |
| `task format` | Write gofmt + Prettier |
| `task fmt` | Write gofmt only |
| `task format:check` | Fail on dirty format (CI gate) |
| `task bench` | Go allocation benchmarks |
| `task test:coverage` | Frontend + Go coverage floors |
| `task lighthouse` | Lighthouse CI on static demo build (all demo-reachable pages) |
| `task showcase` | Capture demo screenshots into `showcase/` |

`task verify` is the local pre-merge gate. Full CI still runs e2e smoke, mutation, coverage floors, Lighthouse, and mobile packaging on top of that.

## Toolchain pins

| Tool | Pin |
|------|-----|
| Go | `go.mod` (1.26.x) |
| Node | `.nvmrc` / mise (`22`) |
| pnpm | `frontend/package.json` `packageManager` (`11.1.2`) |
| Task / golangci-lint / Wails CLI / lefthook | `mise.toml` (optional) |

Install [mise](https://mise.jdx.dev/) and run `mise install` to sync optional CLIs. Without mise, install Go, Node 22, pnpm, and Task from their own docs. Wails CLI matches `go.mod`:

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
task setup
task dev:server
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
