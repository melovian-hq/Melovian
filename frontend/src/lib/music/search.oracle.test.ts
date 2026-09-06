// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import fc from "fast-check";
import { bestMatches, diceCoefficient, normalizeTerm } from "./search";

describe("search oracle", () => {
  it("exact normalized equals always score 1", () => {
    fc.assert(
      fc.property(fc.string({ maxLength: 40 }), (value) => {
        const normalized = normalizeTerm(value);
        expect(diceCoefficient(normalized, normalized)).toBe(1);
        expect(diceCoefficient(value, value)).toBe(1);
      }),
    );
  });

  it("empty query yields no matches", () => {
    fc.assert(
      fc.property(
        fc.array(fc.string({ maxLength: 20 }), { maxLength: 20 }),
        (candidates) => {
          expect(
            bestMatches("   ", candidates, (item) => item, { threshold: 0 }),
          ).toEqual([]);
          expect(
            bestMatches("\t", candidates, (item) => item, { threshold: 0 }),
          ).toEqual([]);
        },
      ),
    );
  });

  it("self candidate is the top match when threshold is zero", () => {
    fc.assert(
      fc.property(
        fc.string({ minLength: 2, maxLength: 20 }),
        fc.array(fc.string({ maxLength: 16 }), { maxLength: 10 }),
        (query, noise) => {
          fc.pre(normalizeTerm(query) !== "");
          const candidates = [query, ...noise];
          const matches = bestMatches(query, candidates, (item) => item, {
            limit: 5,
            threshold: 0,
          });
          expect(matches[0]?.item).toBe(query);
          expect(matches[0]?.score).toBe(1);
        },
      ),
    );
  });
});
