#!/usr/bin/env bash
# Post-process generated Wails bindings. Inserts a DeepSource skipcq comment
# above each generated namespace import so the generated code is not flagged
# as introduced issues in PR analysis. Called by task generate:bindings and
# check-bindings-drift.sh so committed and regenerated output stay identical.
set -euo pipefail

root="$(cd "$(dirname "$0")/../.." && pwd)"

find "$root/frontend/bindings" -name '*.ts' -print0 2>/dev/null | while IFS= read -r -d '' file; do
  if grep -q 'skipcq' "$file"; then
    continue
  fi
  sed -i.bak '/^import \* as /i\
// skipcq: JS-C1003' "$file"
  rm -f "$file.bak"
done
