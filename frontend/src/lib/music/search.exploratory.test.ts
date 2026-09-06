// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import fc from "fast-check";
import { bestMatches, diceCoefficient, normalizeTerm } from "./search";

describe("search exploratory", () => {
  it("explores normalizeTerm never throws on unicode noise", () => {
    fc.assert(
      fc.property(fc.string({ maxLength: 80 }), (value) => {
        const normalized = normalizeTerm(value);
        expect(typeof normalized).toBe("string");
        expect(normalized).toBe(normalized.toLowerCase());
        expect(normalizeTerm(normalized)).toBe(normalized);
      }),
      { numRuns: 200 },
    );
  });

  it("explores diceCoefficient under adversarial pairs", () => {
    fc.assert(
      fc.property(
        fc.string({ maxLength: 40 }),
        fc.string({ maxLength: 40 }),
        (a, b) => {
          const score = diceCoefficient(a, b);
          expect(score).toBeGreaterThanOrEqual(0);
          expect(score).toBeLessThanOrEqual(1);
          expect(diceCoefficient(a, a)).toBe(1);
        },
      ),
      { numRuns: 150 },
    );
  });

  it("explores bestMatches ordering under random catalogs", () => {
    fc.assert(
      fc.property(
        fc.string({ minLength: 1, maxLength: 20 }),
        fc.array(fc.string({ maxLength: 24 }), { maxLength: 30 }),
        fc.integer({ min: 1, max: 8 }),
        (query, candidates, limit) => {
          const matches = bestMatches(query, candidates, (item) => item, {
            limit,
            threshold: 0,
          });
          expect(matches.length).toBeLessThanOrEqual(
            Math.min(limit, candidates.length),
          );
          for (let i = 1; i < matches.length; i++) {
            expect(matches[i - 1]!.score).toBeGreaterThanOrEqual(
              matches[i]!.score,
            );
          }
        },
      ),
      { numRuns: 100 },
    );
  });
});
