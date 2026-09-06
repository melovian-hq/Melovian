#!/usr/bin/env python3
"""Keep generated Android overlay.json free of GOMODCACHE replacements.

Go rejects -overlay entries under GOMODCACHE. The androidMainOnce patch is
applied by prepare-wails-mod.sh via a local module replace instead.
"""

from __future__ import annotations

import json
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]


def main() -> None:
    overlay_path = ROOT / "build/android/overlay.json"
    if not overlay_path.is_file():
        raise SystemExit(f"missing {overlay_path}")
    data = json.loads(overlay_path.read_text())
    replace = data.get("Replace") or {}
    data["Replace"] = {
        src: dst
        for src, dst in replace.items()
        if "/pkg/mod/" not in src.replace("\\", "/")
    }
    overlay_path.write_text(json.dumps(data, indent=2) + "\n")


if __name__ == "__main__":
    main()
