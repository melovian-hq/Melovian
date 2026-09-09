// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { beforeEach, describe, expect, it } from "vitest";
import { compareSemver, resetCompatForTests } from "./index";

describe("compat mutation tests", () => {
  beforeEach(() => {
    resetCompatForTests();
  });

  const propertyCases = [
    { a: "1.0.0", b: "1.0.0", want: 0 },
    { a: "1.0.0", b: "1.0.1", want: -1 },
    { a: "1.10.0", b: "1.2.0", want: 1 },
    { a: "v1.0.0", b: "1.0.0", want: 0 },
    { a: "V1.0.0", b: "1.0.0", want: 0 },
    { a: "1.0.0-rc.1", b: "1.0.0", want: 0 }, // compat ignores prerelease
    { a: "1.0.0+build.1", b: "1.0.0", want: 0 }, // compat ignores build
  ];

  function mutantAlwaysEqual(a: string, b: string): number {
    return 0;
  }

  function mutantNoVPrefix(a: string, b: string): number {
    // Strips only lowercase v and does not pad numeric parts.
    const na = a.replace(/^v/, "").trim();
    const nb = b.replace(/^v/, "").trim();
    if (na < nb) return -1;
    if (na > nb) return 1;
    return 0;
  }

  function mutantIgnoreBuildAndPreOnly(a: string, b: string): number {
    // Removes build and pre, but also removes the core patch if it starts with 0.
    const strip = (v: string) =>
      v.replace(/^v/i, "").split(/[-+]/)[0].replace(/\.0+$/, "");
    const na = strip(a);
    const nb = strip(b);
    if (na < nb) return -1;
    if (na > nb) return 1;
    return 0;
  }

  const mutants = [
    { name: "always_equal", fn: mutantAlwaysEqual },
    { name: "no_v_prefix", fn: mutantNoVPrefix },
    { name: "strip_zeros_wrongly", fn: mutantIgnoreBuildAndPreOnly },
  ];

  it.each(mutants)("kills compareSemver mutant $name", ({ fn }) => {
    for (const c of propertyCases) {
      if (fn(c.a, c.b) !== c.want) {
        return;
      }
    }
    throw new Error(`mutant survived the property suite`);
  });

  it("production compareSemver passes the property suite", () => {
    for (const c of propertyCases) {
      expect(compareSemver(c.a, c.b)).toBe(c.want);
    }
  });
});
