// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { readFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";
import {
  API_VERSION,
  Cap,
  HEADER_API_VERSION,
  HEADER_CAPABILITIES,
  HEADER_CLIENT_VERSION,
  HEADER_SERVER_VERSION,
  MIN_SERVER_VERSION,
} from "./index";

const repoRoot = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "../../../../",
);
const goSource = readFileSync(
  path.join(repoRoot, "internal/compat/compat.go"),
  "utf8",
);

const goStringConstRE = /\b((?:Header|Cap|Min|API)\w*)\s*=\s*"([^"]*)"/g;
const goNumberConstRE = /\b((?:Header|Cap|Min|API)\w*)\s*=\s*(\d+)/g;

function parseGoContractConsts(source: string): Map<string, string> {
  const consts = new Map<string, string>();
  for (const match of source.matchAll(goStringConstRE)) {
    consts.set(match[1], match[2]);
  }
  for (const match of source.matchAll(goNumberConstRE)) {
    consts.set(match[1], match[2]);
  }
  return consts;
}

// capKey converts a Go Cap constant name to the Cap key used in index.ts.
// CapPersonalRadio becomes personalRadio and all caps names like CapWS become ws.
function capKey(goName: string): string {
  const rest = goName.slice("Cap".length);
  if (!rest) return goName;
  if (rest === rest.toUpperCase()) return rest.toLowerCase();
  return rest.charAt(0).toLowerCase() + rest.slice(1);
}

const goConsts = parseGoContractConsts(goSource);

describe("compat contract sync", () => {
  it("keeps header and version constants in sync", () => {
    // MinClientVersion is enforced server side only and has no index.ts export.
    const pairs: Array<[string, string]> = [
      ["HeaderClientVersion", HEADER_CLIENT_VERSION],
      ["HeaderAPIVersion", HEADER_API_VERSION],
      ["HeaderCapabilities", HEADER_CAPABILITIES],
      ["HeaderServerVersion", HEADER_SERVER_VERSION],
      ["MinServerVersion", MIN_SERVER_VERSION],
      ["APIVersion", String(API_VERSION)],
    ];
    const expectedGoNames = new Set(pairs.map(([goName]) => goName));

    const drift: string[] = [];
    for (const [goName, tsValue] of pairs) {
      const goValue = goConsts.get(goName);
      if (goValue === undefined) {
        drift.push(`${goName} is missing from compat.go`);
      } else if (goValue !== tsValue) {
        drift.push(
          `${goName}: compat.go=${JSON.stringify(goValue)} index.ts=${JSON.stringify(tsValue)}`,
        );
      }
    }
    for (const name of goConsts.keys()) {
      if (!name.startsWith("Header") || expectedGoNames.has(name)) continue;
      drift.push(`compat.go exports ${name} with no index.ts counterpart`);
    }
    expect(drift, drift.join("\n")).toEqual([]);
  });

  it("keeps capability constants in sync", () => {
    const tsCaps = new Map(Object.entries(Cap));
    const goCapKeys = new Map<string, string>();
    for (const [goName, goValue] of goConsts) {
      if (goName.startsWith("Cap")) {
        goCapKeys.set(capKey(goName), goValue);
      }
    }

    const drift: string[] = [];
    for (const [key, goValue] of goCapKeys) {
      const tsValue = tsCaps.get(key);
      if (tsValue === undefined) {
        drift.push(
          `Cap.${key} (compat.go ${JSON.stringify(goValue)}) is missing from index.ts`,
        );
      } else if (tsValue !== goValue) {
        drift.push(
          `Cap.${key}: compat.go=${JSON.stringify(goValue)} index.ts=${JSON.stringify(tsValue)}`,
        );
      }
    }
    for (const key of tsCaps.keys()) {
      if (!goCapKeys.has(key)) {
        drift.push(`index.ts Cap.${key} has no compat.go counterpart`);
      }
    }
    expect(drift, drift.join("\n")).toEqual([]);
  });
});
