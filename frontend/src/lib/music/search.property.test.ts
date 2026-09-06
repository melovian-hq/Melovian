// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import fc from "fast-check";
import { bestMatches, diceCoefficient, normalizeTerm } from "./search";

describe("normalizeTerm properties", () => {
  it("is idempotent", () => {
    fc.assert(
      fc.property(fc.string(), (value) => {
        const once = normalizeTerm(value);
        expect(normalizeTerm(once)).toBe(once);
      }),
    );
  });

  it("never contains uppercase letters", () => {
    fc.assert(
      fc.property(fc.string(), (value) => {
        const normalized = normalizeTerm(value);
        expect(normalized).toBe(normalized.toLowerCase());
      }),
    );
  });
});

describe("diceCoefficient properties", () => {
  it("is symmetric and bounded", () => {
    fc.assert(
      fc.property(fc.string(), fc.string(), (a, b) => {
        const scoreAB = diceCoefficient(a, b);
        const scoreBA = diceCoefficient(b, a);
        expect(scoreAB).toBe(scoreBA);
        expect(scoreAB).toBeGreaterThanOrEqual(0);
        expect(scoreAB).toBeLessThanOrEqual(1);
      }),
    );
  });

  it("returns 1 for identical normalized inputs", () => {
    fc.assert(
      fc.property(fc.string(), (value) => {
        expect(diceCoefficient(value, value)).toBe(1);
      }),
    );
  });
});

describe("bestMatches properties", () => {
  it("respects limit and never exceeds candidate count", () => {
    fc.assert(
      fc.property(
        fc.string(),
        fc.array(fc.string(), { maxLength: 20 }),
        fc.integer({ min: 1, max: 10 }),
        (query, candidates, limit) => {
          const matches = bestMatches(query, candidates, (item) => item, {
            limit,
            threshold: 0,
          });
          expect(matches.length).toBeLessThanOrEqual(
            Math.min(limit, candidates.length),
          );
          for (let i = 1; i < matches.length; i++) {
            expect(matches[i - 1].score).toBeGreaterThanOrEqual(
              matches[i].score,
            );
          }
        },
      ),
    );
  });
});
