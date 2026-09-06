#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Quad4 Software
# SPDX-License-Identifier: Apache-2.0
#
# Lightweight Go mutation harness for pure helpers.
# Applies known mutants, expects package tests to fail, then restores sources.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

tmpdir="$(mktemp -d)"
active_file=""
active_backup=""

restore_active() {
  if [ -n "${active_file}" ] && [ -n "${active_backup}" ] && [ -f "${active_backup}" ]; then
    cp "${active_backup}" "${active_file}"
  fi
  active_file=""
  active_backup=""
}

cleanup() {
  restore_active
  rm -rf "$tmpdir"
}

trap cleanup EXIT

kill_count=0
live_count=0
skip_count=0

apply_and_test() {
  local file="$1"
  local description="$2"
  local search="$3"
  local replace="$4"
  local packages="$5"
  local backup

  if ! grep -Fq -- "$search" "$file"; then
    echo "SKIP  $description (pattern not found in $file)"
    skip_count=$((skip_count + 1))
    return
  fi

  backup="$tmpdir/$(echo "$file" | tr '/' '_').orig"
  cp "$file" "$backup"
  active_file="$file"
  active_backup="$backup"

  python3 - "$file" "$search" "$replace" <<'PY'
import pathlib, sys
path, search, replace = sys.argv[1], sys.argv[2], sys.argv[3]
text = pathlib.Path(path).read_text()
if search not in text:
    raise SystemExit(f"search not found: {search!r}")
pathlib.Path(path).write_text(text.replace(search, replace, 1))
PY

  echo -n "MUTANT $description ... "
  if go test $packages -count=1 >/dev/null 2>"$tmpdir/test.err"; then
    echo "LIVED"
    live_count=$((live_count + 1))
  else
    echo "KILLED"
    kill_count=$((kill_count + 1))
  fi

  restore_active
}

echo "Running Go mutation harness..."

apply_and_test \
  internal/httputil/json.go \
  "QueryInt empty-check inversion" \
  'if value == "" {' \
  'if value != "" {' \
  "./internal/httputil"

apply_and_test \
  internal/httputil/json.go \
  "QueryInt atoi error inversion" \
  'if err != nil {' \
  'if err == nil {' \
  "./internal/httputil"

apply_and_test \
  internal/lyrics/validate.go \
  "NormalizeDocument nil check inversion" \
  'if doc == nil {' \
  'if doc != nil {' \
  "./internal/lyrics"

apply_and_test \
  internal/lyrics/validate.go \
  "NormalizeDocument empty-lines check inversion" \
  'if len(lines) == 0 {' \
  'if len(lines) != 0 {' \
  "./internal/lyrics"

apply_and_test \
  internal/cache/cache.go \
  "ResponseCache oversized body check inversion" \
  'if len(entry.Body) > MaxBodyBytes {' \
  'if len(entry.Body) < MaxBodyBytes {' \
  "./internal/cache"

apply_and_test \
  internal/cache/cache.go \
  "ResponseCache get-miss existence inversion" \
  'if !ok {' \
  'if ok {' \
  "./internal/cache"

total=$((kill_count + live_count))
echo
echo "Killed: $kill_count  Lived: $live_count  Skipped: $skip_count"
if [ "$total" -eq 0 ]; then
  echo "No mutants executed" >&2
  exit 1
fi
if [ "$live_count" -gt 0 ]; then
  echo "Mutation harness found surviving mutants" >&2
  exit 1
fi
echo "All executed mutants killed"
