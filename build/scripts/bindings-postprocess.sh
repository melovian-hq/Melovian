#!/usr/bin/env bash
# Post-process generated Wails bindings. Adds a DeepSource skipcq comment so
# the generated namespace imports are not flagged as introduced issues in PR
# analysis. Called by task generate:bindings and check-bindings-drift.sh so
# committed and regenerated output stay identical.
set -euo pipefail

root="$(cd "$(dirname "$0")/../.." && pwd)"

find "$root/frontend/bindings" -name '*.ts' -print0 2>/dev/null | while IFS= read -r -d '' file; do
  if ! grep -q 'skipcq' "$file"; then
    sed -i '1i // skipcq: JS-C1003' "$file"
  fi
done
