#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

if ! command -v wails3 >/dev/null 2>&1; then
  echo "wails3 is required for bindings drift detection" >&2
  exit 1
fi

before="$(mktemp)"
after="$(mktemp)"
trap 'rm -f "$before" "$after"' EXIT

if [ -d frontend/bindings ]; then
  find frontend/bindings -type f | sort | while read -r file; do
    printf '%s\0' "$file"
    sha256sum "$file"
  done >"$before"
fi

wails3 generate bindings -clean=true -ts
bash build/scripts/bindings-postprocess.sh

if [ -d frontend/bindings ]; then
  find frontend/bindings -type f | sort | while read -r file; do
    printf '%s\0' "$file"
    sha256sum "$file"
  done >"$after"
fi

if ! diff -u "$before" "$after"; then
  echo "Bindings drift detected. Run: task generate:bindings" >&2
  git -C "$ROOT" --no-pager diff -- frontend/bindings >&2 || true
  exit 1
fi

echo "Bindings are up to date"
