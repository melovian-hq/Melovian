#!/usr/bin/env bash
# Fail if first-party Go sources are not gofmt-clean.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

unformatted="$(
  find . \
    \( -path ./vendor -o -path ./frontend -o -path ./build/android/wails-v3 -o -path ./build/tools -o -path ./.git -o -path ./bin \) -prune \
    -o -name '*.go' -print \
    | sort \
    | xargs gofmt -l
)"

if [ -n "${unformatted}" ]; then
  echo "gofmt needed on:" >&2
  echo "${unformatted}" >&2
  echo "Run: task fmt" >&2
  exit 1
fi

echo "Go sources are gofmt-clean"
