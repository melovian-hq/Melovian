#!/usr/bin/env bash
# Server-mode development: Go API on MELOVIAN_LISTEN + Vite with /api proxy.
# UI: http://127.0.0.1:${WAILS_VITE_PORT:-9245}
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

LISTEN="${MELOVIAN_LISTEN:-127.0.0.1:17337}"
VITE_PORT="${WAILS_VITE_PORT:-9245}"
API_URL="${MELOVIAN_API_URL:-http://${LISTEN}}"
BIN="${ROOT}/bin/melovian-server"
SERVER_PID=""
VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || cat VERSION)}"

cleanup() {
  if [ -n "${SERVER_PID}" ] && kill -0 "${SERVER_PID}" 2>/dev/null; then
    kill "${SERVER_PID}" 2>/dev/null || true
    wait "${SERVER_PID}" 2>/dev/null || true
  fi
}
trap cleanup EXIT INT TERM

if [ ! -f .env ]; then
  cp .env.example .env
  echo "Created .env from .env.example"
fi

# embed requires frontend/dist. Vite serves the real UI. Keep a stub if missing.
if [ ! -f frontend/dist/index.html ]; then
  mkdir -p frontend/dist
  cat >frontend/dist/index.html <<'EOF'
<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <title>Melovian</title>
  </head>
  <body>
    <p>Open the Vite dev server for the UI (task dev:server).</p>
  </body>
</html>
EOF
  echo "Wrote stub frontend/dist for Go embed"
fi

mkdir -p bin
echo "${VERSION}" > frontend/.build-version
echo "Building server binary (tags=server, version=${VERSION})..."
go build -tags server -ldflags "-X melovian/internal/compat.Version=${VERSION}" -o "${BIN}" .

echo "Starting API on ${LISTEN}..."
MELOVIAN_LISTEN="${LISTEN}" "${BIN}" &
SERVER_PID=$!

# Wait until the listen port accepts connections (up to ~15s).
host="${LISTEN%:*}"
port="${LISTEN##*:}"
ready=0
for _ in $(seq 1 30); do
  if command -v nc >/dev/null 2>&1; then
    if nc -z "${host}" "${port}" 2>/dev/null; then
      ready=1
      break
    fi
  elif (echo >/dev/tcp/"${host}"/"${port}") 2>/dev/null; then
    ready=1
    break
  fi
  if ! kill -0 "${SERVER_PID}" 2>/dev/null; then
    echo "Server exited before becoming ready" >&2
    wait "${SERVER_PID}" || true
    exit 1
  fi
  sleep 0.5
done
if [ "${ready}" != 1 ]; then
  echo "Timed out waiting for ${LISTEN}" >&2
  exit 1
fi

echo "API ready. Vite UI: http://127.0.0.1:${VITE_PORT} (proxy /api -> ${API_URL})"
cd frontend
export MELOVIAN_API_URL="${API_URL}"
export WAILS_VITE_PORT="${VITE_PORT}"
export VITE_APP_VERSION="${VERSION}"
exec pnpm dev --host 127.0.0.1 --port "${VITE_PORT}" --strictPort
