// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import { readdirSync, readFileSync, statSync } from "node:fs";
import { join } from "node:path";

function listSvelteFiles(dir: string, out: string[] = []): string[] {
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

describe("svelte markup hygiene", () => {
  it("does not put // SPDX comments before <script> (they render as text)", () => {
    const root = join(import.meta.dirname, "../..");
    const files = listSvelteFiles(root);
    expect(files.length).toBeGreaterThan(20);

    const offenders: string[] = [];
    for (const file of files) {
      const source = readFileSync(file, "utf8");
      const scriptIdx = source.search(/<script\b/);
      if (scriptIdx < 0) continue;
      const before = source.slice(0, scriptIdx);
      if (/\/\/\s*SPDX|\/\/\s*Copyright/i.test(before)) {
        offenders.push(file.replace(root + "/", ""));
      }
    }
    expect(offenders).toEqual([]);
  });
});
