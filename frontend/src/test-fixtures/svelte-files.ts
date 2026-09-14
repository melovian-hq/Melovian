// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { readdirSync, statSync } from "node:fs";
import { join } from "node:path";

/**
 * Recursively lists .svelte files under dir, skipping node_modules and dist.
 * Shared by hygiene tests that scan the component tree.
 */
export function listSvelteFiles(dir: string, out: string[] = []): string[] {
  for (const name of readdirSync(dir)) {
    const path = join(dir, name);
    const st = statSync(path);
    if (st.isDirectory()) {
      if (name === "node_modules" || name === "dist") continue;
      listSvelteFiles(path, out);
      continue;
    }
    if (name.endsWith(".svelte")) out.push(path);
  }
  return out;
}
