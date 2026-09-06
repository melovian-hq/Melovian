#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Quad4 Software
# SPDX-License-Identifier: Apache-2.0
#
# Verify a mechanical extraction slice.
# Usage:
#   scripts/refactor/check-extraction.sh \
#     --before scripts/refactor/baselines/music.svelte.ts.json \
#     --god frontend/src/lib/config/music.svelte.ts \
#     --dest frontend/src/lib/config/music/settings-ops.ts \
#     --moved updateTranscodingSettings,updateQueueSettings \
#     --max-god-lines 3900

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
INV="$ROOT/scripts/refactor/inventory-methods.mjs"
TMPDIR="${TMPDIR:-/tmp}/melovian-refactor-$$"
mkdir -p "$TMPDIR"
trap 'rm -rf "$TMPDIR"' EXIT

BEFORE=""
GOD=""
DEST=""
MOVED=""
REMAIN=""
MAX_GOD_LINES=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --before) BEFORE="$2"; shift 2 ;;
    --god) GOD="$2"; shift 2 ;;
    --dest) DEST="$2"; shift 2 ;;
    --moved) MOVED="$2"; shift 2 ;;
    --remain) REMAIN="$2"; shift 2 ;;
    --max-god-lines) MAX_GOD_LINES="$2"; shift 2 ;;
    *) echo "Unknown arg: $1" >&2; exit 2 ;;
  esac
done

if [[ -z "$BEFORE" || -z "$GOD" ]]; then
  echo "Need --before and --god" >&2
  exit 2
fi

BEFORE_PATH="$BEFORE"
[[ "$BEFORE" != /* ]] && BEFORE_PATH="$ROOT/$BEFORE"

node "$INV" "$ROOT/$GOD" --out "$TMPDIR/god-after.json"

if [[ -n "$REMAIN" ]]; then
  echo "Checking remain names on god file..."
  node "$INV" --names-present "$TMPDIR/god-after.json" "$REMAIN"
fi

if [[ -n "$DEST" && -n "$MOVED" ]]; then
  echo "Checking moved names in destination..."
  node "$INV" "$ROOT/$DEST" --out "$TMPDIR/dest.json"
  node "$INV" --diff-moved "$BEFORE_PATH" "$TMPDIR/dest.json" "$MOVED"
  echo "Checking wrappers still present on god file..."
  node "$INV" --names-present "$TMPDIR/god-after.json" "$MOVED"
fi

GOD_LINES=$(wc -l < "$ROOT/$GOD" | tr -d ' ')
echo "God file lines now: $GOD_LINES"
if [[ "$MAX_GOD_LINES" -gt 0 && "$GOD_LINES" -gt "$MAX_GOD_LINES" ]]; then
  echo "FAIL: god file has $GOD_LINES lines, expected <= $MAX_GOD_LINES" >&2
  exit 1
fi

echo "OK"
