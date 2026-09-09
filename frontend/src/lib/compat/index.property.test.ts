// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { beforeEach, describe, expect, it } from "vitest";
import fc from "fast-check";
import {
  CLIENT_CAPABILITIES,
  CLIENT_VERSION,
  API_VERSION,
  applyCompatFromConfig,
  compareSemver,
  resetCompatForTests,
  supports,
  versionScheme,
  versionsEqual,
} from "./index";

const semverArb = fc
  .tuple(
    fc.integer({ min: 0, max: 99 }),
    fc.integer({ min: 0, max: 99 }),
    fc.integer({ min: 0, max: 99 }),
  )
  .map(([major, minor, patch]) => `${major}.${minor}.${patch}`);

const versionArb = fc.oneof(
  semverArb,
  semverArb.map((v) => `v${v}`),
  semverArb.map((v) => `${v}-rc.1`),
  semverArb.map((v) => `${v}+build.1`),
  fc.string({ minLength: 1, maxLength: 16 }),
);

describe("compat property tests", () => {
  beforeEach(() => {
    resetCompatForTests();
  });

  it("compareSemver is antisymmetric and in range", () => {
    fc.assert(
      fc.property(versionArb, versionArb, (a, b) => {
        const got = compareSemver(a, b);
        expect(got).toBeGreaterThanOrEqual(-1);
        expect(got).toBeLessThanOrEqual(1);
        const swapped = compareSemver(b, a);
        expect(got + swapped).toBe(0);
      }),
      { numRuns: 200 },
    );
  });

  it("compareSemver ignores v/V prefix, build and prerelease metadata", () => {
    fc.assert(
      fc.property(semverArb, (v) => {
        const bare = compareSemver(v, "0.0.0");
        const withV = compareSemver(`v${v}`, "0.0.0");
        const withVUpper = compareSemver(`V${v}`, "0.0.0");
        const withBuild = compareSemver(`${v}+build.1`, "0.0.0");
        const withPre = compareSemver(`${v}-rc.1`, "0.0.0");
        expect(withV).toBe(bare);
        expect(withVUpper).toBe(bare);
        expect(withBuild).toBe(bare);
        expect(withPre).toBe(bare);
      }),
      { numRuns: 120 },
    );
  });

  it("versionsEqual is an equivalence relation", () => {
    fc.assert(
      fc.property(versionArb, (v) => {
        expect(versionsEqual(v, v)).toBe(true);
      }),
      { numRuns: 120 },
    );
    fc.assert(
      fc.property(semverArb, (v) => {
        expect(versionsEqual(v, `v${v}`)).toBe(true);
      }),
      { numRuns: 120 },
    );
  });

  it("versionScheme is stable under trim and optional leading v", () => {
    fc.assert(
      fc.property(fc.string({ minLength: 1, maxLength: 24 }), (v) => {
        const s = versionScheme(v);
        expect(["git", "semver", "other"]).toContain(s);
      }),
      { numRuns: 200 },
    );
    fc.assert(
      fc.property(semverArb, (v) => {
        expect(versionScheme(`v${v}`)).toBe(versionScheme(v));
        expect(versionScheme(`V${v}`)).toBe(versionScheme(v));
        expect(versionScheme(v.trim())).toBe(versionScheme(v));
      }),
      { numRuns: 120 },
    );
  });

  it("applyCompatFromConfig intersects server capabilities with client ones", () => {
    fc.assert(
      fc.property(
        fc.array(fc.constantFrom(...CLIENT_CAPABILITIES), {
          minLength: 0,
          maxLength: 10,
        }),
        (serverCaps) => {
          const state = applyCompatFromConfig({
            version: CLIENT_VERSION,
            apiVersion: API_VERSION,
            capabilities: serverCaps,
          });
          for (const cap of state.supported) {
            expect(CLIENT_CAPABILITIES).toContain(cap);
            expect(serverCaps).toContain(cap);
          }
          for (const cap of CLIENT_CAPABILITIES) {
            const expected =
              CLIENT_CAPABILITIES.includes(cap) && serverCaps.includes(cap);
            expect(supports(cap)).toBe(expected);
          }
        },
      ),
      { numRuns: 120 },
    );
  });
});
