#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Quad4 Software
# SPDX-License-Identifier: Apache-2.0
#
# Package-level Go coverage floors for high-value pure logic.
# Thresholds sit just under current coverage so CI catches regressions.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

packages=(
  ./internal/cache
  ./internal/httputil
  ./internal/lyrics
  ./internal/localmusic
  ./internal/subsonic
)

# package import path -> minimum statement coverage percent
declare -A mins=(
  [melovian/internal/cache]=85
  [melovian/internal/httputil]=50
  [melovian/internal/lyrics]=65
  [melovian/internal/localmusic]=70
  [melovian/internal/subsonic]=50
)

failed=0

echo "Checking Go coverage floors..."
for pkg_path in "${packages[@]}"; do
  pkg_name="melovian/${pkg_path#./}"
  min="${mins[$pkg_name]}"
  profile="$(mktemp)"
  go test -count=1 -coverprofile="$profile" "$pkg_path" >/dev/null
  pct="$(go tool cover -func="$profile" | awk '/^total:/ { print $NF }')"
  rm -f "$profile"
  num="${pct%\%}"
  if awk -v n="$num" -v m="$min" 'BEGIN { exit !(n + 0 < m + 0) }'; then
    echo "FAIL  $pkg_name coverage ${num}% below floor ${min}%" >&2
    failed=1
  else
    echo "OK    $pkg_name coverage ${num}% (floor ${min}%)"
  fi
done

if [ "$failed" -ne 0 ]; then
  echo "Go coverage floors not met" >&2
  exit 1
fi
echo "All Go coverage floors met"
