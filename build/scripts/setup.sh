#!/usr/bin/env bash
# One-shot contributor bootstrap: .env, frontend deps, optional tool hints.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

hint() {
  printf '  - %s\n' "$1"
}

echo "==> Melovian setup"

if [ ! -f .env ]; then
  cp .env.example .env
  echo "Created .env from .env.example"
else
  echo ".env already present"
fi

if ! command -v pnpm >/dev/null 2>&1; then
  echo "pnpm is required (pinned in frontend/package.json as 11.1.2)" >&2
  exit 1
fi

echo "Installing frontend dependencies..."
(cd frontend && pnpm install)

if command -v lefthook >/dev/null 2>&1; then
  lefthook install
  echo "Installed git hooks via lefthook"
else
  echo "lefthook not found (optional). Install and run: lefthook install"
fi

echo
echo "Toolchain status:"
if command -v go >/dev/null 2>&1; then
  hint "go $(go version | awk '{print $3}')"
else
  hint "go missing (need 1.26+ from go.mod)"
fi
if command -v node >/dev/null 2>&1; then
  hint "node $(node -v) (want .nvmrc / mise pin 22)"
else
  hint "node missing (want 22+)"
fi
hint "pnpm $(pnpm -v)"
if command -v task >/dev/null 2>&1; then
  hint "task $(task --version 2>/dev/null | head -1)"
else
  hint "task missing (optional). https://taskfile.dev"
fi
if command -v wails3 >/dev/null 2>&1; then
  hint "wails3 present (desktop / bindings)"
else
  hint "wails3 missing (desktop + bindings). go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.16"
fi
if command -v golangci-lint >/dev/null 2>&1; then
  hint "golangci-lint present"
else
  hint "golangci-lint missing (optional for task lint:go)"
fi
if command -v mise >/dev/null 2>&1; then
  hint "mise present (mise install to sync mise.toml pins)"
fi

echo
echo "Next:"
hint "task --list"
hint "task dev          # desktop + Vite"
hint "task dev:server   # HTTP API + Vite (no Wails GUI)"
hint "task verify       # local pre-merge gate"
